import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'

import { i18n } from './core.ts'
import { getApiErrorMessage } from './errors.ts'
import { defaultLocale, locales, resolveLocale, translate } from './messages.ts'

test('all backend error codes have English and Chinese translations', () => {
  const source = readFileSync(new URL('../../../internal/errcode/errcode.go', import.meta.url), 'utf8')
  const codes = [...source.matchAll(/\bCode\s*=\s*(\d+)/g)].map((match) => Number(match[1]))
  assert.ok(codes.length > 0)
  for (const code of codes) {
    for (const locale of locales) {
      assert.notEqual(getApiErrorMessage({ code }, locale), translate(locale, 'unknownError'), `${code}: ${locale}`)
    }
    assert.notEqual(getApiErrorMessage({ code }, 'en'), getApiErrorMessage({ code }, 'zh-CN'))
  }
})

test('numeric codes determine the message, independently of server text', () => {
  const error = { code: 10001, message: 'Internal validation details' }
  assert.equal(getApiErrorMessage(error, 'en'), 'The request is invalid. Check your input and try again.')
  assert.equal(getApiErrorMessage(error, 'zh-CN'), '请求无效，请检查输入后重试。')
  assert.equal(error.message, 'Internal validation details')
  assert.equal(getApiErrorMessage({ code: 50003 }, 'zh-CN'), '无法连接数据库，请稍后重试。')
})

test('unknown, retired and malformed errors use a localized fallback', () => {
  for (const error of [null, undefined, 'failed', {}, { code: 50004 }, { code: 12345 }, { code: '10001' },
    { code: NaN }, { message: 'Do not display internal details' }, { code: 'constructor' }]) {
    for (const locale of locales) {
      assert.equal(getApiErrorMessage(error, locale), translate(locale, 'unknownError'))
    }
  }
})

test('stored language is validated and English is the default', () => {
  assert.equal(resolveLocale('zh-CN'), 'zh-CN')
  for (const value of [null, undefined, '', 'en', 'fr', 'zh', {}]) {
    assert.equal(resolveLocale(value), defaultLocale)
  }
})

test('interpolation keeps runtime values intact and switches UI language', () => {
  assert.equal(translate('en', 'buildDetail', { commit: 'abc123', time: '2026-10-06' }), 'Commit abc123 · Built 2026-10-06')
  assert.equal(translate('zh-CN', 'buildDetail', { commit: 'abc123', time: '2026-10-06' }), '提交 abc123 · 构建于 2026-10-06')
  assert.equal(translate('zh-CN', 'versionDetail', { version: '{version}' }), '版本 {version}')
})

test('global translations and subscribers share one language state', () => {
  const { t, error, setLocale } = i18n
  const observed: string[] = []
  const unsubscribe = i18n.subscribe(() => observed.push(t('serviceStatus')))
  try {
    assert.equal(i18n.locale, 'en')
    setLocale('zh-CN')
    assert.equal(i18n.locale, 'zh-CN')
    assert.equal(t('serviceStatus'), '服务状态')
    assert.equal(error({ code: 10001 }), '请求无效，请检查输入后重试。')
    assert.equal(error({ code: 50004 }), '请求失败，请重试。')
    assert.equal(i18n.getServerSnapshot(), 'en')
    setLocale('zh-CN')
    assert.deepEqual(observed, ['服务状态'])
    setLocale('unsupported')
    assert.equal(t('serviceStatus'), 'Service status')
    assert.equal(error({ code: 10001 }), 'The request is invalid. Check your input and try again.')
    assert.deepEqual(observed, ['服务状态', 'Service status'])
    unsubscribe()
    setLocale('zh-CN')
    assert.deepEqual(observed, ['服务状态', 'Service status'])
  } finally {
    unsubscribe()
    setLocale(defaultLocale)
  }
})
