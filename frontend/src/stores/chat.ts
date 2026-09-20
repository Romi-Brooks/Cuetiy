import { create } from 'zustand'
import { conversationAPI, personaAPI } from '../api'
import { getWsUrl } from '../utils/server'
import type { Conversation, Message, Persona } from '../types/api'
import {
  getLocalMessages,
  saveLocalMessages,
  appendLocalMessage,
  clearLocalMessages,
} from '../utils/storage'
import { defaultAvatarUrl, resolveAssetUrl } from '../utils/url'
import { loadHumanizeSettings, randomReplyDelayMs } from './settings'

export interface PromptPart {
  name: string
  content: string
  tokens: number
}

export interface TTSDebugInfo {
  model: string
  api_base: string
  voice: string
  voice_mode: string
  format: string
  style_prompt: string
  emotion?: string
  audio_style_tag?: string
  text_sent: string
  text_original: string
  char_count: number
  truncated: boolean
  audio_url: string
  storage_path: string
  audio_bytes: number
  cache_hit: boolean
  duration_ms: number
  latency_ms: number
  error?: string
}

export interface ContextDebugInfo {
  token_budget: number
  compact_threshold: number
  trigger_tokens: number
  estimated_tokens: number
  occupancy: number
  system_tokens: number
  recent_tokens: number
  user_tokens: number
  system_parts: PromptPart[]
  summary: string
  summary_version: number
  compacted: boolean
  memory: string
  activated_skills: string[]
  emotion: string
  emotion_reason: string
  recent_count: number
  covered_message_id: number
  full_system: string
  /** 本轮真正发给 LLM 的 messages */
  sent_messages?: { role: string; content: string }[]
  /** 与 sent_messages 等价的完整预览文本 */
  llm_preview?: string
}

interface ChatState {
  conversations: Conversation[]
  currentConversationId: number | null
  messages: Message[]
  isLoading: boolean
  isStreaming: boolean
  streamingContent: string
  totalMessages: number
  wsConnected: boolean
  currentPersona: Persona | null
  liveAiAvatar: string
  liveAiNickname: string
  contextDebug: ContextDebugInfo | null
  ttsDebug: TTSDebugInfo | null
  /** 按消息 id 存的 TTS debug（语音条） */
  msgTtsDebug: Record<number, TTSDebugInfo>
  /** 按 audio_url 存（刷新后 id 变了也能对上） */
  ttsDebugByUrl: Record<string, TTSDebugInfo>
  /** 按消息 id 存的当轮 context debug */
  msgCtxDebug: Record<number, ContextDebugInfo>

  fetchConversations: () => Promise<void>
  selectConversation: (id: number) => Promise<void>
  loadMoreMessages: () => Promise<void>
  clearMessages: (convId?: number, keepMemory?: boolean) => Promise<{ message: string; archive_path?: string; archive_count?: number } | void>
  updateConversationConfig: (convId: number, config: Record<string, unknown>) => Promise<void>
  setConversationPersona: (personaId: number | null) => Promise<void>
  createConversation: (title?: string) => Promise<Conversation>
  openPersonaChat: (personaId: number) => Promise<Conversation>
  connectWebSocket: () => void
  sendMessage: (content: string, wantVoice?: boolean) => boolean
  disconnectWebSocket: () => void
  /** 同步当前人格字段（如 background） */
  patchCurrentPersona: (patch: Partial<Persona>) => void
}

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectAttempts = 0
let intentionalClose = false
let typingTimer: ReturnType<typeof setInterval> | null = null
let replyDelayTimer: ReturnType<typeof setTimeout> | null = null

const MAX_RECONNECT_ATTEMPTS = 8
const LAST_CONV_KEY = 'cuetiy:last_conv_id'
const DEBUG_STORE_KEY = 'cuetiy:debug_store'

type DebugPersist = {
  contextDebug?: ContextDebugInfo | null
  ttsDebug?: TTSDebugInfo | null
  msgTtsDebug?: Record<number, TTSDebugInfo>
  ttsDebugByUrl?: Record<string, TTSDebugInfo>
  msgCtxDebug?: Record<number, ContextDebugInfo>
}

function loadDebugStore(): DebugPersist {
  try {
    const raw = localStorage.getItem(DEBUG_STORE_KEY)
    if (!raw) return {}
    return JSON.parse(raw) as DebugPersist
  } catch {
    return {}
  }
}

