# Configuration

Configure via CLI flags, environment variables, or a YAML file. Flags take precedence over env vars, and env vars take precedence over the config file.

## Flags and Environment Variables

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--pulumi.access-token` | `PULUMI_ACCESS_TOKEN` | *(required)* | Pulumi Cloud access token |
| `--pulumi.api-url` | `PULUMI_API_URL` | `https://api.pulumi.com` | Pulumi Cloud API base URL |
| `--pulumi.organizations` | `PULUMI_ORGANIZATIONS` | *(required)* | Organizations to monitor (repeatable, comma-separated) |
| `--pulumi.collect-interval` | `PULUMI_COLLECT_INTERVAL` | `60s` | Polling interval |
| `--pulumi.max-concurrency` | `PULUMI_MAX_CONCURRENCY` | `10` | Max concurrent stack API calls (1-100) |
| `--otlp.endpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4318` | OTLP receiver endpoint (host:port) |
| `--otlp.protocol` | `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | `http/protobuf` or `grpc` |
| `--otlp.insecure` | `OTEL_EXPORTER_OTLP_INSECURE` | `false` | Disable TLS |
| `--otlp.headers` | `OTEL_EXPORTER_OTLP_HEADERS` | *(empty)* | Comma-separated `key=value` pairs, percent-encoded (see [OTLP headers](#otlp-headers)) |
| `--otlp.url-path` | `OTEL_EXPORTER_OTLP_METRICS_URL_PATH` | *(default OTel path)* | Custom URL path for OTLP metrics endpoint |
| `--config.file` | `PULUMI_EXPORTER_CONFIG_FILE` | *(none)* | Path to YAML config file |
| `--web.listen-address` | `PULUMI_EXPORTER_LISTEN_ADDRESS` | `:8080` | Health check listen address |

OTLP environment variable names follow the [OpenTelemetry SDK specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/).

### OTLP headers

`--otlp.headers` / `OTEL_EXPORTER_OTLP_HEADERS` uses the [OpenTelemetry header format](https://opentelemetry.io/docs/specs/otel/protocol/exporter/#specifying-headers-via-environment-variables):

- Entries are separated by commas, and each entry is `key=value`. Only the first `=` splits an entry, so values may contain `=`.
- Values are percent-decoded. Write a literal `%` as `%25` and a literal `,` as `%2C`. Spaces may be written as-is or as `%20`.
- Keys must be valid HTTP header names.
- Empty entries, such as the one a trailing comma leaves, are ignored.

The exporter refuses to start when an entry has no `=`, an invalid key, or an invalid percent-encoding (for example `50%` instead of `50%25`). The error names the entry by its position and never prints the value, because header values are usually credentials.

```bash
OTEL_EXPORTER_OTLP_HEADERS="Authorization=Basic%20dXNlcjpwYXNz,x-scope-orgid=tenant-1"
```

Headers set with the flag or environment variable replace the `otlp.headers` map from the YAML file. YAML values are used verbatim and are not percent-decoded.

> **Upgrading to 0.1.9 from 0.1.8 or earlier:** those versions sent flag and environment header values verbatim. A value that contains a valid escape now changes without any error: `ab%41cd` is sent as `abAcd`. Before upgrading, check your header values for `%` and write each literal `%` as `%25`, or move the headers to the YAML file, where values are not decoded.

## YAML Config File

```yaml
pulumi:
  access-token: "pul-xxx"
  api-url: "https://api.pulumi.com"
  organizations:
    - "my-org"
    - "another-org"
  collect-interval: 60s
  max-concurrency: 10

otlp:
  endpoint: "localhost:4318"
  protocol: "http/protobuf"   # or "grpc"
  insecure: false
  url-path: ""                # e.g. /api/v1/otlp/v1/metrics for Prometheus
  headers:
    Authorization: "Bearer <token>"
```

```bash
./pulumi-exporter --config.file=config.yaml
```

See [`config.example.yaml`](../config.example.yaml) for a full template.

## Multiple Organizations

Monitor multiple orgs simultaneously:

```bash
# Via environment variable
PULUMI_ORGANIZATIONS=my-org,another-org

# Via CLI flags
--pulumi.organizations=my-org --pulumi.organizations=another-org
```

All metrics include an `org` label for filtering and grouping. The Grafana dashboard includes a multi-select Organization dropdown.

## Large Organizations

Each collection cycle makes 2 API calls per stack (resource count + updates) plus 8-10 calls per org. The collect interval must be long enough for all calls to complete. A cycle that exceeds 90% of the interval is cancelled to prevent overlap.

| Stacks | Interval | Concurrency | Notes |
|--------|----------|-------------|-------|
| < 50 | `60s` | `10` | Default settings work fine |
| 50-200 | `2m` | `15` | |
| 200-500 | `3m` | `20` | |
| 500-1000 | `5m` | `30` | Tested: 500+ stacks across 3 orgs completes in ~2 min |
| 1000+ | `10m` | `50` | Watch for API rate limits |

If you see `context deadline exceeded` errors, increase the collect interval.
