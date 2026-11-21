package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	prometheusbridge "go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

func initMetrics(
	res *resource.Resource,
	gatherer prometheus.Gatherer,
	exporter sdkmetric.Exporter,
) func(context.Context) error {
	// Wire the producer explicitly: autoexport's producer configuration is
	// experimental and process-global, and could disable or duplicate this bridge.
	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithProducer(prometheusbridge.NewMetricProducer(prometheusbridge.WithGatherer(gatherer))),
	)
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Prometheus collectors do not accept a context. A blocked Gather can
		// outlive the SDK deadline, so bound the caller's wait independently.
		done := make(chan error, 1)
		go func() { done <- provider.Shutdown(ctx) }()

		var err error
		select {
		case err = <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}

		if err != nil {
			return fmt.Errorf("failed to shutdown metrics provider: %w", err)
		}

		return nil
	}
}

func newMetricExporter(ctx context.Context, protocol string) (sdkmetric.Exporter, error) {
	// Prometheus counters and histograms are cumulative. Exporter temporality
	// preferences cannot turn externally produced data into delta measurements.
	cumulative := func(sdkmetric.InstrumentKind) metricdata.Temporality {
		return metricdata.CumulativeTemporality
	}

	switch protocol {
	case "", "http/protobuf":
		return otlpmetrichttp.New(ctx, otlpmetrichttp.WithTemporalitySelector(cumulative))
	case "grpc":
		return otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithTemporalitySelector(cumulative))
	default:
		return nil, fmt.Errorf("unsupported OTLP metrics protocol: %q", protocol)
	}
}
