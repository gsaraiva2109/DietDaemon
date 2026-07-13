// Add or correct a resolved item. The backend expects the COMPLETE
// ResolvedItem; correct targets a zero-based index, add appends. Keeps the
// product's "honest about uncertainty" principle: fixing/adding is easy.

import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import type { Meal, MacroKey, ResolvedItem, FoodDetail, Macros } from '@/lib/types'
import { MACRO_KEYS, MACRO_META } from '@/lib/types'
import { useCorrectItem, useAddItem, useCatalogSearch } from '@/lib/queries'
import { Button, Spinner } from './ui'
import { CloseIcon } from './icons'
import { sourceLabel } from './FoodCard'
import { round } from '@/lib/format'

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

  const [showCatalog, setShowCatalog] = useState(false)
  const [rawCatalogQuery, setRawCatalogQuery] = useState('')
  const [catalogQuery, setCatalogQuery] = useState('')
  const [preview, setPreview] = useState<FoodDetail | null>(null)
  const catalog = useCatalogSearch(catalogQuery)

  useEffect(() => {
    const id = setTimeout(() => setCatalogQuery(rawCatalogQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawCatalogQuery])

  function setMacro(key: MacroKey, v: number) {
    setMacros((m) => ({ ...m, [key]: v }))
  }

  function scaledMacros(f: FoodDetail): Macros {
    const factor = grams / 100
    return {
      Calories: f.per_100g.Calories * factor,
      Protein: f.per_100g.Protein * factor,
      Carbs: f.per_100g.Carbs * factor,
      Fat: f.per_100g.Fat * factor,
      Fiber: f.per_100g.Fiber * factor,
    }
  }

  function confirmReplace() {
    if (!preview) return
    setName(preview.name)
    setMacros(scaledMacros(preview))
    setPreview(null)
    setShowCatalog(false)
    setRawCatalogQuery('')
    setCatalogQuery('')
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

          <button
            type="button"
            onClick={() => setShowCatalog((v) => !v)}
            className="mb-3 text-sm font-medium text-primary hover:underline"
          >
            {t('correctItemModal.searchCatalogToggle')}
          </button>

          {showCatalog && (
            <div className="mb-4 rounded-lg border border-line bg-surface-2 p-3">
              <input
                value={rawCatalogQuery}
                autoFocus
                onChange={(e) => setRawCatalogQuery(e.target.value)}
                placeholder={t('correctItemModal.catalogSearchPlaceholder')}
                className="mb-2 w-full rounded-lg border border-line bg-bg px-3 py-2 text-sm text-ink outline-none focus:border-primary"
              />

              {catalog.isLoading ? (
                <Spinner label={t('foods.loadingLabel')} />
              ) : !catalog.data?.length ? (
                <p className="py-2 text-sm text-muted">{t('correctItemModal.catalogNoResults')}</p>
              ) : (
                <div className="max-h-40 space-y-1 overflow-y-auto">
                  {catalog.data.map((r: FoodDetail) => (
                    <button
                      key={r.food_id}
                      type="button"
                      onClick={() => setPreview(r)}
                      className="flex w-full items-center justify-between gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-surface"
                    >
                      <span className="min-w-0 truncate text-ink">{r.name}</span>
                      <span className="shrink-0 text-xs text-muted">
                        {sourceLabel(r.source, t)} · {round(r.per_100g.Calories)} kcal
                      </span>
                    </button>
                  ))}
                </div>
              )}

              {preview && (
                <div className="mt-3 rounded-lg border border-primary/30 bg-bg p-3">
                  <p className="mb-2 text-xs font-semibold uppercase tracking-[0.14em] text-muted">
                    {t('correctItemModal.previewTitle')}
                  </p>
                  <p className="mb-2 font-semibold text-ink">{preview.name}</p>
                  <div className="grid grid-cols-5 gap-2 text-xs">
                    {MACRO_KEYS.map((k) => (
                      <div key={k}>
                        <p className="text-muted">{t(`common.macro.${k}`)}</p>
                        <p className="font-semibold text-ink tnum">{round(scaledMacros(preview)[k])}</p>
                      </div>
                    ))}
                  </div>
                  <div className="mt-3 flex justify-end gap-2">
                    <Button variant="ghost" onClick={() => setPreview(null)} className="px-3 py-1.5 text-sm">
                      {t('correctItemModal.cancelReplace')}
                    </Button>
                    <Button onClick={confirmReplace} className="px-3 py-1.5 text-sm">
                      {t('correctItemModal.confirmReplace')}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}

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