function saveDebugStore(partial: DebugPersist) {
  try {
    const next = { ...loadDebugStore(), ...partial }
    if (next.msgTtsDebug) {
      const keys = Object.keys(next.msgTtsDebug)
      if (keys.length > 60) {
        const sorted = keys.map(Number).sort((a, b) => a - b)
        for (const id of sorted.slice(0, keys.length - 60)) {
          delete next.msgTtsDebug[id]
        }
      }
    }
    if (next.msgCtxDebug) {
      const keys = Object.keys(next.msgCtxDebug)
      if (keys.length > 60) {
        const sorted = keys.map(Number).sort((a, b) => a - b)
        for (const id of sorted.slice(0, keys.length - 60)) {
          delete next.msgCtxDebug[id]
        }
      }
    }
    localStorage.setItem(DEBUG_STORE_KEY, JSON.stringify(next))
  } catch {
    /* ignore */
  }
}

let pendingTtsDebug: TTSDebugInfo | null = null
let pendingCtxDebug: ContextDebugInfo | null = null

function stopTypingAnimation() {
  if (typingTimer) {
    clearInterval(typingTimer)
    typingTimer = null
  }
}

function stopReplyDelay() {
  if (replyDelayTimer) {
    clearTimeout(replyDelayTimer)
    replyDelayTimer = null
  }
}

function stopAllHumanize() {
  stopTypingAnimation()
  stopReplyDelay()
}

/**
 * 按字数估算「对方正在输入中」最少停留时间。
 * 接近真人：短句也要有反应时间，长文不能瞬间蹦出。
 */
function estimateTypingHoldMs(charCount: number, hs: ReturnType<typeof loadHumanizeSettings>): number {
  if (!hs.typingEnabled) return 0
  // 每字间隔：默认 40ms；用户可调到 500（一字 0.5s）
  const perChar = Math.max(15, hs.typingCharDelayMs || 40)
  // 起步读消息/想怎么回
  const base = 600
  const minHold = 800
  const maxHold = 12000
  const raw = base + charCount * perChar
  return Math.round(Math.min(maxHold, Math.max(minHold, raw)))
}

function saveLastConvId(id: number | null) {
  try {
    if (id) localStorage.setItem(LAST_CONV_KEY, String(id))
  } catch {
    /* ignore */
  }
}

function loadLastConvId(): number | null {
  try {
    const v = localStorage.getItem(LAST_CONV_KEY)
    if (!v) return null
    const n = Number(v)
    return Number.isFinite(n) && n > 0 ? n : null
  } catch {
    return null
  }
}

function clearReconnectTimer() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

async function loadMessages(
  convId: number,
  set: (partial: Partial<ChatState>) => void,
  get: () => ChatState,
  limit = 50,
  offset = 0,
) {
  if (!convId) return

  // 首屏：先用本地缓存秒开，再请求服务器覆盖
  if (offset === 0) {
    const cached = await getLocalMessages(convId)
    if (cached.length > 0) {
      set({ messages: cached, totalMessages: cached.length })
    }
  }

  set({ isLoading: true })
  try {
    const res = await conversationAPI.getMessages(convId, limit, offset)
    if (get().currentConversationId !== convId) return

    if (offset === 0) {
      const msgs = res.messages || []
      set({ messages: msgs, totalMessages: res.total || 0 })
      if (msgs.length) saveLocalMessages(convId, msgs)
    } else {
      const existing = get().messages
      const existingIds = new Set(existing.map((m) => m.id))
      const older = (res.messages || []).filter((m) => !existingIds.has(m.id))
      set({
        messages: [...older, ...existing],
        totalMessages: res.total || get().totalMessages,
      })
    }
  } catch (e) {
    // 缓存已有内容时静默失败，避免整页空白
    if (get().messages.length === 0) throw e
  } finally {
    if (get().currentConversationId === convId) {
      set({ isLoading: false })
    }
  }
}

