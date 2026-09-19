/** AI 回复显式小段（与后端 service.SplitReplySegments 规则一致） */

export interface ReplySegment {
  index: number
  seg_id: string
  content: string
  start?: number
  end?: number
}

const SHORT_LINE_MAX = 40
const SHORT_MIN_LINE = 2
const EXPLICIT_SEG = /^\s*(?:-{3,}|<<<SEG>>>|【段】)\s*$/

function runeLen(s: string): number {
  return Array.from(s).length
}

function compact(parts: string[]): string[] {
  const out: string[] = []
  for (const p of parts) {
    const t = p.trim()
    if (t) out.push(t)
  }
  return out
}

function shouldSplitShortLines(normalized: string): boolean {
  const lines = normalized.split('\n')
  let n = 0
  for (const line of lines) {
    const t = line.trim()
    if (!t) continue
    if (EXPLICIT_SEG.test(t)) continue
    if (runeLen(t) > SHORT_LINE_MAX) return false
    n++
  }
  return n >= SHORT_MIN_LINE
}

function buildSegments(full: string, raws: string[]): ReplySegment[] {
  const fullRunes = Array.from(full)
  let cursor = 0
  const out: ReplySegment[] = []
  raws.forEach((raw, i) => {
    const needle = Array.from(raw)
    let start = -1
    const from = Math.max(0, cursor)
    outer: for (let p = from; p + needle.length <= fullRunes.length; p++) {
      for (let j = 0; j < needle.length; j++) {
        if (fullRunes[p + j] !== needle[j]) continue outer
      }
      start = p
      break
    }
    if (start < 0) start = cursor
    const end = Math.min(fullRunes.length, start + needle.length)
    cursor = end
    out.push({
      index: i,
      seg_id: String(i),
      content: raw,
      start,
      end,
    })
  })
  return out
}

/**
 * 把完整 AI 回复切分为显式小段。
 * 规则与后端一致：显式分段符 > 空行 > 短句换行回退；空段丢弃。
 */
export function splitReplySegments(content: string): ReplySegment[] {
  let normalized = content.replace(/\r\n/g, '\n').replace(/\r/g, '\n')
  if (!normalized.trim()) return []

  let raws: string[]
  if (normalized.split('\n').some((l) => EXPLICIT_SEG.test(l))) {
    raws = normalized.split(/^\s*(?:-{3,}|<<<SEG>>>|【段】)\s*$/m)
  } else {
    raws = normalized.split(/\n\s*\n+/)
  }

  raws = compact(raws)
  if (raws.length <= 1 && shouldSplitShortLines(normalized)) {
    raws = compact(normalized.split('\n'))
  }
  if (raws.length === 0) return []
  return buildSegments(normalized, raws)
}

/** 拼接可引用的段 ID：message_id:seg_id */
export function formatSegRef(messageId: number | string, segId: string): string {
  return `${messageId}:${segId}`
}
