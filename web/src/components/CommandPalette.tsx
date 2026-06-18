// ⌘K / Ctrl-K command palette: jump between screens and toggle theme/demo
// from anywhere. Mounted once at the app root (inside the router + providers).

import { useEffect, useMemo, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { useTheme } from '@/lib/theme'
import { useDemo } from '@/lib/demo'
import {
  TodayIcon,
  LogIcon,
  HistoryIcon,
  TrendsIcon,
  SummaryIcon,
  SettingsIcon,
  SunIcon,
  MoonIcon,
  SparkleIcon,
  SearchIcon,
  FoodsIcon,
  TemplateIcon,
  BodyIcon,
  GoalIcon,
} from './icons'
import type { SVGProps } from 'react'

interface Command {
  id: string
  label: string
  hint?: string
  Icon: (p: SVGProps<SVGSVGElement>) => React.ReactNode
  run: () => void
}

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState('')
  const [active, setActive] = useState(0)
  const navigate = useNavigate()
  const { theme, toggle } = useTheme()
  const { demo, setDemo } = useDemo()
  const inputRef = useRef<HTMLInputElement>(null)

  const commands = useMemo<Command[]>(() => {
    const go = (to: string) => () => {
      navigate(to)
      setOpen(false)
    }
    return [
      { id: 'today', label: 'Go to Today', hint: '⌘D', Icon: TodayIcon, run: go('/') },
      { id: 'log', label: 'Log a meal', hint: '⌘L', Icon: LogIcon, run: go('/log') },
      { id: 'history', label: 'Go to History', hint: '⌘H', Icon: HistoryIcon, run: go('/history') },
      { id: 'foods', label: 'Go to Foods', Icon: FoodsIcon, run: go('/foods') },
      { id: 'templates', label: 'Go to Templates', Icon: TemplateIcon, run: go('/templates') },
      { id: 'body', label: 'Go to Body', Icon: BodyIcon, run: go('/body') },
      { id: 'goals', label: 'Go to Goals', Icon: GoalIcon, run: go('/goals') },
      { id: 'trends', label: 'Go to Trends', hint: '⌘T', Icon: TrendsIcon, run: go('/trends') },
      { id: 'summary', label: 'Go to Summary', hint: '⌘S', Icon: SummaryIcon, run: go('/summary') },
      { id: 'settings', label: 'Go to Settings', Icon: SettingsIcon, run: go('/settings') },
      {
        id: 'theme',
        label: `Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`,
        Icon: theme === 'dark' ? SunIcon : MoonIcon,
        run: () => {
          toggle()
          setOpen(false)
        },
      },
      {
        id: 'demo',
        label: demo ? 'Turn demo data off' : 'Turn demo data on',
        Icon: SparkleIcon,
        run: () => {
          setDemo(!demo)
          setOpen(false)
        },
      },
    ]
  }, [navigate, theme, toggle, demo, setDemo])

  const results = useMemo(() => {
    const n = q.trim().toLowerCase()
    return n ? commands.filter((c) => c.label.toLowerCase().includes(n)) : commands
  }, [q, commands])

  // Global ⌘K / Ctrl-K toggle.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen((o) => {
          if (!o) {
            // Reset search state when opening — fire-and-forget
            // updates in the same render batch, no cascading effects.
            setQ('')
            setActive(0)
            setTimeout(() => inputRef.current?.focus(), 20)
          }
          return !o
        })
      }
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  // Direct navigation shortcuts: ⌘/Ctrl + L/H/T/S/D. Ignored while typing in a
  // field so they don't hijack text entry.
  useEffect(() => {
    const ROUTES: Record<string, string> = { l: '/log', h: '/history', t: '/trends', s: '/summary', d: '/' }
    function onKey(e: KeyboardEvent) {
      if (!(e.metaKey || e.ctrlKey) || e.altKey || e.shiftKey) return
      const target = e.target as HTMLElement | null
      if (target && (target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName))) return
      const to = ROUTES[e.key.toLowerCase()]
      if (!to) return
      e.preventDefault()
      navigate(to)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [navigate])

  function onListKey(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActive((a) => Math.min(a + 1, results.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((a) => Math.max(a - 1, 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      results[active]?.run()
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="fixed inset-0 grid place-items-start justify-center p-4 pt-[12vh]"
          style={{ zIndex: 1500 }}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        >
          <div className="absolute inset-0 bg-ink/30 backdrop-blur-sm" onClick={() => setOpen(false)} />
          <motion.div
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
            initial={{ opacity: 0, scale: 0.98, y: -8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.98, y: -8 }}
            className="relative w-full max-w-lg overflow-hidden rounded-2xl border border-line bg-surface shadow-lift"
            onKeyDown={onListKey}
          >
            <div className="flex items-center gap-3 border-b border-line px-4 py-3">
              <span className="text-muted">
                <SearchIcon width={18} height={18} />
              </span>
              <input
                ref={inputRef}
                value={q}
                onChange={(e) => { setQ(e.target.value); setActive(0) }}
                placeholder="Type a command…"
                className="flex-1 bg-transparent text-ink outline-none placeholder:text-muted"
              />
              <kbd className="rounded border border-line px-1.5 py-0.5 text-[10px] text-muted">ESC</kbd>
            </div>
            <ul className="max-h-80 overflow-y-auto p-2">
              {results.length === 0 && <li className="px-3 py-6 text-center text-sm text-muted">No commands</li>}
              {results.map((c, i) => (
                <li key={c.id}>
                  <button
                    onMouseMove={() => setActive(i)}
                    onClick={c.run}
                    className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition ${
                      i === active ? 'bg-primary-soft text-primary' : 'text-ink hover:bg-surface-2'
                    }`}
                  >
                    <c.Icon width={18} height={18} />
                    <span className="flex-1">{c.label}</span>
                    {c.hint && (
                      <kbd className="rounded border border-line px-1.5 py-0.5 text-[10px] text-muted">{c.hint}</kbd>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
