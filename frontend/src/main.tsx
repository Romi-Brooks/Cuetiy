import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import { applyPlatformAttrs } from './platform'
import './assets/main.css'

applyPlatformAttrs()

// Android WebView 对 env(safe-area-inset-*) 支持不完整，用 JS 补齐
function initSafeAreaVars() {
  const root = document.documentElement
  // 先用 CSS env，读不到就给 Android 保守值（状态栏关闭 overlay 后 top 通常为 0）
  const readEnv = (name: string, fallback: number) => {
    const probe = document.createElement('div')
    probe.style.cssText = `position:fixed;top:0;left:0;visibility:hidden;padding-top:env(${name})`
    document.body.appendChild(probe)
    const val = parseInt(getComputedStyle(probe).paddingTop, 10)
    document.body.removeChild(probe)
    return Number.isFinite(val) && val > 0 ? val : fallback
  }

  const isAndroid = document.documentElement.dataset.platform === 'android'
  const top = readEnv('safe-area-inset-top', isAndroid ? 0 : 0)
  const bottom = readEnv('safe-area-inset-bottom', isAndroid ? 0 : 0)
  const left = readEnv('safe-area-inset-left', 0)
  const right = readEnv('safe-area-inset-right', 0)

  root.style.setProperty('--safe-top', `${top}px`)
  root.style.setProperty('--safe-bottom', `${bottom}px`)
  root.style.setProperty('--safe-left', `${left}px`)
  root.style.setProperty('--safe-right', `${right}px`)
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
)

initSafeAreaVars()
