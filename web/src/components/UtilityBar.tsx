// Top-right utility controls present on every screen: demo-mode toggle and
// light/dark theme toggle. Reachable on mobile and desktop.

import { motion } from 'framer-motion'
import { useTheme } from '@/lib/theme'
import { useDemo } from '@/lib/demo'
import { SunIcon, MoonIcon, SparkleIcon } from './icons'

export function UtilityBar() {
  const { theme, toggle } = useTheme()
  const { demo, setDemo } = useDemo()

  return (
    <div className="mb-2 flex items-center justify-end gap-2">
      <button
        onClick={() => setDemo(!demo)}
        aria-pressed={demo}
        className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-semibold transition ${
          demo
            ? 'border-transparent bg-primary text-primary-ink'
            : 'border-line bg-surface text-muted hover:text-ink'
        }`}
      >
        <SparkleIcon width={15} height={15} />
        Demo {demo ? 'on' : 'off'}
      </button>
      <button
        onClick={toggle}
        aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
        className="grid size-9 place-items-center rounded-full border border-line bg-surface text-muted transition hover:text-ink"
      >
        <motion.span key={theme} initial={{ rotate: -30, opacity: 0 }} animate={{ rotate: 0, opacity: 1 }}>
          {theme === 'dark' ? <MoonIcon width={18} height={18} /> : <SunIcon width={18} height={18} />}
        </motion.span>
      </button>
    </div>
  )
}

export function DemoBanner() {
  const { demo, setDemo } = useDemo()
  if (!demo) return null
  return (
    <div className="mb-5 flex items-center justify-between gap-3 rounded-xl border border-transparent bg-primary-soft px-4 py-2.5 text-sm text-primary">
      <span className="flex items-center gap-2 font-medium">
        <SparkleIcon width={16} height={16} />
        Demo data — sample meals &amp; trends, no backend needed.
      </span>
      <button onClick={() => setDemo(false)} className="font-semibold underline-offset-2 hover:underline">
        Turn off
      </button>
    </div>
  )
}
