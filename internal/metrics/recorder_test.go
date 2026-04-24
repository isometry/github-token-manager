package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNilRecorderSafety(t *testing.T) {
	var r *Recorder
	ctx := context.Background()

	// All methods must be callable on a nil receiver without panic.
	r.RecordTokenRefresh(ctx, "github-token", ResultSuccess)
	r.RecordTokenRefreshDuration(ctx, "github-token", OperationCreate, time.Second)
	r.RecordGitHubAPICall(ctx, "github-token", time.Second, nil)
	r.RecordGitHubAPICall(ctx, "github-token", time.Second, errors.New("test"))
	r.RecordTokenExpiry(ctx, "github-token", "default", "my-token", time.Now())
	r.RecordReconcileError(ctx, "github-token", ReasonTransient)
	r.EnsureTokenActive(ctx, "github-token", "default/my-token")
	r.RemoveTokenActive(ctx, "github-token", "default/my-token")
	r.RecordSecretOperation(ctx, "github-token", OperationCreate, ResultSuccess)
	r.RecordConfigError(ctx, "github-token", "file")
	if err := r.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown on nil receiver returned error: %v", err)
	}
}

func TestRecorderInstruments(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	r, err := newRecorder(meter)
	if err != nil {
		t.Fatalf("newRecorder: %v", err)
	}

	ctx := context.Background()

	// Record some values.
	r.RecordTokenRefresh(ctx, "github-token", ResultSuccess)
	r.RecordTokenRefresh(ctx, "github-token", ResultSuccess)
	r.RecordTokenRefresh(ctx, "github-clustertoken", ResultError)
	r.RecordTokenRefreshDuration(ctx, "github-token", OperationCreate, 500*time.Millisecond)
	r.RecordGitHubAPICall(ctx, "github-token", 200*time.Millisecond, nil)
	r.RecordGitHubAPICall(ctx, "github-token", 300*time.Millisecond, errors.New("rate limit"))
	r.RecordTokenExpiry(ctx, "github-token", "default", "my-token", time.Unix(1700000000, 0))
	r.RecordReconcileError(ctx, "github-token", ReasonGitHubAPI)
	r.EnsureTokenActive(ctx, "github-token", "default/my-token")
	r.RecordSecretOperation(ctx, "github-token", OperationCreate, ResultSuccess)
	r.RecordConfigError(ctx, "github-app", "app")

	// Collect and verify.
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	metrics := flattenMetrics(rm)

	// Verify token refresh counter.
	assertCounterValue(t, metrics, "token.refresh",
		attribute.String("controller", "github-token"),
		attribute.String("result", ResultSuccess),
		2,
	)
	assertCounterValue(t, metrics, "token.refresh",
		attribute.String("controller", "github-clustertoken"),
		attribute.String("result", ResultError),
		1,
	)

	// Verify reconcile errors counter.
	assertCounterValue(t, metrics, "token.reconcile.errors",
		attribute.String("controller", "github-token"),
		attribute.String("reason", ReasonGitHubAPI),
		1,
	)

	// Verify secret operations counter.
	assertCounterValue(t, metrics, "kubernetes.secret.operations",
		attribute.String("controller", "github-token"),
		attribute.String("operation", OperationCreate),
		attribute.String("result", ResultSuccess),
		1,
	)

	// Verify config errors counter.
	assertCounterValue(t, metrics, "config.errors",
		attribute.String("controller", "github-app"),
		attribute.String("source", "app"),
		1,
	)

	// Verify tokens active up-down counter.
	assertCounterValue(t, metrics, "tokens.active",
		attribute.String("controller", "github-token"),
		1,
	)

	// Verify histogram has data points.
	assertHistogramCount(t, metrics, "token.refresh.duration", 1)
	assertHistogramCount(t, metrics, "github.api.call.duration", 2)

	// Verify github.api.requests counter ticks alongside the duration histogram.
	assertCounterValue(t, metrics, "github.api.requests",
		attribute.String("controller", "github-token"),
		attribute.String("result", ResultSuccess),
		1,
	)
	assertCounterValue(t, metrics, "github.api.requests",
		attribute.String("controller", "github-token"),
		attribute.String("result", ResultError),
		1,
	)

	// Verify gauge value.
	assertGaugeValue(t, metrics, "token.expiry.timestamp", 1700000000)
}

