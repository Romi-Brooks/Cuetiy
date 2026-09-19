import { create } from 'zustand'

/** 拟人化节奏设置：回复前等待 + 逐字上屏 */
export interface HumanizeSettings {
  /** 逐字打字动画 */
  typingEnabled: boolean
  /** 每字间隔 ms，用于估算「输入中」停留时长（不是逐字上屏） */
  typingCharDelayMs: number
  /** 回复前模拟「看到消息再回」的延迟 */
  replyDelayEnabled: boolean
  replyDelayMinMs: number
  replyDelayMaxMs: number
  /** AI 回复后自动 TTS 播放 */
  voiceAutoPlay: boolean
}

interface SettingsState extends HumanizeSettings {
  setTypingEnabled: (v: boolean) => void
  setTypingCharDelayMs: (ms: number) => void
  setReplyDelayEnabled: (v: boolean) => void
  setReplyDelayRange: (min: number, max: number) => void
  setVoiceAutoPlay: (v: boolean) => void
  resetHumanize: () => void
}

const KEY = 'rainyi:humanize'

const defaults: HumanizeSettings = {
  // 开启后：AI 回复到达时仍保持「对方正在输入中」，按字数停够时间再一次性上屏
  // 每字约 40ms + 600ms 起步，下限 0.8s、上限 12s；要更慢可把滑条调大（500≈一字 0.5s）
  typingEnabled: true,
  typingCharDelayMs: 40,
  replyDelayEnabled: false,
  replyDelayMinMs: 400,
  replyDelayMaxMs: 1600,
  voiceAutoPlay: false,
}

function load(): HumanizeSettings {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return { ...defaults }
    const parsed = JSON.parse(raw) as Partial<HumanizeSettings>
    return {
      typingEnabled: parsed.typingEnabled ?? defaults.typingEnabled,
      typingCharDelayMs: clampMs(parsed.typingCharDelayMs ?? defaults.typingCharDelayMs),
      replyDelayEnabled: parsed.replyDelayEnabled ?? defaults.replyDelayEnabled,
      replyDelayMinMs: clampMs(parsed.replyDelayMinMs ?? defaults.replyDelayMinMs, 0, 8000),
      replyDelayMaxMs: clampMs(parsed.replyDelayMaxMs ?? defaults.replyDelayMaxMs, 0, 8000),
      voiceAutoPlay: parsed.voiceAutoPlay ?? defaults.voiceAutoPlay,
    }
  } catch {
    return { ...defaults }
  }
}

function clampMs(n: number, min = 0, max = 500): number {
  if (!Number.isFinite(n)) return min
  return Math.max(min, Math.min(max, Math.round(n)))
}

function persist(partial: Partial<HumanizeSettings>) {
  try {
    const next = { ...load(), ...partial }
    localStorage.setItem(KEY, JSON.stringify(next))
  } catch {
    /* ignore */
  }
}

export function loadHumanizeSettings(): HumanizeSettings {
  return load()
}

export function randomReplyDelayMs(s: HumanizeSettings): number {
  if (!s.replyDelayEnabled) return 0
  const min = Math.max(0, s.replyDelayMinMs)
  const max = Math.max(min, s.replyDelayMaxMs)
  return min + Math.floor(Math.random() * (max - min + 1))
}

export const useSettingsStore = create<SettingsState>((set) => ({
  ...load(),

  setTypingEnabled: (v) => {
    persist({ typingEnabled: v })
    set({ typingEnabled: v })
  },
  setTypingCharDelayMs: (ms) => {
    const v = clampMs(ms)
    persist({ typingCharDelayMs: v })
    set({ typingCharDelayMs: v })
  },
  setReplyDelayEnabled: (v) => {
    persist({ replyDelayEnabled: v })
    set({ replyDelayEnabled: v })
  },
  setReplyDelayRange: (min, max) => {
    const lo = clampMs(min, 0, 8000)
    const hi = clampMs(Math.max(max, lo), 0, 8000)
    persist({ replyDelayMinMs: lo, replyDelayMaxMs: hi })
    set({ replyDelayMinMs: lo, replyDelayMaxMs: hi })
  },
  setVoiceAutoPlay: (v) => {
    persist({ voiceAutoPlay: v })
    set({ voiceAutoPlay: v })
  },
  resetHumanize: () => {
    persist(defaults)
    set({ ...defaults })
  },
}))
