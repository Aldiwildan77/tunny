package tunnycontrol

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/Aldiwildan77/tunny/observability"
)

var (
	meter = otel.Meter("github.com/Aldiwildan77/tunny/control-plane")

	buckets = []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}

	rpcDuration, _ = meter.Float64Histogram(
		"tunny.control.rpc.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Control plane gRPC call duration."),
		metric.WithExplicitBucketBoundaries(buckets...),
	)

	httpDuration, _ = meter.Float64Histogram(
		"tunny.control.http.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Control plane HTTP API request duration."),
		metric.WithExplicitBucketBoundaries(buckets...),
	)
)

func unaryMetrics(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	response, err := handler(ctx, request)
	rpcDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
		attribute.String("rpc.method", info.FullMethod),
		attribute.String("rpc.grpc.status_code", status.Code(err).String()),
	))
	return response, err
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func httpMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: writer, status: http.StatusOK}
		next.ServeHTTP(recorder, request)
		httpDuration.Record(request.Context(), time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("http.request.method", observability.HTTPMethod(request.Method)),
			attribute.String("http.route", httpRoute(request.URL.Path)),
			attribute.Int("http.response.status_code", recorder.status),
		))
	})
}

// Hostnames in /v1/routes/{hostname} are collapsed to keep cardinality bounded.
func httpRoute(path string) string {
	switch {
	case path == "/v1/status", path == "/v1/routes", path == "/v1/providers/health":
		return path
	case strings.HasPrefix(path, "/v1/routes/"):
		return "/v1/routes/{hostname}"
	default:
		return "other"
	}
}
