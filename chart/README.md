# oas-go-template Helm Chart

Deploys the oas-go-template backend (gin server) and frontend (React SPA
served by nginx) into Kubernetes.

## Layout

| Template | What |
|----------|------|
| `server-deployment.yaml` | Server Deployment — liveness on `/healthz`, readiness on `/readyz`, mounts `config.yaml` from a ConfigMap. |
| `server-service.yaml` | ClusterIP Service exposing server on port `server.service.port` (default 8000). |
| `server-configmap.yaml` | Holds `server.config` (a YAML string) — pod rolls when content changes via `checksum/config`. |
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

`server.config` is a YAML string written to a ConfigMap and mounted at
`/app/config.yaml`. The server reads it via the `-c /app/config.yaml` arg.
Edit by overriding the value:

```yaml
# values.prod.yaml
server:
  config: |
    server:
      http_addr: ":8000"
      gin_mode: release
    db:
      driver: postgres
      dsn: ""        # populate from a Secret via env or extraVolumes, NOT here.
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

**Secrets stay out of `server.config`.** Use a Kubernetes Secret +
`envFrom` / `extraVolumes` (extend the chart as needed), or an external
secrets controller.

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
