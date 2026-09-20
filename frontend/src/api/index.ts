import type {
  LoginResponse,
  RegisterResponse,
  ProfileResponse,
  ConversationsResponse,
  MessagesResponse,
  ConversationResponse,
  PersonasResponse,
  PersonaResponse,
  Persona,
  FileRecord,
} from '../types/api'
import { getApiBase } from '../utils/server'

function getToken(): string {
  return localStorage.getItem('token') || ''
}

interface RequestOptions extends Omit<RequestInit, 'headers'> {
  headers?: Record<string, string>
}

async function request<T>(url: string, options: RequestOptions = {}): Promise<T> {
  const config: RequestInit = {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  }

  const token = getToken()
  if (token) {
    config.headers = {
      ...(config.headers as Record<string, string>),
      Authorization: `Bearer ${token}`,
    }
  }

  try {
    const response = await fetch(`${getApiBase()}${url}`, config)

    const text = await response.text()
    let data: unknown = null
    if (text) {
      try {
        data = JSON.parse(text)
      } catch {
        if (!response.ok) {
          throw new Error(`请求失败 (${response.status})`)
        }
        throw new Error('服务器返回了无法解析的内容')
      }
    }

    if (!response.ok) {
      const errMsg =
        (data && typeof data === 'object' && 'error' in data
          ? String((data as { error?: string }).error || '')
          : '') || `请求失败 (${response.status})`
      throw new Error(errMsg)
    }

    if (data === null) {
      return {} as T
    }

    return data as T
  } catch (error) {
    if (error instanceof TypeError) {
      throw new Error('无法连接服务器，请在「我的 → 数据与服务器」填写正确的 API 地址')
    }
    throw error
  }
}

/** 下载导出 JSON（带 Authorization） */
export async function fetchExportJson(path: string, filename: string): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`${getApiBase()}${path}`, { headers })
  if (!res.ok) {
    let msg = `导出失败 (${res.status})`
    try {
      const data = await res.json()
      if (data && typeof data === 'object' && 'error' in data) msg = String(data.error)
    } catch {
      /* ignore */
    }
    throw new Error(msg)
  }
  const blob = await res.blob()
  const a = document.createElement('a')
  const href = URL.createObjectURL(blob)
  a.href = href
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(href)
}

export const chatDataAPI = {
  exportConversation: (convId: number) =>
    fetchExportJson(
      `/conversations/${convId}/export`,
      `cuetiy-chat-${convId}-${new Date().toISOString().slice(0, 10)}.json`,
    ),

  exportAll: () =>
    fetchExportJson(`/export/chat`, `cuetiy-chat-all-${new Date().toISOString().slice(0, 10)}.json`),

  importChat: (payload: unknown) =>
    request<{ message: string; result?: Record<string, unknown>; results?: unknown[] }>(
      '/import/chat',
      { method: 'POST', body: JSON.stringify(payload) },
    ),
}

export const secretsAPI = {
  status: () =>
    request<{
      deepseek_configured: boolean
      mimo_configured: boolean
      tts_enabled: boolean
      grsai_configured?: boolean
      image_gen_enabled?: boolean
      image_gen_model?: string
    }>('/secrets/status'),
  /** 只提交要修改的 key；传空字符串表示清除。响应不回显 key */
  update: (payload: {
    deepseek_api_key?: string
    mimo_api_key?: string
    grsai_api_key?: string
    clear_deepseek?: boolean
    clear_mimo?: boolean
    clear_grsai?: boolean
    image_gen_enabled?: boolean
    image_gen_model?: string
    image_gen_aspect?: string
    image_gen_quality?: string
  }) =>
    request<{
      message: string
      deepseek_configured: boolean
      mimo_configured: boolean
      grsai_configured?: boolean
      image_gen_enabled?: boolean
      image_gen_model?: string
    }>('/secrets', { method: 'PUT', body: JSON.stringify(payload) }),
}

export const imageRefAPI = {
  getMyImageRef: () =>
    request<{
      image_ref: { id: number; url: string; original_name: string; size: number } | null
    }>('/image-ref'),
  uploadMyImageRef: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ message: string; image_ref: { id: number; url: string } }>('/image-ref', {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },
  deleteMyImageRef: () => request<{ message: string }>('/image-ref', { method: 'DELETE' }),
}

export const authAPI = {
  login: (email: string, password: string) =>
    request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),

  register: (username: string, email: string, password: string) =>
    request<RegisterResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    }),

  logout: () =>
    request<{ message: string }>('/auth/logout', {
      method: 'POST',
    }),
}

