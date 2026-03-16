package metrics

import (
	"fmt"

	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Setup creates an OTEL Prometheus exporter registered with the controller-runtime
// metrics registry, and returns a Recorder holding all custom instruments.
func Setup() (*Recorder, error) {
	exporter, err := promexporter.New(
		promexporter.WithRegisterer(crmetrics.Registry),
	)
	if err != nil {
		return nil, fmt.Errorf("creating prometheus exporter: %w", err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("github.com/isometry/github-token-manager")

	recorder, err := newRecorder(meter)
	if err != nil {
		return nil, fmt.Errorf("creating metrics recorder: %w", err)
	}

	recorder.provider = provider
	return recorder, nil
}
