# pulumi-exporter

Standalone OpenTelemetry metrics exporter for [Pulumi Cloud](https://www.pulumi.com/product/pulumi-cloud/). Polls the Pulumi API on a configurable interval and pushes metrics over OTLP to any observability backend.

```
                                              ┌──OTLP/HTTP──▶ DataDog, NewRelic, Dash0, Prometheus 2.47+
Pulumi Cloud API  ◀──poll──  [pulumi-exporter]┤
                                              └──OTLP/gRPC──▶ Dynatrace, OTel Collector, Grafana Alloy
```

The API client is generated from the official [Pulumi Cloud OpenAPI spec](https://www.pulumi.com/blog/announcing-openapi-support-pulumi-cloud/) using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen).

## Install

### Binary (GitHub Releases)

Download the latest release for your platform from the [releases page](https://github.com/dirien/pulumi-exporter/releases). Binaries are available for Linux, macOS, and Windows on both amd64 and arm64.

```bash
# macOS / Linux
curl -sSfL https://github.com/dirien/pulumi-exporter/releases/latest/download/pulumi-exporter_linux_amd64.tar.gz | tar xz
chmod +x pulumi-exporter
sudo mv pulumi-exporter /usr/local/bin/
```

All release artifacts are signed with [Cosign](https://github.com/sigstore/cosign) and include an SBOM.

### Docker

Multi-arch images are published to GitHub Container Registry on every release:

```bash
docker pull ghcr.io/dirien/pulumi-exporter:latest
```

The image is built on `cgr.dev/chainguard/static` (distroless, zero CVEs).

```bash
docker run --rm \
  -e PULUMI_ACCESS_TOKEN=pul-xxx \
  -e PULUMI_ORGANIZATIONS=my-org \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4318 \
  -e OTEL_EXPORTER_OTLP_INSECURE=true \
  -p 8080:8080 \
  ghcr.io/dirien/pulumi-exporter:latest
```

### Build from Source

Requires Go 1.24+.

```bash
git clone https://github.com/dirien/pulumi-exporter.git
cd pulumi-exporter
make build
```

## Quick Start

### Option A: Docker Compose (recommended)

The fastest way to see metrics end-to-end. Spins up Prometheus, Grafana, and the exporter with a pre-built dashboard.

```bash
cd deploy/docker-compose
cp .env.example .env
# Edit .env and set PULUMI_ACCESS_TOKEN and PULUMI_ORGANIZATIONS
docker compose up --build -d
```

Open:
- **Grafana**: http://localhost:3000 (admin / admin) -- dashboard loads as the home page
- **Prometheus**: http://localhost:9090

### Option B: Binary with OTel Collector

#### 1. Get a Pulumi access token

Create one at [app.pulumi.com/account/tokens](https://app.pulumi.com/account/tokens).

#### 2. Start a local OTel Collector

```bash
docker run -d --name otel-collector -p 4318:4318 otel/opentelemetry-collector:latest
```

#### 3. Run the exporter

```bash
export PULUMI_ACCESS_TOKEN=pul-xxx

./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=localhost:4318 \
  --otlp.insecure
```

#### 4. Verify

```bash
curl http://localhost:8080/healthz
# ok
```

## Configuration

Configure via CLI flags, environment variables, or a YAML file. Flags take precedence over env vars, and env vars take precedence over the config file.

### Flags and Environment Variables

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
| `--otlp.headers` | `OTEL_EXPORTER_OTLP_HEADERS` | *(empty)* | Comma-separated `key=value` pairs |
| `--otlp.url-path` | `OTEL_EXPORTER_OTLP_METRICS_URL_PATH` | *(default OTel path)* | Custom URL path for OTLP metrics endpoint |
| `--config.file` | `PULUMI_EXPORTER_CONFIG_FILE` | *(none)* | Path to YAML config file |
| `--web.listen-address` | `PULUMI_EXPORTER_LISTEN_ADDRESS` | `:8080` | Health check listen address |

OTLP environment variable names follow the [OpenTelemetry SDK specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/).

### Multiple Organizations

The exporter supports monitoring multiple Pulumi organizations simultaneously. Configure them as a comma-separated list via the env var or as a YAML list:

```bash
# Via environment variable (comma-separated)
PULUMI_ORGANIZATIONS=my-org,another-org

# Via CLI flags (repeatable)
--pulumi.organizations=my-org --pulumi.organizations=another-org
```

All metrics include an `org` label so you can filter, group, and compare across organizations in your dashboards. The Grafana dashboard includes a multi-select Organization dropdown.

The `org` identity is modeled as an [OTel metric attribute](https://opentelemetry.io/docs/specs/semconv/resource/) (not a Resource attribute), which is the correct pattern for an exporter that observes multiple tenants from a single process -- matching how tools like Prometheus blackbox_exporter handle multi-target monitoring.

### Large Organizations

Each collection cycle makes 2 API calls per stack (resource count + updates) plus 8–10 calls per org. The collect interval must be long enough for all calls to complete. A cycle that exceeds 90% of the interval is cancelled to prevent overlap.

Recommended settings based on total stack count across all monitored orgs:

| Stacks | Interval | Concurrency | Notes |
|--------|----------|-------------|-------|
| < 50 | `60s` | `10` | Default settings work fine |
| 50–200 | `2m` | `15` | |
| 200–500 | `3m` | `20` | |
| 500–1000 | `5m` | `30` | Tested: 500+ stacks across 3 orgs completes in ~2 min |
| 1000+ | `10m` | `50` | Watch for API rate limits |

Example for a large org:

```bash
PULUMI_COLLECT_INTERVAL=5m \
PULUMI_MAX_CONCURRENCY=30 \
  ./pulumi-exporter --pulumi.organizations=my-large-org ...
```

If you see `context deadline exceeded` errors in the logs, increase the collect interval. The exporter automatically paginates all org-level API responses (members, environments, etc.) and parallelizes org-level metric collection. Collection cycles are bounded by a timeout of 90% of the collect interval (minimum 10 seconds) to prevent overlap.

### YAML Config File

```yaml
pulumi:
  access-token: "pul-xxx"
  api-url: "https://api.pulumi.com"
  organizations:
    - "my-org"
    - "another-org"
  collect-interval: 60s

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

See [`config.example.yaml`](config.example.yaml) for a full template.

## Metrics

### Stack Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_stack_resource_count` | Gauge | `org`, `project`, `stack` | Number of resources in a stack |
| `pulumi_update_duration_seconds` | Histogram | `org`, `project`, `stack`, `kind`, `result` | Duration of stack updates (seconds) |
| `pulumi_update_total` | Counter | `org`, `project`, `stack`, `kind`, `result` | Total number of stack updates |
| `pulumi_update_resource_changes` | Counter | `org`, `project`, `stack`, `kind`, `operation` | Resource changes per update |
| `pulumi_deployment_status` | Gauge | `org`, `status` | Deployments by status |
| `pulumi_stack_last_update_timestamp` | Gauge | `org`, `project`, `stack` | Unix timestamp of last update |

### Organization Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `pulumi_org_member_count` | Gauge | `org` | Number of organization members |
| `pulumi_org_team_count` | Gauge | `org` | Number of teams |
| `pulumi_org_environment_count` | Gauge | `org` | Number of ESC environments |
| `pulumi_org_policy_group_count` | Gauge | `org` | Number of policy groups |
| `pulumi_org_policy_pack_count` | Gauge | `org` | Number of policy packs |
| `pulumi_org_policy_violations` | Gauge | `org`, `level`, `kind` | Policy violations by severity and type |
| `pulumi_org_neo_task_count` | Gauge | `org`, `status` | Pulumi Neo AI tasks by status |
| `pulumi_org_policy_total` | Gauge | `org` | Total number of policies |
| `pulumi_org_policy_with_issues` | Gauge | `org` | Number of policies with issues |
| `pulumi_org_governed_resources_total` | Gauge | `org` | Total governed resources |
| `pulumi_org_governed_resources_with_issues` | Gauge | `org` | Governed resources with issues |

### Label Values

- **`kind`**: `update`, `preview`, `destroy`, `refresh`, `import`
- **`result`**: `succeeded`, `failed`, `in-progress`
- **`operation`**: `create`, `update`, `delete`, `same`, `replace`
- **`status`** (deployments): `running`, `succeeded`, `failed`, `not-started`, `accepted`
- **`status`** (Neo tasks): `idle`, `running`
- **`level`** (violations): `advisory`, `mandatory`, `disabled`
- **`kind`** (violations): `preventative`, `audit`

### Histogram Buckets

`pulumi_update_duration_seconds` uses bucket boundaries tuned for IaC operations:

```
5s, 10s, 30s, 1m, 2m, 5m, 10m, 30m
```

## Backend Setup

### Docker Compose (Prometheus + Grafana)

The included Docker Compose stack provides a ready-to-use setup with a pre-built Grafana dashboard.

```bash
cd deploy/docker-compose
cp .env.example .env
# Set PULUMI_ACCESS_TOKEN and PULUMI_ORGANIZATIONS in .env
docker compose up --build -d
```

- **Grafana**: http://localhost:3000 (admin / admin)
- **Prometheus**: http://localhost:9090

The exporter pushes metrics to Prometheus via its native OTLP receiver (`--web.enable-otlp-receiver`). The Grafana dashboard is auto-provisioned with 26 panels covering all 17 metrics.

To stop:

```bash
docker compose down
# To also remove data volumes:
docker compose down -v
```

### Prometheus (standalone, v2.47+)

Enable the OTLP receiver in Prometheus:

```bash
prometheus --config.file=prometheus.yml --web.enable-otlp-receiver
```

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=localhost:9090 \
  --otlp.url-path=/api/v1/otlp/v1/metrics \
  --otlp.insecure
```

### Grafana Alloy / OTel Collector

Point the exporter at your Alloy or Collector's OTLP receiver:

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=alloy:4317 \
  --otlp.protocol=grpc \
  --otlp.insecure
```

### DataDog

Use the DataDog Agent's OTLP ingestion or send directly:

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=<DD_AGENT_HOST>:4318 \
  --otlp.insecure
```

### NewRelic

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=otlp.nr-data.net:4318 \
  --otlp.headers="api-key=<NEWRELIC_LICENSE_KEY>"
```

### Dynatrace

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=<ENV_ID>.live.dynatrace.com:4317 \
  --otlp.protocol=grpc \
  --otlp.headers="Authorization=Api-Token <DT_TOKEN>"
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pulumi-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pulumi-exporter
  template:
    metadata:
      labels:
        app: pulumi-exporter
    spec:
      containers:
        - name: pulumi-exporter
          image: ghcr.io/dirien/pulumi-exporter:latest
          args:
            - --pulumi.organizations=my-org
            - --otlp.endpoint=otel-collector.monitoring:4318
            - --otlp.insecure
          env:
            - name: PULUMI_ACCESS_TOKEN
              valueFrom:
                secretKeyRef:
                  name: pulumi-credentials
                  key: access-token
          ports:
            - name: health
              containerPort: 8080
          livenessProbe:
            httpGet:
              path: /healthz
              port: health
            initialDelaySeconds: 5
          readinessProbe:
            httpGet:
              path: /readyz
              port: health
            initialDelaySeconds: 5
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              memory: 128Mi
```

## Development

### Prerequisites

- Go 1.24+
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2

### Makefile

Run `make help` to see all targets:

| Target | Description |
|--------|-------------|
| `make` | Lint, test, and build (default) |
| `make build` | Build the binary with version ldflags |
| `make test` | Run all tests |
| `make test-race` | Run tests with Go race detector |
| `make test-cover` | Generate HTML coverage report |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code (gofumpt + goimports) |
| `make vet` | Run `go vet` |
| `make generate` | Download Pulumi OpenAPI spec and regenerate Go client |
| `make download-spec` | Download the latest Pulumi Cloud OpenAPI spec |
| `make release-snapshot` | GoReleaser dry run (no publish) |
| `make docker` | Build Docker image locally |
| `make deps` | Download and tidy Go module dependencies |
| `make tools` | Install development tools (oapi-codegen) |
| `make clean` | Remove build artifacts |

### Build and Test

```bash
make build
make test
make test-race
make test-cover
```

### Lint

```bash
make lint
```

The project uses a strict linter configuration (`.golangci.yaml`) with gosec, revive, gocritic, gocyclo, and more. Formatting is enforced by gofumpt and goimports.

### OpenAPI Client Generation

All Pulumi Cloud API calls go through a generated Go client built from the official [Pulumi Cloud OpenAPI spec](https://api.pulumi.com/api/openapi/pulumi-spec.json). The `internal/client/` package provides a typed wrapper around the generated code in `internal/pulumiapi/`.

To regenerate after Pulumi updates their API:

```bash
make generate
```

This downloads the latest spec and runs [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to produce `internal/pulumiapi/client.gen.go`. Generation is scoped to the 13 operations the exporter uses (configured in `oapi-codegen.yaml`):

| Operation | Endpoint |
|-----------|----------|
| `ListUserStacks` | `GET /api/user/stacks` |
| `GetStackResourceCount` | `GET /api/stacks/{org}/{project}/{stack}/resources/count` |
| `GetStackUpdates` | `GET /api/stacks/{org}/{project}/{stack}/updates` |
| `ListStackDeploymentsHandlerV2` | `GET /api/stacks/{org}/{project}/{stack}/deployments` |
| `ListOrgDeployments` | `GET /api/orgs/{org}/deployments` |
| `ListOrganizationMembers` | `GET /api/orgs/{org}/members` |
| `ListTeams` | `GET /api/orgs/{org}/teams` |
| `ListOrgEnvironments_esc` | `GET /api/esc/environments/{org}` |
| `ListPolicyGroups` | `GET /api/orgs/{org}/policygroups` |
| `ListPolicyPacks_orgs` | `GET /api/orgs/{org}/policypacks` |
| `ListPolicyViolationsV2` | `GET /api/orgs/{org}/policyresults/violationsv2` |
| `ListTasks` | `GET /api/preview/agents/{org}/tasks` |
| `GetPolicyResultsMetadata` | `GET /api/orgs/{org}/policyresults/metadata` |

### Project Structure

```
pulumi-exporter/
├── main.go                              # Delegates to cmd/pulumiexporter
├── Makefile                             # Build, test, lint, generate targets
├── oapi-codegen.yaml                    # OpenAPI code generation config
├── cmd/pulumiexporter/
│   └── main.go                          # CLI flags, wiring, signal handling
├── internal/
│   ├── pulumiapi/                       # Generated OpenAPI client (DO NOT EDIT)
│   │   └── client.gen.go               # oapi-codegen output
│   ├── client/                          # Typed wrapper around generated client
│   │   ├── client.go                    # All API methods (stacks, orgs, ESC, Neo)
│   │   └── types.go                     # Response types used by the collector
│   ├── config/                          # CLI flags + env vars + YAML config
│   │   ├── config.go                    # Config struct, RegisterFlags, Validate
│   │   └── config_test.go
│   ├── collector/                       # Metrics collection logic
│   │   ├── collector.go                 # PulumiAPI interface, ticker loop
│   │   ├── instruments.go              # OTel instrument definitions (17 metrics)
│   │   ├── stack.go                     # Per-stack collection
│   │   ├── deployments.go              # Org deployment collection
│   │   ├── org.go                       # Org-level collection (members, teams, ESC, policies, Neo)
│   │   └── collector_test.go           # Mock API + ManualReader tests
│   ├── exporter/                        # OTel MeterProvider setup
│   │   ├── exporter.go                 # OTLP HTTP/gRPC exporter creation
│   │   └── exporter_test.go
│   └── version/                         # Build-time version info (ldflags)
│       └── version.go
├── deploy/
│   └── docker-compose/                  # Ready-to-run observability stack
│       ├── docker-compose.yaml          # Prometheus + Grafana + exporter
│       ├── Dockerfile                   # Multi-stage build for the exporter
│       ├── .env.example
│       ├── prometheus/prometheus.yml
│       └── grafana/
│           ├── provisioning/            # Auto-provisioned datasource + dashboard
│           └── dashboards/
│               └── pulumi-exporter.json # 26-panel Grafana dashboard
├── Dockerfile                           # Chainguard static base (release)
├── .goreleaser.yaml                     # Multi-arch builds, signing, SBOM
├── .golangci.yaml                       # Linter configuration
└── .github/workflows/
    ├── ci.yaml                          # Build, test, release
    └── lint.yaml                        # golangci-lint
```

### Running Locally

**With Docker Compose** (easiest):

```bash
cd deploy/docker-compose
cp .env.example .env
# Set PULUMI_ACCESS_TOKEN in .env
docker compose up --build -d
```

**With `make run`**:

```bash
PULUMI_ACCESS_TOKEN=pul-xxx make run \
  ARGS="--pulumi.organizations=my-org --otlp.endpoint=localhost:4318 --otlp.insecure"
```

**With `go run`**:

```bash
PULUMI_ACCESS_TOKEN=pul-xxx go run . \
  --pulumi.organizations=my-org \
  --otlp.endpoint=localhost:4318 \
  --otlp.insecure
```

### GoReleaser Dry Run

```bash
make release-snapshot
```

## Contributing

1. Fork and clone the repository
2. Install tools: `make tools`
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Make your changes and add tests
5. Verify:
   ```bash
   make test-race
   make lint
   ```
6. Commit using [Conventional Commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add something useful"
   ```
7. Push and open a pull request

### Adding a New Metric

1. Add the instrument to `internal/collector/instruments.go`
2. Record values in the appropriate collector file (`stack.go`, `deployments.go`, or `org.go`)
3. If the metric needs a new API endpoint:
   - Add the operationId to `oapi-codegen.yaml` and run `make generate`
   - Add a wrapper method to `internal/client/client.go`
   - Add response types to `internal/client/types.go`
   - Add the method to the `PulumiAPI` interface in `internal/collector/collector.go`
   - Add a stub to the mock in `internal/collector/collector_test.go`
4. Add tests

### Regenerating After Pulumi API Changes

```bash
make generate
make build
make test
```

## License

Apache-2.0
