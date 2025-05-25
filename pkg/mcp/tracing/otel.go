package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration for OpenTelemetry tracing
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	UseHTTP        bool
	Headers        map[string]string
	Insecure       bool
}

// Tracer wraps OpenTelemetry tracer with convenience methods
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer initializes OpenTelemetry tracing and returns a configured tracer
func NewTracer(ctx context.Context, config Config) (*Tracer, func(context.Context) error, error) {
	// Create exporter
	var exporter *otlptrace.Exporter
	var err error

	if config.UseHTTP {
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(config.Endpoint),
		}
		if config.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(config.Headers))
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	} else {
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(config.Endpoint),
		}
		if config.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(config.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(config.Headers))
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer := tp.Tracer(
		config.ServiceName,
		trace.WithInstrumentationVersion(config.ServiceVersion),
	)

	// Shutdown function
	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	return &Tracer{tracer: tracer}, shutdown, nil
}

// StartSpan starts a new span with the given name
func (t *Tracer) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, opts...)
}

// StartSpanWithKind starts a new span with specific kind
func (t *Tracer) StartSpanWithKind(ctx context.Context, spanName string, kind trace.SpanKind) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName, trace.WithSpanKind(kind))
}

// TraceRequest creates a span for an MCP request
func (t *Tracer) TraceRequest(ctx context.Context, method string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	defaultAttrs := []attribute.KeyValue{
		attribute.String("mcp.method", method),
		attribute.String("mcp.protocol", "2.0"),
	}
	attrs = append(defaultAttrs, attrs...)
	
	return t.tracer.Start(ctx, fmt.Sprintf("MCP %s", method),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)
}

// TraceToolExecution creates a span for tool execution
func (t *Tracer) TraceToolExecution(ctx context.Context, toolName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	defaultAttrs := []attribute.KeyValue{
		attribute.String("mcp.tool.name", toolName),
		attribute.String("mcp.operation", "tool_execution"),
	}
	attrs = append(defaultAttrs, attrs...)
	
	return t.tracer.Start(ctx, fmt.Sprintf("Tool: %s", toolName),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)
}

// TraceResourceOperation creates a span for resource operations
func (t *Tracer) TraceResourceOperation(ctx context.Context, operation, resourceType string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	defaultAttrs := []attribute.KeyValue{
		attribute.String("mcp.resource.operation", operation),
		attribute.String("mcp.resource.type", resourceType),
	}
	attrs = append(defaultAttrs, attrs...)
	
	return t.tracer.Start(ctx, fmt.Sprintf("Resource %s: %s", operation, resourceType),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)
}

// TracePromptOperation creates a span for prompt operations
func (t *Tracer) TracePromptOperation(ctx context.Context, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	defaultAttrs := []attribute.KeyValue{
		attribute.String("mcp.prompt.operation", operation),
	}
	attrs = append(defaultAttrs, attrs...)
	
	return t.tracer.Start(ctx, fmt.Sprintf("Prompt: %s", operation),
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// SetStatus sets the status of the current span
func SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err, opts...)
	}
}

// WithSpan executes a function within a span
func (t *Tracer) WithSpan(ctx context.Context, spanName string, fn func(context.Context) error, opts ...trace.SpanStartOption) error {
	ctx, span := t.tracer.Start(ctx, spanName, opts...)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// WithRequestSpan executes a function within a request span
func (t *Tracer) WithRequestSpan(ctx context.Context, method string, fn func(context.Context) error) error {
	ctx, span := t.TraceRequest(ctx, method)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		SetAttributes(ctx, attribute.Bool("mcp.request.error", true))
	} else {
		span.SetStatus(codes.Ok, "")
		SetAttributes(ctx, attribute.Bool("mcp.request.success", true))
	}

	return err
}

// WithToolSpan executes a function within a tool execution span
func (t *Tracer) WithToolSpan(ctx context.Context, toolName string, fn func(context.Context) error) error {
	ctx, span := t.TraceToolExecution(ctx, toolName)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		SetAttributes(ctx, attribute.Bool("mcp.tool.error", true))
	} else {
		span.SetStatus(codes.Ok, "")
		SetAttributes(ctx, attribute.Bool("mcp.tool.success", true))
	}

	return err
}

// Extract extracts trace context from a carrier
func Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// Inject injects trace context into a carrier
func Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}