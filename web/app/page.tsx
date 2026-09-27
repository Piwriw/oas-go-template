'use client'

import Image from 'next/image'
import { useCallback, useEffect, useState, type FormEvent } from 'react'

import { apiBaseUrl, client } from '../src/api/client'

type ProbeState = {
  kind: 'loading' | 'ok' | 'error'
  value: string
  detail: string
}

type ProbeKey = 'health' | 'ready' | 'version'

type GreetingResponse =
  | { kind: 'idle' | 'sending' }
  | { kind: 'complete'; status: number; body: unknown }
  | { kind: 'network-error'; message: string }

const loadingProbe: ProbeState = {
  kind: 'loading',
  value: 'Checking',
  detail: 'Waiting for the API',
}

const routes = {
  health: '/healthz',
  ready: '/readyz',
  version: '/version',
  greeting: '/v1/greetings',
} as const

const probeLabels: { key: ProbeKey; title: string; path: string }[] = [
  { key: 'health', title: 'Liveness', path: routes.health },
  { key: 'ready', title: 'Readiness', path: routes.ready },
  { key: 'version', title: 'Build', path: routes.version },
]

function Probe({ title, path, state }: { title: string; path: string; state: ProbeState }) {
  return (
    <article className="probe">
      <div className="probe-topline">
        <span className="probe-title">{title}</span>
        <code className="probe-path">GET {path}</code>
      </div>
      <div className={`probe-value probe-${state.kind}`}>
        <span className="probe-dot" aria-hidden="true" />
        <strong>{state.value}</strong>
      </div>
      <p className="probe-detail">{state.detail}</p>
    </article>
  )
}

export default function HomePage() {
  const [probes, setProbes] = useState<Record<ProbeKey, ProbeState>>({
    health: loadingProbe,
    ready: loadingProbe,
    version: loadingProbe,
  })
  const [refreshing, setRefreshing] = useState(true)
  const [checkedAt, setCheckedAt] = useState<Date | null>(null)
  const [name, setName] = useState('')
  const [greeting, setGreeting] = useState<GreetingResponse>({ kind: 'idle' })

  const refresh = useCallback(async () => {
    setRefreshing(true)
    setProbes({ health: loadingProbe, ready: loadingProbe, version: loadingProbe })

    const [health, ready, version] = await Promise.allSettled([
      client.GET(routes.health),
      client.GET(routes.ready),
      client.GET(routes.version),
    ])
    const h = health.status === 'fulfilled' ? health.value : null
    const r = ready.status === 'fulfilled' ? ready.value : null
    const v = version.status === 'fulfilled' ? version.value : null

    setProbes({
      health: h?.data
        ? { kind: 'ok', value: 'Healthy', detail: h.data.version ? `Version ${h.data.version}` : 'Process responding' }
        : { kind: 'error', value: 'Unavailable', detail: h?.error?.message ?? 'Could not reach the API' },
      ready: r?.data
        ? { kind: 'ok', value: 'Ready', detail: 'Dependencies available' }
        : { kind: 'error', value: 'Not ready', detail: r?.error?.message ?? 'Could not reach the API' },
      version: v?.data
        ? {
            kind: 'ok',
            value: v.data.version,
            detail: `Commit ${v.data.gitCommit.slice(0, 8)} · Built ${v.data.buildTime}`,
          }
        : { kind: 'error', value: 'Unavailable', detail: v?.error?.message ?? 'Could not reach the API' },
    })
    setCheckedAt(new Date())
    setRefreshing(false)
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  async function sendGreeting(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setGreeting({ kind: 'sending' })

    try {
      const { data, error, response } = await client.POST(routes.greeting, { body: { name } })
      setGreeting({
        kind: 'complete',
        status: response.status,
        body: data ?? error ?? { message: 'Unexpected empty response' },
      })
    } catch {
      setGreeting({ kind: 'network-error', message: 'Could not reach the API' })
    }
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <Image src="/favicon.svg" alt="" width={36} height={36} priority />
          <span className="brand-name">oas-go-template</span>
          <span className="brand-divider" aria-hidden="true" />
          <span className="brand-context">API Workbench</span>
        </div>
        <span className="topbar-label">Service console</span>
      </header>

      <main className="workspace">
        <section className="status-section" aria-labelledby="status-title">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Service</p>
              <h1 id="status-title">Service status</h1>
            </div>
            <div className="section-actions">
              <span className="checked-at">
                {checkedAt ? `Updated ${checkedAt.toLocaleTimeString('en-US')}` : 'Checking service'}
              </span>
              <button className="secondary-button" type="button" onClick={() => void refresh()} disabled={refreshing}>
                Refresh
              </button>
            </div>
          </div>

          <div className="base-url">
            <span>API base URL</span>
            <code>{apiBaseUrl}</code>
          </div>

          <div className="probe-grid" aria-live="polite">
            {probeLabels.map(({ key, title, path }) => (
              <Probe key={key} title={title} path={path} state={probes[key]} />
            ))}
          </div>
        </section>

        <section className="example-section" aria-labelledby="example-title">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Example API</p>
              <h2 id="example-title">Generate a greeting</h2>
            </div>
            <div className="route-label"><span>POST</span><code>{routes.greeting}</code></div>
          </div>

          <div className="request-tool">
            <form className="request-form" onSubmit={(event) => void sendGreeting(event)}>
              <div className="pane-heading">Request body</div>
              <label htmlFor="greeting-name">Name</label>
              <input
                id="greeting-name"
                name="name"
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={80}
                placeholder="Ada"
                required
                disabled={greeting.kind === 'sending'}
              />
              <button className="primary-button" type="submit" disabled={greeting.kind === 'sending'}>
                {greeting.kind === 'sending' ? 'Sending...' : 'Send request'}
              </button>
            </form>

            <div className="response-pane" aria-live="polite">
              <div className="pane-heading">
                <span>Response</span>
                {greeting.kind === 'complete' && (
                  <span className={`response-status ${greeting.status < 400 ? 'response-ok' : 'response-error'}`}>
                    HTTP {greeting.status}
                  </span>
                )}
                {greeting.kind === 'network-error' && <span className="response-status response-error">Network error</span>}
              </div>
              <pre className={`response-body ${greeting.kind === 'idle' ? 'response-empty' : ''}`}>
                {greeting.kind === 'idle' && 'No response yet'}
                {greeting.kind === 'sending' && 'Waiting for response...'}
                {greeting.kind === 'complete' && JSON.stringify(greeting.body, null, 2)}
                {greeting.kind === 'network-error' && greeting.message}
              </pre>
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}