export const useChatStore = create<ChatState>((set, get) => {
  const scheduleReconnect = () => {
    if (intentionalClose) return
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) return

    clearReconnectTimer()
    const delay = Math.min(1000 * 2 ** reconnectAttempts, 15000)
    reconnectAttempts += 1
    reconnectTimer = setTimeout(() => {
      get().connectWebSocket()
    }, delay)
  }

  return {
    conversations: [],
    currentConversationId: null,
    messages: [],
    isLoading: false,
    isStreaming: false,
    streamingContent: '',
    totalMessages: 0,
    wsConnected: false,
    currentPersona: null,
    liveAiAvatar: '',
    liveAiNickname: '',
    contextDebug: null,
    ttsDebug: null,
    msgTtsDebug: loadDebugStore().msgTtsDebug || {},
    ttsDebugByUrl: loadDebugStore().ttsDebugByUrl || {},
    msgCtxDebug: loadDebugStore().msgCtxDebug || {},

    fetchConversations: async () => {
      const res = await conversationAPI.getConversations()
      set({ conversations: res.conversations || [] })
    },

    selectConversation: async (id) => {
      stopAllHumanize()
      saveLastConvId(id)
      const persisted = loadDebugStore()
      set({
        currentConversationId: id,
        liveAiAvatar: '',
        liveAiNickname: '',
        messages: [],
        totalMessages: 0,
        contextDebug: persisted.contextDebug || null,
        ttsDebug: persisted.ttsDebug || null,
        msgTtsDebug: persisted.msgTtsDebug || {},
        ttsDebugByUrl: persisted.ttsDebugByUrl || {},
        msgCtxDebug: persisted.msgCtxDebug || {},
        isStreaming: false,
        streamingContent: '',
      })
      try {
        await loadMessages(id, set, get)
      } catch (e) {
        console.error('加载消息失败:', e)
      }
      try {
        const res = await personaAPI.getConversationPersona(id)
        if (get().currentConversationId === id) {
          const p = res.persona || null
          set({ currentPersona: p })
          if (p?.background) {
            // 供聊天底图使用；MainChat 也会再拉一次 getPersona 兜底
          }
        }
      } catch {
        if (get().currentConversationId === id) {
          set({ currentPersona: null })
        }
      }
    },

    loadMoreMessages: async () => {
      const { messages, totalMessages, currentConversationId, isLoading } = get()
      if (!currentConversationId || isLoading || messages.length >= totalMessages) return
      set({ isLoading: true })
      try {
        // 用最旧一条真实消息的 id 做游标，避免乐观临时 id 干扰 offset
        const oldestServer = messages.find((m) => m.id > 0 && m.id < 1e12)
        const beforeId = oldestServer?.id || 0
        const res = await conversationAPI.getMessages(currentConversationId, 50, 0, beforeId)
        if (get().currentConversationId !== currentConversationId) return
        const existingIds = new Set(messages.map((m) => m.id))
        const older = (res.messages || []).filter((m) => !existingIds.has(m.id) && m.id !== beforeId)
        set({
          messages: [...older, ...messages],
          totalMessages: res.total || get().totalMessages,
        })
      } finally {
        if (get().currentConversationId === currentConversationId) {
          set({ isLoading: false })
        }
      }
    },

    clearMessages: async (convId, keepMemory = false) => {
      const id = convId || get().currentConversationId
      if (!id) return
      const res = await conversationAPI.clearMessages(id, keepMemory)
      set({ messages: [], totalMessages: 0 })
      clearLocalMessages(id)
      return res
    },

    updateConversationConfig: async (convId, config) => {
      const res = await conversationAPI.updateConfig(convId, config)
      const conversations = get().conversations.map((c) =>
        c.id === convId ? res.conversation : c,
      )
      const patch: Partial<ChatState> = { conversations }
      if (config.ai_avatar) patch.liveAiAvatar = config.ai_avatar as string
      if (config.ai_nickname) patch.liveAiNickname = config.ai_nickname as string
      set(patch)
    },

    setConversationPersona: async (personaId) => {
      const convId = get().currentConversationId
      if (!convId) return
      const personaIdVal = personaId || null
      const res = await personaAPI.setConversationPersona(convId, personaIdVal)
      set({
        conversations: get().conversations.map((c) => (c.id === convId ? res.conversation : c)),
      })
      if (personaIdVal) {
        const detail = await personaAPI.getPersona(personaIdVal)
        set({ currentPersona: detail.persona })
      } else {
        set({ currentPersona: null })
      }
    },

    createConversation: async (title = '情感陪伴') => {
      const res = await conversationAPI.createConversation(title)
      set({ conversations: [res.conversation, ...get().conversations] })
      return res.conversation
    },

    /** 打开某人格的专属会话（一人格一会话；已存在则复用） */
    openPersonaChat: async (personaId) => {
      const res = await personaAPI.openPersonaConversation(personaId)
      const conv = res.conversation
      const list = get().conversations.filter((c) => c.id !== conv.id)
      set({ conversations: [conv, ...list] })
      if (res.persona) {
        set({ currentPersona: res.persona })
      }
      await get().selectConversation(conv.id)
      return conv
    },

    patchCurrentPersona: (patch) => {
      const cur = get().currentPersona
      if (!cur) return
      set({ currentPersona: { ...cur, ...patch } })
    },

    connectWebSocket: () => {
      const token = localStorage.getItem('token')
      if (!token) return

      intentionalClose = false
      clearReconnectTimer()

      if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
        return
      }

      if (ws) {
        try {
          ws.close()
        } catch {
          /* ignore */
        }
        ws = null
      }

      const wsPath = getWsUrl()
      ws = new WebSocket(`${wsPath}${wsPath.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}`)

      ws.onopen = () => {
        reconnectAttempts = 0
        set({ wsConnected: true })
      }

      ws.onmessage = (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data) as {
            type: string
            content?: string
            message_id?: number
            conversation_id?: number
            debug?: unknown
            tts_debug?: TTSDebugInfo
            context_debug?: ContextDebugInfo
            message_type?: 'text' | 'voice' | 'image'
            audio_url?: string
            audio_duration_ms?: number
            segments?: Message['segments']
            image_url?: string
            image_debug?: unknown
          }
          switch (data.type) {
            case 'context_debug': {
              if (data.debug) {
                const dbg = data.debug as ContextDebugInfo
                pendingCtxDebug = dbg
                set({ contextDebug: dbg })
                saveDebugStore({ contextDebug: dbg })
              }
              break
            }
            case 'tts_debug': {
              if (data.debug) {
                const dbg = data.debug as TTSDebugInfo
                pendingTtsDebug = dbg
                set({ ttsDebug: dbg })
                saveDebugStore({ ttsDebug: dbg })
              }
              break
            }
            case 'image_message': {
              const convId = get().currentConversationId
              if (!convId) break
              const msg: Message = {
                id: data.message_id || Date.now(),
                conversation_id: convId,
                role: 'assistant',
                message_type: 'image',
                content: data.content || '',
                image_url: data.image_url,
                attachment_url: data.image_url,
                attachment_type: 'image',
                has_attachment: true,
                created_at: new Date().toISOString(),
                is_deleted: false,
              }
              set({ messages: [...get().messages, msg] })
              appendLocalMessage(convId, msg)
              break
            }
            case 'ai_start':
              // 新一轮：清上一轮 pending TTS（context_debug 在 ai_start 之前到达，不能清）
              pendingTtsDebug = null
              set({ isStreaming: true, streamingContent: '' })
              break
            case 'stream':
              // 仅累积，不展示；结束后一次性上屏
              set({ streamingContent: get().streamingContent + (data.content || '') })
              break
            case 'complete': {
              stopAllHumanize()
              const streamingContent = get().streamingContent
              const content = data.content || streamingContent
              const currentConversationId = get().currentConversationId
              const msgId = data.message_id || Date.now()
              const messageType = (data.message_type as 'text' | 'voice') || 'text'

              const finalize = (full: string) => {
                // complete 内嵌 debug 优先；pending 仅作兜底
                const ttsFromComplete = data.tts_debug || null
                const ctxFromComplete = data.context_debug || null
                const tts = ttsFromComplete || pendingTtsDebug
                const ctx = ctxFromComplete || pendingCtxDebug
                pendingTtsDebug = null
                pendingCtxDebug = null

                const newMsg: Message = {
                  id: msgId,
                  conversation_id: currentConversationId!,
                  role: 'assistant',
                  message_type: messageType,
                  content: full,
                  audio_url: data.audio_url,
                  audio_duration_ms: data.audio_duration_ms,
                  segments: data.segments?.length ? data.segments : undefined,
                  created_at: new Date().toISOString(),
                  is_deleted: false,
                }
                const nextById = { ...get().msgTtsDebug }
                const nextByUrl = { ...get().ttsDebugByUrl }
                const nextCtx = { ...get().msgCtxDebug }

                if (ctx) {
                  nextCtx[msgId] = ctx
                }
                if (messageType === 'voice' && tts) {
                  nextById[msgId] = tts
                  const urls = [data.audio_url, tts.audio_url].filter(Boolean) as string[]
                  for (const u of urls) nextByUrl[u] = tts
                }

                saveDebugStore({
                  msgTtsDebug: nextById,
                  ttsDebugByUrl: nextByUrl,
                  msgCtxDebug: nextCtx,
                  // 总体面板：始终对齐「刚刚完成的这一轮」
                  contextDebug: ctx || get().contextDebug,
                  ttsDebug: messageType === 'voice' && tts ? tts : get().ttsDebug,
                })

                const patch: Partial<ChatState> = {
                  isStreaming: false,
                  streamingContent: '',
                  messages: [...get().messages, newMsg],
                  msgTtsDebug: nextById,
                  ttsDebugByUrl: nextByUrl,
                  msgCtxDebug: nextCtx,
                }
                if (ctx) patch.contextDebug = ctx
                // 文字轮不再展示上一轮 TTS
                if (messageType === 'voice' && tts) {
                  patch.ttsDebug = tts
                } else if (messageType === 'text') {
                  patch.ttsDebug = null
                }
                set(patch)
                if (currentConversationId) appendLocalMessage(currentConversationId, newMsg)
              }

              const hs = loadHumanizeSettings()
              const charCount = Array.from(content || '').length
              const holdMs = estimateTypingHoldMs(charCount, hs)

              if (holdMs > 0) {
                set({ isStreaming: true, streamingContent: '' })
                replyDelayTimer = setTimeout(() => {
                  replyDelayTimer = null
                  finalize(content)
                }, holdMs)
                break
              }

              finalize(content)
              break
            }
            case 'error':
              set({ isStreaming: false, streamingContent: '' })
              console.error('AI error:', data.content)
              break
          }
        } catch (e) {
          console.error('WS parse error:', e)
        }
      }

      ws.onclose = () => {
        set({ wsConnected: false, isStreaming: false })
        ws = null
        scheduleReconnect()
      }
      ws.onerror = () => {
        set({ wsConnected: false })
      }
    },

    sendMessage: (content, wantVoice = false) => {
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        get().connectWebSocket()
        return false
      }
      const currentConversationId = get().currentConversationId
      if (!currentConversationId) return false

      const userMsg: Message = {
        id: Date.now(),
        conversation_id: currentConversationId,
        role: 'user',
        message_type: 'text',
        content,
        created_at: new Date().toISOString(),
        is_deleted: false,
      }

      const hs = loadHumanizeSettings()
      const thinkMs = randomReplyDelayMs(hs)

      set({
        messages: [...get().messages, userMsg],
        isStreaming: thinkMs <= 0,
        streamingContent: '',
      })
      appendLocalMessage(currentConversationId, userMsg)

      if (thinkMs > 0) {
        stopReplyDelay()
        replyDelayTimer = setTimeout(() => {
          replyDelayTimer = null
          if (get().currentConversationId === currentConversationId) {
            set({ isStreaming: true })
          }
        }, thinkMs)
      }

      try {
        ws.send(
          JSON.stringify({
            conversation_id: currentConversationId,
            content,
            want_voice: wantVoice,
          }),
        )
      } catch (e) {
        console.error('WebSocket send failed:', e)
        stopAllHumanize()
        set({ isStreaming: false })
        return false
      }
      return true
    },

    disconnectWebSocket: () => {
      intentionalClose = true
      clearReconnectTimer()
      reconnectAttempts = 0
      if (ws) {
        ws.close()
        ws = null
      }
      set({ wsConnected: false, isStreaming: false })
    },
  }
})

export function selectAiNickname(s: ChatState): string {
  const conv = s.conversations.find((c) => c.id === s.currentConversationId)
  return (
    s.liveAiNickname ||
    s.currentPersona?.nickname ||
    s.currentPersona?.name ||
    conv?.ai_nickname ||
    'Cuetiy'
  )
}

export function selectAiAvatar(s: ChatState): string {
  const conv = s.conversations.find((c) => c.id === s.currentConversationId)
  return (
    resolveAssetUrl(
      s.liveAiAvatar || s.currentPersona?.avatar || conv?.ai_avatar || '',
    ) || defaultAvatarUrl()
  )
}

export { loadLastConvId, saveLastConvId }

export function selectCurrentConversation(s: ChatState): Conversation | null {
  return s.conversations.find((c) => c.id === s.currentConversationId) || null
}

export function selectPersonaName(s: ChatState): string {
  return s.currentPersona?.name || '默认'
}
