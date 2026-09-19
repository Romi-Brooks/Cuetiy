import { create } from 'zustand'

interface ThemeState {
  isDark: boolean
  toggleTheme: () => void
  applyTheme: () => void
  initTheme: () => void
}

function readDark(): boolean {
  return localStorage.getItem('darkMode') === 'true'
}

function applyClass(isDark: boolean) {
  document.documentElement.classList.toggle('dark', isDark)
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  isDark: readDark(),

  toggleTheme: () => {
    const next = !get().isDark
    localStorage.setItem('darkMode', String(next))
    applyClass(next)
    set({ isDark: next })
  },

  applyTheme: () => {
    applyClass(get().isDark)
  },

  initTheme: () => {
    applyClass(get().isDark)
  },
}))
