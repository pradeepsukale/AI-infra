package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"promethius/utils"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Employee struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Address string `json:"address"`
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

	shutdown := utils.InitTracer("employee-service")
	defer shutdown()

	log.Println("fdfgdfdf")

	// NewHandler instruments inbound HTTP requests by extracting trace context
	// and creating a server span for each request.
	handler := otelhttp.NewHandler(
		http.HandlerFunc(employeeHandler),
		"employee-handler",
	)

	http.Handle("/employee/", handler)

	log.Println("dsds Server started on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
