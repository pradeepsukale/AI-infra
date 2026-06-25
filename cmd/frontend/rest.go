package main

import (
	"context"
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
	"go.uber.org/zap"
)

var logger *zap.Logger

var httpDuration metric.Float64Histogram

func hello(w http.ResponseWriter, r *http.Request) {

	logger.Info(
		"request hello called",
		zap.Any("context", r.Context()),
	)
	// otel.Tracer returns an instrumentation-scoped tracer used to create spans
	// for this service's custom business operations.
	tracer := otel.Tracer("hello-service")

	// Start creates a span from the incoming request context, preserving any
	// distributed trace context extracted by the otelhttp server middleware.
	ctx, span := tracer.Start(
		r.Context(),
		"business-logic",
	)
	defer span.End()

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

	logger.Info(
		"About to call employee",
		zap.String("empId", "101"),
		zap.Any("context", ctx),
	)

	req, err1 := http.NewRequestWithContext(
		ctx,
		"GET",
		"http://employee-service/employee/101",
		nil,
	)

	if err1 != nil {
		logger.Error("Failed to create employee service request", zap.Error(err1), zap.Any("context", ctx))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)

	if err != nil {
		logger.Error("Employee service request failed", zap.Error(err), zap.Any("context", ctx))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	span.AddEvent("Got API response for emp 101")

	//fmt.Println(err.Error())

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	log.Println(string(body))

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

	logger.Info(
		"got caller details",
		zap.String("caller", caller),
		zap.Any("context", ctx),
	)

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

	var shutDownLogger func(context.Context) error
	logger, shutDownLogger = utils.InitLogger(context.Background(), "hello-service-new")
	defer shutDownLogger(context.Background())

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

	logger.Info(
		"init completed")

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
