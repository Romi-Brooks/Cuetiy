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
  /** 段与段之间随机间隔的下限 ms */
  segmentRevealMinMs: number
  /** 段与段之间随机间隔的上限 ms */
  segmentRevealMaxMs: number
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
  setSegmentRevealRange: (min: number, max: number) => void
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
  // 默认 3–5s 随机：固定短间隔会显得像机器刷屏
  segmentRevealMinMs: 3000,
  segmentRevealMaxMs: 5000,
  defaultKeepMemoryOnClear: false,
  voiceAutoPlay: false,
}

function load(): HumanizeSettings {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return { ...defaults }
    const parsed = JSON.parse(raw) as Partial<HumanizeSettings> & {
      segmentRevealDelayMs?: number
    }
    // 旧字段固定间隔 → 升级为 3–5s 随机（旧值过短时强制抬高）
    const legacy = parsed.segmentRevealDelayMs
    const min =
      parsed.segmentRevealMinMs ??
      (legacy != null && legacy >= 2000 ? legacy : defaults.segmentRevealMinMs)
    const max =
      parsed.segmentRevealMaxMs ??
      (legacy != null && legacy >= 3000 ? Math.max(legacy + 1000, 5000) : defaults.segmentRevealMaxMs)
    return {
      typingEnabled: parsed.typingEnabled ?? defaults.typingEnabled,
      typingCharDelayMs: clampMs(parsed.typingCharDelayMs ?? defaults.typingCharDelayMs),
      replyDelayEnabled: parsed.replyDelayEnabled ?? defaults.replyDelayEnabled,
      replyDelayMinMs: clampMs(parsed.replyDelayMinMs ?? defaults.replyDelayMinMs, 0, 8000),
      replyDelayMaxMs: clampMs(parsed.replyDelayMaxMs ?? defaults.replyDelayMaxMs, 0, 8000),
      segmentRevealEnabled: parsed.segmentRevealEnabled ?? defaults.segmentRevealEnabled,
      segmentRevealMinMs: clampMs(min, 1000, 8000),
      segmentRevealMaxMs: clampMs(Math.max(max, min), 1000, 8000),
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

/** 分段上屏：段与段之间随机等待（默认 3–5s） */
export function randomSegmentRevealGapMs(s: HumanizeSettings): number {
  const min = Math.max(1000, s.segmentRevealMinMs || 3000)
  const max = Math.max(min, s.segmentRevealMaxMs || 5000)
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
  setSegmentRevealRange: (min, max) => {
    const lo = clampMs(min, 1000, 8000)
    const hi = clampMs(Math.max(max, lo), 1000, 8000)
    persist({ segmentRevealMinMs: lo, segmentRevealMaxMs: hi })
    set({ segmentRevealMinMs: lo, segmentRevealMaxMs: hi })
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