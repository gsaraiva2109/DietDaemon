// Collapses a run of adjacent tool-call parts into one card — the
// Copilot-style "reasoning" panel: collapsed by default once done, expanded
// automatically while any call in the run is still going, toggle to inspect.

import { useEffect, useState, type ReactNode } from 'react'
import { motion } from 'framer-motion'
import { SparkleIcon, ChevronDown } from './icons'
import { fadeUp } from '@/lib/motion'

interface ToolCallGroupProps {
  running: boolean
  count: number
  children: ReactNode
}

export function ToolCallGroup({ running, count, children }: ToolCallGroupProps) {
  const [open, setOpen] = useState(running)

  useEffect(() => {
    if (running) setOpen(true)
  }, [running])

  return (
    <motion.div
      variants={fadeUp}
      initial="hidden"
      animate="show"
      className="my-1.5 overflow-hidden rounded-lg border border-line bg-surface-2"
    >
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 px-3 py-2 text-xs text-muted transition hover:text-ink"
      >
        <span className="text-primary">
          <SparkleIcon width={14} height={14} />
        </span>
        <span className="flex-1 text-left">
          {running ? 'Running' : 'Ran'} {count} tool{count === 1 ? '' : 's'}
        </span>
        {running && <span className="size-3 animate-spin rounded-full border-2 border-line border-t-primary" />}
        <ChevronDown width={14} height={14} className={`transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && <div className="flex flex-col divide-y divide-line border-t border-line px-3">{children}</div>}
    </motion.div>
  )
}
