import { create } from 'zustand'

/** 动作标签（<emotional>…</emotional>）开关与解析结果缓存 */
export interface ParsedAction {
  text: string
  at: number
}

interface ActionState {
  /** 总开关：开启后才展示动作（后续可映射 emoji/表情包） */
  enabled: boolean
  /** 最近解析到的动作（本轮对话内存，供后续映射） */
  recent: ParsedAction[]
  setEnabled: (v: boolean) => void
  pushAction: (text: string) => void
  clear: () => void
}

const KEY = 'cuetiy:actions_enabled'

function loadEnabled(): boolean {
  try {
    return localStorage.getItem(KEY) === '1'
  } catch {
    return false
  }
}

export const useActionStore = create<ActionState>((set, get) => ({
  enabled: loadEnabled(),
  recent: [],
  setEnabled: (v) => {
    try {
      localStorage.setItem(KEY, v ? '1' : '0')
    } catch {
      /* ignore */
    }
    set({ enabled: v })
  },
  pushAction: (text) => {
    if (!text) return
    const list = [...get().recent, { text, at: Date.now() }]
    // 只留最近 50 条
    set({ recent: list.slice(-50) })
  },
  clear: () => set({ recent: [] }),
}))

const emotionalRegex = /<emotional>([\s\S]*?)<\/emotional>/gi

/** 从消息文本抽出动作；默认不改变原文（原文仍存库） */
export function extractEmotionalActions(content: string): string[] {
  if (!content) return []
  const out: string[] = []
  emotionalRegex.lastIndex = 0
  let m: RegExpExecArray | null
  while ((m = emotionalRegex.exec(content)) !== null) {
    const t = m[1]?.trim()
    if (t) out.push(t)
  }
  return out
}

/** 气泡展示用：去掉 emotional 标签；开关关闭时动作不打印 */
export function stripEmotionalForDisplay(content: string): string {
  if (!content) return ''
  return content.replace(emotionalRegex, '').replace(/\n{3,}/g, '\n\n').trim()
}
