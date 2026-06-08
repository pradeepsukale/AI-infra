package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"

	"promethius/utils"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var httpDuration metric.Float64Histogram

func hello(w http.ResponseWriter, r *http.Request) {

	// otel.Tracer returns an instrumentation-scoped tracer used to create spans
	// for this service's custom business operations.
	tracer := otel.Tracer("hello-service")

	// Start creates a span from the incoming request context, preserving any
	// distributed trace context extracted by the otelhttp server middleware.
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
		// NewTransport wraps the default HTTP transport so outbound requests
		// create client spans and inject trace headers into downstream calls.
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

	// A child span records work that happens inside the business-logic span.
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

	httpDuration.Record(
		ctx,
		duration,
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("route", "/api/hello"),
			attribute.String("caller", caller),
		),
	)

	w.Write([]byte("Hello 🚀"))
}

func main() {
	shutdownTracer := utils.InitTracer("hello-service-new")
	defer shutdownTracer()

	shutdownMeter := utils.InitMeter("hello-service-new")
	defer shutdownMeter()

	meter := otel.Meter("hello-service")
	var err error
	httpDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds_new",
		metric.WithDescription("Request latency"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2),
	)
	if err != nil {
		log.Fatal(err)
	}

	// NewHandler instruments inbound HTTP requests by extracting trace context
	// and creating a server span for each request.
	handler := otelhttp.NewHandler(
		http.HandlerFunc(hello),
		"hello-handler",
	)

	http.Handle("/api/hello", handler)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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
