export type Platform = 'web' | 'android' | 'desktop'

function detect(): Platform {
  const w = window as unknown as {
    Capacitor?: { isNativePlatform?: () => boolean; getPlatform?: () => string }
    __TAURI__?: unknown
  }
  if (w.__TAURI__) return 'desktop'
  if (w.Capacitor?.isNativePlatform?.()) {
    const p = w.Capacitor.getPlatform?.()
    if (p === 'android' || p === 'ios') return 'android'
  }
  return 'web'
}

export const platform: Platform = detect()
export const isNative = platform !== 'web'
export const isAndroid = platform === 'android'
export const isDesktop = platform === 'desktop'

export function applyPlatformAttrs() {
  document.documentElement.dataset.platform = platform
  document.body.dataset.platform = platform
}
