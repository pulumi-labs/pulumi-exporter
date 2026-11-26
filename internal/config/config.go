// Package config provides configuration loading from flags, env vars, and YAML files.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"golang.org/x/net/http/httpguts"
	"gopkg.in/yaml.v3"
)

const (
	protocolHTTPProtobuf = "http/protobuf"
	protocolGRPC         = "grpc"

	minConcurrency = 1
	maxConcurrency = 100
)

var (
	// ErrMissingAccessToken is returned when no Pulumi access token is configured.
	ErrMissingAccessToken = errors.New("pulumi access token is required")
	// ErrMissingOrganizations is returned when no Pulumi organization is configured.
	ErrMissingOrganizations = errors.New("at least one pulumi organization is required")
	// ErrInvalidConcurrency is returned when max-concurrency is out of range.
	ErrInvalidConcurrency = errors.New("invalid max-concurrency")
	// ErrUnsupportedProtocol is returned for an OTLP protocol other than http/protobuf or grpc.
	ErrUnsupportedProtocol = errors.New("unsupported OTLP protocol")
	// ErrMalformedHeader is returned for an OTLP header entry without "=".
	ErrMalformedHeader = errors.New("expected key=value")
	// ErrInvalidHeaderKey is returned for an OTLP header key that is not a valid HTTP field name.
	ErrInvalidHeaderKey = errors.New("invalid HTTP header field name")
	// ErrInvalidHeaderValue is returned for an OTLP header value with an invalid percent-encoding.
	ErrInvalidHeaderValue = errors.New("invalid percent-encoding in header value (write a literal % as %25)")
)

// Config holds the complete application configuration.
type Config struct {
	Pulumi    PulumiConfig    `yaml:"pulumi"`
	Exporters ExportersConfig `yaml:"otlp"`
}

// PulumiConfig holds Pulumi Cloud API configuration.
type PulumiConfig struct {
	AccessToken     string        `yaml:"access-token"`
	APIURL          string        `yaml:"api-url"`
	Organizations   []string      `yaml:"organizations"`
	CollectInterval time.Duration `yaml:"collect-interval"`
	MaxConcurrency  int           `yaml:"max-concurrency"`
}

// ExportersConfig holds exporter configuration.
type ExportersConfig struct {
	Endpoint string            `yaml:"endpoint"`
	URLPath  string            `yaml:"url-path"`
	Protocol string            `yaml:"protocol"`
	Insecure bool              `yaml:"insecure"`
	Headers  map[string]string `yaml:"headers"`
}

// RegisterFlags registers CLI flags on the given kingpin application and returns a Config.
// Call this before kingpin.Parse(). After Parse(), the Config will be populated with
// flag values, env var overrides, and defaults.
func RegisterFlags(app *kingpin.Application) *Config {
	cfg := &Config{}

	app.Flag("pulumi.access-token", "Pulumi Cloud access token.").
		Envar("PULUMI_ACCESS_TOKEN").
		StringVar(&cfg.Pulumi.AccessToken)

	app.Flag("pulumi.api-url", "Pulumi Cloud API URL.").
		Default("https://api.pulumi.com").
		Envar("PULUMI_API_URL").
		StringVar(&cfg.Pulumi.APIURL)

	app.Flag("pulumi.organizations", "Pulumi organizations to monitor.").
		Envar("PULUMI_ORGANIZATIONS").
		StringsVar(&cfg.Pulumi.Organizations)

	app.Flag("pulumi.collect-interval", "Metrics collection interval.").
		Default("60s").
		Envar("PULUMI_COLLECT_INTERVAL").
		DurationVar(&cfg.Pulumi.CollectInterval)

	app.Flag("pulumi.max-concurrency", "Maximum number of concurrent stack API calls (1-100).").
		Default("10").
		Envar("PULUMI_MAX_CONCURRENCY").
		IntVar(&cfg.Pulumi.MaxConcurrency)

	app.Flag("otlp.endpoint", "OTLP exporter endpoint.").
		Default("localhost:4318").
		Envar("OTEL_EXPORTER_OTLP_ENDPOINT").
		StringVar(&cfg.Exporters.Endpoint)

	app.Flag("otlp.protocol", "OTLP exporter protocol (http/protobuf or grpc).").
		Default(protocolHTTPProtobuf).
		Envar("OTEL_EXPORTER_OTLP_PROTOCOL").
		StringVar(&cfg.Exporters.Protocol)

	app.Flag("otlp.url-path", "OTLP metrics URL path (e.g. /api/v1/otlp/v1/metrics for Prometheus).").
		Envar("OTEL_EXPORTER_OTLP_METRICS_URL_PATH").
		StringVar(&cfg.Exporters.URLPath)

	app.Flag("otlp.insecure", "Disable TLS for OTLP exporter.").
		Default("false").
		Envar("OTEL_EXPORTER_OTLP_INSECURE").
		BoolVar(&cfg.Exporters.Insecure)

	return cfg
}

