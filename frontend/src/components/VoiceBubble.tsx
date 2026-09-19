import { useEffect, useMemo, useRef, useState } from 'react'
import { stripEmotionalForDisplay } from '../stores/actions'
import { resolveAssetUrl } from '../utils/url'

/** 微信式语音条：点按播放，可展开「转文字」（展示 content，不是 TTS 朗读文字） */
export function VoiceBubble({
  audioUrl,
  durationMs,
  text,
  isAI,
  nickname,
  onDebug,
  debugOpen,
}: {
  audioUrl: string
  durationMs?: number
  /** 语音对应的原文，用于「转文字」 */
  text?: string
  isAI: boolean
  nickname?: string
  onDebug?: () => void
  debugOpen?: boolean
}) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const lastSrcRef = useRef('')
  const [playing, setPlaying] = useState(false)
  const [showText, setShowText] = useState(false)
  const [progress, setProgress] = useState(0)

  const secs = Math.max(1, Math.round((durationMs || 3000) / 1000))
  // 时长决定气泡宽度，类似微信
  const width = Math.min(220, 72 + secs * 10)
  // 转文字：像微信一样只显示纯文本，不带情绪/动作标签
  const plainText = useMemo(() => stripEmotionalForDisplay(text || ''), [text])

  useEffect(() => {
    return () => {
      audioRef.current?.pause()
    }
  }, [])

  const toggle = () => {
    const resolved = resolveAssetUrl(audioUrl)
    if (!resolved) return
    if (lastSrcRef.current !== resolved) {
      audioRef.current?.pause()
      audioRef.current = new Audio(resolved)
      lastSrcRef.current = resolved
      audioRef.current.onended = () => {
        setPlaying(false)
        setProgress(0)
      }
      audioRef.current.ontimeupdate = () => {
        const a = audioRef.current
        if (!a || !a.duration) return
        setProgress(a.currentTime / a.duration)
      }
    }
    const a = audioRef.current
    if (!a) return
    if (playing) {
      a.pause()
      setPlaying(false)
    } else {
      a.play()
        .then(() => setPlaying(true))
        .catch(() => setPlaying(false))
    }
  }

  const bubbleBase =
    'inline-flex items-center gap-2 px-3 py-2.5 rounded-2xl cursor-pointer select-none max-w-full'
  const bubbleClass = isAI
    ? `${bubbleBase} bg-white dark:bg-wechat-bubble-ai-dark text-gray-800 dark:text-gray-100 rounded-bl-sm shadow-sm`
    : `${bubbleBase} bg-wechat-bubble dark:bg-wechat-bubble-dark text-gray-800 dark:text-gray-100 rounded-br-sm`

  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        className={bubbleClass}
        style={{ width, justifyContent: isAI ? 'flex-start' : 'flex-end' }}
        onClick={toggle}
        title={playing ? '停止' : '播放语音'}
      >
        {isAI && (
          <svg className="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="currentColor">
            <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z" />
          </svg>
        )}
        <div className="flex-1 h-0.5 bg-black/10 dark:bg-white/20 rounded-full overflow-hidden">
          <div
            className="h-full bg-wechat-green transition-all"
            style={{ width: `${Math.round(progress * 100)}%` }}
          />
        </div>
        {!isAI && (
          <svg className="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="currentColor">
            <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02z" />
          </svg>
        )}
        <span className="text-xs tabular-nums shrink-0">{secs}"</span>
      </button>
      <div className={`flex items-center gap-2 text-[10px] text-gray-400 ${isAI ? '' : 'justify-end'}`}>
        {isAI && nickname && <span>{nickname}</span>}
        {plainText && (
          <button
            type="button"
            className="hover:text-wechat-green transition-colors"
            onClick={() => setShowText((v) => !v)}
          >
            {showText ? '收起文字' : '转文字'}
          </button>
        )}
        {onDebug && (
          <button
            type="button"
            className="hover:text-wechat-green transition-colors"
            onClick={onDebug}
          >
            {debugOpen ? '收起 Debug' : 'Debug'}
          </button>
        )}
      </div>
      {showText && plainText && (
        <div
          className={`text-xs text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-800/60 rounded-xl px-3 py-2 max-w-[80%] whitespace-pre-wrap break-words ${
            isAI ? '' : 'self-end'
          }`}
        >
          {plainText}
        </div>
      )}
    </div>
  )
}
