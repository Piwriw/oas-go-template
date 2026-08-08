# oas-go-template web

Next.js App Router frontend. It is independent from the Go backend and is
deployed as a static export served by CDN or Nginx.

## Dev

```bash
npm install
npm run dev
```

## Build

```bash
npm run build
# outputs to out/
```

For local production verification, serve the exported directory with any
static file server:

```bash
npm run build
python3 -m http.server 8080 --directory out
```

The Docker image uses the static `out/` export and serves it on port `8080`.

## API Client

The frontend OAS client is intentionally not included in this template. If
needed, generate a TypeScript client from `../spec/openapi.yaml` with
`openapi-typescript` and/or `openapi-fetch`, placing generated files under
`src/api/`.
