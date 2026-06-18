// Correct one resolved item. The backend expects the COMPLETE ResolvedItem
// back at a zero-based index; we edit a copy and POST the whole object. Keeps
// the product's "honest about uncertainty" principle: fixing a guess is easy.

import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import type { Meal, MacroKey, ResolvedItem } from '@/lib/types'
import { MACRO_KEYS, MACRO_META } from '@/lib/types'
import { useCorrectItem } from '@/lib/queries'
import { Button } from './ui'
import { CloseIcon } from './icons'

interface Props {
  meal: Meal
  index: number
  onClose: () => void
}

export function CorrectItemModal({ meal, index, onClose }: Props) {
  const item = meal.Items[index]
  const correct = useCorrectItem(meal.ID)
  const [name, setName] = useState(item.Match.Name)
  const [grams, setGrams] = useState(item.Parsed.NormalizedGrams)
  const [macros, setMacros] = useState({ ...item.Macros })

  function setMacro(key: MacroKey, v: number) {
    setMacros((m) => ({ ...m, [key]: v }))
  }

  function submit() {
    const corrected: ResolvedItem = {
      ...item,
      Parsed: { ...item.Parsed, NormalizedGrams: grams },
      Match: { ...item.Match, Name: name },
      Macros: macros,
    }
    correct.mutate({ index, corrected }, { onSuccess: onClose })
  }

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 grid place-items-center p-4"
        style={{ zIndex: 1300 }}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
      >
        <div
          className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
          style={{ zIndex: 1200 }}
          onClick={onClose}
        />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label={`Correct ${item.Match.Name}`}
          initial={{ opacity: 0, scale: 0.97, y: 8 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.97 }}
          className="relative w-full max-w-md rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1300 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">Correct item</p>
              <h2 className="mt-1 text-xl font-bold text-ink">{item.Parsed.RawPhrase}</h2>
            </div>
            <button onClick={onClose} aria-label="Close" className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          <label className="mb-3 block">
            <span className="mb-1 block text-xs font-medium text-muted">Food name</span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none focus:border-primary"
            />
          </label>

          <label className="mb-4 block">
            <span className="mb-1 block text-xs font-medium text-muted">Grams</span>
            <input
              type="number"
              value={grams}
              onChange={(e) => setGrams(Number(e.target.value))}
              className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none focus:border-primary tnum"
            />
          </label>

          <div className="grid grid-cols-2 gap-3">
            {MACRO_KEYS.map((k) => (
              <label key={k} className="block">
                <span className="mb-1 block text-xs font-medium text-muted">
                  {MACRO_META[k].label} ({MACRO_META[k].unit})
                </span>
                <input
                  type="number"
                  value={macros[k]}
                  onChange={(e) => setMacro(k, Number(e.target.value))}
                  className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none focus:border-primary tnum"
                />
              </label>
            ))}
          </div>

          {correct.isError && (
            <p className="mt-3 text-sm font-medium text-accent" role="alert">
              {correct.error instanceof Error ? correct.error.message : 'Failed to save'}
            </p>
          )}

          <div className="mt-6 flex justify-end gap-2">
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={submit} disabled={correct.isPending}>
              {correct.isPending ? 'Saving…' : 'Save correction'}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
