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

## Internationalization

The workbench supports English (`en`, the default) and Simplified Chinese
(`zh-CN`). The header's language selector stores the choice in localStorage
under `oas-go-template.locale`. Static HTML and the first client render use
English; the saved choice is restored after hydration. Switching languages
updates the page, document language, metadata, timestamps, and existing errors
without repeating API requests. If storage is unavailable, switching still works
for the current session.

Import all public i18n APIs from `src/i18n`. A single `i18n` instance owns the
current language, UI translation, error translation, and saved preference.
`I18nProvider` is mounted once in the root layout to restore the preference and
update document metadata. Components use `useI18n()` to subscribe to language
changes; browser utilities can use the same `i18n` instance directly:

```tsx
import { useI18n } from '../src/i18n'

const { t, error, locale, locales, setLocale } = useI18n()
t('serviceStatus')
error({ code: 10001 })
setLocale('zh-CN')
```

```ts
import { i18n } from '../src/i18n'

i18n.t('serviceStatus')
i18n.error({ code: 10001 })
```

Both translation functions automatically use the current global language.
Call them during rendering to keep displayed results in sync with language
changes. React subscriptions use `useSyncExternalStore`;
`src/i18n/messages.ts` owns typed dictionaries and interpolation. Add each new
UI key to both dictionaries. This uses React and native browser APIs with no
extra runtime dependencies or locale routing.

`error(apiError)` translates the numeric `code` in an API error. The mapping follows
`../internal/errcode/errcode.go`; add translations and a mapping whenever a new
backend code is introduced. Unknown codes or malformed errors use a localized
generic fallback; network failures use a separate translated message. Backend
logs and response `message` fields remain English. The response pane keeps raw
JSON unchanged and displays a translated explanation above failed responses.

Run `npm test` (Node.js 22+) to check error-code coverage against the Go source,
fallbacks, shared language state/subscriptions, language selection, and
interpolation. CI also runs these tests.

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
