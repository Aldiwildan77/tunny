package health

import (
	"context"
	"errors"
	"log"
	"net"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Aldiwildan77/tunny/observability"
)

const (
	errProviderNotFound   = "provider_not_found"
	errUnreachable        = "provider_unreachable"
	errHandshake          = "handshake"
	errUnexpectedResponse = "unexpected_response"
	errTimeout            = "timeout"
	errCanceled           = "canceled"

	attrState = "tunny.health.state"
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/health")

	checkDuration, _ = meter.Float64Histogram(
		"tunny.health.check.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of active PING/PONG provider health checks."),
		metric.WithExplicitBucketBoundaries(observability.DialBuckets...),
	)

	transitions, _ = meter.Int64Counter(
		"tunny.health.transitions",
		metric.WithUnit("{transition}"),
		metric.WithDescription("Provider health state changes, labelled with the new state."),
	)

	healthy, _ = meter.Int64ObservableGauge(
		"tunny.health.healthy",
		metric.WithDescription("1 when the provider is healthy, 0 when unhealthy."),
	)
)

func recordCheck(ctx context.Context, provider string, duration time.Duration, errType string) {
	attrs := append([]attribute.KeyValue{attribute.String(observability.AttrProvider, provider)}, observability.Outcome(errType)...)
	checkDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func recordTransition(provider string, state State) {
	transitions.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String(observability.AttrProvider, provider),
		attribute.String(attrState, string(state)),
	))
}

func classifyCheckError(ctx context.Context, errType string, err error) string {
	if err == nil {
		return ""
	}
	var netErr net.Error
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return errCanceled
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errTimeout
	case errors.As(err, &netErr) && netErr.Timeout():
		return errTimeout
	default:
		return errType
	}
}

func (m *Manager) registerMetrics(ctx context.Context) {
	registration, err := meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		for _, status := range m.Snapshot() {
			value := int64(1)
			if status.State == StateUnhealthy {
				value = 0
			}
			observer.ObserveInt64(healthy, value, metric.WithAttributes(attribute.String(observability.AttrProvider, status.Provider)))
		}
		return nil
	}, healthy)
	if err != nil {
		log.Printf("register health metrics: %v", err)
		return
	}

	go func() {
		<-ctx.Done()
		_ = registration.Unregister()
	}()
}
