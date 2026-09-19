import { useEffect, useMemo, useState } from 'react'
import type { Message } from '../types/api'
import { formatTime } from '../utils'
import { resolveAssetUrl } from '../utils/url'
import { splitReplySegments, type ReplySegment } from '../utils/segments'
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
}: {
  message: Message
  isAI: boolean
  aiNickname: string
  aiAvatar: string
  userNickname: string
  userAvatar: string
}) {
  const actionsEnabled = useActionStore((s) => s.enabled)
  const pushAction = useActionStore((s) => s.pushAction)

  const content = message?.content || ''

  const { segments, actions } = useMemo(() => {
    const acts = extractEmotionalActions(content)
    // AI：优先服务端 complete 下发的 segments；历史消息本地按同一规则切分
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

  useEffect(() => {
    if (!isAI || actions.length === 0) return
    for (const a of actions) pushAction(a)
  }, [isAI, actions, pushAction])

  if (isAI) {
    const multi = segments.length > 1
    return (
      <div className="flex items-start gap-3 mb-4">
        <div className="flex-shrink-0">
          <Avatar src={aiAvatar} name={aiNickname} fallbackClass="bg-purple-500" />
        </div>
        <div className="flex flex-col max-w-[70%]">
          <span className="text-xs text-gray-400 mb-1 px-1">{aiNickname}</span>
          <div className="flex flex-col gap-2">
            {segments.map((seg) => (
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
          {multi && (
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
