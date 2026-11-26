// Package pulumiexporter provides the entrypoint for the pulumi-exporter application.
package pulumiexporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"

	"github.com/pulumi-labs/pulumi-exporter/internal/appinfo"
	"github.com/pulumi-labs/pulumi-exporter/internal/client"
	"github.com/pulumi-labs/pulumi-exporter/internal/collector"
	"github.com/pulumi-labs/pulumi-exporter/internal/config"
	"github.com/pulumi-labs/pulumi-exporter/internal/exporter"
)

const (
	// readHeaderTimeout bounds how long the health server waits for request headers.
	readHeaderTimeout = 5 * time.Second
	// The shutdown steps have separate budgets that add up to 25s, under the
	// 30s default Kubernetes termination grace period.

	// serverShutdownTimeout bounds the graceful shutdown of the health server.
	serverShutdownTimeout = 5 * time.Second
	// collectorStopTimeout bounds the wait for an in-flight collection.
	collectorStopTimeout = 10 * time.Second
	// flushTimeout bounds the final exporter flush.
	flushTimeout = 10 * time.Second
)

// Main is the entrypoint for the pulumi-exporter application. It returns the
// process exit code.
func Main() int {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// options holds the settings that are flags only and not part of config.Config.
type options struct {
	listenAddr string
}

func run() error {
	cfg, opts, err := parseConfig(os.Args[1:])
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.InfoContext(ctx, "starting pulumi-exporter", "version", appinfo.Version)

	// Create the Pulumi API client first: it holds no resources, so a bad API
	// URL fails before the exporter exists.
	apiClient, err := client.NewClient(cfg.Pulumi.APIURL, cfg.Pulumi.AccessToken)
	if err != nil {
		return fmt.Errorf("creating API client: %w", err)
	}

	exp, err := exporter.NewExporter(ctx, &exporter.OTLPConfig{
		Endpoint: cfg.Exporters.Endpoint,
		URLPath:  cfg.Exporters.URLPath,
		Protocol: cfg.Exporters.Protocol,
		Insecure: cfg.Exporters.Insecure,
		Headers:  cfg.Exporters.Headers,
	}, appinfo.Version)
	if err != nil {
		return fmt.Errorf("creating exporter: %w", err)
	}

	coll, err := collector.NewCollector(apiClient, cfg, exp.Meter(), logger)
	if err != nil {
		flushCtx, flushCancel := context.WithTimeout(context.WithoutCancel(ctx), flushTimeout)
		defer flushCancel()
		return errors.Join(fmt.Errorf("creating collector: %w", err), exp.Shutdown(flushCtx))
	}

	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
		if runErr := coll.Run(ctx); runErr != nil && ctx.Err() == nil {
			logger.ErrorContext(ctx, "collector error", "error", runErr)
		}
	}()

	srv := newHealthServer(opts.listenAddr)
	go func() {
		logger.InfoContext(ctx, "health check server listening", "address", opts.listenAddr)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "health check server error", "error", serveErr)
		}
	}()

	<-ctx.Done()
	// signal.NotifyContext sets a cause that names the received signal.
	logger.InfoContext(ctx, "shutting down", "reason", context.Cause(ctx).Error())
	// Stop capturing signals, so a second Ctrl-C or SIGTERM force-stops the
	// process while shutdown is still running.
	cancel()

	shutdown(context.WithoutCancel(ctx), logger, srv, collectorDone, exp)
	return nil
}

// shutdown stops the health server, waits for the collector to finish its
// in-flight collection, and then flushes the exporter. Each step has its own
// timeout, so a stuck collector cannot use up the time the flush needs.
func shutdown(ctx context.Context, logger *slog.Logger, srv *http.Server, collectorDone <-chan struct{}, exp *exporter.Exporter) {
	srvCtx, srvCancel := context.WithTimeout(ctx, serverShutdownTimeout)
	defer srvCancel()
	if err := srv.Shutdown(srvCtx); err != nil {
		logger.ErrorContext(srvCtx, "health check server shutdown error", "error", err)
	}

	// Let an in-flight collection finish recording before the exporter flushes.
	select {
	case <-collectorDone:
	case <-time.After(collectorStopTimeout):
		logger.WarnContext(ctx, "collector did not stop before the timeout", "timeout", collectorStopTimeout.String())
	}

	flushCtx, flushCancel := context.WithTimeout(ctx, flushTimeout)
	defer flushCancel()
	if err := exp.Shutdown(flushCtx); err != nil {
		logger.ErrorContext(flushCtx, "exporter shutdown error", "error", err)
	}

	logger.InfoContext(ctx, "pulumi-exporter stopped")
}

// parseConfig parses flags, environment variables, and the optional config
// file into a validated configuration.
func parseConfig(args []string) (*config.Config, options, error) {
	app := kingpin.New("pulumi-exporter", "OpenTelemetry Pulumi Cloud metrics exporter.")
	app.Version(appinfo.Print("pulumi-exporter"))
	app.HelpFlag.Short('h')

	cfg := config.RegisterFlags(app)

	configFile := app.Flag("config.file", "Path to configuration file.").
		Envar("PULUMI_EXPORTER_CONFIG_FILE").
		String()

	headersRaw := app.Flag("otlp.headers", "OTLP headers as comma-separated key=value pairs.").
		Envar("OTEL_EXPORTER_OTLP_HEADERS").
		String()

	listenAddr := app.Flag("web.listen-address", "Address to listen on for health checks.").
		Default(":8080").
		Envar("PULUMI_EXPORTER_LISTEN_ADDRESS").
		String()

	if _, err := app.Parse(args); err != nil {
		return nil, options{}, fmt.Errorf("parsing flags: %w", err)
	}

	if *configFile != "" {
		if err := cfg.LoadFile(*configFile); err != nil {
			return nil, options{}, fmt.Errorf("loading config: %w", err)
		}
	}

	if err := cfg.ParseHeaders(*headersRaw); err != nil {
		return nil, options{}, fmt.Errorf("configuration error: %w", err)
	}
	cfg.NormalizeOrganizations()

	if err := cfg.Validate(); err != nil {
		return nil, options{}, fmt.Errorf("configuration error: %w", err)
	}

	return cfg, options{listenAddr: *listenAddr}, nil
}

// newHealthServer returns an HTTP server that answers the liveness and
// readiness probes.
func newHealthServer(addr string) *http.Server {
	ok := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", ok)
	mux.HandleFunc("/readyz", ok)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}
