// Save the current set of resolved items as a reusable meal template. The
// backend stores the COMPLETE ResolvedItem list; we just name it.

import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import type { ResolvedItem } from '@/lib/types'
import { useCreateTemplate } from '@/lib/queries'
import { Button } from './ui'
import { CloseIcon } from './icons'
import { formatGrams } from '@/lib/format'
import { scaleIn } from '@/lib/motion'

interface Props {
  items: ResolvedItem[]
  onClose: () => void
}

export function SaveTemplateModal({ items, onClose }: Props) {
  const create = useCreateTemplate()
  const [name, setName] = useState('')
  const error = create.error
  const disabled = !name.trim() || !items.length || create.isPending

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  function submit() {
    if (disabled) return
    create.mutate({ name: name.trim(), items }, { onSuccess: onClose })
  }

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 grid place-items-center p-4"
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
          aria-label="Save meal as template"
          variants={scaleIn}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative w-full max-w-md rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                Save template
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">Name this meal</h2>
            </div>
            <button onClick={onClose} aria-label="Close" className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          <label className="mb-4 block">
            <span className="mb-1 block text-xs font-medium text-muted">Template name</span>
            <input
              value={name}
              autoFocus
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && submit()}
              placeholder="e.g. Post-workout breakfast"
              className="w-full rounded-full border border-line bg-bg px-4 py-2 text-ink outline-none transition focus:border-primary"
            />
          </label>

          <div className="mb-2 text-xs font-medium text-muted">
            {items.length} item{items.length === 1 ? '' : 's'}
          </div>
          <ul className="mb-2 max-h-56 divide-y divide-line overflow-y-auto rounded-lg border border-line">
            {items.map((it, i) => (
              <li key={i} className="flex items-center justify-between gap-3 px-3 py-2">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-ink">
                    {it.Match.Name || it.Parsed.RawPhrase}
                  </p>
                  <p className="text-xs text-muted">{formatGrams(it.Parsed.NormalizedGrams)}</p>
                </div>
                <span className="shrink-0 text-sm font-semibold text-ink tnum">
                  {Math.round(it.Macros.Calories)} kcal
                </span>
              </li>
            ))}
            {!items.length && (
              <li className="px-3 py-4 text-center text-sm text-muted">No items to save.</li>
            )}
          </ul>

          {error && (
            <p className="mt-3 text-sm font-medium text-accent" role="alert">
              {error instanceof Error ? error.message : 'Failed to save template'}
            </p>
          )}

          <div className="mt-6 flex justify-end gap-2">
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={submit} disabled={disabled}>
              {create.isPending ? 'Saving…' : 'Save template'}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
