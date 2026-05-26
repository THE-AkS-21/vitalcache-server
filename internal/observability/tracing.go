package observability

import (
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTracer sets up the OpenTelemetry Tracer
func InitTracer(environment string) (*sdktrace.TracerProvider, error) {
	// For local dev, export traces to stdout.
	// For prod, replace stdouttrace with otlptracegrpc for Jaeger/Tempo.
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("vitalcache-server"),
			semconv.DeploymentEnvironmentKey.String(environment),
		)),
	)

	otel.SetTracerProvider(tp)
	log.Println("OpenTelemetry Tracer initialized successfully")
	return tp, nil
}
