package telemetry

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	collector "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// TODO: Add a lightweight production metric-name inventory test with an explicit
// expected list. Revisit how to keep that inventory exhaustive as collectors are
// added across packages, including collectors not exercised by the test fixture.

// Tests using process-wide OTel environment settings cannot run in parallel.
func cleanEnv(t *testing.T) {
	t.Helper()

	// Init can replace the tracer provider when a common OTLP endpoint is set.
	original := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(original) })

	keys := []string{
		"OTEL_METRICS_EXPORTER",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
		"OTEL_METRIC_EXPORT_INTERVAL",
		"OTEL_METRIC_EXPORT_TIMEOUT",
		"OTEL_SERVICE_NAME",
		"OTEL_RESOURCE_ATTRIBUTES",
		"OTEL_METRICS_PRODUCERS",
		"OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE",
		"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
}

func useGatherer(t *testing.T, gatherer prometheus.Gatherer) {
	t.Helper()
	original := prometheus.DefaultGatherer
	prometheus.DefaultGatherer = gatherer
	t.Cleanup(func() { prometheus.DefaultGatherer = original })
}

func newTestCounter() prometheus.Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "requests_total",
		Help:        "Requests",
		ConstLabels: prometheus.Labels{"route": "/api/v3/messages"},
	})
	counter.Add(3)
	return counter
}

func httpReceiver(t *testing.T) (*httptest.Server, <-chan *collector.ExportMetricsServiceRequest) {
	t.Helper()

	received := make(chan *collector.ExportMetricsServiceRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/metrics", r.URL.Path)
		assert.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		request := new(collector.ExportMetricsServiceRequest)
		if !assert.NoError(t, proto.Unmarshal(body, request)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- request
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	t.Cleanup(server.Close)
	return server, received
}

func receive(
	t *testing.T,
	received <-chan *collector.ExportMetricsServiceRequest,
) *collector.ExportMetricsServiceRequest {
	t.Helper()
	select {
	case request := <-received:
		return request
	case <-time.After(3 * time.Second):
		t.Fatal("no OTLP metrics received")
		return nil
	}
}

func attrs(values []*common.KeyValue) map[string]string {
	result := map[string]string{}
	for _, value := range values {
		result[value.Key] = value.Value.GetStringValue()
	}
	return result
}

func TestOTLPHTTPShutdownFlush(t *testing.T) {
	cleanEnv(t)
	server, received := httpReceiver(t)
	t.Setenv("OTEL_METRICS_EXPORTER", "otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://unused.invalid:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", server.URL+"/v1/metrics")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "3600000")
	t.Setenv("OTEL_SERVICE_NAME", "configured-service")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES",
		"service.name=lower-priority,service.instance.id=pod-1,deployment.environment.name=test",
	)
	reg := prometheus.NewRegistry()
	useGatherer(t, reg)

	shutdown, err := Init(t.Context(), "traQ")
	require.NoError(t, err)
	// Echo and GORM also register their collectors after telemetry initialization.
	reg.MustRegister(newTestCounter())
	require.NoError(t, shutdown(context.Background()))
	request := receive(t, received)
	require.Len(t, request.ResourceMetrics, 1)
	rm := request.ResourceMetrics[0]
	attributes := attrs(rm.Resource.Attributes)
	assert.Equal(t, "configured-service", attributes["service.name"])
	assert.Equal(t, "pod-1", attributes["service.instance.id"])
	assert.Equal(t, "test", attributes["deployment.environment.name"])
	require.Len(t, rm.ScopeMetrics, 1)
	metrics := rm.ScopeMetrics[0].Metrics
	require.Len(t, metrics, 1)
	assert.Equal(t, "requests_total", metrics[0].Name)
	sum := metrics[0].GetSum()
	require.NotNil(t, sum)
	require.Len(t, sum.DataPoints, 1)
	assert.Equal(t, float64(3), sum.DataPoints[0].GetAsDouble())
	assert.Equal(t, map[string]string{"route": "/api/v3/messages"}, attrs(sum.DataPoints[0].Attributes))
}

type grpcReceiver struct {
	collector.UnimplementedMetricsServiceServer
	received chan *collector.ExportMetricsServiceRequest
}

func (s *grpcReceiver) Export(
	_ context.Context,
	request *collector.ExportMetricsServiceRequest,
) (*collector.ExportMetricsServiceResponse, error) {
	s.received <- request
	return &collector.ExportMetricsServiceResponse{}, nil
}

