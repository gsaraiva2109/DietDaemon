// Top-right utility controls present on every screen: demo-mode toggle and
// light/dark theme toggle. Reachable on mobile and desktop.

import { useDemo, DEMO_TOGGLE_ENABLED } from '@/lib/demo'
import { SparkleIcon } from './icons'
import { ThemeToggle } from './ThemeToggle'

export function UtilityBar() {
  const { demo, setDemo } = useDemo()

  return (
    <div className="mb-2 flex items-center justify-end gap-2">
      {DEMO_TOGGLE_ENABLED && (
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
      )}
      <ThemeToggle />
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
