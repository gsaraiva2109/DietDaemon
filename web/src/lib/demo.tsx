// Demo mode: fills the whole UI with realistic sample data so it never looks
// empty while testing, with no backend running. Toggled from the nav, persisted
// in localStorage. The query hooks (queries.ts) read `useDemo()` and return
// this sample data instead of hitting the API.

import { createContext, use, useState, type ReactNode } from 'react'

const KEY = 'dd.demo'

// demoAvailable is true when demo mode can be toggled: always in dev, or when
// the VITE_ENABLE_DEMO env var is set during a production build. Vite statically
// replaces import.meta.env at build time, so both branches tree-shake cleanly.
export function demoAvailable(): boolean {
  const v = import.meta.env.VITE_ENABLE_DEMO
  return import.meta.env.DEV || v === 'true' || v === '1'
}

interface DemoValue {
  demo: boolean
  setDemo: (v: boolean) => void
}
const DemoContext = createContext<DemoValue | null>(null)

export function DemoProvider({ children }: { children: ReactNode }) {
  const [demo, set] = useState<boolean>(() =>
    demoAvailable() ? localStorage.getItem(KEY) === '1' : false,
  )
  function setDemo(v: boolean) {
    set(v)
    localStorage.setItem(KEY, v ? '1' : '0')
  }
  return <DemoContext value={{ demo, setDemo }}>{children}</DemoContext>
}

export function useDemo(): DemoValue {
  const ctx = use(DemoContext)
  if (!ctx) throw new Error('useDemo must be used within DemoProvider')
  return ctx
}
