package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Employee struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Address string `json:"address"`
}

func initTracer() func() {

	ctx := context.Background()

	// The OTLP gRPC exporter sends completed spans to the OpenTelemetry
	// Collector service running inside Kubernetes.
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("otel-collector:4317"),
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
				semconv.ServiceName("employee-service"),
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

func employeeHandler(w http.ResponseWriter, r *http.Request) {

	// otel.Tracer returns an instrumentation-scoped tracer used to create spans
	// for this service's custom business operations.
	tracer := otel.Tracer("employee-service")

	// Start creates a span from the incoming request context, continuing the
	// distributed trace extracted by the otelhttp server middleware.
	ctx, span := tracer.Start(
		r.Context(),
		"get-employee",
	)

	fmt.Print("fdfd")

	defer span.End()

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {
		http.Error(w, "employee id missing", http.StatusBadRequest)
		return
	}

	employeeID := parts[2]

	span.SetAttributes(
		attribute.String("employee.id", employeeID),
	)

	// A child span records work that happens inside the get-employee span.
	_, dbSpan := tracer.Start(
		ctx,
		"fake-db-call",
	)

	time.Sleep(300 * time.Millisecond)

	dbSpan.End()

	employee := Employee{
		ID:      employeeID,
		Name:    "Pradeep",
		Age:     42,
		Address: "Hyderabad",
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(employee)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {

	shutdown := initTracer()
	defer shutdown()

	// NewHandler instruments inbound HTTP requests by extracting trace context
	// and creating a server span for each request.
	handler := otelhttp.NewHandler(
		http.HandlerFunc(employeeHandler),
		"employee-handler",
	)

	http.Handle("/employee/", handler)

	log.Println("Server started on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
