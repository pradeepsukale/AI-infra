package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"promethius/utils"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type Employee struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Address string `json:"address"`
}

var logger *zap.Logger

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
	defer span.End()

	logger.Info("Received request at employeeHandler", zap.Any("context", ctx))

	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {
		logger.Error("employee id missing in request path", zap.Any("context", ctx))
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

	logger.Info("Returning employee details", zap.String("employee.id", employeeID), zap.Any("context", ctx))

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(employee)
	if err != nil {
		logger.Error("Failed to encode response", zap.Error(err), zap.Any("context", ctx))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {

	var shutDownLogger func(context.Context) error
	logger, shutDownLogger = utils.InitLogger(context.Background(), "employee-service")
	defer shutDownLogger(context.Background())

	configs := utils.IntiConfig()
	logger.Info("Database connection parameters",
		zap.String("db_host", configs.DbHost),
		zap.String("db_port", configs.DbPort),
		zap.String("db_name", configs.DbName),
		zap.String("db_user", configs.DbUser),
		zap.String("db_pass", configs.DbPass),
	)

	shutdown := utils.InitTracer("employee-service")
	defer shutdown()

	logger.Info("Starting employee-service server")

	// NewHandler instruments inbound HTTP requests by extracting trace context
	// and creating a server span for each request.
	handler := otelhttp.NewHandler(
		http.HandlerFunc(employeeHandler),
		"employee-handler",
	)

	http.Handle("/employee/", handler)

	logger.Info("Server started on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Fatal("Server listen and serve failed", zap.Error(err))
	}
}
