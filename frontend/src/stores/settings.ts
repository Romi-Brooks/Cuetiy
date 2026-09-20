import { create } from 'zustand'

/** 拟人化节奏设置：回复前等待 + 分段延时上屏 */
export interface HumanizeSettings {
  /** 逐字打字动画（影响「输入中」停留时长） */
  typingEnabled: boolean
  /** 每字间隔 ms，用于估算「输入中」停留时长 */
  typingCharDelayMs: number
  /** 回复前模拟「看到消息再回」的延迟 */
  replyDelayEnabled: boolean
  replyDelayMinMs: number
  replyDelayMaxMs: number
  /** 多段回复按节奏逐段上屏（参照拟人输入节奏） */
  segmentRevealEnabled: boolean
  /** 段与段之间的间隔 ms */
  segmentRevealDelayMs: number
  /** 清空聊天时是否默认保留记忆卡（可被会话配置覆盖） */
  defaultKeepMemoryOnClear: boolean
  /** AI 回复后自动 TTS 播放 */
  voiceAutoPlay: boolean
}

interface SettingsState extends HumanizeSettings {
  setTypingEnabled: (v: boolean) => void
  setTypingCharDelayMs: (ms: number) => void
  setReplyDelayEnabled: (v: boolean) => void
  setReplyDelayRange: (min: number, max: number) => void
  setSegmentRevealEnabled: (v: boolean) => void
  setSegmentRevealDelayMs: (ms: number) => void
  setDefaultKeepMemoryOnClear: (v: boolean) => void
  setVoiceAutoPlay: (v: boolean) => void
  resetHumanize: () => void
}

const KEY = 'cuetiy:humanize'

const defaults: HumanizeSettings = {
  typingEnabled: true,
  typingCharDelayMs: 40,
  replyDelayEnabled: false,
  replyDelayMinMs: 400,
  replyDelayMaxMs: 1600,
  segmentRevealEnabled: true,
  segmentRevealDelayMs: 700,
  defaultKeepMemoryOnClear: false,
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
      segmentRevealEnabled: parsed.segmentRevealEnabled ?? defaults.segmentRevealEnabled,
      segmentRevealDelayMs: clampMs(parsed.segmentRevealDelayMs ?? defaults.segmentRevealDelayMs, 200, 3000),
      defaultKeepMemoryOnClear: parsed.defaultKeepMemoryOnClear ?? defaults.defaultKeepMemoryOnClear,
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
  setSegmentRevealEnabled: (v) => {
    persist({ segmentRevealEnabled: v })
    set({ segmentRevealEnabled: v })
  },
  setSegmentRevealDelayMs: (ms) => {
    const v = clampMs(ms, 200, 3000)
    persist({ segmentRevealDelayMs: v })
    set({ segmentRevealDelayMs: v })
  },
  setDefaultKeepMemoryOnClear: (v) => {
    persist({ defaultKeepMemoryOnClear: v })
    set({ defaultKeepMemoryOnClear: v })
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
