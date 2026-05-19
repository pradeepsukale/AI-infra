package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var httpDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds_new",
		Help:    "Request latency",
		Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2},
	},
	[]string{"method", "route", "caller"},
)

func init() {
	prometheus.MustRegister(httpDuration)
}

func hello(w http.ResponseWriter, r *http.Request) {

	tracer := otel.Tracer("hello-service")

	ctx, span := tracer.Start(
		r.Context(),
		"business-logic",
	)

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.Path),
		attribute.String("user.agent", r.UserAgent()),
	)

	span.AddEvent("About to call Employee API")

	client := http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
		),
	}

	req, err1 := http.NewRequestWithContext(
		ctx,
		"GET",
		"http://employee-service/employee/101",
		nil,
	)

	if err1 != nil {
		fmt.Println(err1.Error())
		return
	}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	span.AddEvent("Got API response for emp 101")

	//fmt.Println(err.Error())

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	log.Println(string(body))

	time.Sleep(500 * time.Millisecond)

	defer span.End()

	time.Sleep(500 * time.Millisecond)

	// child span
	_, dbSpan := tracer.Start(
		ctx,
		"fake-db-call",
	)

	time.Sleep(300 * time.Millisecond)

	dbSpan.End()

	start := time.Now()

	caller := getCallerService(r)

	// simulate latency
	time.Sleep(time.Duration(rand.Intn(400)) * time.Millisecond)

	duration := time.Since(start).Seconds()

	httpDuration.WithLabelValues(r.Method, "/api/hello", caller).Observe(duration)

	w.Write([]byte("Hello 🚀"))
}

func main() {
	ctx := context.Background()

	// OTEL Collector service inside k8s
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("otel-collector:4317"),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		panic(err)
	}

	// Tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("hello-service-new"),
			),
		),
	)

	otel.SetTracerProvider(tp)

	// ADD THIS LINE
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	// HTTP handler with tracing middleware
	handler := otelhttp.NewHandler(
		http.HandlerFunc(hello),
		"hello-handler",
	)

	http.Handle("/api/hello", handler)
	http.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(":8080", nil)
}

func getCallerService(r *http.Request) string {
	caller := r.Header.Get("X-Caller-Service")

	// default fallback
	if caller == "" {
		return "unknown"
	}

	// allow only known services (VERY IMPORTANT)
	switch caller {
	case "a10nsp", "olt", "leaf":
		return caller
	default:
		return "other"
	}
}
