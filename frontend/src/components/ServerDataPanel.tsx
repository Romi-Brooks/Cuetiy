import { useCallback, useEffect, useRef, useState } from 'react'
import { chatDataAPI, secretsAPI } from '../api'
import { platform } from '../platform'
import {
  describeServerMode,
  getStoredApiBase,
  normalizeServerInput,
  pingServer,
  setApiBase,
} from '../utils/server'
import { Toast } from './Toast'

export function ServerDataPanel({
  currentConversationId,
}: {
  currentConversationId: number | null
}) {
  const [apiInput, setApiInput] = useState(() => getStoredApiBase())
  const [savedApi, setSavedApi] = useState(() => getStoredApiBase())
  const [pingMsg, setPingMsg] = useState('')
  const [pingOk, setPingOk] = useState<boolean | null>(null)
  const [busy, setBusy] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  const [deepseekCfg, setDeepseekCfg] = useState(false)
  const [mimoCfg, setMimoCfg] = useState(false)
  const [dsInput, setDsInput] = useState('')
  const [mimoInput, setMimoInput] = useState('')
  const [keyBusy, setKeyBusy] = useState(false)

  const showToast = useCallback((msg: string) => setToast(msg), [])

  const refreshSecrets = useCallback(async () => {
    try {
      const s = await secretsAPI.status()
      setDeepseekCfg(!!s.deepseek_configured)
      setMimoCfg(!!s.mimo_configured)
    } catch {
      /* 未登录或无权限时忽略 */
    }
  }, [])

  useEffect(() => {
    void refreshSecrets()
  }, [refreshSecrets])

  const saveKeys = useCallback(async () => {
    const payload: Parameters<typeof secretsAPI.update>[0] = {}
    if (dsInput.trim()) payload.deepseek_api_key = dsInput.trim()
    if (mimoInput.trim()) payload.mimo_api_key = mimoInput.trim()
    if (!Object.keys(payload).length) {
      showToast('请先输入要更新的 Key')
      return
    }
    setKeyBusy(true)
    try {
      const res = await secretsAPI.update(payload)
      setDeepseekCfg(!!res.deepseek_configured)
      setMimoCfg(!!res.mimo_configured)
      setDsInput('')
      setMimoInput('')
      showToast(res.message || '密钥已保存')
    } catch (e) {
      showToast((e as Error).message || '保存失败')
    } finally {
      setKeyBusy(false)
    }
  }, [dsInput, mimoInput, showToast])

  const clearKey = useCallback(
    async (which: 'deepseek' | 'mimo') => {
      setKeyBusy(true)
      try {
        const res = await secretsAPI.update(
          which === 'deepseek' ? { clear_deepseek: true } : { clear_mimo: true },
        )
        setDeepseekCfg(!!res.deepseek_configured)
        setMimoCfg(!!res.mimo_configured)
        if (which === 'deepseek') setDsInput('')
        else setMimoInput('')
        showToast('已清除')
      } catch (e) {
        showToast((e as Error).message || '清除失败')
      } finally {
        setKeyBusy(false)
      }
    },
    [showToast],
  )

  const saveServer = useCallback(async () => {
    const normalized = normalizeServerInput(apiInput)
    setApiBase(normalized)
    setSavedApi(normalized)
    setApiInput(normalized)
    setBusy(true)
    setPingMsg('探测中...')
    try {
      const r = await pingServer()
      setPingOk(r.ok)
      setPingMsg(r.message)
      showToast(r.ok ? '服务器地址已保存' : '已保存，但当前无法连接')
    } finally {
      setBusy(false)
    }
  }, [apiInput, showToast])

  const doExportAll = useCallback(async () => {
    setBusy(true)
    try {
      await chatDataAPI.exportAll()
      showToast('已导出全部会话 JSON')
    } catch (e) {
      showToast((e as Error).message || '导出失败')
    } finally {
      setBusy(false)
    }
  }, [showToast])

  const doExportCurrent = useCallback(async () => {
    if (!currentConversationId) {
      showToast('当前没有可导出的会话')
      return
    }
    setBusy(true)
    try {
      await chatDataAPI.exportConversation(currentConversationId)
      showToast('已导出当前会话 JSON')
    } catch (e) {
      showToast((e as Error).message || '导出失败')
    } finally {
      setBusy(false)
    }
  }, [currentConversationId, showToast])

  const doImportFile = useCallback(
    async (file: File) => {
      setBusy(true)
      try {
        const text = await file.text()
        const json = JSON.parse(text) as unknown
        const res = await chatDataAPI.importChat(json)
        showToast(res.message || '导入成功')
      } catch (e) {
        const msg = e instanceof Error ? e.message : '导入失败'
        showToast(msg.includes('Unexpected') ? 'JSON 文件格式不正确' : msg)
      } finally {
        setBusy(false)
        if (fileRef.current) fileRef.current.value = ''
      }
    },
    [showToast],
  )

  return (
    <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-sm overflow-hidden">
      <Toast message={toast} onDone={() => setToast(null)} />
      <div className="app-header px-5 py-4 border-b border-gray-100 dark:border-gray-700">
        <h2 className="text-base font-semibold text-gray-800 dark:text-gray-100">数据与服务器</h2>
        <p className="text-xs text-gray-400 mt-1">
          平台 {platform === 'desktop' ? 'Desktop' : platform === 'android' ? 'Android' : 'Web'} ·{' '}
          {describeServerMode()}
        </p>
      </div>

      <div className="px-5 py-4 space-y-4">
        <div className="space-y-2">
          <div>
            <span className="text-sm text-gray-500 dark:text-gray-400">服务器 API 地址</span>
            <p className="text-xs text-gray-400 mt-0.5">
              thin 包请填写，例如 http://服务器IP:8080；本地/同源可留空
            </p>
          </div>
          <div className="flex flex-col sm:flex-row gap-2">
            <input
              className="flex-1 rounded-xl border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-800 dark:text-gray-100 outline-none focus:border-wechat-green"
              placeholder="http://服务器IP:8080"
              value={apiInput}
              onChange={(e) => setApiInput(e.target.value)}
              autoCapitalize="off"
              autoCorrect="off"
              spellCheck={false}
            />
            <button
              type="button"
              disabled={busy}
              className="shrink-0 rounded-xl bg-wechat-green text-white text-sm font-medium px-4 py-2 hover:bg-wechat-green-dark disabled:opacity-50"
              onClick={() => void saveServer()}
            >
              {busy ? '处理中...' : '保存并测试'}
            </button>
          </div>
          {pingMsg && (
            <p className={`text-xs ${pingOk ? 'text-wechat-green' : 'text-red-500'}`}>{pingMsg}</p>
          )}
          <p className="text-[11px] text-gray-400 break-all">
            当前：{savedApi || '(同源 / 构建默认)'}
          </p>
        </div>

        <div className="border-t border-gray-100 dark:border-gray-700 pt-4 space-y-3">
          <div>
            <span className="text-sm text-gray-500 dark:text-gray-400">AI 服务密钥</span>
            <p className="text-xs text-gray-400 mt-0.5">
              保存在服务器/本机 .env，接口<strong>不回显</strong>明文。仅填写需要更新的项。
            </p>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                DeepSeek API Key
                <span className={`ml-2 ${deepseekCfg ? 'text-wechat-green' : 'text-amber-500'}`}>
                  {deepseekCfg ? '已配置' : '未配置'}
                </span>
              </span>
              {deepseekCfg && (
                <button
                  type="button"
                  className="text-[11px] text-red-400"
                  disabled={keyBusy}
                  onClick={() => void clearKey('deepseek')}
                >
                  清除
                </button>
              )}
            </div>
            <input
              type="password"
              autoComplete="off"
              className="w-full rounded-xl border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-800 dark:text-gray-100 outline-none focus:border-wechat-green"
              placeholder={deepseekCfg ? '已配置 · 输入新 Key 可覆盖' : 'sk-...'}
              value={dsInput}
              onChange={(e) => setDsInput(e.target.value)}
              spellCheck={false}
            />
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                MiMo TTS/ASR Key
                <span className={`ml-2 ${mimoCfg ? 'text-wechat-green' : 'text-amber-500'}`}>
                  {mimoCfg ? '已配置' : '未配置'}
                </span>
              </span>
              {mimoCfg && (
                <button
                  type="button"
                  className="text-[11px] text-red-400"
                  disabled={keyBusy}
                  onClick={() => void clearKey('mimo')}
                >
                  清除
                </button>
              )}
            </div>
            <input
              type="password"
              autoComplete="off"
              className="w-full rounded-xl border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-sm text-gray-800 dark:text-gray-100 outline-none focus:border-wechat-green"
              placeholder={mimoCfg ? '已配置 · 输入新 Key 可覆盖' : '输入 MiMo API Key'}
              value={mimoInput}
              onChange={(e) => setMimoInput(e.target.value)}
              spellCheck={false}
            />
          </div>

          <button
            type="button"
            disabled={keyBusy || (!dsInput.trim() && !mimoInput.trim())}
            className="rounded-xl bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-sm px-4 py-2 disabled:opacity-40"
            onClick={() => void saveKeys()}
          >
            {keyBusy ? '保存中...' : '保存密钥'}
          </button>
        </div>

        <div className="border-t border-gray-100 dark:border-gray-700 pt-4 space-y-2">
          <span className="text-sm text-gray-500 dark:text-gray-400">聊天数据</span>
          <p className="text-xs text-gray-400">
            导出为 cuetiy-chat-export JSON；导入会在当前账号下新建会话（ID 重映射）
          </p>
          <div className="flex flex-col sm:flex-row flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              className="rounded-xl border border-gray-200 dark:border-gray-600 text-sm px-3 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
              onClick={() => void doExportCurrent()}
            >
              导出当前会话
            </button>
            <button
              type="button"
              disabled={busy}
              className="rounded-xl border border-gray-200 dark:border-gray-600 text-sm px-3 py-2 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
              onClick={() => void doExportAll()}
            >
              导出全部会话
            </button>
            <button
              type="button"
              disabled={busy}
              className="rounded-xl bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-sm px-3 py-2 hover:opacity-90 disabled:opacity-50"
              onClick={() => fileRef.current?.click()}
            >
              导入 JSON
            </button>
            <input
              ref={fileRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0]
                if (f) void doImportFile(f)
              }}
            />
          </div>
        </div>
      </div>
    </div>
  )
}
