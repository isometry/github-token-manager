package metrics

import (
	"fmt"
	"os"

	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const serviceName = "github-token-manager"

// Setup creates an OTEL Prometheus exporter registered with the controller-runtime
// metrics registry, and returns a Recorder holding all custom instruments.
//
// version is surfaced as service.version on the OTEL Resource and therefore on
// the `target_info` metric. Pass the build-time version string (or "" if unknown).
func Setup(version string) (*Recorder, error) {
	exporter, err := promexporter.New(
		promexporter.WithRegisterer(crmetrics.Registry),
		promexporter.WithoutScopeInfo(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating prometheus exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		semconv.ServiceInstanceID(instanceID()),
	))
	if err != nil {
		return nil, fmt.Errorf("building metrics resource: %w", err)
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithResource(res),
	)
	meter := provider.Meter(serviceName)

	recorder, err := newRecorder(meter)
	if err != nil {
		return nil, fmt.Errorf("creating metrics recorder: %w", err)
	}

	recorder.provider = provider
	return recorder, nil
}

// instanceID returns the pod name (via POD_NAME / HOSTNAME downward API) for
// use as service.instance.id, falling back to the hostname or an empty string.
func instanceID() string {
	if v := os.Getenv("POD_NAME"); v != "" {
		return v
	}
	if v, err := os.Hostname(); err == nil {
		return v
	}
	return ""
}
