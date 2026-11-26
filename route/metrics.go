package route

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/route")

	routeCount, _ = meter.Int64ObservableGauge(
		"tunny.route.count",
		metric.WithUnit("{entry}"),
		metric.WithDescription("Entries in the route table by kind: hostname routes, failover policies, and resolved IPs."),
	)

	kindHostname = metric.WithAttributes(attribute.String("tunny.route.kind", "hostname"))
	kindPolicy   = metric.WithAttributes(attribute.String("tunny.route.kind", "policy"))
	kindIP       = metric.WithAttributes(attribute.String("tunny.route.kind", "ip"))
)

func (t *Table) registerMetrics() {
	_, err := meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		t.mu.RLock()
		defer t.mu.RUnlock()

		observer.ObserveInt64(routeCount, int64(len(t.Routes)), kindHostname)
		observer.ObserveInt64(routeCount, int64(len(t.Policies)), kindPolicy)
		observer.ObserveInt64(routeCount, int64(len(t.IPs)), kindIP)
		return nil
	}, routeCount)
	if err != nil {
		log.Printf("register route metrics: %v", err)
	}
}
