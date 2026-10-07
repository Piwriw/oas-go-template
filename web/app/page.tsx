'use client'

import Image from 'next/image'
import { useCallback, useEffect, useState, type FormEvent } from 'react'

import { apiBaseUrl, client } from '../src/api/client'
import { useI18n } from '../src/i18n'

type ProbeState =
  | { kind: 'loading' }
  | { kind: 'ok'; version?: string; gitCommit?: string; buildTime?: string }
  | { kind: 'error'; error: unknown; network: boolean }

type ProbeKey = 'health' | 'ready' | 'version'

type GreetingResponse =
  | { kind: 'idle' | 'sending' | 'network-error' }
  | { kind: 'complete'; status: number; body: unknown }

const loadingProbe: ProbeState = { kind: 'loading' }

const routes = {
  health: '/healthz',
  ready: '/readyz',
  version: '/version',
  greeting: '/v1/greetings',
} as const

const probeKeys: ProbeKey[] = ['health', 'ready', 'version']

function Probe({ probeKey, state }: { probeKey: ProbeKey; state: ProbeState }) {
  const { t, error } = useI18n()
  let value = t('checking')
  let detail = t('waitingApi')
  if (state.kind === 'error') {
    value = t(probeKey === 'ready' ? 'notReady' : 'unavailable')
    detail = state.network ? t('networkMessage') : error(state.error)
  } else if (state.kind === 'ok') {
    if (probeKey === 'health') {
      value = t('healthy')
      detail = state.version ? t('versionDetail', { version: state.version }) : t('processResponding')
    } else if (probeKey === 'ready') {
      value = t('readyValue')
      detail = t('dependenciesAvailable')
    } else {
      value = state.version ?? ''
      detail = t('buildDetail', { commit: state.gitCommit?.slice(0, 8) ?? '', time: state.buildTime ?? '' })
    }
  }
  return (
    <article className="probe">
      <div className="probe-topline">
        <span className="probe-title">{t(probeKey)}</span>
        <code className="probe-path">GET {routes[probeKey]}</code>
      </div>
      <div className={`probe-value probe-${state.kind}`}>
        <span className="probe-dot" aria-hidden="true" />
        <strong>{value}</strong>
      </div>
      <p className="probe-detail">{detail}</p>
    </article>
  )
}

export default function HomePage() {
  const { locale, locales, setLocale, t, error } = useI18n()
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
        ? { kind: 'ok', version: h.data.version }
        : { kind: 'error', error: h?.error, network: health.status === 'rejected' },
      ready: r?.data
        ? { kind: 'ok' }
        : { kind: 'error', error: r?.error, network: ready.status === 'rejected' },
      version: v?.data
        ? { kind: 'ok', ...v.data }
        : { kind: 'error', error: v?.error, network: version.status === 'rejected' },
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
        body: data ?? error ?? null,
      })
    } catch {
      setGreeting({ kind: 'network-error' })
    }
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <Image src="/favicon.svg" alt="" width={36} height={36} priority />
          <span className="brand-name">oas-go-template</span>
          <span className="brand-divider" aria-hidden="true" />
          <span className="brand-context">{t('workbench')}</span>
        </div>
        <div className="topbar-actions">
          <span className="topbar-label">{t('console')}</span>
          <select
            className="language-select"
            aria-label={t('language')}
            value={locale}
            onChange={(event) => setLocale(event.target.value)}
          >
            {locales.map((language) => (
              <option key={language} value={language} lang={language}>
                {language === 'en' ? 'English' : '简体中文'}
              </option>
            ))}
          </select>
        </div>
      </header>

      <main className="workspace">
        <section className="status-section" aria-labelledby="status-title">
          <div className="section-heading">
            <div>
              <p className="eyebrow">{t('service')}</p>
              <h1 id="status-title">{t('serviceStatus')}</h1>
            </div>
            <div className="section-actions">
              <span className="checked-at">
                {checkedAt ? t('updated', { time: checkedAt.toLocaleTimeString(locale) }) : t('checkingService')}
              </span>
              <button className="secondary-button" type="button" onClick={() => void refresh()} disabled={refreshing}>
                {t('refresh')}
              </button>
            </div>
          </div>

          <div className="base-url">
            <span>{t('baseUrl')}</span>
            <code>{apiBaseUrl}</code>
          </div>

          <div className="probe-grid" aria-live="polite">
            {probeKeys.map((key) => (
              <Probe key={key} probeKey={key} state={probes[key]} />
            ))}
          </div>
        </section>

        <section className="example-section" aria-labelledby="example-title">
          <div className="section-heading">
            <div>
              <p className="eyebrow">{t('exampleApi')}</p>
              <h2 id="example-title">{t('generateGreeting')}</h2>
            </div>
            <div className="route-label"><span>POST</span><code>{routes.greeting}</code></div>
          </div>

          <div className="request-tool">
            <form className="request-form" onSubmit={(event) => void sendGreeting(event)}>
              <div className="pane-heading">{t('requestBody')}</div>
              <label htmlFor="greeting-name">{t('name')}</label>
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
                {greeting.kind === 'sending' ? t('sending') : t('sendRequest')}
              </button>
            </form>

            <div className="response-pane" aria-live="polite">
              <div className="pane-heading">
                <span>{t('response')}</span>
                {greeting.kind === 'complete' && (
                  <span className={`response-status ${greeting.status < 400 ? 'response-ok' : 'response-error'}`}>
                    HTTP {greeting.status}
                  </span>
                )}
                {greeting.kind === 'network-error' && <span className="response-status response-error">{t('networkError')}</span>}
              </div>
              {greeting.kind === 'complete' && greeting.status >= 400 && (
                <p className="response-error-message" role="alert">
                  {error(greeting.body)}
                </p>
              )}
              <pre className={`response-body ${greeting.kind === 'idle' ? 'response-empty' : ''}`}>
                {greeting.kind === 'idle' && t('noResponse')}
                {greeting.kind === 'sending' && t('waitingResponse')}
                {greeting.kind === 'complete' && (greeting.body === null ? t('emptyResponse') : JSON.stringify(greeting.body, null, 2))}
                {greeting.kind === 'network-error' && t('networkMessage')}
              </pre>
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}
