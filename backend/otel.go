package main

import (
	"context"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func initOpenTelemetry(ctx context.Context, serviceName string) (shutdown func(context.Context) error, _ error) {
	endpoint, insecure := otlpEndpointFromEnv()
	headers := otlpHeadersFromEnv()

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(headers))
	}

	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

func otlpEndpointFromEnv() (endpoint string, insecure bool) {
	if v := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); v != "" {
		return normalizeOTLPEndpoint(v)
	}
	if v := strings.TrimSpace(os.Getenv("ELASTIC_APM_SERVER_URL")); v != "" {
		return normalizeOTLPEndpoint(v)
	}
	return "localhost:8200", true
}

func normalizeOTLPEndpoint(v string) (endpoint string, insecure bool) {
	// Accept either:
	// - host:port
	// - http(s)://host:port
	if strings.Contains(v, "://") {
		u, err := url.Parse(v)
		if err == nil && u.Host != "" {
			return u.Host, u.Scheme != "https"
		}
	}
	// Assume host:port.
	return strings.TrimRight(v, "/"), true
}

func otlpHeadersFromEnv() map[string]string {
	// Elastic APM Server supports OTLP with the same secret token.
	// For OTLP/HTTP this is typically passed as Authorization header.
	if token := strings.TrimSpace(os.Getenv("ELASTIC_APM_SECRET_TOKEN")); token != "" {
		return map[string]string{"Authorization": "Bearer " + token}
	}
	return nil
}
