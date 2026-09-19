import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  selectAiAvatar,
  selectAiNickname,
  selectPersonaName,
  useChatStore,
  loadLastConvId,
} from '../stores/chat'
import { useUserAvatar, useUserEmail, useUserStore, useUsername } from '../stores/user'
import { useThemeStore } from '../stores/theme'
import { useSettingsStore } from '../stores/settings'
import { personaAPI, ttsAPI } from '../api'
import { compressImageToBlob } from '../utils/image'
import { resolveAssetUrl } from '../utils/url'
import type { Message, PersonaFromServer, PersonaFile } from '../types/api'
import { ChatBubble } from '../components/ChatBubble'
import { VoiceBubble } from '../components/VoiceBubble'
import { MsgDebugModal } from '../components/MsgDebugModal'
import { TimeStamp } from '../components/TimeStamp'
import { EmojiPicker } from '../components/EmojiPicker'
import { Toast } from '../components/Toast'
import { ServerDataPanel } from '../components/ServerDataPanel'
import { useActionStore } from '../stores/actions'

type TabKey = 'chat' | 'persona' | 'my'

const tabs: { key: TabKey; label: string; icon: React.ReactNode }[] = [
  {
    key: 'chat',
    label: '对话',
    icon: (
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    ),
  },
  {
    key: 'persona',
    label: '人格',
    icon: (
      <>
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
        <circle cx="12" cy="7" r="4" />
      </>
    ),
  },
  {
    key: 'my',
    label: '我的',
    icon: (
      <>
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
      </>
    ),
  },
]

const personaColors = [
  '#6366f1',
  '#8b5cf6',
  '#ec4899',
  '#f43f5e',
  '#f97316',
  '#eab308',
  '#22c55e',
  '#14b8a6',
  '#06b6d4',
  '#3b82f6',
]

function personaColor(name: string): string {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return personaColors[Math.abs(hash) % personaColors.length]
}

function moduleBadgeClass(category: string): string {
  const map: Record<string, string> = {
    persona_base: 'bg-pink-100 text-pink-600 dark:bg-pink-900/30 dark:text-pink-400',
    persona_tone: 'bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400',
    forbidden_rules: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400',
    emotion_companion: 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400',
    professional_skills: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400',
    style_switch: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400',
    trigger_rules: 'bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400',
  }
  return map[category] || 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
}

function TabIcon({ children }: { children: React.ReactNode }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className="w-5 h-5"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {children}
    </svg>
  )
}

function shouldShowTimestamp(messages: Message[], idx: number) {
  if (idx === 0) return true
  const prev = messages[idx - 1]
  const msg = messages[idx]
  if (!prev?.created_at || !msg?.created_at) return true
  return new Date(msg.created_at).getTime() - new Date(prev.created_at).getTime() > 5 * 60 * 1000
}

