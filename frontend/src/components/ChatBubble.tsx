import { useEffect, useMemo, useState } from 'react'
import type { Message } from '../types/api'
import { formatTime } from '../utils'
import { resolveAssetUrl } from '../utils/url'
import { splitReplySegments, type ReplySegment } from '../utils/segments'
import { loadHumanizeSettings } from '../stores/settings'
import {
  extractEmotionalActions,
  stripEmotionalForDisplay,
  useActionStore,
} from '../stores/actions'

function Avatar({
  src,
  name,
  fallbackClass,
}: {
  src: string
  name: string
  fallbackClass: string
}) {
  const [error, setError] = useState(false)
  const resolved = resolveAssetUrl(src)
  useEffect(() => setError(false), [resolved])
  if (error || !resolved) {
    return (
      <div
        className={`w-10 h-10 rounded-full flex items-center justify-center text-white text-sm font-medium ${fallbackClass}`}
      >
        {name?.charAt(0) || 'R'}
      </div>
    )
  }
  return (
    <img
      src={resolved}
      alt={name}
      className="w-10 h-10 rounded-full object-cover"
      onError={() => setError(true)}
    />
  )
}

export function ChatBubble({
  message,
  isAI,
  aiNickname,
  aiAvatar,
  userNickname,
  userAvatar,
  /** 最新 AI 回复且开启拟人节奏时，分段延时上屏 */
  animateSegments = false,
}: {
  message: Message
  isAI: boolean
  aiNickname: string
  aiAvatar: string
  userNickname: string
  userAvatar: string
  animateSegments?: boolean
}) {
  const actionsEnabled = useActionStore((s) => s.enabled)
  const pushAction = useActionStore((s) => s.pushAction)

  const content = message?.content || ''

  const { segments, actions } = useMemo(() => {
    const acts = extractEmotionalActions(content)
    const raw: ReplySegment[] =
      isAI && message?.segments?.length
        ? message.segments
        : isAI
          ? splitReplySegments(content)
          : [{ index: 0, seg_id: '0', content, start: 0, end: content.length }]

    const mapped = raw.map((seg) => {
      const display =
        isAI && !actionsEnabled ? stripEmotionalForDisplay(seg.content) : seg.content
      return { ...seg, display }
    })
    return { segments: mapped, actions: acts }
  }, [content, isAI, actionsEnabled, message?.segments])

  // 分段延时上屏：历史消息与关闭拟人时直接全量
  const [visibleCount, setVisibleCount] = useState(segments.length)
  useEffect(() => {
    const hs = loadHumanizeSettings()
    if (!isAI || !animateSegments || !hs.segmentRevealEnabled || segments.length <= 1) {
      setVisibleCount(segments.length)
      return
    }
    setVisibleCount(1)
    let shown = 1
    const gap = Math.max(200, hs.segmentRevealDelayMs || 700)
    const timer = setInterval(() => {
      shown += 1
      setVisibleCount(shown)
      if (shown >= segments.length) clearInterval(timer)
    }, gap)
    return () => clearInterval(timer)
  }, [isAI, animateSegments, segments.length, message?.id])

  useEffect(() => {
    if (!isAI || actions.length === 0) return
    for (const a of actions) pushAction(a)
  }, [isAI, actions, pushAction])

  if (isAI) {
    const imgUrl = message?.image_url || (message?.attachment_type === 'image' ? message?.attachment_url : '')
    if (message?.message_type === 'image' && imgUrl) {
      return (
        <div className="flex items-start gap-3 mb-4">
          <div className="flex-shrink-0">
            <Avatar src={aiAvatar} name={aiNickname} fallbackClass="bg-purple-500" />
          </div>
          <div className="flex flex-col max-w-[70%]">
            <span className="text-xs text-gray-400 mb-1 px-1">{aiNickname}</span>
            {content?.trim() && (
              <div className="chat-bubble-ai whitespace-pre-wrap break-words mb-2">{content}</div>
            )}
            <img
              src={resolveAssetUrl(imgUrl)}
              alt="AI 图片"
              className="max-w-[220px] sm:max-w-[260px] rounded-2xl border border-gray-200 dark:border-gray-600 object-cover bg-gray-100 dark:bg-gray-700"
              loading="lazy"
            />
            {message?.created_at && (
              <span className="text-xs text-gray-400 mt-1 px-1">
                {formatTime(message.created_at)}
              </span>
            )}
          </div>
        </div>
      )
    }

    const multi = segments.length > 1
    const shown = segments.slice(0, Math.max(1, visibleCount))
    return (
      <div className="flex items-start gap-3 mb-4">
        <div className="flex-shrink-0">
          <Avatar src={aiAvatar} name={aiNickname} fallbackClass="bg-purple-500" />
        </div>
        <div className="flex flex-col max-w-[70%]">
          <span className="text-xs text-gray-400 mb-1 px-1">{aiNickname}</span>
          <div className="flex flex-col gap-2">
            {shown.map((seg) => (
              <div
                key={`${message?.id ?? 'm'}:${seg.seg_id}`}
                data-seg-id={`${message?.id ?? '0'}:${seg.seg_id}`}
                data-seg-index={seg.index}
                className="flex items-start gap-1.5"
              >
                {multi && (
                  <span
                    className="mt-1.5 shrink-0 text-[10px] leading-none text-gray-400 dark:text-gray-500 tabular-nums select-none"
                    title={`段 ${seg.index + 1} / ${segments.length}`}
                  >
                    #{seg.index + 1}
                  </span>
                )}
                <div className="chat-bubble-ai whitespace-pre-wrap break-words min-w-0">
                  {seg.display}
                </div>
              </div>
            ))}
          </div>
          {multi && visibleCount >= segments.length && (
            <span className="text-[10px] text-gray-400 dark:text-gray-500 mt-0.5 px-1">
              共 {segments.length} 段
            </span>
          )}
          {message?.created_at && (
            <span className="text-xs text-gray-400 mt-1 px-1">
              {formatTime(message.created_at)}
            </span>
          )}
        </div>
      </div>
    )
  }

  const display = segments[0]?.display ?? content

  return (
    <div className="flex items-start gap-3 mb-4 w-full justify-end">
      <div className="flex flex-col max-w-[70%]">
        <span className="text-xs text-gray-400 mb-1 px-1 text-right">{userNickname || '我'}</span>
        <div className="chat-bubble-user inline-block max-w-full whitespace-pre-wrap break-words">
          {display}
        </div>
        {message?.created_at && (
          <span className="text-xs text-gray-400 mt-1 px-1 text-right">
            {formatTime(message.created_at)}
          </span>
        )}
      </div>
      <div className="flex-shrink-0">
        <Avatar src={userAvatar} name={userNickname} fallbackClass="bg-wechat-green" />
      </div>
    </div>
  )
}
