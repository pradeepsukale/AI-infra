package utils

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const otelCollectorEndpoint = "otel-collector:4317"

func InitTracer(serviceName string) func() {
	ctx := context.Background()

	// The OTLP gRPC exporter sends completed spans to the OpenTelemetry
	// Collector service running inside Kubernetes.
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(otelCollectorEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	// The tracer provider owns span processors and resource metadata, including
	// the service.name used by backends such as Jaeger.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(serviceName),
			),
		),
	)

	// Register this provider globally so otel.Tracer uses the configured
	// exporter and resource for every span created by this process.
	otel.SetTracerProvider(tp)

	// Configure W3C TraceContext and Baggage propagation so trace IDs flow
	// across HTTP service boundaries.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		_ = tp.Shutdown(ctx)
	}
}

func InitMeter(serviceName string) func() {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(otelCollectorEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter),
		),
		sdkmetric.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(serviceName),
			),
		),
	)

	otel.SetMeterProvider(mp)

	return func() {
		_ = mp.Shutdown(ctx)
	}
}
