/** 后端返回的资源路径前缀（头像 / 语音 / 静态文件） */
import { getServerOrigin } from './server'

const RELATIVE_ASSET_PREFIX = /^(?:\/storage\/|\/static\/)/i

/**
 * 把后端相对路径（/storage/*、/static/*）解析为可访问的绝对地址。
 * - Capacitor / Tauri thin：origin 来自运行时配置的 API 地址
 * - 网页同源 / 反代：保持相对路径即可
 * - 已是 http(s)/data/blob 等绝对 URL：原样返回
 */
export function resolveAssetUrl(url?: string | null): string {
  if (!url) return ''
  if (/^(?:[a-z][a-z0-9+.-]*:|\/\/)/i.test(url)) return url
  if (!url.startsWith('/')) return url
  if (!RELATIVE_ASSET_PREFIX.test(url)) return url

  const origin = getServerOrigin()
  if (!origin) return url
  return origin + url
}

export const DEFAULT_AVATAR_PATH = '/static/default-avatar.svg'

export function defaultAvatarUrl(): string {
  return resolveAssetUrl(DEFAULT_AVATAR_PATH)
}
