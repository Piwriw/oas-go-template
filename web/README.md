# oas-go-template web

Next.js App Router frontend. It is independent from the Go backend and is
deployed as a static export served by CDN or Nginx.

## Dev

Start the Go backend from the repo root with `make run`, then start the frontend:

```bash
npm ci
npm run dev
```

The workbench shows `/healthz`, `/readyz`, and `/version`, and submits
`POST /v1/greetings` through the typed API client. If the backend is down, it
shows an offline state and lets you retry.

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

`src/api/schema.gen.ts` is generated from `../spec/openapi.yaml` by
`openapi-typescript` — run `make gen-web` from the repo root. It holds **types
only**; never hand-edit it.

Run `make lint-oas` from the repo root to validate the same contract with the
Redocly CLI pinned in this package's lockfile.

The runtime client is `src/api/client.ts`, a hand-written `openapi-fetch`
instance typed against those paths. It defaults to `http://localhost:8000`;
set `NEXT_PUBLIC_API_BASE_URL` at build time to point elsewhere. Next inlines
`NEXT_PUBLIC_*` into the bundle, so this is fixed at build time and cannot be
changed by restarting the container.

For a Docker image targeting another backend, run from the repo root:

```bash
make web-docker NEXT_PUBLIC_API_BASE_URL=https://api.example.com
```

```ts
import { client } from './api/client'

const { data, error } = await client.GET('/version')
```
