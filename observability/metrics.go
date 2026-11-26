package observability

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	AttrRoute     = "tunny.route"
	AttrProvider  = "tunny.provider"
	AttrOutcome   = "tunny.outcome"
	AttrErrorType = "error.type"
	AttrIngress   = "tunny.ingress"
	AttrTransport = "network.transport"
	AttrDirection = "network.io.direction"

	OutcomeSuccess = "success"
	OutcomeError   = "error"

	DirectionReceive  = "receive"
	DirectionTransmit = "transmit"

	IngressSOCKS5   = "socks5"
	IngressHTTP     = "http"
	IngressTUN      = "tun"
	IngressProvider = "provider"
)

var (
	DialBuckets       = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	ConnectionBuckets = []float64{0.1, 0.5, 1, 5, 10, 30, 60, 300, 900, 1800, 3600}
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/observability")

	connectionActive, _ = meter.Int64UpDownCounter(
		"tunny.connection.active",
		metric.WithUnit("{connection}"),
		metric.WithDescription("Client connections currently open."),
	)

	connectionDuration, _ = meter.Float64Histogram(
		"tunny.connection.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Lifetime of closed client connections."),
		metric.WithExplicitBucketBoundaries(ConnectionBuckets...),
	)

	connectionIO, _ = meter.Int64Counter(
		"tunny.connection.io",
		metric.WithUnit("By"),
		metric.WithDescription("Bytes received from and transmitted to clients."),
	)

	activeConnections atomic.Int64
)

var knownHTTPMethods = map[string]bool{
	"CONNECT": true, "DELETE": true, "GET": true, "HEAD": true, "OPTIONS": true,
	"PATCH": true, "POST": true, "PUT": true, "TRACE": true,
}

func HTTPMethod(method string) string {
	if knownHTTPMethods[method] {
		return method
	}
	return "_OTHER"
}

func ActiveConnections() int64 {
	return activeConnections.Load()
}

// Outcome returns the outcome attributes for an error type, where "" means success.
func Outcome(errType string) []attribute.KeyValue {
	if errType == "" {
		return []attribute.KeyValue{attribute.String(AttrOutcome, OutcomeSuccess)}
	}
	return []attribute.KeyValue{
		attribute.String(AttrOutcome, OutcomeError),
		attribute.String(AttrErrorType, errType),
	}
}

func DialErrorType(ctx context.Context, err error) string {
	var dnsErr *net.DNSError
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.As(err, &dnsErr):
		return "dns"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "refused"
	case errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH):
		return "unreachable"
	case errors.As(err, &netErr) && netErr.Timeout():
		return "timeout"
	default:
		return "dial"
	}
}

type trackedConn struct {
	net.Conn

	start       time.Time
	attrs       metric.MeasurementOption
	receiveOpt  metric.AddOption
	transmitOpt metric.AddOption
	closeOnce   sync.Once
}

func TrackConn(conn net.Conn, ingress, transport string) net.Conn {
	base := []attribute.KeyValue{
		attribute.String(AttrIngress, ingress),
		attribute.String(AttrTransport, transport),
	}
	attrs := metric.WithAttributeSet(attribute.NewSet(base...))

	connectionActive.Add(context.Background(), 1, attrs)
	activeConnections.Add(1)

	return &trackedConn{
		Conn:        conn,
		start:       time.Now(),
		attrs:       attrs,
		receiveOpt:  metric.WithAttributeSet(attribute.NewSet(append(base, attribute.String(AttrDirection, DirectionReceive))...)),
		transmitOpt: metric.WithAttributeSet(attribute.NewSet(append(base, attribute.String(AttrDirection, DirectionTransmit))...)),
	}
}

func (c *trackedConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		connectionIO.Add(context.Background(), int64(n), c.receiveOpt)
	}
	return n, err
}

func (c *trackedConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		connectionIO.Add(context.Background(), int64(n), c.transmitOpt)
	}
	return n, err
}

func (c *trackedConn) CloseWrite() error {
	if cw, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.closeOnce.Do(func() {
		connectionActive.Add(context.Background(), -1, c.attrs)
		activeConnections.Add(-1)
		connectionDuration.Record(context.Background(), time.Since(c.start).Seconds(), c.attrs)
	})
	return err
}

type trackedListener struct {
	net.Listener
	ingress string
}

func TrackListener(listener net.Listener, ingress string) net.Listener {
	return &trackedListener{Listener: listener, ingress: ingress}
}

func (l *trackedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return TrackConn(conn, l.ingress, "tcp"), nil
}
