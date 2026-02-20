# pulumi-exporter

Standalone OpenTelemetry metrics exporter for [Pulumi Cloud](https://www.pulumi.com/product/pulumi-cloud/). Polls the Pulumi API on a configurable interval and pushes metrics over OTLP to any observability backend.

```
                                              ┌──OTLP/HTTP──▶ DataDog, NewRelic, Dash0, Prometheus 2.47+
Pulumi Cloud API  ◀──poll──  [pulumi-exporter]┤
                                              └──OTLP/gRPC──▶ Dynatrace, OTel Collector, Grafana Alloy
```

## Quick Start

```bash
cd deploy/docker-compose
cp .env.example .env
# Set PULUMI_ACCESS_TOKEN and PULUMI_ORGANIZATIONS in .env
make compose-up
```

Open **Grafana** at http://localhost:3000 (admin / admin) — the dashboard loads automatically.

## Install

### Helm (recommended for Kubernetes)

```bash
helm install pulumi-exporter oci://ghcr.io/dirien/charts/pulumi-exporter \
  --set existingSecret=pulumi-credentials \
  --set "pulumiOrganizations={my-org}" \
  --set otlp.endpoint=otel-collector:4318 \
  --set otlp.insecure=true
```

See the [chart README](charts/pulumi-exporter/README.md) for all values.

### Docker

```bash
docker run --rm \
  -e PULUMI_ACCESS_TOKEN=pul-xxx \
  -e PULUMI_ORGANIZATIONS=my-org \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4318 \
  -e OTEL_EXPORTER_OTLP_INSECURE=true \
  ghcr.io/dirien/pulumi-exporter:latest
```

### Binary

```bash
curl -sSfL https://github.com/dirien/pulumi-exporter/releases/latest/download/pulumi-exporter_linux_amd64.tar.gz | tar xz
./pulumi-exporter --pulumi.organizations=my-org --otlp.endpoint=localhost:4318 --otlp.insecure
```

All release artifacts are signed with [Cosign](https://github.com/sigstore/cosign) and include an SBOM.

## Metrics

17 metrics across stacks and organizations:

| Scope | Metrics |
|-------|---------|
| **Stack** | `resource_count`, `last_update_timestamp`, `update_total`, `update_duration_seconds`, `update_resource_changes`, `deployment_status` |
| **Organization** | `member_count`, `team_count`, `environment_count`, `policy_group_count`, `policy_pack_count`, `policy_violations`, `neo_task_count` |
| **Compliance** | `policy_total`, `policy_with_issues`, `governed_resources_total`, `governed_resources_with_issues` |

All metric names are prefixed with `pulumi_` (stack) or `pulumi_org_` (org/compliance). Full details in [docs/metrics.md](docs/metrics.md).

## Makefile

Run `make help` to see all targets:

| Target | Description |
|--------|-------------|
| `make` | Lint, test, and build |
| `make build` | Build the binary |
| `make test-race` | Run tests with race detector |
| `make lint` | Run golangci-lint |
| `make generate` | Regenerate OpenAPI client |
| `make helm-test` | Lint + template + helm-docs + ct lint |
| `make helm-test-e2e` | Full e2e: kind + build + ct install |
| `make compose-up` | Start local Prometheus + Grafana stack |
| `make compose-down` | Stop local stack |
| `make compose-logs` | Tail exporter logs |
| `make compose-restart` | Rebuild + restart exporter only |

## Documentation

| Document | Contents |
|----------|----------|
| [Configuration](docs/configuration.md) | Flags, env vars, YAML config, multi-org, large orgs |
| [Metrics Reference](docs/metrics.md) | All 17 metrics with types, labels, and histogram buckets |
| [Backend Setup](docs/backends.md) | Prometheus, Grafana Alloy, DataDog, NewRelic, Dynatrace |
| [Kubernetes & Helm](docs/kubernetes.md) | Helm chart, raw manifests, chart CI/CD |
| [Development](docs/development.md) | Build, test, project structure, OpenAPI, contributing |

## License

Apache-2.0
