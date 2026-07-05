// Compose a meal template from scratch by picking foods straight from the
// personal library, no prior meal log required. Mirrors SaveTemplateModal's
// shell; the food-search-and-pick pattern mirrors Aliases.tsx.

import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import type { FoodDetail, Macros } from '@/lib/types'
import { useSearchFoods, useComposeTemplate } from '@/lib/queries'
import { Button } from './ui'
import { CloseIcon, SearchIcon } from './icons'
import { formatNumber } from '@/lib/format'
import { scaleIn } from '@/lib/motion'

interface Props {
  onClose: () => void
}

interface Picked {
  food: FoodDetail
  grams: number
}

function scale(m: Macros, grams: number): Macros {
  const f = grams / 100
  return { Calories: m.Calories * f, Protein: m.Protein * f, Carbs: m.Carbs * f, Fat: m.Fat * f, Fiber: m.Fiber * f }
}

export function ComposeTemplateModal({ onClose }: Props) {
  const compose = useComposeTemplate()
  const [name, setName] = useState('')
  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')
  const [picked, setPicked] = useState<Picked[]>([])
  const error = compose.error

  useEffect(() => {
    const id = setTimeout(() => setQuery(rawQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawQuery])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const search = useSearchFoods(query)
  const results = (search.data ?? []).filter((f) => !picked.some((p) => p.food.food_id === f.food_id)).slice(0, 8)

  const total = picked.reduce((sum, p) => {
    const m = scale(p.food.per_100g, p.grams)
    return {
      Calories: sum.Calories + m.Calories,
      Protein: sum.Protein + m.Protein,
      Carbs: sum.Carbs + m.Carbs,
      Fat: sum.Fat + m.Fat,
      Fiber: sum.Fiber + m.Fiber,
    }
  }, { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 })

  const disabled = !name.trim() || !picked.length || compose.isPending

  function addFood(food: FoodDetail) {
    setPicked((p) => [...p, { food, grams: 100 }])
    setRawQuery('')
    setQuery('')
  }

  function setGrams(foodID: string, grams: number) {
    setPicked((p) => p.map((it) => (it.food.food_id === foodID ? { ...it, grams } : it)))
  }

  function removeFood(foodID: string) {
    setPicked((p) => p.filter((it) => it.food.food_id !== foodID))
  }

  function submit() {
    if (disabled) return
    compose.mutate(
      {
        name: name.trim(),
        items: picked.map((p) => ({ food_id: p.food.food_id, grams: p.grams })),
      },
      { onSuccess: onClose },
    )
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
          aria-label="Compose a template from your food library"
          variants={scaleIn}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative flex max-h-[85vh] w-full max-w-md flex-col rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                New from scratch
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">Compose a template</h2>
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
              placeholder="e.g. Post-workout breakfast"
              className="w-full rounded-full border border-line bg-bg px-4 py-2 text-ink outline-none transition focus:border-primary"
            />
          </label>

          <div className="relative mb-1">
            <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
              <SearchIcon width={16} height={16} />
            </span>
            <input
              value={rawQuery}
              onChange={(e) => setRawQuery(e.target.value)}
              placeholder="Search your food library"
              aria-label="Search foods to add"
              className="w-full rounded-full border border-line bg-bg py-2 pl-9 pr-4 text-sm text-ink outline-none transition focus:border-primary"
            />
          </div>

          {query.length > 0 && (
            <ul className="mb-3 max-h-40 divide-y divide-line overflow-y-auto rounded-lg border border-line">
              {search.isLoading ? (
                <li className="px-3 py-2 text-sm text-muted">Searching…</li>
              ) : results.length === 0 ? (
                <li className="px-3 py-2 text-sm text-muted">No matching foods.</li>
              ) : (
                results.map((f) => (
                  <li key={f.food_id}>
                    <button
                      onClick={() => addFood(f)}
                      className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm text-ink transition hover:bg-surface-2"
                    >
                      <span className="truncate">{f.name}</span>
                      <span className="shrink-0 text-xs text-muted">
                        {formatNumber(f.per_100g.Calories)} kcal/100g
                      </span>
                    </button>
                  </li>
                ))
              )}
            </ul>
          )}

          <div className="mb-2 text-xs font-medium text-muted">
            {picked.length} item{picked.length === 1 ? '' : 's'}
          </div>
          <ul className="mb-3 flex-1 divide-y divide-line overflow-y-auto rounded-lg border border-line">
            {picked.map((p) => {
              const m = scale(p.food.per_100g, p.grams)
              return (
                <li key={p.food.food_id} className="flex items-center gap-3 px-3 py-2">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-ink">{p.food.name}</p>
                    <p className="tnum text-xs text-muted">{formatNumber(m.Calories)} kcal</p>
                  </div>
                  <label className="flex shrink-0 items-center gap-1">
                    <input
                      type="number"
                      min={1}
                      value={p.grams}
                      onChange={(e) => setGrams(p.food.food_id, Number(e.target.value) || 0)}
                      aria-label={`Grams of ${p.food.name}`}
                      className="w-16 rounded-full border border-line bg-bg px-2 py-1 text-right text-sm text-ink outline-none transition focus:border-primary"
                    />
                    <span className="text-xs text-muted">g</span>
                  </label>
                  <button
                    onClick={() => removeFood(p.food.food_id)}
                    aria-label={`Remove ${p.food.name}`}
                    className="grid size-7 shrink-0 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
                  >
                    <CloseIcon width={13} height={13} />
                  </button>
                </li>
              )
            })}
            {!picked.length && (
              <li className="px-3 py-4 text-center text-sm text-muted">
                Search above to add foods.
              </li>
            )}
          </ul>

          {picked.length > 0 && (
            <div className="mb-3 flex items-center justify-between rounded-lg bg-surface-2 px-3 py-2 text-sm">
              <span className="font-medium text-ink">Total</span>
              <span className="tnum text-muted">
                {formatNumber(total.Calories)} kcal · {formatNumber(total.Protein)}P ·{' '}
                {formatNumber(total.Carbs)}C · {formatNumber(total.Fat)}F
              </span>
            </div>
          )}

          {error && (
            <p className="mb-2 text-sm font-medium text-accent" role="alert">
              {error instanceof Error ? error.message : 'Failed to save template'}
            </p>
          )}

          <div className="mt-1 flex justify-end gap-2">
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={submit} disabled={disabled}>
              {compose.isPending ? 'Saving…' : 'Save template'}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
