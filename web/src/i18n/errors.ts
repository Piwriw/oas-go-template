import { translate, type Locale, type MessageKey } from './messages.ts'

// Keep these stable public codes aligned with internal/errcode/errcode.go.
const errorKeys = new Map<number, MessageKey>([
  [10001, 'invalidRequest'],
  [10002, 'requestBodyTooLarge'],
  [20001, 'forbidden'],
  [30001, 'notFound'],
  [30002, 'methodNotAllowed'],
  [50001, 'dbUnavailable'], // Reserved deprecated code: historical responses still translate.
  [50002, 'dbHandle'],
  [50003, 'dbPing'],
  [99001, 'internalError'],
])

export function getApiErrorMessage(error: unknown, locale: Locale): string {
  const code = error !== null && typeof error === 'object' && 'code' in error ? error.code : undefined
  const key = typeof code === 'number' ? errorKeys.get(code) : undefined
  return translate(locale, key ?? 'unknownError')
}
