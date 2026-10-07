import { getApiErrorMessage } from './errors.ts'
import { defaultLocale, locales, localeStorageKey, resolveLocale, translate, type MessageKey } from './messages.ts'

let locale = defaultLocale
const listeners = new Set<() => void>()

// One browser-wide language store shared by React and plain TypeScript callers.
export const i18n = {
  locales,
  get locale() {
    return locale
  },
  t(key: MessageKey, values?: Record<string, string | number>): string {
    return translate(locale, key, values)
  },
  error(error: unknown): string {
    return getApiErrorMessage(error, locale)
  },
  setLocale(value: unknown) {
    const nextLocale = resolveLocale(value)
    if (typeof window !== 'undefined') {
      try {
        window.localStorage.setItem(localeStorageKey, nextLocale)
      } catch {
        // Language switching also works without persistent storage.
      }
    }
    if (nextLocale === locale) return
    locale = nextLocale
    listeners.forEach((listener) => listener())
  },
  subscribe(listener: () => void) {
    listeners.add(listener)
    return () => { listeners.delete(listener) }
  },
  getSnapshot: () => locale,
  getServerSnapshot: () => defaultLocale,
}
