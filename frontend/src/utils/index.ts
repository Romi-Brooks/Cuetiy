import type { Message } from '../types/api'

export { resolveAssetUrl, defaultAvatarUrl, DEFAULT_AVATAR_PATH } from './url'

export function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const oneDay = 24 * 60 * 60 * 1000

  if (diff < oneDay && date.getDate() === now.getDate()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }

  if (diff < 2 * oneDay) {
    return '昨天 ' + date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }

  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}/${date.getDate()} ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  }

  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()}`
}

export function formatConversationTime(dateStr: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const oneDay = 24 * 60 * 60 * 1000

  if (diff < oneDay && date.getDate() === now.getDate()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }

  if (diff < 2 * oneDay) {
    return '昨天'
  }

  if (diff < 7 * oneDay) {
    const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
    return days[date.getDay()]
  }

  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}/${date.getDate()}`
  }

  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()}`
}

export function truncate(str: string, len = 30): string {
  if (!str) return ''
  return str.length > len ? str.substring(0, len) + '...' : str
}

export function getLastMessage(messages: Message[] | null | undefined): string {
  if (!messages || messages.length === 0) return ''
  const last = messages[messages.length - 1]
  return truncate(last.content, 50)
}
