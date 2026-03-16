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
	tokenExpiry          metric.Float64Gauge
	reconcileErrors      metric.Int64Counter
	tokensActive         metric.Int64UpDownCounter
	secretOperations     metric.Int64Counter
	configErrors         metric.Int64Counter
	githubTokenRequests  metric.Int64Counter

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

	if r.tokenRefresh, err = meter.Int64Counter("gtm.token.refresh",
		metric.WithUnit("{refresh}"),
		metric.WithDescription("Total number of token refresh operations"),
	); err != nil {
		return nil, err
	}

	if r.tokenRefreshDuration, err = meter.Float64Histogram("gtm.token.refresh.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of token refresh operations"),
	); err != nil {
		return nil, err
	}

	if r.githubAPIDuration, err = meter.Float64Histogram("gtm.github.api_call.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of GitHub API calls"),
	); err != nil {
		return nil, err
	}

	if r.tokenExpiry, err = meter.Float64Gauge("gtm.token.expiry.timestamp",
		metric.WithUnit("s"),
		metric.WithDescription("Unix timestamp when the token expires"),
	); err != nil {
		return nil, err
	}

	if r.reconcileErrors, err = meter.Int64Counter("gtm.reconcile.errors",
		metric.WithUnit("{error}"),
		metric.WithDescription("Total number of reconciliation errors"),
	); err != nil {
		return nil, err
	}

	if r.tokensActive, err = meter.Int64UpDownCounter("gtm.tokens.active",
		metric.WithUnit("{token}"),
		metric.WithDescription("Number of currently active tokens"),
	); err != nil {
		return nil, err
	}

	if r.secretOperations, err = meter.Int64Counter("gtm.secret.operations",
		metric.WithUnit("{operation}"),
		metric.WithDescription("Total number of secret operations"),
	); err != nil {
		return nil, err
	}

	if r.configErrors, err = meter.Int64Counter("gtm.config.errors",
		metric.WithUnit("{error}"),
		metric.WithDescription("Total number of configuration errors"),
	); err != nil {
		return nil, err
	}

	if r.githubTokenRequests, err = meter.Int64Counter("gtm.github.token.requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Total GitHub Installation Access Token requests"),
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

// RecordGitHubAPIDuration records the duration of a GitHub API call.
func (r *Recorder) RecordGitHubAPIDuration(ctx context.Context, d time.Duration, err error) {
	if r == nil {
		return
	}
	result := ResultSuccess
	if err != nil {
		result = ResultError
	}
	r.githubAPIDuration.Record(ctx, d.Seconds(),
		metric.WithAttributes(
			attribute.String("result", result),
		),
	)
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

// RecordSecretOperation records a secret create/update/delete operation.
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

// RecordConfigError records a configuration loading error.
func (r *Recorder) RecordConfigError(ctx context.Context, source string) {
	if r == nil {
		return
	}
	r.configErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("source", source),
		),
	)
}

// RecordGitHubTokenRequest records a GitHub Installation Access Token request.
func (r *Recorder) RecordGitHubTokenRequest(ctx context.Context, err error) {
	if r == nil {
		return
	}
	result := ResultSuccess
	if err != nil {
		result = ResultError
	}
	r.githubTokenRequests.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("result", result),
		),
	)
}
