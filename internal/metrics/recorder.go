package metrics

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Attribute value constants for result and operation labels.
const (
	ResultSuccess = "success"
	ResultError   = "error"

	OperationCreate = "create"
	OperationUpdate = "update"
	OperationDelete = "delete"

	ReasonTransient    = "transient"
	ReasonGitHubAPI    = "github_api"
	ReasonConfig       = "config"
	ReasonOwnership    = "ownership"
	ReasonSecretCreate = "secret_create"
	ReasonSecretUpdate = "secret_update"
	ReasonStatusUpdate = "status_update"
)

// Recorder holds all custom OTEL metric instruments for the operator.
// All recording methods are nil-receiver safe.
type Recorder struct {
	provider *sdkmetric.MeterProvider

	tokenRefresh         metric.Int64Counter
	tokenRefreshDuration metric.Float64Histogram
	githubAPIDuration    metric.Float64Histogram
	githubAPIRequests    metric.Int64Counter
	tokenExpiry          metric.Float64Gauge
	reconcileErrors      metric.Int64Counter
	tokensActive         metric.Int64UpDownCounter
	secretOperations     metric.Int64Counter
	configErrors         metric.Int64Counter

	activeTokens sync.Map
}

// Shutdown shuts down the underlying MeterProvider, flushing any remaining data.
// It is nil-receiver safe.
func (r *Recorder) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	return r.provider.Shutdown(ctx)
}

func newRecorder(meter metric.Meter) (*Recorder, error) {
	var r Recorder
	var err error

	if r.tokenRefresh, err = meter.Int64Counter("token.refresh",
		metric.WithUnit("{refresh}"),
		metric.WithDescription("Total number of token refresh operations"),
	); err != nil {
		return nil, err
	}

	if r.tokenRefreshDuration, err = meter.Float64Histogram("token.refresh.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of token refresh operations"),
	); err != nil {
		return nil, err
	}

	if r.githubAPIDuration, err = meter.Float64Histogram("github.api.call.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of GitHub API calls"),
	); err != nil {
		return nil, err
	}

	if r.githubAPIRequests, err = meter.Int64Counter("github.api.requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Total number of GitHub API requests"),
	); err != nil {
		return nil, err
	}

	if r.tokenExpiry, err = meter.Float64Gauge("token.expiry.timestamp",
		metric.WithUnit("s"),
		metric.WithDescription("Unix timestamp when the token expires"),
	); err != nil {
		return nil, err
	}

	if r.reconcileErrors, err = meter.Int64Counter("token.reconcile.errors",
		metric.WithUnit("{error}"),
		metric.WithDescription("Total number of token-reconcile errors (by reason)"),
	); err != nil {
		return nil, err
	}

	if r.tokensActive, err = meter.Int64UpDownCounter("tokens.active",
		metric.WithUnit("{token}"),
		metric.WithDescription("Number of currently active tokens"),
	); err != nil {
		return nil, err
	}

	if r.secretOperations, err = meter.Int64Counter("kubernetes.secret.operations",
		metric.WithUnit("{operation}"),
		metric.WithDescription("Total number of Kubernetes Secret operations performed by the operator"),
	); err != nil {
		return nil, err
	}

	if r.configErrors, err = meter.Int64Counter("config.errors",
		metric.WithUnit("{error}"),
		metric.WithDescription("Total number of configuration errors"),
	); err != nil {
		return nil, err
	}

	return &r, nil
}

// RecordTokenRefresh records a token refresh operation with its result.
func (r *Recorder) RecordTokenRefresh(ctx context.Context, controllerName, result string) {
	if r == nil {
		return
	}
	r.tokenRefresh.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("controller", controllerName),
			attribute.String("result", result),
		),
	)
}

// RecordTokenRefreshDuration records the duration of a token refresh operation.
func (r *Recorder) RecordTokenRefreshDuration(ctx context.Context, controllerName, operation string, d time.Duration) {
	if r == nil {
		return
	}
	r.tokenRefreshDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			attribute.String("controller", controllerName),
			attribute.String("operation", operation),
		),
	)
}

// RecordGitHubAPICall records a completed GitHub API call: increments the
// requests counter and observes the duration histogram with the same attributes.
func (r *Recorder) RecordGitHubAPICall(ctx context.Context, controllerName string, d time.Duration, err error) {
	if r == nil {
		return
	}
	result := ResultSuccess
	if err != nil {
		result = ResultError
	}
	attrs := metric.WithAttributes(
		attribute.String("controller", controllerName),
		attribute.String("result", result),
	)
	r.githubAPIRequests.Add(ctx, 1, attrs)
	r.githubAPIDuration.Record(ctx, d.Seconds(), attrs)
}

// RecordTokenExpiry records the expiry timestamp for a token.
func (r *Recorder) RecordTokenExpiry(ctx context.Context, controllerName, namespace, name string, expiresAt time.Time) {
	if r == nil {
		return
	}
	r.tokenExpiry.Record(ctx, float64(expiresAt.Unix()),
		metric.WithAttributes(
			attribute.String("controller", controllerName),
			attribute.String("namespace", namespace),
			attribute.String("name", name),
		),
	)
}

// RecordReconcileError records a reconciliation error with its reason.
func (r *Recorder) RecordReconcileError(ctx context.Context, controllerName, reason string) {
	if r == nil {
		return
	}
	r.reconcileErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("controller", controllerName),
			attribute.String("reason", reason),
		),
	)
}

// EnsureTokenActive idempotently marks a token as active. The first call for a
// given tokenKey increments the active-token counter; subsequent calls are no-ops.
// This makes the counter self-healing across controller restarts.
func (r *Recorder) EnsureTokenActive(ctx context.Context, controllerName, tokenKey string) {
	if r == nil {
		return
	}
	if _, loaded := r.activeTokens.LoadOrStore(tokenKey, struct{}{}); loaded {
		return
	}
	r.tokensActive.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("controller", controllerName),
		),
	)
}

// RemoveTokenActive idempotently marks a token as inactive. Only decrements the
// counter if the tokenKey was previously tracked via EnsureTokenActive.
func (r *Recorder) RemoveTokenActive(ctx context.Context, controllerName, tokenKey string) {
	if r == nil {
		return
	}
	if _, loaded := r.activeTokens.LoadAndDelete(tokenKey); !loaded {
		return
	}
	r.tokensActive.Add(ctx, -1,
		metric.WithAttributes(
			attribute.String("controller", controllerName),
		),
	)
}

// RecordSecretOperation records a Kubernetes Secret create/update/delete operation.
func (r *Recorder) RecordSecretOperation(ctx context.Context, controllerName, operation, result string) {
	if r == nil {
		return
	}
	r.secretOperations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("controller", controllerName),
			attribute.String("operation", operation),
			attribute.String("result", result),
		),
	)
}

// RecordConfigError records a configuration loading error. source identifies the
// subsystem that failed (e.g. "ghapp", "app").
func (r *Recorder) RecordConfigError(ctx context.Context, controllerName, source string) {
	if r == nil {
		return
	}
	r.configErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("controller", controllerName),
			attribute.String("source", source),
		),
	)
}
