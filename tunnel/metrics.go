package tunnel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Aldiwildan77/tunny/observability"
)

const (
	errCreateEndpoint = "create_endpoint"
	errDial           = "dial"
	errDeviceWrite    = "device_write"
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/tunnel")

	packets, _ = meter.Int64Counter(
		"tunny.tunnel.packets",
		metric.WithUnit("{packet}"),
		metric.WithDescription("IP packets read from (receive) or written to (transmit) the TUN device."),
	)

	packetIO, _ = meter.Int64Counter(
		"tunny.tunnel.io",
		metric.WithUnit("By"),
		metric.WithDescription("Bytes of IP packets read from or written to the TUN device."),
	)

	forwardErrors, _ = meter.Int64Counter(
		"tunny.tunnel.forward.errors",
		metric.WithUnit("{flow}"),
		metric.WithDescription("TCP or UDP flows that could not be forwarded."),
	)
)

type packetOptions struct {
	success metric.AddOption
	failure metric.AddOption
	io      metric.AddOption
}

var packetOpts = map[string]packetOptions{
	observability.DirectionReceive:  newPacketOptions(observability.DirectionReceive),
	observability.DirectionTransmit: newPacketOptions(observability.DirectionTransmit),
}

func newPacketOptions(direction string) packetOptions {
	dir := attribute.String(observability.AttrDirection, direction)
	return packetOptions{
		success: metric.WithAttributeSet(attribute.NewSet(append(observability.Outcome(""), dir)...)),
		failure: metric.WithAttributeSet(attribute.NewSet(append(observability.Outcome(errDeviceWrite), dir)...)),
		io:      metric.WithAttributeSet(attribute.NewSet(dir)),
	}
}

func recordPacket(ctx context.Context, direction string, size int, err error) {
	opts := packetOpts[direction]
	if err != nil {
		packets.Add(ctx, 1, opts.failure)
		return
	}
	packets.Add(ctx, 1, opts.success)
	packetIO.Add(ctx, int64(size), opts.io)
}

func recordForwardError(ctx context.Context, transport, errType string) {
	forwardErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String(observability.AttrTransport, transport),
		attribute.String(observability.AttrErrorType, errType),
	))
}
