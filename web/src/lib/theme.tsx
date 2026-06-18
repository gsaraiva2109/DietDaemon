// Light/dark theme. Applies `.dark` on <html>, persists the choice, and
// defaults to the OS preference on first visit. Tokens for both live in
// index.css.

import { createContext, use, useEffect, useState, type ReactNode } from 'react'

type Theme = 'light' | 'dark'
const KEY = 'dd.theme'

interface ThemeValue {
  theme: Theme
  toggle: () => void
}
const ThemeContext = createContext<ThemeValue | null>(null)

function initial(): Theme {
  const saved = localStorage.getItem(KEY)
  if (saved === 'light' || saved === 'dark') return saved
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(initial)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    localStorage.setItem(KEY, theme)
  }, [theme])

  return (
    <ThemeContext value={{ theme, toggle: () => setTheme((t) => (t === 'dark' ? 'light' : 'dark')) }}>
      {children}
    </ThemeContext>
  )
}

export function useTheme(): ThemeValue {
  const ctx = use(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
