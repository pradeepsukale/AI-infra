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

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("otel-collector:4317"),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		panic(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("employee-service"),
			),
		),
	)

	otel.SetTracerProvider(tp)

	// ADD THIS LINE
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		_ = tp.Shutdown(ctx)
	}
}

func employeeHandler(w http.ResponseWriter, r *http.Request) {

	tracer := otel.Tracer("employee-service")

	// CONTINUES incoming trace automatically
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

	// Child span
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
