package observability

import (
	"context"
	"errors"
	"fmt"

	"github.com/Aldiwildan77/tunny/config"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Daemon     string
	Version    string
	InstanceID string
	Endpoint   string
	Insecure   bool
}

type Provider struct {
	metrics *sdkmetric.MeterProvider
	traces  *sdktrace.TracerProvider
}

func New(ctx context.Context, cfg Config) (*Provider, error) {
	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var metricOpts []otlpmetricgrpc.Option
	var traceOpts []otlptracegrpc.Option
	if cfg.Endpoint != "" {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithEndpoint(cfg.Endpoint))
		traceOpts = append(traceOpts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, traceOpts...)
	if err != nil {
		_ = metricExporter.Shutdown(ctx)
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)

	otel.SetMeterProvider(mp)
	otel.SetTracerProvider(tp)

	p := &Provider{
		metrics: mp,
		traces:  tp,
	}

	if err := otelruntime.Start(otelruntime.WithMeterProvider(mp)); err != nil {
		_ = p.Shutdown(ctx)
		return nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	return p, nil
}

func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		attribute.String("service.namespace", config.APP_NAME),
		attribute.String("service.name", fmt.Sprintf("%s-%s", config.APP_NAME, cfg.Daemon)),
		attribute.String("service.version", cfg.Version),
	}
	if cfg.InstanceID != "" {
		attrs = append(attrs, attribute.String("service.instance.id", cfg.InstanceID))
	}

	// WithFromEnv comes last so OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES override the defaults.
	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithFromEnv(),
	)
}

func (p *Provider) Meter(name string) metric.Meter {
	return p.metrics.Meter(name)
}

func (p *Provider) Tracer(name string) trace.Tracer {
	return p.traces.Tracer(name)
}

func (p *Provider) Shutdown(ctx context.Context) error {
	metricErr := p.metrics.Shutdown(ctx)
	traceErr := p.traces.Shutdown(ctx)

	return errors.Join(metricErr, traceErr)
}
