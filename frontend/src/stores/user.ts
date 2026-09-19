import { create } from 'zustand'
import { authAPI, userAPI } from '../api'
import type { User } from '../types/api'
import { defaultAvatarUrl, resolveAssetUrl } from '../utils/url'

function loadSavedUser(): User | null {
  const saved = localStorage.getItem('user')
  if (!saved) return null
  try {
    return JSON.parse(saved) as User
  } catch {
    return null
  }
}

interface UserState {
  token: string
  userInfo: User | null
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  fetchProfile: () => Promise<void>
  logout: () => Promise<void>
  restoreSession: () => void
  setAvatar: (avatar: string) => void
}

export const useUserStore = create<UserState>((set, get) => ({
  token: localStorage.getItem('token') || '',
  userInfo: loadSavedUser(),

  login: async (email, password) => {
    const res = await authAPI.login(email, password)
    localStorage.setItem('token', res.token)
    localStorage.setItem('user', JSON.stringify(res.user))
    set({ token: res.token, userInfo: res.user })
  },

  register: async (username, email, password) => {
    const res = await authAPI.register(username, email, password)
    localStorage.setItem('token', res.token)
    localStorage.setItem('user', JSON.stringify(res.user))
    set({ token: res.token, userInfo: res.user })
  },

  fetchProfile: async () => {
    const res = await userAPI.getProfile()
    localStorage.setItem('user', JSON.stringify(res.user))
    set({ userInfo: res.user })
  },

  logout: async () => {
    const token = get().token
    if (token) {
      try {
        await authAPI.logout()
      } catch {
        /* 网络失败也继续本地登出 */
      }
    }
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    set({ token: '', userInfo: null })
  },

  restoreSession: () => {
    set({ userInfo: loadSavedUser() })
  },

  setAvatar: (avatar) => {
    const userInfo = get().userInfo
    if (!userInfo) return
    const next = { ...userInfo, avatar }
    localStorage.setItem('user', JSON.stringify(next))
    set({ userInfo: next })
  },
}))

export function useIsLoggedIn() {
  return useUserStore((s) => !!s.token)
}

export function useUsername() {
  return useUserStore((s) => s.userInfo?.username || '用户')
}

export function useUserAvatar() {
  return useUserStore((s) => resolveAssetUrl(s.userInfo?.avatar) || defaultAvatarUrl())
}

export function useUserEmail() {
  return useUserStore((s) => s.userInfo?.email || '')
}
