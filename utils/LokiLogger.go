package utils

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(ctx context.Context) (*zap.Logger, func(context.Context) error) {

	// ==========================================
	// 1. NETWORK EXPORTER SETUP
	// ==========================================
	// This creates the network pipe. It tells the OpenTelemetry SDK how to package
	// and ship logs over the network using gRPC to your OTel Collector endpoint.
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(OtelCollectorEndpoint),
		otlploggrpc.WithInsecure(), // Uses HTTP/2 without TLS (perfect for inside Minikube)
	)
	if err != nil {
		panic(err)
	}

	// ==========================================
	// 2. RESOURCE ATTRIBUTES (METADATA)
	// ==========================================
	// This attaches global labels to *every single log line* sent by this app.
	// When these hit Loki, they become indexed fields (labels) like `service_name="my-go-app"`.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("my-go-app"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("dev"),
		),
	)
	if err != nil {
		panic(err)
	}

	// ==========================================
	// 3. THE BATCH PROCESSOR (Performance Engine)
	// ==========================================
	// EXPLANATION: If your app sends 1,000 logs per second, you don't want to make
	// 1,000 individual network calls to the OTel Collector. That would kill your app's performance.
	// The BatchProcessor acts as an in-memory waiting room. It collects logs in RAM and
	// fires them across the network in large batches only when:
	//   - A certain number of logs are waiting (e.g., 512 logs), OR
	//   - A specific time limit has passed (e.g., every 1 second).
	processor := sdklog.NewBatchProcessor(exporter)

	// The LoggerProvider is the master manager. It stitches the metadata (Resource)
	// and the performance batching engine (Processor) together into a functional OTel unit.
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(processor),
		sdklog.WithResource(res),
	)

	// ==========================================
	// 4. LOCAL CONSOLE OUTPUT (Standard Container Logs)
	// ==========================================
	// EXPLANATION: When you run `kubectl logs <pod-name>`, Kubernetes reads whatever your
	// container prints directly to its system console (Stdout).
	// This section ensures your logs *still* print locally to the container console, so you
	// don't lose the ability to debug using basic kubectl commands.
	productionCfg := zap.NewProductionConfig()                            // Uses standard production settings (JSON, Info level)
	consoleEncoder := zapcore.NewJSONEncoder(productionCfg.EncoderConfig) // Turns logs into structured JSON strings

	// Core #1: Writes the JSON formatted logs directly to the standard terminal output (os.Stdout)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zap.InfoLevel)

	// ==========================================
	// 5. OPENTELEMETRY BRIDGE CORE
	// ==========================================
	// EXPLANATION: This is a translation layer. Zap doesn't naturally know how to talk
	// to OpenTelemetry. The `otelzap` bridge acts as an interpreter.
	// Core #2: It intercepts Zap logs, translates them into the binary OTel format,
	// and passes them directly to the OTel LoggerProvider we built in step 3.
	otelCore := otelzap.NewCore("my-go-app", otelzap.WithLoggerProvider(provider))

	// ==========================================
	// 6. THE ZAP TEE (Dual Routing)
	// ==========================================
	// EXPLANATION: A "Tee" works just like a plumbing pipe split. It takes a single
	// log entry from your code and clones it into two directions simultaneously.
	// Every time you call `logger.Info()`, it executes:
	//   Direction A -> consoleCore (prints to standard terminal for `kubectl logs`)
	//   Direction B -> otelCore    (batches it and sends it over gRPC to the OTel Collector -> Loki)
	combinedCore := zapcore.NewTee(consoleCore, otelCore)
	logger := zap.New(combinedCore, zap.AddCaller()) // zap.AddCaller() adds the filename and line number (e.g. main.go:34)

	// ==========================================
	// 7. THE SHUTDOWN HANDLER (Memory Flush)
	// ==========================================
	// EXPLANATION: Because the BatchProcessor holds logs in memory (RAM) to optimize network speed,
	// if your application crashes or exits unexpectedly, the logs currently sitting in the
	// "waiting room" would be lost forever.
	// This shutdown hook returns a function that you must trigger (usually via `defer`) in your main.go.
	shutdownHook := func(sCtx context.Context) error {
		_ = logger.Sync()              // Forces Zap to clear out any remaining OS terminal streams.
		return provider.Shutdown(sCtx) // Tells OpenTelemetry: "We are shutting down! Immediately flush whatever logs are left in RAM over the network to the collector right now, then close the connection cleanly."
	}

	return logger, shutdownHook
}
