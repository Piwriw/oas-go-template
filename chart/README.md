# oas-go-template Helm Chart

Deploys the oas-go-template backend (gin server) and frontend (React SPA
served by nginx) into Kubernetes.

## Layout

| Template | What |
|----------|------|
| `server-deployment.yaml` | Server Deployment — liveness on `/healthz`, readiness on `/readyz`, mounts `config.yaml` from a ConfigMap or existing Secret. |
| `server-service.yaml` | ClusterIP Service exposing server on port `server.service.port` (default 8000). |
| `server-configmap.yaml` | Holds `server.config` when no existing config Secret is selected — pod rolls when content changes via `checksum/config`. |
| `web-deployment.yaml` | Optional frontend Deployment (nginx on port 8080). |
| `web-service.yaml` | Optional frontend Service. |
| `ingress.yaml` | Optional Ingress. Off by default; per-path backend selectable via `service` + `port` (e.g. `/` → web, `/api` → server). |
| `hpa.yaml` | Optional HPA on server (CPU/memory). |
| `serviceaccount.yaml` | Dedicated ServiceAccount when `serviceAccount.create=true`. |

## Quickstart

```bash
# Render locally — no cluster connection required.
make helm-template

# Lint before installing.
make helm-lint

# Install into a cluster.
helm install my-release ./chart \
  --namespace my-ns --create-namespace \
  --set server.image.repository=my-registry/oas-go-template \
  --set server.image.tag=v1.0.0 \
  --set web.image.repository=my-registry/oas-go-template-web \
  --set web.image.tag=v1.0.0

# Smoke test from your machine.
kubectl port-forward -n my-ns svc/my-release-oas-go-template-server 8000:8000
curl http://localhost:8000/healthz
```

## Config management

The server always reads `/app/config.yaml` via its `-c` argument. For
non-secret configuration, `server.config` is written to a ConfigMap and
mounted at that path:

```yaml
# values.prod.yaml
server:
  config: |
    server:
      http_addr: ":8000"
      gin_mode: release
      read_header_timeout: 5s
      read_timeout: 15s
      write_timeout: 30s
      idle_timeout: 60s
      max_header_bytes: 1048576
      max_body_bytes: 1048576
    db:
      driver: ""
      dsn: ""
    log:
      format: json
      level: info
    otel:
      enabled: true
      exporter_otlp_endpoint: "http://otel-collector.observability:4318"
```

```bash
helm upgrade my-release ./chart -f values.prod.yaml
```

For credentials, create or provision a Secret containing the complete YAML
file, then select it through `server.existingConfigSecret`:

```bash
kubectl create secret generic my-server-config \
  --namespace my-ns \
  --from-file=config.yaml=./config.prod.yaml
```

```yaml
# values.prod.yaml
server:
  existingConfigSecret:
    name: my-server-config
    key: config.yaml
```

When `name` is set, `server.config` is ignored and the chart does not create
the server ConfigMap. The Secret must exist in the release namespace. A change
to the Secret name or key rolls the Deployment; after a content-only update,
restart the Deployment or use a reloader controller.

## Switching OTel on

The default config points the OTLP exporter at `http://otel-collector:4318`
inside the cluster. If you don't run a collector, set `otel.enabled=false`
or override the endpoint:

```bash
helm install my-release ./chart \
  --set-string server.config='{otel: {enabled: false}}'
```

## Disabling the frontend

If you serve the SPA from a CDN or a separate release:

```yaml
web:
  enabled: false
```

All web templates are guarded with `{{- if .Values.web.enabled }}`.
