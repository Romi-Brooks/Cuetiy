/** 运行时服务器配置：thin 包（EXE/APK）由用户在 App 内填写，不打进安装包 */

import { platform } from '../platform'

const API_KEY = 'cuetiy:api_base'
/** unified 桌面包默认本机后端 */
const UNIFIED_DEFAULT_API = 'http://127.0.0.1:8080'

function rawApiBase(): string {
  try {
    const stored = (localStorage.getItem(API_KEY) || '').trim()
    if (stored) return stored
  } catch {
    /* ignore */
  }
  const envUrl = (import.meta.env.VITE_API_URL || '').trim()
  if (envUrl) return envUrl
  // Desktop 未配置时默认本机（unified / 本机后端）；Web 仍走同源
  if (platform === 'desktop') return UNIFIED_DEFAULT_API
  return ''
}

/** 用户可填服务器根地址（http://host:port）或完整 /api 前缀 */
export function getStoredApiBase(): string {
  try {
    return (localStorage.getItem(API_KEY) || '').trim()
  } catch {
    return ''
  }
}

export function setApiBase(value: string) {
  const v = value.trim().replace(/\/+$/, '')
  try {
    if (v) localStorage.setItem(API_KEY, v)
    else localStorage.removeItem(API_KEY)
  } catch {
    /* ignore */
  }
}

/** 请求用的 API 前缀；未配置时网页走同源 /api */
export function getApiBase(): string {
  const url = rawApiBase()
  if (!url) return '/api'
  if (url.endsWith('/api')) return url
  return `${url}/api`
}

/** 资源绝对地址用的 origin；同源网页返回空串 */
export function getServerOrigin(): string {
  const url = rawApiBase()
  if (!url) return ''
  const withScheme = /^https?:\/\//i.test(url) ? url : `http://${url}`
  try {
    return new URL(withScheme).origin
  } catch {
    return ''
  }
}

/** WebSocket 地址（含 /api/ws/chat） */
export function getWsUrl(): string {
  try {
    const custom = (localStorage.getItem('cuetiy:ws_url') || import.meta.env.VITE_WS_URL || '').trim()
    if (custom) {
      if (custom.includes('/api/ws/chat')) return custom
      const base = custom.replace(/\/+$/, '')
      return `${base}/api/ws/chat`
    }
  } catch {
    /* fallthrough */
  }
  const origin = getServerOrigin()
  if (origin) {
    const u = new URL(origin)
    const proto = u.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${u.host}/api/ws/chat`
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/api/ws/chat`
}

/** 展示用：当前数据来源说明 */
export function describeServerMode(): string {
  const stored = getStoredApiBase()
  if (stored) return stored
  const url = rawApiBase()
  if (!url) return '同源 / 默认构建配置'
  if (platform === 'desktop' && url === UNIFIED_DEFAULT_API) {
    return `${url}（桌面默认本机）`
  }
  return url
}

/** 解析用户输入：补全 http://，去掉末尾斜杠 */
export function normalizeServerInput(input: string): string {
  let v = input.trim().replace(/\/+$/, '')
  if (!v) return ''
  if (!/^https?:\/\//i.test(v)) v = `http://${v}`
  return v
}

/** 简单探测服务器是否可达 */
export async function pingServer(timeoutMs = 4000): Promise<{ ok: boolean; message: string }> {
  const base = getApiBase()
  const url = `${base}/auth/login`
  const ctrl = new AbortController()
  const timer = setTimeout(() => ctrl.abort(), timeoutMs)
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'ping@cuetiy.local', password: 'invalid' }),
      signal: ctrl.signal,
    })
    // 任意 HTTP 响应都说明后端在听
    return { ok: true, message: `服务器可达（HTTP ${res.status}）` }
  } catch (e) {
    const msg = e instanceof Error && e.name === 'AbortError' ? '连接超时' : '无法连接服务器'
    return { ok: false, message: msg }
  } finally {
    clearTimeout(timer)
  }
}
