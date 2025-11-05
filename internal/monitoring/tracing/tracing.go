package tracing

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	initOnce       sync.Once
	tracerProvider *sdktrace.TracerProvider
	tracerName     = "gcli2api-go"
)

// Init configures OpenTelemetry tracing if OTLP endpoint is present.
// It returns a shutdown function that should be invoked during server shutdown.
func Init(ctx context.Context) (func(context.Context) error, error) {
	var initErr error
	initOnce.Do(func() {
		endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		if endpoint == "" {
			tracerProvider = nil
			return
		}

		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(endpoint),
		}

		insecureFlag := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"))
		if insecureFlag == "" || strings.EqualFold(insecureFlag, "true") || strings.EqualFold(insecureFlag, "1") {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		exporter, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			initErr = err
			return
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				attribute.String("service.name", tracerName),
				attribute.String("service.version", version.Version),
				attribute.String("service.instance.id", hostname()),
			),
			resource.WithProcess(),
			resource.WithTelemetrySDK(),
			resource.WithFromEnv(),
		)
		if err != nil {
			initErr = err
			return
		}

		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter,
				sdktrace.WithBatchTimeout(5*time.Second),
			),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	})

	if initErr != nil {
		return func(context.Context) error { return nil }, initErr
	}
	if tracerProvider == nil {
		return func(context.Context) error { return nil }, nil
	}
	return tracerProvider.Shutdown, nil
}

// Tracer returns a named tracer, defaulting to the global provider.
func Tracer(component string) trace.Tracer {
	name := tracerName
	if strings.TrimSpace(component) != "" {
		name = name + "/" + component
	}
	return otel.Tracer(name)
}

// StartSpan is a convenience wrapper around Tracer(component).Start.
func StartSpan(ctx context.Context, component, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer(component).Start(ctx, spanName, opts...)
}

func hostname() string {
	if host, err := os.Hostname(); err == nil && host != "" {
		return host
	}
	return "unknown"
}
