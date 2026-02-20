# Backend Setup

The exporter pushes metrics over OTLP. Point it at any OTLP-compatible receiver.

## Docker Compose (Prometheus + Grafana)

The included stack provides a ready-to-use setup with a pre-built Grafana dashboard (26 panels, 17 metrics).

```bash
cd deploy/docker-compose
cp .env.example .env
# Set PULUMI_ACCESS_TOKEN and PULUMI_ORGANIZATIONS in .env
make compose-up
```

- **Grafana**: http://localhost:3000 (admin / admin)
- **Prometheus**: http://localhost:9090

```bash
make compose-down      # stop
make compose-logs      # tail exporter logs
make compose-restart   # rebuild + restart exporter only
```

## Prometheus (standalone, v2.47+)

Enable the OTLP receiver:

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

## Grafana Alloy / OTel Collector

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=alloy:4317 \
  --otlp.protocol=grpc \
  --otlp.insecure
```

## Honeycomb

You need an Ingest API key (prefix `hcaik_`). Create one under Settings > API Keys > Create Ingest Key.

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=api.honeycomb.io:443 \
  --otlp.protocol=grpc \
  --otlp.headers="x-honeycomb-team=<HONEYCOMB_INGEST_KEY>"
```

Metrics appear in the environment tied to the ingest key. Note: Honeycomb Management API keys (`hcamk_` prefix) won't work here — you need an ingest key.

## DataDog

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=<DD_AGENT_HOST>:4318 \
  --otlp.insecure
```

## NewRelic

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=otlp.nr-data.net:4318 \
  --otlp.headers="api-key=<NEWRELIC_LICENSE_KEY>"
```

## Dynatrace

```bash
./pulumi-exporter \
  --pulumi.organizations=my-org \
  --otlp.endpoint=<ENV_ID>.live.dynatrace.com:4317 \
  --otlp.protocol=grpc \
  --otlp.headers="Authorization=Api-Token <DT_TOKEN>"
```