export function MainChat() {
  const navigate = useNavigate()
  const chat = useChatStore()
  const username = useUsername()
  const userEmail = useUserEmail()
  const userAvatar = useUserAvatar()
  const logout = useUserStore((s) => s.logout)
  const setAvatar = useUserStore((s) => s.setAvatar)
  const actionsEnabled = useActionStore((s) => s.enabled)
  const setActionsEnabled = useActionStore((s) => s.setEnabled)
  const recentActions = useActionStore((s) => s.recent)
  const isDark = useThemeStore((s) => s.isDark)
  const toggleTheme = useThemeStore((s) => s.toggleTheme)
  const typingEnabled = useSettingsStore((s) => s.typingEnabled)
  const typingCharDelayMs = useSettingsStore((s) => s.typingCharDelayMs)
  const setTypingEnabled = useSettingsStore((s) => s.setTypingEnabled)
  const setTypingCharDelayMs = useSettingsStore((s) => s.setTypingCharDelayMs)
  const replyDelayEnabled = useSettingsStore((s) => s.replyDelayEnabled)
  const replyDelayMinMs = useSettingsStore((s) => s.replyDelayMinMs)
  const replyDelayMaxMs = useSettingsStore((s) => s.replyDelayMaxMs)
  const setReplyDelayEnabled = useSettingsStore((s) => s.setReplyDelayEnabled)
  const setReplyDelayRange = useSettingsStore((s) => s.setReplyDelayRange)
  const [wantVoice, setWantVoice] = useState(false)
  const [userVoice, setUserVoice] = useState<{ url: string; original_name?: string } | null>(null)
  const [uploadingVoice, setUploadingVoice] = useState(false)
  const [asrBusy, setAsrBusy] = useState(false)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const asrChunksRef = useRef<BlobPart[]>([])

  const aiNickname = useChatStore(selectAiNickname)
  const aiAvatar = useChatStore(selectAiAvatar)
  const personaName = useChatStore(selectPersonaName)
  const currentPersona = useChatStore((s) => s.currentPersona)

  const [activeTab, setActiveTab] = useState<TabKey>('chat')
  const [isMobile, setIsMobile] = useState(() => window.innerWidth < 1024)
  const [inputContent, setInputContent] = useState('')
  const [showEmojiPicker, setShowEmojiPicker] = useState(false)
  const [aiAvatarError, setAiAvatarError] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const [uploadingAvatar, setUploadingAvatar] = useState(false)
  const [clearingChat, setClearingChat] = useState(false)

  const [personas, setPersonas] = useState<PersonaFromServer[]>([])
  const [personaSearchQuery, setPersonaSearchQuery] = useState('')
  const [personaFilesMap, setPersonaFilesMap] = useState<Record<number, PersonaFile[]>>({})
  const [showAddPersonaModal, setShowAddPersonaModal] = useState(false)
  const [addPersonaSystemName, setAddPersonaSystemName] = useState('')
  const [addPersonaNickname, setAddPersonaNickname] = useState('')
  const [addPersonaFiles, setAddPersonaFiles] = useState<File[]>([])
  const [addPersonaFileErrors, setAddPersonaFileErrors] = useState<string[]>([])
  const [submittingAddPersona, setSubmittingAddPersona] = useState(false)
  const [editingPersona, setEditingPersona] = useState<PersonaFromServer | null>(null)
  const [editPersonaName, setEditPersonaName] = useState('')
  const [editPersonaDesc, setEditPersonaDesc] = useState('')
  const [editPersonaAvatarUrl, setEditPersonaAvatarUrl] = useState('')
  const [submittingEditPersona, setSubmittingEditPersona] = useState(false)
  const [uploadingPersonaAvatar, setUploadingPersonaAvatar] = useState(false)

  const [debugExpandedMsgId, setDebugExpandedMsgId] = useState<number | null>(null)
  const [debugMsgContent, setDebugMsgContent] = useState('')
  const debugSystemCache = useRef('')
  const [showContextDebug, setShowContextDebug] = useState(false)
  const [debugPartTab, setDebugPartTab] = useState(0)
  const [msgDebugOpen, setMsgDebugOpen] = useState(false)

  const messagesContainer = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const loadMoreSentinel = useRef<HTMLDivElement | null>(null)
  const bottomSentinel = useRef<HTMLDivElement | null>(null)
  const isAutoScrolling = useRef(false)
  const loadMoreObserver = useRef<IntersectionObserver | null>(null)
  const lastMessageCount = useRef(0)
  const lastFirstMessageId = useRef<number | null>(null)

  const showToast = useCallback((msg: string) => setToast(msg), [])

  const filteredPersonas = useMemo(() => {
    if (!personaSearchQuery) return personas
    const q = personaSearchQuery.toLowerCase()
    return personas.filter((p) => p.name.toLowerCase().includes(q))
  }, [personas, personaSearchQuery])

  const fetchPersonas = useCallback(async () => {
    try {
      const res = await personaAPI.getPersonas()
      setPersonas(res.personas || [])
    } catch {
      /* ignore */
    }
  }, [])

  const handleLogout = useCallback(async () => {
    await logout()
    navigate('/login')
  }, [logout, navigate])

  const handleClearChat = useCallback(async () => {
    if (!chat.currentConversationId || clearingChat) return
    const ok = window.confirm(
      '确定清空当前对话？\n会先归档到服务器文件再删除消息；人格 Skills 与长期记忆会保留。',
    )
    if (!ok) return
    setClearingChat(true)
    try {
      const res = await chat.clearMessages()
      showToast(res?.archive_path ? '已清空（已归档可溯源）' : '已清空')
    } catch {
      showToast('清空失败')
    } finally {
      setClearingChat(false)
    }
  }, [chat, clearingChat, showToast])

  /** 录音 → ASR 转文字 → 填入输入框 */
  const toggleAsrRecord = useCallback(async () => {
    if (asrBusy) {
      mediaRecorderRef.current?.stop()
      return
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      // 优先浏览器能给的 wav/mp4；多数环境只有 webm，后端仅支持 wav/mp3
      const mime =
        MediaRecorder.isTypeSupported('audio/wav')
          ? 'audio/wav'
          : MediaRecorder.isTypeSupported('audio/mp4')
            ? 'audio/mp4'
            : 'audio/webm'
      const rec = new MediaRecorder(stream, { mimeType: mime })
      asrChunksRef.current = []
      rec.ondataavailable = (e) => {
        if (e.data.size > 0) asrChunksRef.current.push(e.data)
      }
      rec.onstop = async () => {
        stream.getTracks().forEach((t) => t.stop())
        const blob = new Blob(asrChunksRef.current, { type: mime })
        const ext = mime.includes('wav') ? 'wav' : mime.includes('mp4') ? 'mp4' : 'webm'
        setAsrBusy(true)
        showToast('正在识别...')
        try {
          const res = await ttsAPI.transcribe(blob, `audio.${ext}`, 'zh')
          if (res?.text) {
            setInputContent((prev) => (prev ? prev + res.text : res.text))
            showToast('已转为文字')
          } else {
            showToast('未识别到内容')
          }
        } catch (e) {
          console.error(e)
          showToast('转文字失败（需 wav/mp3）')
        } finally {
          setAsrBusy(false)
        }
      }
      mediaRecorderRef.current = rec
      rec.start()
      setAsrBusy(true)
      showToast('录音中，再点一次结束')
    } catch {
      showToast('无法访问麦克风')
      setAsrBusy(false)
    }
  }, [asrBusy, showToast])

  const scrollToBottom = useCallback(() => {
    isAutoScrolling.current = true
    requestAnimationFrame(() => {
      bottomSentinel.current?.scrollIntoView({ behavior: 'smooth' })
      setTimeout(() => {
        isAutoScrolling.current = false
      }, 300)
    })
  }, [])

  const autoGrow = useCallback(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    const lineHeight = parseInt(getComputedStyle(el).lineHeight) || 20
    el.style.height = `${Math.min(el.scrollHeight, lineHeight * 4 + 8)}px`
  }, [])

  const sendMessage = useCallback(() => {
    const text = inputContent.trim()
    if (!text || chat.isStreaming) return
    const ok = chat.sendMessage(text, wantVoice)
    if (!ok) {
      showToast('连接中断，正在重连，请稍后重试')
      return
    }
    setInputContent('')
    setWantVoice(false)
    autoGrow()
    scrollToBottom()
  }, [inputContent, chat, autoGrow, scrollToBottom, showToast, wantVoice])

  const switchToPersona = useCallback(
    async (persona: PersonaFromServer) => {
      try {
        await chat.openPersonaChat(persona.id)
        showToast(`已打开「${persona.nickname || persona.name}」的对话`)
        setActiveTab('chat')
      } catch (e) {
        console.error(e)
        showToast('打开对话失败')
      }
    },
    [chat, showToast],
  )

  const openEditPersona = useCallback((p: PersonaFromServer) => {
    setEditingPersona(p)
    setEditPersonaName(p.nickname || p.name)
    setEditPersonaDesc(p.description || '')
    setEditPersonaAvatarUrl(p.avatar || '')
    setSubmittingEditPersona(false)
  }, [])

  const openPersonaSettings = useCallback(
    async (p: PersonaFromServer) => {
      openEditPersona(p)
      try {
        const res = await personaAPI.getPersona(p.id)
        setPersonaFilesMap((m) => ({ ...m, [p.id]: res.persona_files || [] }))
      } catch {
        setPersonaFilesMap((m) => ({ ...m, [p.id]: [] }))
      }
    },
    [openEditPersona],
  )

  const toggleMsgDebug = useCallback(
    async (msgId: number, msgIdx: number) => {
      if (debugExpandedMsgId === msgId && msgDebugOpen) {
        setMsgDebugOpen(false)
        setDebugExpandedMsgId(null)
        return
      }

      // 优先：本轮 context_debug（真正发给 LLM 的组装结果，含压缩后 system/history）
      const ctx = chat.msgCtxDebug[msgId] || null
      if (ctx?.llm_preview) {
        setDebugMsgContent(ctx.llm_preview)
        setDebugExpandedMsgId(msgId)
        setMsgDebugOpen(true)
        return
      }
      if (ctx?.sent_messages?.length || ctx?.full_system) {
        const sys = ctx.full_system || ctx.sent_messages?.find((m) => m.role === 'system')?.content || ''
        const hist = (ctx.sent_messages || [])
          .filter((m) => m.role !== 'system')
          .map((m) => `${m.role === 'assistant' ? 'Assistant' : 'User'}: ${m.content}`)
          .join('\n\n')
        const badge = ctx.compacted
          ? `[compact] 本轮已压缩 · summary v${ctx.summary_version} · recent=${ctx.recent_count} · covered_id=${ctx.covered_message_id}\n\n`
          : ctx.summary
            ? `[summary v${ctx.summary_version}] HISTORY 仅含摘要覆盖后的消息 · recent=${ctx.recent_count}\n\n`
            : ''
        setDebugMsgContent(
          `${badge}=== System（本轮实际） ===\n${sys}\n\n=== History（压缩后 recent + 当前输入） ===\n${hist}`,
        )
        setDebugExpandedMsgId(msgId)
        setMsgDebugOpen(true)
        return
      }

      // 回退：历史消息可能没有当轮组装记录（旧会话 / 刷新前未缓存）
      // 此时展示人格全文 + 界面可见历史，并明确标注「非压缩后实际输入」
      if (!debugSystemCache.current && chat.currentConversationId) {
        try {
          const res = await personaAPI.getDebugPrompt(chat.currentConversationId)
          debugSystemCache.current = res.system_prompt || ''
        } catch {
          debugSystemCache.current = ''
        }
      }
      const messages = chat.messages.slice(0, msgIdx + 1)
      const history = messages
        .map((m) => {
          const tag = m.role === 'assistant' ? 'Assistant' : 'User'
          const body =
            m.message_type === 'voice'
              ? `[语音条 ${Math.round((m.audio_duration_ms || 0) / 1000)}s]`
              : m.content
          return `${tag}: ${body}`
        })
        .join('\n\n')

      setDebugMsgContent(
        `⚠️ 本条无「当轮组装」缓存。\n` +
          `以下 = 人格编译全文 + 界面可见历史，**不是**压缩后的实际 LLM 输入。\n` +
          `请看「上下文」页签，或点 AI 昵称打开本轮 Debug。\n\n` +
          `=== System（人格全文，非 L0 分层） ===\n${debugSystemCache.current}\n\n` +
          `=== Conversation Context（界面历史，未压缩） ===\n${history}`,
      )
      setDebugExpandedMsgId(msgId)
      setMsgDebugOpen(true)
    },
    [debugExpandedMsgId, msgDebugOpen, chat],
  )

  // lifecycle
  useEffect(() => {
    const onResize = () => {
      setIsMobile(window.innerWidth < 1024)
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        await chat.fetchConversations()
      } catch {
        /* ignore */
      }
      if (cancelled) return
      const state = useChatStore.getState()
      // 默认打开上次对话；否则第一条
      const lastId = loadLastConvId()
      const prefer =
        (lastId && state.conversations.find((c) => c.id === lastId)) ||
        state.conversations[0]
      if (prefer && !state.currentConversationId) {
        await state.selectConversation(prefer.id)
      }
      setAiAvatarError(false)
      state.connectWebSocket()
      scrollToBottom()
      autoGrow()
      fetchPersonas()
      try {
        const v = await ttsAPI.getMyVoice()
        setUserVoice(v.voice ? { url: v.voice.url, original_name: v.voice.original_name } : null)
      } catch {
        /* ignore */
      }
    })()
    return () => {
      cancelled = true
      useChatStore.getState().disconnectWebSocket()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    const el = loadMoreSentinel.current
    if (!el) return
    loadMoreObserver.current?.disconnect()
    loadMoreObserver.current = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !useChatStore.getState().isLoading && useChatStore.getState().messages.length > 0) {
          useChatStore.getState().loadMoreMessages()
        }
      },
      { threshold: 0.1 },
    )
    loadMoreObserver.current.observe(el)
    return () => loadMoreObserver.current?.disconnect()
  }, [chat.messages.length, chat.currentConversationId])

  // 上翻加载历史时保持滚动位置；新消息或流式输出时才自动滚底
  useEffect(() => {
    const el = messagesContainer.current
    const first = chat.messages[0]
    const firstId = first?.id ?? null
    const prevFirstId = lastFirstMessageId.current
    const prevCount = lastMessageCount.current
    const count = chat.messages.length

    if (
      el &&
      prevFirstId !== null &&
      firstId !== null &&
      firstId !== prevFirstId &&
      count > prevCount
    ) {
      // 前置了更旧消息：保持底部不跳
      const delta = el.scrollHeight - el.scrollTop - el.clientHeight
      requestAnimationFrame(() => {
        el.scrollTop = el.scrollHeight - el.clientHeight - delta
      })
    } else if (!isAutoScrolling.current && !chat.isLoading) {
      scrollToBottom()
    }

    lastMessageCount.current = count
    lastFirstMessageId.current = firstId
  }, [chat.messages, chat.isLoading, scrollToBottom])

  useEffect(() => {
    if (chat.isStreaming) scrollToBottom()
  }, [chat.isStreaming, scrollToBottom])

  useEffect(() => {
    setAiAvatarError(false)
    debugSystemCache.current = ''
    setDebugExpandedMsgId(null)
    setDebugMsgContent('')
  }, [chat.currentConversationId])

  return (
    <div className="h-[100dvh] flex flex-col overflow-hidden bg-white dark:bg-wechat-bg-dark md:flex-row">
      {/* Desktop tab bar */}
      {!isMobile && (
        <div className="hidden md:flex flex-col items-center w-16 bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 py-4 shrink-0 desktop-tabbar">
          {tabs.map((t) => (
            <button
              key={t.key}
              className={`w-12 h-12 rounded-xl flex flex-col items-center justify-center gap-0.5 transition-colors ${
                activeTab === t.key
                  ? 'bg-wechat-green/10 text-wechat-green'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'
              }`}
              onClick={() => setActiveTab(t.key)}
            >
              <TabIcon>{t.icon}</TabIcon>
              <span className="text-[10px] font-medium">{t.label}</span>
            </button>
          ))}
          <div className="flex-1" />
          <button
            className="w-12 h-12 rounded-xl flex flex-col items-center justify-center text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
            onClick={handleLogout}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-5 h-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
          </button>
        </div>
      )}

      {/* Main area: content + mobile tabs */}
      <div className="flex-1 flex flex-col min-h-0 min-w-0 overflow-hidden">
      {/* CHAT：入口在人格页，这里只展示当前会话 */}
      {activeTab === 'chat' && (
        <div className="flex-1 flex flex-col min-h-0 min-w-0 overflow-hidden">
          <main className="flex-1 flex flex-col min-w-0 min-h-0">
            {chat.currentConversationId ? (
              <>
                <div className="app-header flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="w-9 h-9 rounded-full overflow-hidden flex-shrink-0">
                      {!aiAvatarError ? (
                        <img
                          src={resolveAssetUrl(aiAvatar)}
                          alt={aiNickname}
                          className="w-full h-full object-cover"
                          onError={() => setAiAvatarError(true)}
                        />
                      ) : (
                        <div className="w-full h-full bg-purple-500 flex items-center justify-center text-white text-sm font-medium">
                          {aiNickname.charAt(0)}
                        </div>
                      )}
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 min-w-0">
                        <h2 className="text-base font-semibold text-gray-800 dark:text-gray-100 truncate">
                          {aiNickname}
                        </h2>
                        <button
                          type="button"
                          className="shrink-0 text-[11px] font-medium px-2 py-0.5 rounded-full border border-wechat-green/40 text-wechat-green bg-wechat-green/5 hover:bg-wechat-green/15 transition-colors"
                          onClick={() => setShowContextDebug(true)}
                          title="查看上下文 / TTS Debug"
                        >
                          Debug
                        </button>
                      </div>
                      {chat.isStreaming && (
                        <p className="text-xs text-wechat-green">对方正在输入中</p>
                      )}
                      <p className="text-xs text-gray-400">
                        {chat.wsConnected ? '在线' : '连接中...'}
                      </p>
                    </div>
                  </div>
                  <button
                    className="shrink-0 text-xs text-gray-500 dark:text-gray-400 border border-gray-200 dark:border-gray-600 rounded-lg px-2.5 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors disabled:opacity-50"
                    disabled={clearingChat}
                    onClick={handleClearChat}
                    title="清空当前对话（先归档）"
                  >
                    {clearingChat ? '清空中...' : '清空记录'}
                  </button>
                </div>

                <div
                  ref={messagesContainer}
                  className="flex-1 overflow-y-auto overscroll-contain px-4 py-4 space-y-1 bg-gray-50 dark:bg-gray-800/50"
                >
                  <div ref={loadMoreSentinel} className="h-4 flex items-center justify-center">
                    {chat.isLoading && <span className="text-xs text-gray-400">加载中...</span>}
                  </div>
                  {chat.messages.map((msg, idx) => (
                    <div key={msg.id ?? idx}>
                      {shouldShowTimestamp(chat.messages, idx) && <TimeStamp time={msg.created_at} />}
                      {msg.message_type === 'voice' && msg.audio_url ? (
                        <div
                          className={`flex items-start gap-3 mb-2 ${
                            msg.role === 'user' ? 'flex-row-reverse' : ''
                          }`}
                        >
                          <div className="w-10 h-10 rounded-full overflow-hidden shrink-0 bg-purple-500 flex items-center justify-center text-white text-sm">
                            {msg.role === 'user' ? username.charAt(0) : aiNickname.charAt(0)}
                          </div>
                          <VoiceBubble
                            audioUrl={resolveAssetUrl(msg.audio_url)}
                            durationMs={msg.audio_duration_ms}
                            text={msg.content}
                            isAI={msg.role === 'assistant'}
                            nickname={aiNickname}
                            onDebug={
                              msg.role === 'assistant'
                                ? () => toggleMsgDebug(msg.id, idx)
                                : undefined
                            }
                            debugOpen={msgDebugOpen && debugExpandedMsgId === msg.id}
                          />
                        </div>
                      ) : (
                        <>
                          <ChatBubble
                            message={msg}
                            isAI={msg.role === 'assistant'}
                            aiNickname={aiNickname}
                            aiAvatar={aiAvatar}
                            userNickname={username}
                            userAvatar={userAvatar}
                          />
                          {msg.role === 'assistant' && (
                            <div className="flex justify-start pl-14 -mt-2 mb-3">
                              <button
                                className="text-[10px] text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer transition-colors"
                                onClick={() => toggleMsgDebug(msg.id, idx)}
                              >
                                {msgDebugOpen && debugExpandedMsgId === msg.id
                                  ? '收起 Debug'
                                  : 'Debug'}
                              </button>
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  ))}
                  {/* 微信式：流式期间不在消息区打点，仅标题显示「对方正在输入中」 */}
                  <div ref={bottomSentinel} />
                </div>

                <div className="bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 px-4 py-3 shrink-0">
                  <div className="flex items-end gap-2 sm:gap-3">
                    <div className="flex items-center gap-1 pb-1 shrink-0">
                      <button
                        className="flex items-center justify-center w-9 h-9 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                        onClick={() => setShowEmojiPicker(true)}
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="w-5 h-5"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <circle cx="12" cy="12" r="10" />
                          <path d="M8 14s1.5 2 4 2 4-2 4-2" />
                          <line x1="9" y1="9" x2="9.01" y2="9" />
                          <line x1="15" y1="9" x2="15.01" y2="9" />
                        </svg>
                      </button>
                      <button
                        type="button"
                        title="语音转文字"
                        className={`flex items-center justify-center w-9 h-9 rounded-lg transition-colors ${
                          asrBusy
                            ? 'bg-red-100 text-red-500 dark:bg-red-900/30'
                            : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'
                        }`}
                        onClick={toggleAsrRecord}
                      >
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="w-5 h-5"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
                          <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
                          <line x1="12" y1="19" x2="12" y2="23" />
                          <line x1="8" y1="23" x2="16" y2="23" />
                        </svg>
                      </button>
                      <button
                        type="button"
                        title={wantVoice ? '本条要语音（已开）' : '本条要语音'}
                        className={`flex items-center justify-center w-9 h-9 rounded-lg transition-colors ${
                          wantVoice
                            ? 'bg-wechat-green/15 text-wechat-green'
                            : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'
                        }`}
                        onClick={() => setWantVoice((v) => !v)}
                      >
                        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
                          <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z" />
                        </svg>
                      </button>
                    </div>
                    <div className="flex-1 min-w-0">
                      <textarea
                        ref={textareaRef}
                        value={inputContent}
                        onChange={(e) => {
                          setInputContent(e.target.value)
                          autoGrow()
                        }}
                        className="w-full resize-none bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200 rounded-xl px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-wechat-green/30 transition-all leading-relaxed"
                        rows={1}
                        placeholder="输入消息..."
                        disabled={chat.isStreaming}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' && !e.shiftKey) {
                            e.preventDefault()
                            sendMessage()
                          }
                        }}
                      />
                    </div>
                    <button
                      className={`flex-shrink-0 flex items-center justify-center w-10 h-10 rounded-xl transition-colors ${
                        inputContent.trim() && !chat.isStreaming
                          ? 'bg-wechat-green hover:bg-wechat-green-dark text-white'
                          : 'bg-gray-200 dark:bg-gray-700 text-gray-400 dark:text-gray-500 cursor-not-allowed'
                      }`}
                      disabled={!inputContent.trim() || chat.isStreaming}
                      onClick={sendMessage}
                    >
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        className="w-5 h-5"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <line x1="22" y1="2" x2="11" y2="13" />
                        <polygon points="22 2 15 22 11 13 2 9 22 2" />
                      </svg>
                    </button>
                  </div>
                </div>
              </>
            ) : (
              <div className="flex-1 flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-800/50 text-gray-400 dark:text-gray-500">
                <h3 className="text-lg font-medium text-gray-600 dark:text-gray-400 mb-2">
                  欢迎使用 RainYi
                </h3>
                <p className="text-sm text-center max-w-xs px-4">
                  到「人格」页选择角色并点「对话」，即可开始专属聊天
                </p>
                <button
                  className="mt-4 btn-primary text-sm"
                  onClick={() => setActiveTab('persona')}
                >
                  去选人格
                </button>
              </div>
            )}
          </main>
        </div>
      )}

      {/* PERSONA */}
      {activeTab === 'persona' && (
        <div className="flex-1 flex flex-col bg-white dark:bg-gray-900 overflow-hidden min-h-0">
          <div className="app-header px-5 py-4 border-b border-gray-200 dark:border-gray-700 space-y-3 shrink-0">
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <input
                  value={personaSearchQuery}
                  onChange={(e) => setPersonaSearchQuery(e.target.value)}
                  type="text"
                  placeholder="搜索人格"
                  className="w-full pl-4 pr-4 py-2 text-sm bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-xl border-none outline-none focus:ring-2 focus:ring-wechat-green/30 transition-all"
                />
              </div>
              <button
                className="shrink-0 flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-white bg-wechat-green hover:bg-wechat-green-dark rounded-xl transition-colors"
                onClick={() => {
                  setAddPersonaSystemName('')
                  setAddPersonaNickname('')
                  setAddPersonaFiles([])
                  setAddPersonaFileErrors([])
                  setSubmittingAddPersona(false)
                  setShowAddPersonaModal(true)
                }}
              >
                添加
              </button>
            </div>
            <p className="text-xs text-gray-400">{filteredPersonas.length} 个角色</p>
          </div>

          <div className="flex-1 overflow-y-auto overscroll-contain pb-4">
            {filteredPersonas.map((p) => (
              <div
                key={p.id}
                className="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors border-b border-gray-100 dark:border-gray-800/50"
                onClick={() => openPersonaSettings(p)}
              >
                <div
                  className="w-12 h-12 rounded-full shrink-0 overflow-hidden"
                  style={{ backgroundColor: personaColor(p.name) }}
                >
                  {p.avatar ? (
                    <img src={resolveAssetUrl(p.avatar)} className="w-full h-full object-cover" alt="" />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-white text-base font-bold">
                      {p.name.charAt(0).toUpperCase()}
                    </div>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-semibold text-gray-800 dark:text-gray-200">
                      {p.nickname || p.name}
                    </p>
                    {p.is_built_in && (
                      <span className="text-[10px] text-wechat-green bg-wechat-green/10 rounded-full px-1.5 py-0.5">
                        官方
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-gray-400 truncate mt-0.5">
                    {p.description || '这个人很懒，什么都没留下~'}
                  </p>
                </div>
                <button
                  className="text-xs text-wechat-green border border-wechat-green/30 rounded-lg px-3 py-1 hover:bg-wechat-green/5 transition-colors shrink-0"
                  onClick={(e) => {
                    e.stopPropagation()
                    switchToPersona(p)
                  }}
                >
                  对话
                </button>
              </div>
            ))}
            {filteredPersonas.length === 0 && (
              <div className="flex flex-col items-center justify-center py-24 text-gray-400">
                <p className="text-sm">
                  {personaSearchQuery ? '没有匹配的角色' : '暂无可用角色'}
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* MY */}
      {activeTab === 'my' && (
        <div className="flex-1 flex flex-col bg-gray-50 dark:bg-gray-900 overflow-hidden min-h-0">
          <div className="flex-1 overflow-y-auto overscroll-contain">
            <div className="max-w-2xl mx-auto px-4 py-4 space-y-4 pb-[var(--safe-bottom)]">
              <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-sm overflow-hidden">
                <div className="app-header px-5 py-4 border-b border-gray-100 dark:border-gray-700">
                  <h2 className="text-base font-semibold text-gray-800 dark:text-gray-100">用户信息</h2>
                </div>
                <div className="px-5 py-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500 dark:text-gray-400">用户名</span>
                    <span className="text-sm font-medium text-gray-800 dark:text-gray-200">
                      {username}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500 dark:text-gray-400">邮箱</span>
                    <span className="text-sm font-medium text-gray-800 dark:text-gray-200">
                      {userEmail || '未设置'}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-500 dark:text-gray-400">头像</span>
                    <div className="flex items-center gap-2">
                      <img
                        src={resolveAssetUrl(userAvatar)}
                        className="w-8 h-8 rounded-full object-cover border border-gray-200 dark:border-gray-600"
                        alt=""
                      />
                      <label className="text-xs text-wechat-green cursor-pointer hover:text-wechat-green-dark transition-colors">
                        <span>{uploadingAvatar ? '上传中...' : '更换'}</span>
                        <input
                          type="file"
                          accept="image/*"
                          className="hidden"
                          disabled={uploadingAvatar}
                          onChange={async (e) => {
                            const raw = e.target.files?.[0]
                            if (!raw) return
                            setUploadingAvatar(true)
                            try {
                              let file = raw
                              if (file.size > 512 * 1024) {
                                showToast('正在压缩图片...')
                                file = await compressImageToBlob(file, {
                                  maxSizeBytes: 1024 * 1024,
                                  maxEdge: 720,
                                  quality: 0.85,
                                })
                              }
                              if (file.size > 1024 * 1024) {
                                showToast('压缩后仍超过 1MB，请换小一点的图')
                                return
                              }
                              const res = await personaAPI.uploadAvatar(file)
                              setAvatar(res.file.url)
                              showToast('头像已更新')
                            } catch {
                              showToast('上传失败')
                            } finally {
                              setUploadingAvatar(false)
                              e.target.value = ''
                            }
                          }}
                        />
                      </label>
                    </div>
                  </div>
                  <div className="flex items-center justify-between">
                    <div className="flex-1 pr-3">
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        拟人输入节奏
                      </span>
                      <p className="text-xs text-gray-400 mt-0.5">
                        回复到达后仍显示「对方正在输入中」，按字数停够时间再整段发出，避免瞬间蹦字
                      </p>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={typingEnabled}
                      className={`relative w-11 h-6 rounded-full transition-colors shrink-0 ${
                        typingEnabled ? 'bg-wechat-green' : 'bg-gray-300 dark:bg-gray-600'
                      }`}
                      onClick={() => setTypingEnabled(!typingEnabled)}
                    >
                      <span
                        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
                          typingEnabled ? 'translate-x-5' : ''
                        }`}
                      />
                    </button>
                  </div>
                  {typingEnabled && (
                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                        <span>每字节奏（估算停留）</span>
                        <span className="font-medium text-gray-700 dark:text-gray-200">
                          {typingCharDelayMs} ms/字
                        </span>
                      </div>
                      <input
                        type="range"
                        min={15}
                        max={500}
                        step={5}
                        value={typingCharDelayMs}
                        onChange={(e) => setTypingCharDelayMs(Number(e.target.value))}
                        className="w-full accent-emerald-500"
                      />
                      <p className="text-[11px] text-gray-400">
                        约 {Math.round(typingCharDelayMs)}ms×字数 + 0.6s 起步，总停留约 0.8～12s。拉到 500 接近一字 0.5 秒。
                      </p>
                    </div>
                  )}
                  <div className="flex items-center justify-between">
                    <div className="flex-1 pr-3">
                      <span className="text-sm text-gray-500 dark:text-gray-400">
                        回复前思考延迟
                      </span>
                      <p className="text-xs text-gray-400 mt-0.5">
                        发出后先等一下再开始「输入中」（模拟看到消息）
                      </p>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={replyDelayEnabled}
                      className={`relative w-11 h-6 rounded-full transition-colors shrink-0 ${
                        replyDelayEnabled ? 'bg-wechat-green' : 'bg-gray-300 dark:bg-gray-600'
                      }`}
                      onClick={() => setReplyDelayEnabled(!replyDelayEnabled)}
                    >
                      <span
                        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
                          replyDelayEnabled ? 'translate-x-5' : ''
                        }`}
                      />
                    </button>
                  </div>
                  {replyDelayEnabled && (
                    <div className="space-y-1.5">
                      <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                        <span>思考延迟范围</span>
                        <span className="font-medium text-gray-700 dark:text-gray-200">
                          {replyDelayMinMs}–{replyDelayMaxMs} ms
                        </span>
                      </div>
                      <div className="flex gap-2 items-center">
                        <input
                          type="range"
                          min={0}
                          max={4000}
                          step={100}
                          value={replyDelayMinMs}
                          onChange={(e) =>
                            setReplyDelayRange(Number(e.target.value), replyDelayMaxMs)
                          }
                          className="flex-1 accent-emerald-500"
                        />
                        <input
                          type="range"
                          min={0}
                          max={8000}
                          step={100}
                          value={replyDelayMaxMs}
                          onChange={(e) =>
                            setReplyDelayRange(replyDelayMinMs, Number(e.target.value))
                          }
                          className="flex-1 accent-emerald-500"
                        />
                      </div>
                    </div>
                  )}
                  <div className="flex items-center justify-between">
                    <div className="flex-1 pr-3">
                      <span className="text-sm text-gray-500 dark:text-gray-400">语音条</span>
                      <p className="text-xs text-gray-400 mt-0.5">
                        文字消息不可朗读。点输入区喇叭「要语音」后，本轮回复为语音条；
                        AI 自选发送已关闭，后续再完善
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-sm text-gray-500 dark:text-gray-400">我的音色样本</span>
                      <p className="text-xs text-gray-400 mt-0.5">
                        上传 wav/mp3（≤10MB），TTS 自动用 voiceclone
                      </p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {userVoice && (
                        <button
                          className="text-xs text-red-400"
                          onClick={async () => {
                            try {
                              await ttsAPI.deleteMyVoice()
                              setUserVoice(null)
                              showToast('音色已删除')
                            } catch {
                              showToast('删除失败')
                            }
                          }}
                        >
                          删除
                        </button>
                      )}
                      <label className="text-xs text-wechat-green cursor-pointer border border-wechat-green/30 rounded-lg px-3 py-1.5">
                        <span>{uploadingVoice ? '上传中...' : userVoice ? '更换' : '上传'}</span>
                        <input
                          type="file"
                          accept=".wav,.mp3,audio/wav,audio/mpeg"
                          className="hidden"
                          disabled={uploadingVoice}
                          onChange={async (e) => {
                            const file = e.target.files?.[0]
                            if (!file) return
                            setUploadingVoice(true)
                            try {
                              const res = await ttsAPI.uploadMyVoice(file)
                              setUserVoice({
                                url: res.voice?.url || '',
                                original_name: file.name,
                              })
                              showToast('音色已保存，将用于语音合成')
                            } catch (err) {
                              console.error(err)
                              showToast('上传失败')
                            } finally {
                              setUploadingVoice(false)
                              e.target.value = ''
                            }
                          }}
                        />
                      </label>
                    </div>
                  </div>
                  {userVoice && (
                    <p className="text-xs text-gray-400">
                      当前：{userVoice.original_name || '已上传样本'}
                    </p>
                  )}
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-sm text-gray-500 dark:text-gray-400">黑夜模式</span>
                      <p className="text-xs text-gray-400 mt-0.5">深色界面，更省眼</p>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={isDark}
                      className={`relative w-11 h-6 rounded-full transition-colors ${
                        isDark ? 'bg-wechat-green' : 'bg-gray-300 dark:bg-gray-600'
                      }`}
                      onClick={toggleTheme}
                    >
                      <span
                        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
                          isDark ? 'translate-x-5' : ''
                        }`}
                      />
                    </button>
                  </div>
                  <div className="flex items-center justify-between">
                    <div>
                      <span className="text-sm text-gray-500 dark:text-gray-400">动作标签</span>
                      <p className="text-xs text-gray-400 mt-0.5">
                        解析 &lt;emotional&gt; 动作；关闭时不打印，后续可映射表情
                      </p>
                    </div>
                    <button
                      type="button"
                      role="switch"
                      aria-checked={actionsEnabled}
                      className={`relative w-11 h-6 rounded-full transition-colors ${
                        actionsEnabled ? 'bg-wechat-green' : 'bg-gray-300 dark:bg-gray-600'
                      }`}
                      onClick={() => setActionsEnabled(!actionsEnabled)}
                    >
                      <span
                        className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
                          actionsEnabled ? 'translate-x-5' : ''
                        }`}
                      />
                    </button>
                  </div>
                  {recentActions.length > 0 && (
                    <p className="text-xs text-gray-400">
                      最近动作：{recentActions.slice(-3).map((a) => a.text).join(' / ')}
                    </p>
                  )}
                  <div className="pt-2">
                    <button
                      className="w-full text-sm text-red-500 border border-red-300 dark:border-red-700 rounded-lg py-2.5 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors font-medium"
                      onClick={handleLogout}
                    >
                      退出登录
                    </button>
                  </div>
                </div>
              </div>
              <ServerDataPanel currentConversationId={chat.currentConversationId} />
              <div className="h-6" />
            </div>
          </div>
        </div>
      )}

      {/* Mobile bottom tabs — 常驻底部，竖屏也能随时切换 */}
      {isMobile && (
        <div className="shrink-0 z-30 flex bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 pb-[var(--safe-bottom)]">
          {tabs.map((t) => (
            <button
              key={t.key}
              className={`flex-1 flex flex-col items-center justify-center py-2 gap-0.5 min-h-[52px] transition-colors ${
                activeTab === t.key
                  ? 'text-wechat-green'
                  : 'text-gray-500 dark:text-gray-400'
              }`}
              onClick={() => setActiveTab(t.key)}
            >
              <TabIcon>{t.icon}</TabIcon>
              <span className="text-[10px] font-medium">{t.label}</span>
            </button>
          ))}
        </div>
      )}
      </div>

      <Toast message={toast} onDone={() => setToast(null)} />
      <EmojiPicker visible={showEmojiPicker} onClose={() => setShowEmojiPicker(false)} />

      <MsgDebugModal
        open={msgDebugOpen && debugExpandedMsgId != null}
        onClose={() => {
          setMsgDebugOpen(false)
          setDebugExpandedMsgId(null)
        }}
        llmText={debugMsgContent}
        ttsDebug={(() => {
          if (debugExpandedMsgId == null) return null
          const m = chat.messages.find((x) => x.id === debugExpandedMsgId)
          const byId = chat.msgTtsDebug[debugExpandedMsgId]
          const byUrl = m?.audio_url ? chat.ttsDebugByUrl[m.audio_url] : null
          return byId || byUrl || null
        })()}
        contextDebug={
          debugExpandedMsgId != null ? chat.msgCtxDebug[debugExpandedMsgId] || null : null
        }
        title="消息 Debug"
      />

      {/* Context Debug 面板：点击 AI 昵称打开 */}
      {showContextDebug && (
        <div
          className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/40"
          onClick={(e) => {
            if (e.target === e.currentTarget) setShowContextDebug(false)
          }}
        >
          <div className="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-xl w-full sm:max-w-lg max-h-[85vh] flex flex-col overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between shrink-0">
              <div>
                <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                  上下文 Debug
                </h3>
                <p className="text-xs text-gray-400 mt-0.5">
                  本轮发给模型的组装明细（发送前已推送）
                </p>
              </div>
              <button
                className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                onClick={() => setShowContextDebug(false)}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>

            <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
              {!chat.contextDebug ? (
                <p className="text-sm text-gray-400 py-8 text-center">
                  暂无数据，发送一条消息后即可看到本轮组装信息
                </p>
              ) : (
                <>
                  {(() => {
                    const d = chat.contextDebug!
                    const pct = Math.min(100, Math.round((d.occupancy || 0) * 100))
                    const triggerPct = Math.round((d.compact_threshold || 0.55) * 100)
                    const over = pct >= triggerPct
                    return (
                      <>
                        <div className="bg-gray-50 dark:bg-gray-900/50 rounded-xl p-4 space-y-3">
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-gray-500 dark:text-gray-400">上下文占用</span>
                            <span className={`font-semibold ${over ? 'text-amber-500' : 'text-wechat-green'}`}>
                              {d.estimated_tokens} / {d.token_budget} tok（{pct}%）
                            </span>
                          </div>
                          <div className="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden relative">
                            <div
                              className={`h-full rounded-full ${over ? 'bg-amber-400' : 'bg-wechat-green'}`}
                              style={{ width: `${pct}%` }}
                            />
                            <div
                              className="absolute top-0 bottom-0 w-0.5 bg-red-400"
                              style={{ left: `${triggerPct}%` }}
                              title={`压缩线 ${triggerPct}%`}
                            />
                          </div>
                          <p className="text-[11px] text-gray-400">
                            压缩线 {triggerPct}% · 触发约 {d.trigger_tokens} tok
                            {d.compacted ? ' · 本轮已触发 AI 压缩' : ''}
                          </p>
                          <div className="grid grid-cols-3 gap-2 text-center">
                            {[
                              ['System', d.system_tokens],
                              ['Recent', d.recent_tokens],
                              ['User', d.user_tokens],
                            ].map(([label, val]) => (
                              <div key={label as string} className="bg-white dark:bg-gray-800 rounded-lg py-2">
                                <div className="text-[10px] text-gray-400">{label}</div>
                                <div className="text-sm font-medium text-gray-700 dark:text-gray-200">
                                  {val as number}
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>

                        <div className="flex flex-wrap gap-2">
                          <span className="text-xs px-2 py-1 rounded-full bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400">
                            人格：{currentPersona?.name || personaName || '默认'}
                          </span>
                          <span className="text-xs px-2 py-1 rounded-full bg-pink-100 text-pink-600 dark:bg-pink-900/30 dark:text-pink-400">
                            情绪：{d.emotion || 'neutral'}
                          </span>
                          {d.emotion_reason && (
                            <span className="text-xs px-2 py-1 rounded-full bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-300">
                              {d.emotion_reason}
                            </span>
                          )}
                          {(d.activated_skills || []).length === 0 ? (
                            <span className="text-xs px-2 py-1 rounded-full bg-gray-100 text-gray-400 dark:bg-gray-700">
                              无激活 L2 技能
                            </span>
                          ) : (
                            d.activated_skills.map((s) => (
                              <span
                                key={s}
                                className="text-xs px-2 py-1 rounded-full bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"
                              >
                                {s}
                              </span>
                            ))
                          )}
                          <span className="text-xs px-2 py-1 rounded-full bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-300">
                            摘要 v{d.summary_version || 0}
                          </span>
                          <span className="text-xs px-2 py-1 rounded-full bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-300">
                            近期 {d.recent_count} 条
                          </span>
                        </div>

                        {d.summary && (
                          <div>
                            <h4 className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">
                              滚动摘要（压缩后）
                            </h4>
                            <pre className="text-xs bg-amber-50 dark:bg-amber-900/20 text-gray-700 dark:text-gray-200 rounded-xl px-3 py-2 whitespace-pre-wrap break-words max-h-32 overflow-y-auto">
                              {d.summary}
                            </pre>
                          </div>
                        )}

                        {d.memory && (
                          <div>
                            <h4 className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">
                              长期记忆卡
                            </h4>
                            <pre className="text-xs bg-green-50 dark:bg-green-900/20 text-gray-700 dark:text-gray-200 rounded-xl px-3 py-2 whitespace-pre-wrap break-words max-h-28 overflow-y-auto">
                              {d.memory}
                            </pre>
                          </div>
                        )}

                        <div>
                          <h4 className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">
                            提示词构成
                          </h4>
                          {d.system_parts?.length ? (
                            <>
                              <div className="flex flex-wrap gap-1.5 mb-2">
                                {d.system_parts.map((p, i) => (
                                  <button
                                    key={p.name}
                                    className={`text-[11px] px-2 py-1 rounded-lg border transition-colors ${
                                      debugPartTab === i
                                        ? 'border-wechat-green text-wechat-green bg-wechat-green/5'
                                        : 'border-gray-200 dark:border-gray-600 text-gray-500'
                                    }`}
                                    onClick={() => setDebugPartTab(i)}
                                  >
                                    {p.name}
                                    <span className="ml-1 opacity-60">{p.tokens}t</span>
                                  </button>
                                ))}
                              </div>
                              <pre className="text-xs bg-gray-900 text-gray-100 rounded-xl px-3 py-3 whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
                                {d.system_parts[Math.min(debugPartTab, d.system_parts.length - 1)]?.content}
                              </pre>
                            </>
                          ) : (
                            <pre className="text-xs bg-gray-900 text-gray-100 rounded-xl px-3 py-3 whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
                              {d.full_system || '（空）'}
                            </pre>
                          )}
                        </div>

                        {chat.ttsDebug && (
                          <div>
                            <h4 className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1.5">
                              MiMo TTS（发给语音合成的请求）
                            </h4>
                            <div className="flex flex-wrap gap-1.5 mb-2">
                              <span className="text-[11px] px-2 py-1 rounded-full bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300">
                                {chat.ttsDebug.model}
                              </span>
                              <span className="text-[11px] px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                                {chat.ttsDebug.voice_mode}
                              </span>
                              <span className="text-[11px] px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                                {chat.ttsDebug.format}
                              </span>
                              <span className="text-[11px] px-2 py-1 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                                {chat.ttsDebug.char_count} 字 · {chat.ttsDebug.latency_ms}ms
                                {chat.ttsDebug.cache_hit ? ' · cache' : ''}
                                {chat.ttsDebug.truncated ? ' · 已截断' : ''}
                              </span>
                            </div>
                            <div className="text-[11px] text-gray-400 mb-1">
                              voice：{chat.ttsDebug.voice || '—'} · url：
                              {chat.ttsDebug.audio_url || '—'}
                            </div>
                            <div className="text-[11px] text-gray-400 mb-1">
                              风格指令：
                              {chat.ttsDebug.style_prompt || '—'}
                            </div>
                            <pre className="text-xs bg-indigo-50 dark:bg-indigo-900/20 text-gray-700 dark:text-gray-200 rounded-xl px-3 py-2 whitespace-pre-wrap break-words max-h-32 overflow-y-auto">
                              {chat.ttsDebug.text_sent || chat.ttsDebug.error || '（无）'}
                            </pre>
                            {chat.ttsDebug.error && (
                              <p className="text-[11px] text-red-400 mt-1">{chat.ttsDebug.error}</p>
                            )}
                          </div>
                        )}
                      </>
                    )
                  })()}
                </>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Add persona modal */}
      {showAddPersonaModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
          onClick={(e) => {
            if (e.target === e.currentTarget) setShowAddPersonaModal(false)
          }}
        >
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-lg mx-4 overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100 dark:border-gray-700">
              <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">添加人格</h3>
              <p className="text-xs text-gray-400 mt-0.5">设置名称并上传 .md 技能文件</p>
            </div>
            <div className="px-5 py-4 space-y-4">
              <div>
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">人格名（英文）</label>
                <input
                  value={addPersonaSystemName}
                  onChange={(e) => setAddPersonaSystemName(e.target.value)}
                  className="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200"
                  placeholder="例：rain（用于系统标识）"
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">昵称（显示名称）</label>
                <input
                  value={addPersonaNickname}
                  onChange={(e) => setAddPersonaNickname(e.target.value)}
                  className="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200"
                  placeholder="例：Rain"
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">技能文件 (.md)</label>
                <label className="mt-1 flex flex-col items-center justify-center border-2 border-dashed border-gray-200 dark:border-gray-600 rounded-xl px-4 py-6 cursor-pointer hover:border-wechat-green/50 transition-colors">
                  <span className="text-sm text-gray-400">点击选择文件，支持多选</span>
                  <span className="text-xs text-gray-400 mt-1">每个文件不超过 10KB</span>
                  <input
                    type="file"
                    accept=".md"
                    multiple
                    className="hidden"
                    onChange={(e) => {
                      const files = e.target.files
                      if (!files || files.length === 0) return
                      const errors: string[] = []
                      const valid: File[] = []
                      for (const f of Array.from(files)) {
                        if (!f.name.endsWith('.md')) {
                          errors.push(`跳过非 .md 文件: ${f.name}`)
                          continue
                        }
                        if (f.size > 10 * 1024) {
                          errors.push(`${f.name} 超过 10KB`)
                          continue
                        }
                        valid.push(f)
                      }
                      setAddPersonaFiles((prev) => [...prev, ...valid])
                      setAddPersonaFileErrors(errors)
                      e.target.value = ''
                    }}
                  />
                </label>
              </div>
              {addPersonaFiles.length > 0 && (
                <div className="space-y-1.5">
                  <p className="text-xs font-medium text-gray-500 dark:text-gray-400">
                    已选 {addPersonaFiles.length} 个文件
                  </p>
                  {addPersonaFiles.map((f, i) => (
                    <div
                      key={`${f.name}-${i}`}
                      className="flex items-center justify-between bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-xs font-medium text-gray-700 dark:text-gray-300 truncate">
                          {f.name}
                        </p>
                        <p className="text-xs text-gray-400">
                          优先级 {i} · {(f.size / 1024).toFixed(1)} KB
                        </p>
                      </div>
                      <button
                        className="text-xs text-red-400 hover:text-red-600 shrink-0 ml-2"
                        onClick={() => setAddPersonaFiles((prev) => prev.filter((_, j) => j !== i))}
                      >
                        移除
                      </button>
                    </div>
                  ))}
                  {addPersonaFileErrors.map((err) => (
                    <p key={err} className="text-xs text-red-400">
                      {err}
                    </p>
                  ))}
                </div>
              )}
            </div>
            <div className="px-5 py-4 border-t border-gray-100 dark:border-gray-700 flex justify-end gap-2">
              <button
                className="text-sm text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-750 rounded-lg px-4 py-2"
                onClick={() => setShowAddPersonaModal(false)}
              >
                取消
              </button>
              <button
                className="text-sm text-white bg-wechat-green hover:bg-wechat-green-dark rounded-lg px-4 py-2 font-medium disabled:opacity-50"
                disabled={
                  !addPersonaSystemName.trim() ||
                  !addPersonaNickname.trim() ||
                  addPersonaFiles.length === 0 ||
                  submittingAddPersona
                }
                onClick={async () => {
                  const systemName = addPersonaSystemName.trim()
                  const nickname = addPersonaNickname.trim()
                  if (!systemName || !nickname || addPersonaFiles.length === 0) return
                  setSubmittingAddPersona(true)
                  try {
                    const res = await personaAPI.createPersona({
                      name: systemName,
                      nickname,
                      description: `自定义人格: ${nickname}`,
                    })
                    await personaAPI.uploadSkillFile(res.persona.id, addPersonaFiles)
                    showToast(`人格 "${nickname}" 创建成功`)
                    setShowAddPersonaModal(false)
                    await fetchPersonas()
                  } catch {
                    showToast('添加失败')
                  } finally {
                    setSubmittingAddPersona(false)
                  }
                }}
              >
                {submittingAddPersona ? '提交中...' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit persona modal */}
      {editingPersona && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
          onClick={(e) => {
            if (e.target === e.currentTarget) setEditingPersona(null)
          }}
        >
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-lg mx-4 overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100 dark:border-gray-700">
              <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">人格设置</h3>
              <p className="text-xs text-gray-400 mt-0.5">
                编辑 {editingPersona.nickname || editingPersona.name}
              </p>
            </div>
            <div className="px-5 py-4 space-y-4 max-h-[60vh] overflow-y-auto">
              <div className="flex items-center gap-4">
                <div
                  className="w-16 h-16 rounded-full flex items-center justify-center text-white text-xl font-bold shrink-0"
                  style={{ backgroundColor: personaColor(editingPersona.name) }}
                >
                  {editPersonaAvatarUrl ? (
                    <img src={resolveAssetUrl(editPersonaAvatarUrl)} className="w-full h-full rounded-full object-cover" alt="" />
                  ) : (
                    editingPersona.name.charAt(0).toUpperCase()
                  )}
                </div>
                <label className="text-xs text-wechat-green cursor-pointer border border-wechat-green/30 rounded-lg px-3 py-1.5">
                  <span>{uploadingPersonaAvatar ? '上传中...' : '更换头像'}</span>
                  <input
                    type="file"
                    accept="image/*"
                    className="hidden"
                    disabled={uploadingPersonaAvatar}
                    onChange={async (e) => {
                      const raw = e.target.files?.[0]
                      if (!raw || !editingPersona) return
                      setUploadingPersonaAvatar(true)
                      try {
                        let file = raw
                        if (file.size > 512 * 1024) {
                          showToast('正在压缩图片...')
                          file = await compressImageToBlob(file, {
                            maxSizeBytes: 1024 * 1024,
                            maxEdge: 720,
                            quality: 0.85,
                          })
                        }
                        if (file.size > 1024 * 1024) {
                          showToast('压缩后仍超过 1MB，请换小一点的图')
                          return
                        }
                        const res = await personaAPI.uploadPersonaAvatar(editingPersona.id, file)
                        setEditPersonaAvatarUrl(res.avatar)
                        setEditingPersona({ ...editingPersona, avatar: res.avatar })
                        showToast('头像已更新')
                      } catch {
                        showToast('上传失败')
                      } finally {
                        setUploadingPersonaAvatar(false)
                        e.target.value = ''
                      }
                    }}
                  />
                </label>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">人格名</label>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 px-1">
                  {editingPersona.name}
                </p>
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">昵称</label>
                <input
                  value={editPersonaName}
                  onChange={(e) => setEditPersonaName(e.target.value)}
                  className="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200"
                />
              </div>
              <div>
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">描述</label>
                <input
                  value={editPersonaDesc}
                  onChange={(e) => setEditPersonaDesc(e.target.value)}
                  className="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200"
                  placeholder="个性签名"
                />
              </div>
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-xs font-medium text-gray-500 dark:text-gray-400">技能文件</label>
                  <label className="text-xs text-wechat-green cursor-pointer">
                    <span>上传</span>
                    <input
                      type="file"
                      accept=".md"
                      multiple
                      className="hidden"
                      onChange={async (e) => {
                        const files = e.target.files
                        if (!files || files.length === 0 || !editingPersona) return
                        try {
                          const res = await personaAPI.uploadSkillFile(
                            editingPersona.id,
                            Array.from(files),
                          )
                          showToast(`成功上传 ${res.uploaded?.length || 0} 个文件`)
                          const detail = await personaAPI.getPersona(editingPersona.id)
                          setPersonaFilesMap((m) => ({
                            ...m,
                            [editingPersona.id]: detail.persona_files || [],
                          }))
                        } catch {
                          showToast('上传失败')
                        }
                        e.target.value = ''
                      }}
                    />
                  </label>
                </div>
                {personaFilesMap[editingPersona.id]?.length ? (
                  <div className="space-y-1.5">
                    {personaFilesMap[editingPersona.id].map((pf) => (
                      <div
                        key={pf.id}
                        className="flex items-center justify-between bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2"
                      >
                        <div className="flex-1 min-w-0">
                          <p className="text-xs font-medium text-gray-700 dark:text-gray-300 truncate">
                            {pf.file_name}
                          </p>
                          <div className="flex items-center gap-1.5 mt-0.5">
                            <span
                              className={`text-xs px-1.5 py-0.5 rounded-full ${moduleBadgeClass(pf.module_category)}`}
                            >
                              {pf.module_category}
                            </span>
                            <span className="text-xs text-gray-400">优先级 {pf.priority}</span>
                          </div>
                        </div>
                        <button
                          className="text-xs text-red-400 hover:text-red-600 shrink-0 ml-2"
                          onClick={async () => {
                            if (!window.confirm('确定要删除该文件吗？')) return
                            try {
                              await personaAPI.deleteSkillFile(editingPersona.id, pf.id)
                              const detail = await personaAPI.getPersona(editingPersona.id)
                              setPersonaFilesMap((m) => ({
                                ...m,
                                [editingPersona.id]: detail.persona_files || [],
                              }))
                            } catch {
                              showToast('删除失败')
                            }
                          }}
                        >
                          删除
                        </button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-gray-400 py-2">暂无技能文件</p>
                )}
              </div>
            </div>
            <div className="px-5 py-4 border-t border-gray-100 dark:border-gray-700 flex justify-between gap-2">
              {!editingPersona.is_built_in && (
                <button
                  className="text-sm text-red-400 border border-red-200 dark:border-red-800 rounded-lg px-4 py-2"
                  onClick={async () => {
                    if (!window.confirm('确定要删除该人格及其所有文件吗？')) return
                    try {
                      await personaAPI.deletePersona(editingPersona.id)
                      showToast('人格已删除')
                      setEditingPersona(null)
                      await fetchPersonas()
                    } catch {
                      showToast('删除失败')
                    }
                  }}
                >
                  删除人格
                </button>
              )}
              <div className="flex gap-2 ml-auto">
                <button
                  className="text-sm text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-750 rounded-lg px-4 py-2"
                  onClick={() => setEditingPersona(null)}
                >
                  取消
                </button>
                <button
                  className="text-sm text-white bg-wechat-green hover:bg-wechat-green-dark rounded-lg px-4 py-2 font-medium disabled:opacity-50"
                  disabled={!editPersonaName.trim() || submittingEditPersona}
                  onClick={async () => {
                    if (!editingPersona || !editPersonaName.trim()) return
                    setSubmittingEditPersona(true)
                    try {
                      await personaAPI.updatePersona(editingPersona.id, {
                        nickname: editPersonaName.trim(),
                        description: editPersonaDesc.trim(),
                      })
                      showToast('人格已更新')
                      setEditingPersona(null)
                      await fetchPersonas()
                    } catch {
                      showToast('更新失败')
                    } finally {
                      setSubmittingEditPersona(false)
                    }
                  }}
                >
                  {submittingEditPersona ? '提交中...' : '保存'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
