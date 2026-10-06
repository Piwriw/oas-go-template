'use client'

import { useEffect, useSyncExternalStore, type ReactNode } from 'react'

import { i18n } from './core'
import { getApiErrorMessage } from './errors'
import { localeStorageKey, translate } from './messages'

export function I18nProvider({ children }: { children: ReactNode }) {
  const { locale } = useI18n()

  useEffect(() => {
    try {
      i18n.setLocale(localStorage.getItem(localeStorageKey))
    } catch {
      // Storage may be disabled; the default locale still works.
    }
  }, [])

  useEffect(() => {
    document.documentElement.lang = locale
    document.title = `${translate(locale, 'workbench')} | oas-go-template`
    document.querySelector('meta[name="description"]')?.setAttribute('content', translate(locale, 'description'))
  }, [locale])

  return children
}

// oxlint-disable-next-line react/only-export-components -- Keep the language hook with its browser initializer.
export function useI18n(): typeof i18n {
  const locale = useSyncExternalStore(i18n.subscribe, i18n.getSnapshot, i18n.getServerSnapshot)
  // Bind translations to React's snapshot so hydration also starts in English.
  return {
    ...i18n,
    locale,
    t: (key, values) => translate(locale, key, values),
    error: (error) => getApiErrorMessage(error, locale),
  }
}
