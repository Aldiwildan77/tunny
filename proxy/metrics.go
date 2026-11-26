package proxy

import (
	"context"
	"errors"
	"net"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Aldiwildan77/tunny/observability"
)

const (
	errProviderNotFound    = "provider_not_found"
	errProviderUnreachable = "provider_unreachable"
	errHandshake           = "handshake"
	errProviderRejected    = "provider_rejected"
	errNoHealthyProvider   = "no_healthy_provider"
	errCanceled            = "canceled"
	errTimeout             = "timeout"
)

// Instruments bind to the global MeterProvider lazily, so they start
// exporting once observability.New calls otel.SetMeterProvider.
var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/proxy")

	requestDuration, _ = meter.Float64Histogram(
		"tunny.route.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to establish a routed connection, including provider failover."),
		metric.WithExplicitBucketBoundaries(observability.DialBuckets...),
	)

	attemptDuration, _ = meter.Float64Histogram(
		"tunny.route.provider.attempt.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time spent on a single provider attempt for a routed connection."),
		metric.WithExplicitBucketBoundaries(observability.DialBuckets...),
	)

	directDuration, _ = meter.Float64Histogram(
		"tunny.direct.dial.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to dial a destination that matches no route."),
		metric.WithExplicitBucketBoundaries(observability.DialBuckets...),
	)
)

func recordRequest(ctx context.Context, network, route, provider string, duration time.Duration, errType string) {
	attrs := append(routeAttributes(network, route), observability.Outcome(errType)...)
	if errType == "" {
		attrs = append(attrs, attribute.String(observability.AttrProvider, provider))
	}
	requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func recordAttempt(ctx context.Context, network, route, provider string, duration time.Duration, errType string) {
	attrs := append(routeAttributes(network, route), observability.Outcome(errType)...)
	attrs = append(attrs, attribute.String(observability.AttrProvider, provider))
	attemptDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func recordDirect(ctx context.Context, network string, duration time.Duration, errType string) {
	attrs := append([]attribute.KeyValue{attribute.String(observability.AttrTransport, network)}, observability.Outcome(errType)...)
	directDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func routeAttributes(network, route string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(observability.AttrTransport, network),
		attribute.String(observability.AttrRoute, route),
	}
}

func classifyError(ctx context.Context, err error, fallback string) string {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return errCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errTimeout
	}
	return fallback
}
