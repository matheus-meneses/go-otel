package trace

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	_ "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ProviderConfig represents the provider configuration and used to create a new
// `Provider` type.
type ProviderConfig struct {
	ProviderEndpoint string
	ServiceName      string
	ServiceVersion   string
	Environment      string
	// Set this to `true` if you want to disable tracing completly.
	Disabled bool
}

// Provider represents the tracer provider. Depending on the `config.Disabled`
// parameter, it will either use a "live" provider or a "no operations" version.
// The "no operations" means, tracing will be globally disabled.
type Provider struct {
	provider trace.TracerProvider
}

// NewProvider returns a new `Provider` type. It uses Jaeger exporter and globally sets
// the tracer provider as well as the global tracer for spans.
func NewProvider(config ProviderConfig) (Provider, error) {
	if config.Disabled {
		return Provider{provider: noop.NewTracerProvider()}, nil
	}

	exp, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(config.ProviderEndpoint),
			otlptracegrpc.WithInsecure()))
	//otlptracehttp.NewClient(
	//	otlptracehttp.WithEndpoint(config.ProviderEndpoint),
	//	otlptracehttp.WithInsecure()))
	if err != nil {
		return Provider{}, err
	}

	prv := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(sdkresource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		)),
	)
	otel.SetTracerProvider(prv)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	_ = struct{ trace.Tracer }{otel.Tracer(config.ServiceName)}

	return Provider{provider: prv}, nil
}

// Close shuts down the tracer provider only if it was not "no operations"
// version.
func (p Provider) Close(ctx context.Context) error {
	if prv, ok := p.provider.(*sdktrace.TracerProvider); ok {
		return prv.Shutdown(ctx)
	}

	return nil
}
