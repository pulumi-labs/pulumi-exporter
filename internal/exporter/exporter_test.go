package exporter

import (
	"errors"
	"testing"
)

func TestNewExporterHTTP(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4318",
		Protocol: protocolHTTPProtobuf,
		Insecure: true,
	}

	exp, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err != nil {
		t.Fatalf("NewExporter() returned unexpected error: %v", err)
	}
	defer shutdown(t, exp)

	if m := exp.Meter(); m == nil {
		t.Fatal("Meter() returned nil")
	}
}

func TestNewExporterGRPC(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4317",
		Protocol: protocolGRPC,
		Insecure: true,
	}

	exp, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err != nil {
		t.Fatalf("NewExporter() returned unexpected error: %v", err)
	}
	defer shutdown(t, exp)

	if m := exp.Meter(); m == nil {
		t.Fatal("Meter() returned nil")
	}
}

func TestNewExporterInvalidProtocol(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4317",
		Protocol: "invalid",
		Insecure: true,
	}

	_, err := NewExporter(ctx, cfg, "0.0.1-test")
	if !errors.Is(err, ErrUnsupportedProtocol) {
		t.Fatalf("NewExporter() error = %v, want %v", err, ErrUnsupportedProtocol)
	}
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cfg := &OTLPConfig{
		Endpoint: "localhost:4318",
		Protocol: protocolHTTPProtobuf,
		Insecure: true,
	}

	exp, err := NewExporter(ctx, cfg, "0.0.1-test")
	if err != nil {
		t.Fatalf("NewExporter() returned unexpected error: %v", err)
	}

	shutdown(t, exp)
}

// shutdown shuts the exporter down. It may return a connection error when no
// collector is running, which is expected in tests, so the error is only logged.
func shutdown(t *testing.T, exp *Exporter) {
	t.Helper()

	if err := exp.Shutdown(t.Context()); err != nil {
		t.Logf("shutdown: %v", err)
	}
}
