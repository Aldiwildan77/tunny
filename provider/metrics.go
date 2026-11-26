package provider

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Aldiwildan77/tunny/observability"
)

const (
	requestConnect = "connect"
	requestPing    = "ping"
	requestInvalid = "invalid"
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/provider")

	requests, _ = meter.Int64Counter(
		"tunny.provider.requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Requests received by the provider, by type."),
	)

	targetDialDuration, _ = meter.Float64Histogram(
		"tunny.provider.target.dial.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time for the provider to dial the requested destination."),
		metric.WithExplicitBucketBoundaries(observability.DialBuckets...),
	)
)

func recordRequest(ctx context.Context, requestType string) {
	requests.Add(ctx, 1, metric.WithAttributes(attribute.String("tunny.request.type", requestType)))
}

func recordTargetDial(ctx context.Context, duration time.Duration, errType string) {
	targetDialDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(observability.Outcome(errType)...))
}
