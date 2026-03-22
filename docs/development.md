# Development

## Prerequisites

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2
- [helm](https://helm.sh/docs/intro/install/) 3.8+
- [helm-docs](https://github.com/norwoodj/helm-docs)
- [ct](https://github.com/helm/chart-testing) (chart-testing)
- [kind](https://kind.sigs.k8s.io/) (for e2e chart tests)

Install Go tools:

```bash
make tools
```

## Makefile Targets

Run `make help` to see all targets:

### Build & Test

| Target | Description |
|--------|-------------|
| `make` | Lint, test, and build (default) |
| `make build` | Build the binary with version ldflags |
| `make test` | Run all tests |
| `make test-race` | Run tests with Go race detector |
| `make test-cover` | Generate HTML coverage report |
| `make clean` | Remove build artifacts |

### Code Quality

| Target | Description |
|--------|-------------|
| `make lint` | Run golangci-lint |
| `make fmt` | Format code (gofumpt + goimports) |
| `make vet` | Run `go vet` |

### Code Generation

| Target | Description |
|--------|-------------|
| `make generate` | Download Pulumi OpenAPI spec and regenerate Go client |
| `make download-spec` | Download the latest OpenAPI spec only |

### Helm

| Target | Description |
|--------|-------------|
| `make helm-lint` | `helm lint` the chart |
| `make helm-template` | Render templates with test values |
| `make helm-docs` | Regenerate chart README via helm-docs |
| `make ct-lint` | chart-testing lint (schema, YAML, maintainers) |
| `make helm-test` | All of the above in sequence |
| `make kind-create` | Create a kind cluster for e2e testing |
| `make ct-install` | Build image, load into kind, ct install |
| `make kind-delete` | Delete the kind cluster |
| `make helm-test-e2e` | kind-create + ct-install + kind-delete |

### Docker Compose

| Target | Description |
|--------|-------------|
| `make compose-up` | Start local Prometheus + Grafana + exporter stack |
| `make compose-down` | Stop local stack |
| `make compose-logs` | Tail exporter logs |
| `make compose-restart` | Rebuild and restart exporter only |

### Release

| Target | Description |
|--------|-------------|
| `make release-snapshot` | GoReleaser dry run (no publish) |
| `make docker` | Build Docker image locally |
| `make deps` | Download and tidy Go module dependencies |
| `make tools` | Install development tools (oapi-codegen, helm-docs) |

## Project Structure

```
pulumi-exporter/
├── main.go                              # Delegates to cmd/pulumiexporter
├── Makefile                             # Build, test, lint, helm, compose targets
├── oapi-codegen.yaml                    # OpenAPI code generation config
├── cmd/pulumiexporter/
│   └── main.go                          # CLI flags, wiring, signal handling
├── internal/
│   ├── pulumiapi/                       # Generated OpenAPI client (DO NOT EDIT)
│   │   └── client.gen.go
│   ├── client/                          # Typed wrapper around generated client
│   │   ├── client.go
│   │   └── types.go
│   ├── config/                          # CLI flags + env vars + YAML config
│   ├── collector/                       # Metrics collection logic
│   │   ├── collector.go                 # PulumiAPI interface, ticker loop
│   │   ├── instruments.go              # OTel instrument definitions (17 metrics)
│   │   ├── stack.go                     # Per-stack collection
│   │   ├── deployments.go              # Org deployment collection
│   │   ├── org.go                       # Org-level collection
│   │   └── collector_test.go
│   ├── exporter/                        # OTel MeterProvider setup
│   └── appinfo/                         # Build-time version info (ldflags)
├── dashboards/                          # Grafana dashboard JSON
├── charts/pulumi-exporter/              # Helm chart
├── deploy/
│   ├── docker-compose/                  # Prometheus + Grafana + exporter
│   ├── pulumi-yaml/                     # Pulumi YAML deployment
│   ├── pulumi-typescript/               # Pulumi TypeScript deployment
│   └── pulumi-python/                   # Pulumi Python deployment
├── docs/                                # Documentation
├── .github/
│   ├── configs/                         # ct-lint, cr, lintconf configs
│   ├── dependabot.yaml                  # Automated dependency updates
│   └── workflows/                       # CI, lint, helm-publish, lint-and-test
├── .goreleaser.yaml                     # Multi-arch builds, signing, SBOM
└── .golangci.yaml                       # Linter configuration
```

## OpenAPI Client Generation

All Pulumi Cloud API calls go through a generated Go client built from the official [Pulumi Cloud OpenAPI spec](https://api.pulumi.com/api/openapi/pulumi-spec.json). The `internal/client/` package provides a typed wrapper around the generated code.

```bash
make generate
```

Generation is scoped to the 13 operations the exporter uses (configured in `oapi-codegen.yaml`):

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

## Contributing

1. Fork and clone the repository
2. Install tools: `make tools`
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Make your changes and add tests
5. Verify:
   ```bash
   make test-race
   make lint
   make helm-test
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
4. Add Grafana panels to `dashboards/pulumi-exporter.json`
5. Run `make helm-docs` if chart values changed
6. Add tests
