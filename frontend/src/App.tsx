import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { Login } from './views/Login'
import { MainChat } from './views/MainChat'
import { useThemeStore } from './stores/theme'
import { useUserStore } from './stores/user'
import { isAndroid, isNative } from './platform'

async function initNativeChrome() {
  if (!isNative) return
  try {
    const { StatusBar, Style } = await import('@capacitor/status-bar')
    const { Capacitor } = await import('@capacitor/core')
    if (!Capacitor.isNativePlatform()) return

    // Android: 不让 WebView 钻到状态栏底下，避免遮挡
    // iOS: 允许 overlay + env(safe-area-inset-*)
    if (isAndroid) {
      await StatusBar.setOverlaysWebView({ overlay: false })
      await StatusBar.setStyle({ style: Style.Dark })
      await StatusBar.setBackgroundColor({ color: '#1f2937' })
    } else {
      await StatusBar.setOverlaysWebView({ overlay: true })
      await StatusBar.setStyle({ style: Style.Dark })
    }
  } catch {
    /* browser preview */
  }
}

export default function App() {
  const initTheme = useThemeStore((s) => s.initTheme)
  const restoreSession = useUserStore((s) => s.restoreSession)
  const token = useUserStore((s) => s.token)
  const fetchProfile = useUserStore((s) => s.fetchProfile)

  useEffect(() => {
    initTheme()
    restoreSession()
    initNativeChrome()
  }, [initTheme, restoreSession])

  // 有 token 时拉一次 profile，同步头像/昵称等服务端字段（PC 改完手机能跟上）
  useEffect(() => {
    if (!token) return
    fetchProfile().catch(() => {
      /* 离线或 token 失效时沿用本地缓存 */
    })
  }, [token, fetchProfile])

  return (
    <div className="h-full bg-wechat-bg dark:bg-wechat-bg-dark text-wechat-text dark:text-wechat-text-dark transition-colors duration-200">
      <Routes>
        <Route path="/" element={<Navigate to="/login" replace />} />
        <Route path="/login" element={<Login />} />
        <Route
          path="/chat"
          element={
            <RequireAuth>
              <MainChat />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </div>
  )
}

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useUserStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}