export const conversationAPI = {
  getConversations: () => request<ConversationsResponse>('/conversations'),

  getMessages: (convId: number, limit = 50, offset = 0, beforeId = 0) => {
    const params = new URLSearchParams()
    params.set('limit', String(limit))
    params.set('offset', String(offset))
    if (beforeId > 0) params.set('before_id', String(beforeId))
    return request<MessagesResponse>(`/conversations/${convId}/messages?${params}`)
  },

  clearMessages: (convId: number, keepMemory = false) =>
    request<{ message: string; archive_path?: string; archive_count?: number; keep_memory?: boolean }>(
      `/conversations/${convId}/messages?keep_memory=${keepMemory ? 'true' : 'false'}`,
      { method: 'DELETE' },
    ),

  updateConfig: (convId: number, config: Record<string, unknown>) =>
    request<ConversationResponse>(`/conversations/${convId}/config`, {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  createConversation: (title: string) =>
    request<ConversationResponse>('/conversations', {
      method: 'POST',
      body: JSON.stringify({ title }),
    }),
}

export const userAPI = {
  getProfile: () => request<ProfileResponse>('/user/profile'),

  updateProfile: (data: Record<string, unknown>) =>
    request<ProfileResponse>('/user/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
}

export const personaAPI = {
  getPersonas: () => request<PersonasResponse>('/personas'),

  getPersona: (id: number) => request<PersonaResponse>(`/personas/${id}`),

  createPersona: (data: Record<string, unknown>) =>
    request<PersonaResponse>('/personas', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  updatePersona: (id: number, data: Record<string, unknown>) =>
    request<PersonaResponse>(`/personas/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deletePersona: (id: number) =>
    request<void>(`/personas/${id}`, {
      method: 'DELETE',
    }),

  uploadPersonaBackground: (personaId: number, file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ message: string; background: string; persona: Persona }>(
      `/personas/${personaId}/background`,
      {
        method: 'POST',
        headers: {},
        body: formData,
      },
    )
  },

  deletePersonaBackground: (personaId: number) =>
    request<{ message: string; background: string; persona: Persona }>(
      `/personas/${personaId}/background`,
      { method: 'DELETE' },
    ),

  uploadSkillFile: (personaId: number, files: File[]) => {
    const formData = new FormData()
    for (const file of files) {
      formData.append('file', file)
    }
    return request<{ message: string; uploaded?: string[]; errors?: string[] }>(
      `/personas/${personaId}/files`,
      {
        method: 'POST',
        headers: {},
        body: formData,
      },
    )
  },

  getDebugPrompt: (convId: number) =>
    request<{ system_prompt: string }>(`/personas/${convId}/debug`),

  deleteSkillFile: (personaId: number, nodeId: number) =>
    request<void>(`/personas/${personaId}/files/${nodeId}`, {
      method: 'DELETE',
    }),

  loadFromDirectory: () =>
    request<{ message: string; count: number }>('/personas/load', {
      method: 'POST',
    }),

  getConversationPersona: (convId: number) =>
    request<PersonaResponse>(`/conversations/${convId}/persona`),

  openPersonaConversation: (personaId: number) =>
    request<{ message: string; conversation: import('../types/api').Conversation; created: boolean; persona: import('../types/api').Persona }>(
      `/personas/${personaId}/conversation`,
      { method: 'POST' },
    ),

  setConversationPersona: (convId: number, personaId: number | null) =>
    request<ConversationResponse>(`/conversations/${convId}/persona`, {
      method: 'PUT',
      body: JSON.stringify({ persona_id: personaId }),
    }),

  uploadAvatar: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ message: string; file: FileRecord }>('/upload/avatar', {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },

  uploadAIAvatar: (file: File, conversationId: number) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('conversation_id', String(conversationId))
    return request<{ message: string; file: FileRecord }>('/upload/avatar/ai', {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },

  uploadPersonaAvatar: (personaId: number, file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ message: string; avatar: string }>(`/personas/${personaId}/avatar`, {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },

  uploadImage: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ message: string; file: FileRecord }>('/upload/image', {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },
}

export const ttsAPI = {
  synthesize: (text: string, style?: string) =>
    request<{ message: string; url: string; path?: string }>('/tts', {
      method: 'POST',
      body: JSON.stringify({ text, style: style || '' }),
    }),

  transcribe: async (file: Blob, filename = 'audio.webm', language = 'zh') => {
    const formData = new FormData()
    formData.append('file', file, filename)
    formData.append('language', language)
    return request<{ message: string; text: string }>('/asr', {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },

  getMyVoice: () =>
    request<{ voice: { id: number; url: string; original_name: string; size: number } | null }>(
      '/voice',
    ),

  uploadMyVoice: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ message: string; voice: { id: number; url: string } }>('/voice', {
      method: 'POST',
      headers: {},
      body: formData,
    })
  },

  deleteMyVoice: () =>
    request<{ message: string }>('/voice', { method: 'DELETE' }),
}