func TestOTLPGRPCShutdownFlush(t *testing.T) {
	cleanEnv(t)
	reg := prometheus.NewRegistry()
	reg.MustRegister(newTestCounter())
	useGatherer(t, reg)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	receiver := &grpcReceiver{received: make(chan *collector.ExportMetricsServiceRequest, 1)}
	collector.RegisterMetricsServiceServer(server, receiver)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://"+listener.Addr().String())
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "3600000")
	shutdown, err := Init(t.Context(), "traQ")
	require.NoError(t, err)
	require.NoError(t, shutdown(context.Background()))
	request := receive(t, receiver.received)
	require.Len(t, request.ResourceMetrics, 1)
	attributes := attrs(request.ResourceMetrics[0].Resource.Attributes)
	assert.Equal(t, "traQ", attributes["service.name"])
	assert.NotEmpty(t, attributes["service.instance.id"])
}

func TestMetricsActivation(t *testing.T) {
	tests := map[string]struct {
		env map[string]string
	}{
		"success: unset": {},
		"success: unset ignores resource settings": {env: map[string]string{
			"OTEL_RESOURCE_ATTRIBUTES": "invalid",
		}},
		"success: traces only": {env: map[string]string{
			"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://localhost:4318/v1/traces",
		}},
		"success: none overrides endpoint": {env: map[string]string{
			"OTEL_METRICS_EXPORTER":       "none",
			"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4318",
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cleanEnv(t)
			for key, value := range test.env {
				t.Setenv(key, value)
			}
			// Disabled metrics must not collect from the registry or parse their protocol.
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", "invalid")
			useGatherer(t, nil)
			shutdown, err := Init(t.Context(), "traQ")
			require.NoError(t, err)
			require.NoError(t, shutdown(t.Context()))
		})
	}
}

func TestMetricsConfiguration(t *testing.T) {
	tests := map[string]struct {
		key       string
		value     string
		wantError string
	}{
		"fail: unsupported exporter": {
			key:       "OTEL_METRICS_EXPORTER",
			value:     "prometheus",
			wantError: "unsupported OTEL_METRICS_EXPORTER",
		},
		"fail: invalid protocol": {
			key:       "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
			value:     "invalid",
			wantError: "unsupported OTLP metrics protocol",
		},
		"fail: invalid resource": {
			key:       "OTEL_RESOURCE_ATTRIBUTES",
			value:     "invalid",
			wantError: "failed to create telemetry resource",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "http://localhost:4318/v1/metrics")
			t.Setenv(test.key, test.value)

			_, err := Init(t.Context(), "traQ")
			require.ErrorContains(t, err, test.wantError)
		})
	}
}

// uncancellableTestCollector simulates a stuck Prometheus collection, which has
// no context to observe. Shutdown must return without waiting for its release.
type uncancellableTestCollector struct {
	unblockCollection <-chan struct{}
}

func (c uncancellableTestCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("blocked", "Blocked collection", nil, nil)
}

func (c uncancellableTestCollector) Collect(chan<- prometheus.Metric) {
	<-c.unblockCollection
}

// discardTestExporter keeps shutdown tests independent of network behavior.
type discardTestExporter struct{}

func (discardTestExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (discardTestExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (discardTestExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	return nil
}

func (discardTestExporter) ForceFlush(context.Context) error { return nil }
func (discardTestExporter) Shutdown(context.Context) error   { return nil }

func TestMetricsShutdown(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		callerTimeout time.Duration
		wantElapsed   time.Duration
	}{
		"fail: collection timeout": {
			wantElapsed: 5 * time.Second,
		},
		"fail: caller deadline during collection": {
			callerTimeout: 50 * time.Millisecond,
			wantElapsed:   50 * time.Millisecond,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				reg := prometheus.NewRegistry()
				unblockCollection := make(chan struct{})
				// Release the collector after the assertion so synctest can finish.
				defer close(unblockCollection)
				reg.MustRegister(uncancellableTestCollector{unblockCollection: unblockCollection})
				shutdown := initMetrics(resource.Empty(), reg, discardTestExporter{})
				ctx := t.Context()
				if test.callerTimeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, test.callerTimeout)
					defer cancel()
				}

				start := time.Now()
				require.ErrorIs(t, shutdown(ctx), context.DeadlineExceeded)

				assert.Equal(t, test.wantElapsed, time.Since(start))
			})
		})
	}
}
