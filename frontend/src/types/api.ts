export interface User {
  id: number
  username: string
  email: string
  avatar: string
  created_at: string
  updated_at: string
}

export interface Conversation {
  id: number
  user_id: number
  title: string
  ai_nickname: string
  ai_avatar: string
  persona_id: number | null
  /** 清空聊天时是否保留记忆卡 */
  keep_memory_on_clear?: boolean
  /** 亲密模式（NSFW）开关 */
  nsfw_enabled?: boolean
  created_at: string
  updated_at: string
  avatar?: string | null
  last_message?: { content: string; role: string; created_at: string } | null
}

export interface ReplySegment {
  index: number
  seg_id: string
  content: string
  start?: number
  end?: number
}

export interface Message {
  id: number
  conversation_id: number
  role: 'user' | 'assistant'
  /** text=文字气泡；voice=语音条；image=AI 图片 */
  message_type?: 'text' | 'voice' | 'image'
  content: string
  audio_url?: string
  audio_duration_ms?: number
  /** AI 图片 URL（image_message） */
  image_url?: string
  attachment_url?: string
  attachment_type?: string
  has_attachment?: boolean
  /** AI 完整回复的显式小段；无则前端按规则切分 */
  segments?: ReplySegment[]
  created_at: string
  is_deleted: boolean
}

export interface Persona {
  id: number
  user_id: number
  name: string
  nickname: string
  description: string
  dir_name: string
  avatar: string
  /** 聊天页人格背景图 */
  background?: string
  is_active: boolean
  created_at: string
  updated_at: string
  is_built_in?: boolean
  file_count?: number
}

export type PersonaFromServer = Persona

export interface PersonaFile {
  id: number
  persona_id: number
  file_name: string
  storage_path: string
  priority: number
  module_category: string
  file_size: number
  created_at: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface RegisterResponse {
  token: string
  user: User
}

export interface ProfileResponse {
  user: User
}

export interface ConversationsResponse {
  conversations: Conversation[]
}

export interface MessagesResponse {
  messages: Message[]
  total: number
}

export interface ConversationResponse {
  conversation: Conversation
}

export interface PersonasResponse {
  personas: PersonaFromServer[]
}

export interface PersonaResponse {
  persona: PersonaFromServer
  persona_files: PersonaFile[]
}

export interface FileRecord {
  id: number
  user_id: number
  file_type: string
  reference_id: number
  reference_type: string
  original_name: string
  storage_path: string
  url: string
  size: number
  mime_type: string
  created_at: string
}
