import createClient from 'openapi-fetch'

import type { paths } from './schema.gen'

/**
 * Base URL of the Go backend. Next inlines `NEXT_PUBLIC_*` at build time, so
 * this is baked into the static export — set it when building the bundle, not
 * when starting the nginx container.
 */
const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8000'

/**
 * Typed API client. This file is hand-written: `openapi-typescript` emits types
 * only (`./schema.gen.ts`), so the runtime client lives here. Never hand-edit
 * `*.gen.ts` — edit `spec/openapi.yaml` and run `make gen-web` instead.
 */
export const client = createClient<paths>({ baseUrl })
