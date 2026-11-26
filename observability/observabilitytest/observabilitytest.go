package observabilitytest

import (
	"context"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var (
	once   sync.Once
	reader *sdkmetric.ManualReader
)

// Install must run before the code under test records anything: the global
// MeterProvider only accepts a delegate once per process.
func Install() {
	once.Do(func() {
		reader = sdkmetric.NewManualReader()
		otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	})
}

// Points returns every data point of the named metric keyed by its encoded
// attributes. Histograms report their count; counters and gauges their value.
func Points(t testing.TB, name string) map[string]float64 {
	t.Helper()
	Install()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	points := make(map[string]float64)
	encoder := attribute.DefaultEncoder()
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}
			switch data := m.Data.(type) {
			case metricdata.Histogram[float64]:
				for _, dp := range data.DataPoints {
					points[dp.Attributes.Encoded(encoder)] += float64(dp.Count)
				}
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					points[dp.Attributes.Encoded(encoder)] += float64(dp.Value)
				}
			case metricdata.Gauge[int64]:
				for _, dp := range data.DataPoints {
					points[dp.Attributes.Encoded(encoder)] = float64(dp.Value)
				}
			default:
				t.Fatalf("unsupported metric data %T for %s", m.Data, name)
			}
		}
	}
	return points
}
