// Add or correct a resolved item. The backend expects the COMPLETE
// ResolvedItem; correct targets a zero-based index, add appends. Keeps the
// product's "honest about uncertainty" principle: fixing/adding is easy.

import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import type { Meal, MacroKey, ResolvedItem } from '@/lib/types'
import { MACRO_KEYS, MACRO_META } from '@/lib/types'
import { useCorrectItem, useAddItem } from '@/lib/queries'
import { Button } from './ui'
import { CloseIcon } from './icons'

interface Props {
  meal: Meal
  /** index to correct; omit (undefined) to add a new item */
  index?: number
  onClose: () => void
}

const BLANK: ResolvedItem = {
  Parsed: { RawPhrase: '', Quantity: 0, Unit: 'g', NormalizedGrams: 0, Locale: '' },
  Match: { FoodID: '', Name: '', Source: 'manual', Per100g: { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }, MatchScore: 1 },
  Macros: { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 },
}

export function CorrectItemModal({ meal, index, onClose }: Props) {
  const { t } = useTranslation()
  const isAdd = index === undefined
  const base = isAdd ? BLANK : meal.Items[index]
  const correct = useCorrectItem(meal.ID)
  const add = useAddItem(meal.ID)
  const pending = correct.isPending || add.isPending
  const error = correct.error ?? add.error

  const [name, setName] = useState(base.Match.Name)
  const [grams, setGrams] = useState(base.Parsed.NormalizedGrams)
  const [macros, setMacros] = useState({ ...base.Macros })

  function setMacro(key: MacroKey, v: number) {
    setMacros((m) => ({ ...m, [key]: v }))
  }

  function submit() {
    const result: ResolvedItem = {
      ...base,
      Parsed: { ...base.Parsed, RawPhrase: base.Parsed.RawPhrase || name, NormalizedGrams: grams, Quantity: grams || base.Parsed.Quantity },
      Match: { ...base.Match, Name: name },
      Macros: macros,
    }
    if (isAdd) add.mutate(result, { onSuccess: onClose })
    else correct.mutate({ index, corrected: result }, { onSuccess: onClose })
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
        <div className="absolute inset-0 bg-ink/30 backdrop-blur-sm" style={{ zIndex: 1200 }} onClick={onClose} />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label={isAdd ? t('correctItemModal.ariaAddItem') : t('correctItemModal.ariaCorrectItem', { name: base.Match.Name })}
          initial={{ opacity: 0, scale: 0.97, y: 8 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.97 }}
          className="relative w-full max-w-md rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1300 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                {isAdd ? t('correctItemModal.eyebrowAdd') : t('correctItemModal.eyebrowCorrect')}
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">{isAdd ? t('correctItemModal.titleAdd') : base.Parsed.RawPhrase}</h2>
            </div>
            <button onClick={onClose} aria-label={t('correctItemModal.close')} className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          <label className="mb-3 block">
            <span className="mb-1 block text-xs font-medium text-muted">{t('correctItemModal.foodNameLabel')}</span>
            <input
              value={name}
              autoFocus={isAdd}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('correctItemModal.foodNamePlaceholder')}
              className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none focus:border-primary"
            />
          </label>

          <label className="mb-4 block">
            <span className="mb-1 block text-xs font-medium text-muted">{t('correctItemModal.gramsLabel')}</span>
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
                  {t(`common.macro.${k}`)} ({MACRO_META[k].unit})
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

          {error && (
            <p className="mt-3 text-sm font-medium text-accent" role="alert">
              {error instanceof Error ? error.message : t('correctItemModal.saveFailed')}
            </p>
          )}

          <div className="mt-6 flex justify-end gap-2">
            <Button variant="ghost" onClick={onClose}>
              {t('correctItemModal.cancel')}
            </Button>
            <Button onClick={submit} disabled={pending || (isAdd && !name.trim())}>
              {pending ? t('correctItemModal.saving') : isAdd ? t('correctItemModal.addItemButton') : t('correctItemModal.saveCorrection')}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
