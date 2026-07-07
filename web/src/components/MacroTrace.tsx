// Slide-up modal that shows the resolved macro trace for each meal item:
// food name, source, confidence tier, and per-macro breakdown.

import { useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { MACRO_KEYS, MACRO_META, type ResolvedItem } from '@/lib/types'
import { confidenceTier, confidenceColor } from '@/lib/format'
import { Pill } from '@/components/ui'
import { CloseIcon } from '@/components/icons'
import { easeOut } from '@/lib/motion'

export function MacroTrace({
  items,
  onClose,
}: {
  items: ResolvedItem[]
  onClose: () => void
}) {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0"
        style={{ zIndex: 1500 }}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
      >
        <div
          className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
          style={{ zIndex: 1400 }}
          onClick={onClose}
        />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label="Macro trace"
          className="absolute bottom-0 left-0 right-0 max-h-[80vh] rounded-t-2xl border border-b-0 border-line bg-surface p-5 shadow-lift"
          style={{ zIndex: 1500 }}
          initial={{ y: '100%' }}
          animate={{ y: 0 }}
          exit={{ y: '100%' }}
          transition={{ duration: 0.4, ease: easeOut }}
        >
          <div className="mb-5 flex items-center justify-between">
            <h2 className="text-lg font-bold text-ink">Macro trace</h2>
            <button onClick={onClose} aria-label="Close" className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          <div className="flex flex-col gap-3 overflow-y-auto pr-1">
            {items.length === 0 && (
              <p className="py-6 text-center text-sm text-muted">No items to trace.</p>
            )}
            {items.map((item, i) => {
              const tier = confidenceTier(item.Match.MatchScore)
              return (
                <div key={i} className="rounded-lg border border-line p-4">
                  <p className="truncate font-semibold text-ink">{item.Match.Name}</p>
                  <div className="mt-1.5 flex flex-wrap items-center gap-2 text-xs">
                    <Pill tone="neutral">{item.Match.Source}</Pill>
                    <span
                      className={`tnum font-medium ${confidenceColor(item.Match.MatchScore) || 'text-ink'}`}
                    >
                      {Math.round(item.Match.MatchScore * 100)}%
                    </span>
                    <span className="text-muted">{tier}</span>
                  </div>
                  <dl className="mt-3 grid grid-cols-5 gap-2 border-t border-line pt-3">
                    {MACRO_KEYS.map((k) => (
                      <div key={k}>
                        <dt className="text-[10px] uppercase tracking-[0.1em] text-muted">
                          {MACRO_META[k].label}
                        </dt>
                        <dd className="font-semibold text-ink tnum">
                          {Math.round(item.Macros[k])}
                        </dd>
                      </div>
                    ))}
                  </dl>
                </div>
              )
            })}
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
