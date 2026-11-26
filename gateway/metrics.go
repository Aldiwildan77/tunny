package gateway

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Aldiwildan77/tunny/observability"
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/gateway")

	httpRequestDuration, _ = meter.Float64Histogram(
		"tunny.http.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("HTTP proxy request duration. CONNECT requests last for the whole tunnel."),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300, 900, 3600),
	)
)

func recordHTTPRequest(ctx context.Context, method string, status int, duration time.Duration) {
	httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("http.request.method", observability.HTTPMethod(method)),
		attribute.Int("http.response.status_code", status),
	))
}
