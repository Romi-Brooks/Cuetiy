import { useState } from 'react'
import type { ContextDebugInfo, TTSDebugInfo } from '../stores/chat'
import type { ReplySegment } from '../types/api'
import { splitReplySegments } from '../utils/segments'

type Tab = 'llm' | 'ctx' | 'tts'

/** 单条消息 Debug 弹窗：LLM / 上下文 / TTS / 分段 */
export function MsgDebugModal({
  open,
  onClose,
  llmText,
  ttsDebug,
  contextDebug,
  segments,
  title,
}: {
  open: boolean
  onClose: () => void
  llmText: string
  ttsDebug?: TTSDebugInfo | null
  contextDebug?: ContextDebugInfo | null
  /** 完整回复的显式小段；缺省时按 LLM 文本本地切分 */
  segments?: ReplySegment[] | null
  title?: string
}) {
  const [tab, setTab] = useState<Tab>('llm')
  if (!open) return null

  const segs =
    segments && segments.length > 0 ? segments : splitReplySegments(llmText || '')

  const tabs: { key: Tab; label: string; disabled?: boolean }[] = [
    { key: 'llm', label: 'LLM' },
    { key: 'ctx', label: '上下文', disabled: !contextDebug },
    { key: 'tts', label: 'TTS', disabled: !ttsDebug },
  ]
  let active = tab
  if (active === 'ctx' && !contextDebug) active = 'llm'
  if (active === 'tts' && !ttsDebug) active = 'llm'

  return (
    <div
      className="fixed inset-0 z-[70] flex items-end sm:items-center justify-center bg-black/40"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-xl w-full sm:max-w-lg max-h-[80vh] flex flex-col overflow-hidden">
        <div className="px-5 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between shrink-0">
          <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">
            {title || '消息 Debug'}
          </h3>
          <button
            className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
            onClick={onClose}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>

        <div className="flex gap-1 px-4 pt-3 pb-1 shrink-0">
          {tabs.map((t) => (
            <button
              key={t.key}
              disabled={t.disabled}
              className={`text-xs px-3 py-1.5 rounded-lg border transition-colors disabled:opacity-40 ${
                active === t.key
                  ? 'border-wechat-green text-wechat-green bg-wechat-green/5'
                  : 'border-gray-200 dark:border-gray-600 text-gray-500'
              }`}
              onClick={() => !t.disabled && setTab(t.key)}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-3">
          {active === 'llm' && (
            <div className="space-y-2">
              <div className="flex flex-wrap gap-1.5">
                <span className="px-2 py-1 rounded-full bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300">
                  分段 {segs.length}
                </span>
                {segs.length > 1 &&
                  segs.map((s) => (
                    <span
                      key={s.seg_id}
                      className="px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300"
                      title={s.content.slice(0, 40)}
                    >
                      #{s.index + 1} {s.content.slice(0, 8)}
                      {s.content.length > 8 ? '…' : ''}
                    </span>
                  ))}
              </div>
              {segs.length > 1 && (
                <div className="space-y-1.5">
                  {segs.map((s) => (
                    <div
                      key={s.seg_id}
                      className="text-[11px] text-gray-500 dark:text-gray-400 border border-dashed border-gray-200 dark:border-gray-600 rounded-lg px-2 py-1.5"
                    >
                      <span className="font-medium">seg {s.seg_id}</span>
                      {s.start != null && s.end != null && (
                        <span className="ml-2 opacity-70">
                          [{s.start}, {s.end})
                        </span>
                      )}
                      <div className="mt-1 text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-words">
                        {s.content}
                      </div>
                    </div>
                  ))}
                </div>
              )}
              <pre className="text-xs bg-gray-900 text-gray-100 rounded-xl px-3 py-3 whitespace-pre-wrap break-words max-h-[40vh] overflow-y-auto">
                {llmText || '（无）'}
              </pre>
            </div>
          )}

          {active === 'ctx' && contextDebug && (
            <div className="space-y-2 text-xs">
              <div className="flex flex-wrap gap-1.5">
                <span className="px-2 py-1 rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                  {contextDebug.estimated_tokens} / {contextDebug.token_budget} tok
                  （{Math.round((contextDebug.occupancy || 0) * 100)}%）
                </span>
                <span className="px-2 py-1 rounded-full bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-300">
                  情绪 {contextDebug.emotion || '—'}
                </span>
                {(contextDebug.activated_skills || []).map((s) => (
                  <span key={s} className="px-2 py-1 rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
                    {s}
                  </span>
                ))}
                <span className="px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  摘要 v{contextDebug.summary_version || 0} · 近期 {contextDebug.recent_count}
                  {contextDebug.compacted ? ' · 本轮已压缩' : ''}
                </span>
              </div>
              <div className="text-[11px] text-gray-400">
                system {contextDebug.system_tokens}t · recent {contextDebug.recent_tokens}t · user {contextDebug.user_tokens}t
                {contextDebug.covered_message_id ? ` · covered_id=${contextDebug.covered_message_id}` : ''}
              </div>
              {contextDebug.compacted && (
                <p className="text-[11px] text-amber-600 dark:text-amber-400">
                  压缩已生效：旧消息并入滚动摘要，HISTORY 仅含摘要之后的 recent，不再发全文。
                </p>
              )}
              {contextDebug.summary && (
                <pre className="bg-amber-50 dark:bg-amber-900/20 text-gray-700 dark:text-gray-200 rounded-xl px-3 py-2 whitespace-pre-wrap break-words max-h-36 overflow-y-auto">
                  {contextDebug.summary}
                </pre>
              )}
              {contextDebug.system_parts?.length ? (
                <div className="space-y-1">
                  {contextDebug.system_parts.map((p) => (
                    <div key={p.name}>
                      <div className="text-[11px] font-medium text-gray-500 dark:text-gray-400">
                        {p.name} · {p.tokens}t
                      </div>
                      <pre className="text-[11px] bg-gray-50 dark:bg-gray-900/40 rounded-lg px-2 py-1.5 whitespace-pre-wrap break-words max-h-28 overflow-y-auto">
                        {p.content}
                      </pre>
                    </div>
                  ))}
                </div>
              ) : (
                contextDebug.full_system && (
                  <pre className="bg-gray-900 text-gray-100 rounded-xl px-3 py-2 whitespace-pre-wrap break-words max-h-40 overflow-y-auto">
                    {contextDebug.full_system}
                  </pre>
                )
              )}
            </div>
          )}

          {active === 'tts' && ttsDebug && (
            <div className="space-y-2">
              <div className="flex flex-wrap gap-1.5">
                <span className="text-[11px] px-2 py-1 rounded-full bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300">
                  {ttsDebug.model}
                </span>
                <span className="text-[11px] px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  {ttsDebug.voice_mode}
                </span>
                <span className="text-[11px] px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  {ttsDebug.format}
                </span>
                <span className="text-[11px] px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  {ttsDebug.char_count} 字 · {ttsDebug.latency_ms}ms
                  {ttsDebug.cache_hit ? ' · cache' : ''}
                </span>
              </div>
              <div className="text-[11px] text-gray-400">
                base：{ttsDebug.api_base}
                <br />
                voice：{ttsDebug.voice || '—'}
                <br />
                emotion：{ttsDebug.emotion || '—'}
                {ttsDebug.audio_style_tag ? ` · tag=(${ttsDebug.audio_style_tag})` : ''}
                <br />
                audio：{ttsDebug.audio_url || '—'} · {ttsDebug.audio_bytes} bytes
              </div>
              <div className="text-[11px] text-gray-400">风格：{ttsDebug.style_prompt || '—'}</div>
              <div>
                <div className="text-[11px] font-medium text-gray-500 dark:text-gray-400 mb-1">
                  发给 TTS 的文本
                </div>
                <pre className="text-xs bg-indigo-50 dark:bg-indigo-900/20 text-gray-700 dark:text-gray-200 rounded-xl px-3 py-2 whitespace-pre-wrap break-words max-h-40 overflow-y-auto">
                  {ttsDebug.text_sent || '（无）'}
                </pre>
              </div>
              {ttsDebug.error && <p className="text-[11px] text-red-400">{ttsDebug.error}</p>}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
