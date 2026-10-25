package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

// Init initializes OpenTelemetry tracing and metrics with independent signal settings.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	commonEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	tracesEnabled := commonEndpoint != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
	metricsExporter := os.Getenv("OTEL_METRICS_EXPORTER")
	metricsEnabled := metricsExporter != "none" && (metricsExporter != "" ||
		commonEndpoint != "" || os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != "")

	if !tracesEnabled && !metricsEnabled {
		return noopShutdown, nil
	}

	if metricsEnabled && metricsExporter != "" && metricsExporter != "otlp" {
		return nil, fmt.Errorf("unsupported OTEL_METRICS_EXPORTER: %q (use otlp or none)", metricsExporter)
	}

	instanceID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to create service instance ID: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceInstanceIDKey.String(instanceID.String()),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telemetry resource: %w", err)
	}

	shutdownTracer := noopShutdown
	if tracesEnabled {
		shutdownTracer, err = initTracer(ctx, res)
		if err != nil {
			return nil, err
		}
	}

	if !metricsEnabled {
		return shutdownTracer, nil
	}

	protocol := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL")
	if protocol == "" {
		protocol = os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	}

	exporter, err := newMetricExporter(ctx, protocol)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("failed to create OTLP metrics exporter: %w", err),
			shutdownTracer(context.Background()),
		)
	}

	shutdownMetrics := initMetrics(res, prometheus.DefaultGatherer, exporter)

	return func(ctx context.Context) error {
		return errors.Join(shutdownMetrics(ctx), shutdownTracer(ctx))
	}, nil
}

func noopShutdown(context.Context) error { return nil }
