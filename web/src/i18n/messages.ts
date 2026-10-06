export const locales = ['en', 'zh-CN'] as const
export type Locale = (typeof locales)[number]
export const defaultLocale: Locale = 'en'
export const localeStorageKey = 'oas-go-template.locale'

const en = {
  workbench: 'API Workbench',
  description: 'Service status and typed API example for oas-go-template.',
  console: 'Service console',
  language: 'Language',
  service: 'Service',
  serviceStatus: 'Service status',
  updated: 'Updated {time}',
  checkingService: 'Checking service',
  refresh: 'Refresh',
  baseUrl: 'API base URL',
  health: 'Liveness',
  ready: 'Readiness',
  version: 'Build',
  checking: 'Checking',
  waitingApi: 'Waiting for the API',
  healthy: 'Healthy',
  readyValue: 'Ready',
  unavailable: 'Unavailable',
  notReady: 'Not ready',
  versionDetail: 'Version {version}',
  processResponding: 'Process responding',
  dependenciesAvailable: 'Dependencies available',
  buildDetail: 'Commit {commit} · Built {time}',
  exampleApi: 'Example API',
  generateGreeting: 'Generate a greeting',
  requestBody: 'Request body',
  name: 'Name',
  sending: 'Sending...',
  sendRequest: 'Send request',
  response: 'Response',
  noResponse: 'No response yet',
  waitingResponse: 'Waiting for response...',
  networkError: 'Network error',
  networkMessage: 'Could not reach the API. Check your connection and try again.',
  invalidRequest: 'The request is invalid. Check your input and try again.',
  requestBodyTooLarge: 'The request body is too large.',
  forbidden: 'You do not have permission to perform this action.',
  notFound: 'The requested resource was not found.',
  methodNotAllowed: 'This request method is not allowed.',
  dbUnavailable: 'The database is unavailable. Please try again later.',
  dbHandle: 'The database connection could not be accessed. Please try again later.',
  dbPing: 'The database could not be reached. Please try again later.',
  internalError: 'An internal server error occurred. Please try again later.',
  unknownError: 'The request failed. Please try again.',
  emptyResponse: 'The API returned an empty response.',
}

export type MessageKey = keyof typeof en

const zh: Record<MessageKey, string> = {
  workbench: 'API 工作台',
  description: 'oas-go-template 的服务状态与类型安全 API 示例。',
  console: '服务控制台',
  language: '语言',
  service: '服务',
  serviceStatus: '服务状态',
  updated: '更新于 {time}',
  checkingService: '正在检查服务',
  refresh: '刷新',
  baseUrl: 'API 基础地址',
  health: '存活检查',
  ready: '就绪检查',
  version: '构建信息',
  checking: '检查中',
  waitingApi: '等待 API 响应',
  healthy: '健康',
  readyValue: '已就绪',
  unavailable: '不可用',
  notReady: '未就绪',
  versionDetail: '版本 {version}',
  processResponding: '进程正常响应',
  dependenciesAvailable: '依赖服务可用',
  buildDetail: '提交 {commit} · 构建于 {time}',
  exampleApi: 'API 示例',
  generateGreeting: '生成问候语',
  requestBody: '请求体',
  name: '姓名',
  sending: '发送中...',
  sendRequest: '发送请求',
  response: '响应',
  noResponse: '暂无响应',
  waitingResponse: '等待响应...',
  networkError: '网络错误',
  networkMessage: '无法连接 API，请检查网络后重试。',
  invalidRequest: '请求无效，请检查输入后重试。',
  requestBodyTooLarge: '请求体过大。',
  forbidden: '您没有执行此操作的权限。',
  notFound: '未找到请求的资源。',
  methodNotAllowed: '不支持此请求方法。',
  dbUnavailable: '数据库不可用，请稍后重试。',
  dbHandle: '无法获取数据库连接，请稍后重试。',
  dbPing: '无法连接数据库，请稍后重试。',
  internalError: '服务器内部错误，请稍后重试。',
  unknownError: '请求失败，请重试。',
  emptyResponse: 'API 返回了空响应。',
}

const messages: Record<Locale, Record<MessageKey, string>> = { en, 'zh-CN': zh }

export function resolveLocale(value: unknown): Locale {
  return value === 'zh-CN' ? value : defaultLocale
}

export function translate(locale: Locale, key: MessageKey, values: Record<string, string | number> = {}): string {
  return messages[locale][key].replace(/\{(\w+)\}/g, (placeholder, name: string) =>
    Object.hasOwn(values, name) ? String(values[name]) : placeholder,
  )
}
