import { ttsAPI } from '../api'
import { stripEmotionalForDisplay } from '../stores/actions'
import { resolveAssetUrl } from './url'

let currentAudio: HTMLAudioElement | null = null

export function stopVoicePlayback() {
  if (currentAudio) {
    currentAudio.pause()
    currentAudio = null
  }
}

/** 合成并播放；同 URL 有缓存，后端也会按 hash 命中 */
export async function speakText(text: string, opts?: { style?: string; force?: boolean }) {
  const clean = stripEmotionalForDisplay(text)
  if (!clean) return
  try {
    stopVoicePlayback()
    const res = await ttsAPI.synthesize(clean, opts?.style)
    if (!res?.url) return
    const audio = new Audio(resolveAssetUrl(res.url))
    currentAudio = audio
    await audio.play().catch(() => {
      /* 自动播放策略可能拦截，用户手势后再点喇叭即可 */
    })
  } catch (e) {
    console.warn('TTS failed:', e)
  }
}
