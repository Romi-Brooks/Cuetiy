import { useEffect, useMemo, useState } from 'react'
import type { Message } from '../types/api'
import { formatTime } from '../utils'
import { resolveAssetUrl } from '../utils/url'
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

  const { display, actions } = useMemo(() => {
    const acts = extractEmotionalActions(content)
    const display = isAI
      ? actionsEnabled
        ? content // 开启动作展示时保留原文（后续可换成 emoji）
        : stripEmotionalForDisplay(content)
      : content
    return { display, actions: acts }
  }, [content, isAI, actionsEnabled])

  // 解析到动作时记入 store（供后续表情映射），与是否展示无关
  useEffect(() => {
    if (!isAI || actions.length === 0) return
    for (const a of actions) pushAction(a)
  }, [isAI, actions, pushAction])

  if (isAI) {
    return (
      <div className="flex items-start gap-3 mb-4">
        <div className="flex-shrink-0">
          <Avatar src={aiAvatar} name={aiNickname} fallbackClass="bg-purple-500" />
        </div>
        <div className="flex flex-col max-w-[70%]">
          <span className="text-xs text-gray-400 mb-1 px-1">{aiNickname}</span>
          <div className="chat-bubble-ai whitespace-pre-wrap break-words">{display}</div>
          {message?.created_at && (
            <span className="text-xs text-gray-400 mt-1 px-1">
              {formatTime(message.created_at)}
            </span>
          )}
        </div>
      </div>
    )
  }

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