func TestActiveTokenIdempotency(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter("test")

	r, err := newRecorder(meter)
	if err != nil {
		t.Fatalf("newRecorder: %v", err)
	}

	ctx := context.Background()

	// First call increments.
	r.EnsureTokenActive(ctx, "github-token", "default/tok-a")
	assertActiveCount(t, reader, ctx, 1)

	// Duplicate call is a no-op.
	r.EnsureTokenActive(ctx, "github-token", "default/tok-a")
	assertActiveCount(t, reader, ctx, 1)

	// Second distinct key increments again.
	r.EnsureTokenActive(ctx, "github-token", "default/tok-b")
	assertActiveCount(t, reader, ctx, 2)

	// Remove one key.
	r.RemoveTokenActive(ctx, "github-token", "default/tok-a")
	assertActiveCount(t, reader, ctx, 1)

	// Duplicate remove is a no-op.
	r.RemoveTokenActive(ctx, "github-token", "default/tok-a")
	assertActiveCount(t, reader, ctx, 1)

	// Remove the other key.
	r.RemoveTokenActive(ctx, "github-token", "default/tok-b")
	assertActiveCount(t, reader, ctx, 0)
}

func assertActiveCount(t *testing.T, reader *metric.ManualReader, ctx context.Context, expected int64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	metrics := flattenMetrics(rm)
	m, ok := metrics["tokens.active"]
	if !ok {
		t.Fatal("metric tokens.active not found")
	}
	data, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", m.Data)
	}
	var total int64
	for _, dp := range data.DataPoints {
		total += dp.Value
	}
	if total != expected {
		t.Errorf("tokens.active: got %d, want %d", total, expected)
	}
}

// flattenMetrics returns a map of metric name -> metricdata.Metrics.
func flattenMetrics(rm metricdata.ResourceMetrics) map[string]metricdata.Metrics {
	result := make(map[string]metricdata.Metrics)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			result[m.Name] = m
		}
	}
	return result
}

func assertCounterValue(t *testing.T, metrics map[string]metricdata.Metrics, name string, args ...any) {
	t.Helper()

	// Last arg is the expected value.
	expected := args[len(args)-1].(int)
	attrs := make([]attribute.KeyValue, 0, len(args)-1)
	for _, a := range args[:len(args)-1] {
		attrs = append(attrs, a.(attribute.KeyValue))
	}

	m, ok := metrics[name]
	if !ok {
		t.Errorf("metric %q not found", name)
		return
	}

	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		for _, dp := range data.DataPoints {
			if hasAttributes(dp.Attributes, attrs) {
				if dp.Value != int64(expected) {
					t.Errorf("metric %q: got %d, want %d", name, dp.Value, expected)
				}
				return
			}
		}
		t.Errorf("metric %q: no data point with matching attributes %v", name, attrs)
	default:
		t.Errorf("metric %q: unexpected data type %T", name, m.Data)
	}
}

func assertHistogramCount(t *testing.T, metrics map[string]metricdata.Metrics, name string, minCount uint64) {
	t.Helper()

	m, ok := metrics[name]
	if !ok {
		t.Errorf("metric %q not found", name)
		return
	}

	data, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Errorf("metric %q: expected Histogram, got %T", name, m.Data)
		return
	}

	var totalCount uint64
	for _, dp := range data.DataPoints {
		totalCount += dp.Count
	}
	if totalCount < minCount {
		t.Errorf("metric %q: got count %d, want >= %d", name, totalCount, minCount)
	}
}

func assertGaugeValue(t *testing.T, metrics map[string]metricdata.Metrics, name string, expected float64) {
	t.Helper()

	m, ok := metrics[name]
	if !ok {
		t.Errorf("metric %q not found", name)
		return
	}

	data, ok := m.Data.(metricdata.Gauge[float64])
	if !ok {
		t.Errorf("metric %q: expected Gauge[float64], got %T", name, m.Data)
		return
	}

	if len(data.DataPoints) == 0 {
		t.Errorf("metric %q: no data points", name)
		return
	}

	if data.DataPoints[0].Value != expected {
		t.Errorf("metric %q: got %f, want %f", name, data.DataPoints[0].Value, expected)
	}
}

func hasAttributes(set attribute.Set, want []attribute.KeyValue) bool {
	for _, kv := range want {
		v, ok := set.Value(kv.Key)
		if !ok || v != kv.Value {
			return false
		}
	}
	return true
}
