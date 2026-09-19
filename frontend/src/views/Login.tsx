import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useUserStore } from '../stores/user'
import { platform } from '../platform'
import {
  getStoredApiBase,
  normalizeServerInput,
  pingServer,
  setApiBase,
} from '../utils/server'

function validateEmail(email: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
}

export function Login() {
  const navigate = useNavigate()
  const login = useUserStore((s) => s.login)
  const register = useUserStore((s) => s.register)

  const [activeTab, setActiveTab] = useState<'login' | 'register'>('login')
  const [loginForm, setLoginForm] = useState({ email: '', password: '' })
  const [loginErrors, setLoginErrors] = useState({ email: '', password: '' })
  const [loginError, setLoginError] = useState('')
  const [loginLoading, setLoginLoading] = useState(false)

  const [registerForm, setRegisterForm] = useState({ username: '', email: '', password: '' })
  const [registerErrors, setRegisterErrors] = useState({ username: '', email: '', password: '' })
  const [registerError, setRegisterError] = useState('')
  const [registerLoading, setRegisterLoading] = useState(false)

  const [serverOpen, setServerOpen] = useState(() => platform !== 'web' || !!getStoredApiBase())
  const [serverInput, setServerInput] = useState(() => getStoredApiBase())
  const [serverMsg, setServerMsg] = useState('')
  const [serverOk, setServerOk] = useState<boolean | null>(null)
  const [serverBusy, setServerBusy] = useState(false)

  async function saveServer() {
    const normalized = normalizeServerInput(serverInput)
    setApiBase(normalized)
    setServerInput(normalized)
    setServerBusy(true)
    setServerMsg('探测中...')
    try {
      const r = await pingServer()
      setServerOk(r.ok)
      setServerMsg(r.message)
    } finally {
      setServerBusy(false)
    }
  }

  function validateLogin() {
    let valid = true
    const errs = { email: '', password: '' }
    if (!loginForm.email) {
      errs.email = '请输入邮箱'
      valid = false
    } else if (!validateEmail(loginForm.email)) {
      errs.email = '邮箱格式不正确'
      valid = false
    }
    if (!loginForm.password) {
      errs.password = '请输入密码'
      valid = false
    }
    setLoginErrors(errs)
    return valid
  }

  function validateRegister() {
    let valid = true
    const errs = { username: '', email: '', password: '' }
    if (!registerForm.username) {
      errs.username = '请输入用户名'
      valid = false
    }
    if (!registerForm.email) {
      errs.email = '请输入邮箱'
      valid = false
    } else if (!validateEmail(registerForm.email)) {
      errs.email = '邮箱格式不正确'
      valid = false
    }
    if (!registerForm.password) {
      errs.password = '请输入密码'
      valid = false
    }
    setRegisterErrors(errs)
    return valid
  }

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoginError('')
    if (!validateLogin()) return
    setLoginLoading(true)
    try {
      await login(loginForm.email, loginForm.password)
      navigate('/chat')
    } catch (err) {
      setLoginError((err as Error).message || '登录失败，请检查邮箱和密码')
    } finally {
      setLoginLoading(false)
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    setRegisterError('')
    if (!validateRegister()) return
    setRegisterLoading(true)
    try {
      await register(registerForm.username, registerForm.email, registerForm.password)
      navigate('/chat')
    } catch (err) {
      setRegisterError((err as Error).message || '注册失败，请稍后重试')
    } finally {
      setRegisterLoading(false)
    }
  }

  return (
    <div className="h-[100dvh] bg-gray-100 dark:bg-wechat-bg-dark flex items-center justify-center p-4 overflow-y-auto">
      <div className="w-full max-w-md py-8">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-wechat-green mb-2">RainYi</h1>
          <p className="text-sm text-gray-500 dark:text-wechat-text-secondary-dark">
            你的专属情感陪伴机器人
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-lg p-6 sm:p-8">
          <div className="mb-4">
            <button
              type="button"
              className="text-xs text-gray-500 dark:text-gray-400 hover:text-wechat-green"
              onClick={() => setServerOpen((v) => !v)}
            >
              {serverOpen ? '收起服务器设置' : '服务器设置（thin 包必配）'}
            </button>
            {serverOpen && (
              <div className="mt-3 rounded-xl border border-gray-200 dark:border-gray-600 p-3 space-y-2">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {platform === 'web'
                    ? '网页版默认同源；若前端与后端分离，请填写后端地址'
                    : 'APK / Desktop thin 需填写后端地址，例如 http://192.168.1.10:8080'}
                </p>
                <div className="flex flex-col sm:flex-row gap-2">
                  <input
                    className="flex-1 input-field"
                    placeholder="http://服务器IP:8080"
                    value={serverInput}
                    onChange={(e) => setServerInput(e.target.value)}
                    autoCapitalize="off"
                    autoCorrect="off"
                    spellCheck={false}
                  />
                  <button
                    type="button"
                    className="btn-primary shrink-0 px-4 py-2 text-sm"
                    disabled={serverBusy}
                    onClick={() => void saveServer()}
                  >
                    {serverBusy ? '测试中...' : '保存并测试'}
                  </button>
                </div>
                {serverMsg && (
                  <p className={`text-xs ${serverOk ? 'text-wechat-green' : 'text-red-500'}`}>
                    {serverMsg}
                  </p>
                )}
              </div>
            )}
          </div>

          <div className="flex mb-6 bg-gray-100 dark:bg-gray-700 rounded-xl p-1">
            {(['login', 'register'] as const).map((tab) => (
              <button
                key={tab}
                className={`flex-1 py-2 text-sm font-medium rounded-lg transition-all ${
                  activeTab === tab
                    ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                    : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200'
                }`}
                onClick={() => setActiveTab(tab)}
              >
                {tab === 'login' ? '登录' : '注册'}
              </button>
            ))}
          </div>

          {activeTab === 'login' && (
            <form onSubmit={handleLogin}>
              <div className="space-y-4">
                <div>
                  <input
                    type="email"
                    placeholder="邮箱"
                    className={`input-field w-full ${loginErrors.email ? 'border-red-400 focus:border-red-400' : ''}`}
                    value={loginForm.email}
                    onChange={(e) => {
                      setLoginForm({ ...loginForm, email: e.target.value })
                      setLoginErrors({ ...loginErrors, email: '' })
                    }}
                  />
                  {loginErrors.email && (
                    <p className="text-red-500 text-xs mt-1 ml-1">{loginErrors.email}</p>
                  )}
                </div>
                <div>
                  <input
                    type="password"
                    placeholder="密码"
                    className={`input-field w-full ${loginErrors.password ? 'border-red-400 focus:border-red-400' : ''}`}
                    value={loginForm.password}
                    onChange={(e) => {
                      setLoginForm({ ...loginForm, password: e.target.value })
                      setLoginErrors({ ...loginErrors, password: '' })
                    }}
                  />
                  {loginErrors.password && (
                    <p className="text-red-500 text-xs mt-1 ml-1">{loginErrors.password}</p>
                  )}
                </div>
              </div>

              {loginError && <p className="text-red-500 text-sm text-center mt-4">{loginError}</p>}

              <button type="submit" className="btn-primary w-full mt-6 py-3" disabled={loginLoading}>
                {loginLoading ? (
                  <span className="flex items-center justify-center gap-2">
                    <span className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    登录中...
                  </span>
                ) : (
                  '登录'
                )}
              </button>
            </form>
          )}

          {activeTab === 'register' && (
            <form onSubmit={handleRegister}>
              <div className="space-y-4">
                <div>
                  <input
                    type="text"
                    placeholder="用户名"
                    className={`input-field w-full ${registerErrors.username ? 'border-red-400 focus:border-red-400' : ''}`}
                    value={registerForm.username}
                    onChange={(e) => {
                      setRegisterForm({ ...registerForm, username: e.target.value })
                      setRegisterErrors({ ...registerErrors, username: '' })
                    }}
                  />
                  {registerErrors.username && (
                    <p className="text-red-500 text-xs mt-1 ml-1">{registerErrors.username}</p>
                  )}
                </div>
                <div>
                  <input
                    type="email"
                    placeholder="邮箱"
                    className={`input-field w-full ${registerErrors.email ? 'border-red-400 focus:border-red-400' : ''}`}
                    value={registerForm.email}
                    onChange={(e) => {
                      setRegisterForm({ ...registerForm, email: e.target.value })
                      setRegisterErrors({ ...registerErrors, email: '' })
                    }}
                  />
                  {registerErrors.email && (
                    <p className="text-red-500 text-xs mt-1 ml-1">{registerErrors.email}</p>
                  )}
                </div>
                <div>
                  <input
                    type="password"
                    placeholder="密码"
                    className={`input-field w-full ${registerErrors.password ? 'border-red-400 focus:border-red-400' : ''}`}
                    value={registerForm.password}
                    onChange={(e) => {
                      setRegisterForm({ ...registerForm, password: e.target.value })
                      setRegisterErrors({ ...registerErrors, password: '' })
                    }}
                  />
                  {registerErrors.password && (
                    <p className="text-red-500 text-xs mt-1 ml-1">{registerErrors.password}</p>
                  )}
                </div>
              </div>

              {registerError && (
                <p className="text-red-500 text-sm text-center mt-4">{registerError}</p>
              )}

              <button
                type="submit"
                className="btn-primary w-full mt-6 py-3"
                disabled={registerLoading}
              >
                {registerLoading ? (
                  <span className="flex items-center justify-center gap-2">
                    <span className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    注册中...
                  </span>
                ) : (
                  '注册'
                )}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