// LoadFile loads a YAML configuration file and overlays it onto the existing config.
func (c *Config) LoadFile(path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err = yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	return nil
}

// ParseHeaders parses a comma-separated list of key=value pairs into the Headers map.
// The format follows the OpenTelemetry OTEL_EXPORTER_OTLP_HEADERS specification:
// values are percent-decoded, and keys must be valid HTTP header field names.
// Empty entries, such as the one a trailing comma produces, are skipped.
//
// Header values are usually credentials, so errors name the entry by its
// 1-based position and never include the raw entry or value.
func (c *Config) ParseHeaders(raw string) error {
	headers := make(map[string]string)
	position := 0
	for pair := range strings.SplitSeq(raw, ",") {
		position++
		if strings.TrimSpace(pair) == "" {
			continue
		}

		rawKey, rawValue, found := strings.Cut(pair, "=")
		if !found {
			return fmt.Errorf("parsing OTLP header entry %d: %w", position, ErrMalformedHeader)
		}

		key := strings.TrimSpace(rawKey)
		if !httpguts.ValidHeaderFieldName(key) {
			return fmt.Errorf("parsing OTLP header entry %d: %w", position, ErrInvalidHeaderKey)
		}

		value, err := url.PathUnescape(rawValue)
		if err != nil {
			// The url.EscapeError text quotes part of the value, so drop it.
			return fmt.Errorf("parsing OTLP header %q (entry %d): %w", key, position, ErrInvalidHeaderValue)
		}

		headers[key] = strings.TrimSpace(value)
	}

	if len(headers) > 0 {
		c.Exporters.Headers = headers
	}
	return nil
}

// NormalizeOrganizations splits comma-separated organization values into individual entries.
// This handles the case where PULUMI_ORGANIZATIONS env var contains "org1,org2,org3".
func (c *Config) NormalizeOrganizations() {
	var normalized []string
	for _, org := range c.Pulumi.Organizations {
		for part := range strings.SplitSeq(org, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				normalized = append(normalized, trimmed)
			}
		}
	}
	c.Pulumi.Organizations = normalized
}

// Validate checks that required configuration values are present.
func (c *Config) Validate() error {
	if c.Pulumi.AccessToken == "" {
		return ErrMissingAccessToken
	}

	if len(c.Pulumi.Organizations) == 0 {
		return ErrMissingOrganizations
	}

	if c.Pulumi.MaxConcurrency < minConcurrency || c.Pulumi.MaxConcurrency > maxConcurrency {
		return fmt.Errorf("max-concurrency must be between %d and %d, got %d: %w",
			minConcurrency, maxConcurrency, c.Pulumi.MaxConcurrency, ErrInvalidConcurrency)
	}

	switch c.Exporters.Protocol {
	case protocolHTTPProtobuf, protocolGRPC:
		// valid
	default:
		return fmt.Errorf("%w: %q (must be %s or %s)",
			ErrUnsupportedProtocol, c.Exporters.Protocol, protocolHTTPProtobuf, protocolGRPC)
	}

	return nil
}
