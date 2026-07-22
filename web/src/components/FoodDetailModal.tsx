// Full food detail in a modal, fetched fresh by id. Shows the complete
// per-100g breakdown, serving info, provenance, aliases, and a shortcut to log.

import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion, type Variants } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useFood,
  useAddAlias,
  useDeleteAlias,
  useRemoveFromLibrary,
  useAddToLibrary,
  useDeleteCustomFood,
  useAddServingUnit,
  useDeleteServingUnit,
} from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { Button, Pill, Spinner } from './ui'
import { CloseIcon, LogIcon } from './icons'
import { sourceLabel } from './FoodCard'
import { MACRO_KEYS, type FoodDetail, type Macros } from '@/lib/types'
import { formatNumber, round } from '@/lib/format'
import { easeOut } from '@/lib/motion'

const PER_100G_BASIS = '100g'

function scaleMacros(per100g: Macros, grams: number): Macros {
  const factor = grams / 100
  return {
    Calories: per100g.Calories * factor,
    Protein: per100g.Protein * factor,
    Carbs: per100g.Carbs * factor,
    Fat: per100g.Fat * factor,
    Fiber: per100g.Fiber * factor,
  }
}

const scaleInDialog: Variants = {
  hidden: { opacity: 0, scale: 0.96, y: 8 },
  show: { opacity: 1, scale: 1, y: 0, transition: { duration: 0.4, ease: easeOut } },
}

export function FoodDetailModal({ foodID, onClose, onEditCustom }: {
  foodID: string
  onClose: () => void
  onEditCustom?: (food: FoodDetail) => void
}) {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const food = useFood(foodID)
  const navigate = useNavigate()
  const addAlias = useAddAlias(foodID)
  const deleteAlias = useDeleteAlias(foodID)
  const removeFromLibrary = useRemoveFromLibrary(foodID)
  const addToLibrary = useAddToLibrary(foodID)
  const deleteCustom = useDeleteCustomFood(foodID)
  const addServingUnit = useAddServingUnit(foodID)
  const deleteServingUnit = useDeleteServingUnit(foodID)
  const [aliasValue, setAliasValue] = useState('')
  const [confirmRemove, setConfirmRemove] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [basis, setBasis] = useState(PER_100G_BASIS)
  const [unitLabel, setUnitLabel] = useState('')
  const [unitGrams, setUnitGrams] = useState('')

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const f = food.data

  function logit() {
    if (!f) return
    navigate(`/log?text=${encodeURIComponent(f.name)}`)
    onClose()
  }

  function submitAlias() {
    const v = aliasValue.trim()
    if (!v || demo) return
    addAlias.mutate(v)
    setAliasValue('')
  }

  function submitUnit() {
    const grams = Number(unitGrams)
    if (!unitLabel.trim() || !(grams > 0) || demo) return
    addServingUnit.mutate(
      { label: unitLabel.trim(), grams },
      { onSuccess: () => { setUnitLabel(''); setUnitGrams('') } },
    )
  }

  const servingUnits = useMemo(() => f?.serving_units ?? [], [f])
  // Basis resets whenever the selected unit no longer exists (e.g. it was
  // just deleted) rather than silently showing stale scaled macros.
  const activeBasis = basis === PER_100G_BASIS || servingUnits.some((u) => u.id === basis) ? basis : PER_100G_BASIS
  const displayedMacros = useMemo(() => {
    if (!f) return undefined
    if (activeBasis === PER_100G_BASIS) return f.per_100g
    const unit = servingUnits.find((u) => u.id === activeBasis)
    return unit ? scaleMacros(f.per_100g, unit.grams) : f.per_100g
  }, [f, activeBasis, servingUnits])

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
          aria-label={f ? f.name : t('foodDetailModal.ariaLabelFallback')}
          variants={scaleInDialog}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative w-full max-w-md rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <button
            onClick={onClose}
            aria-label={t('foodDetailModal.close')}
            className="absolute right-4 top-4 text-muted hover:text-ink"
          >
            <CloseIcon />
          </button>

          {food.isLoading || !f ? (
            <div className="py-10">
              <Spinner label={t('foodDetailModal.loading')} />
            </div>
          ) : (
            <>
              <h2 className="pr-8 text-xl font-bold text-ink">{f.name}</h2>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <Pill tone={f.source === 'food_library' || f.source === 'custom' ? 'primary' : 'neutral'}>
                  {sourceLabel(f.source, t)}
                </Pill>
                {f.category && <Pill tone="muted">{f.category}</Pill>}
              </div>

              {servingUnits.length > 0 && (
                <div className="mt-4 flex items-center gap-2">
                  <label className="text-[11px] font-medium uppercase tracking-[0.1em] text-muted">
                    {t('foodDetailModal.macroBasis')}
                  </label>
                  <select
                    value={activeBasis}
                    onChange={(e) => setBasis(e.target.value)}
                    className="rounded-lg border border-line bg-bg px-2 py-1 text-sm text-ink outline-none focus:border-primary"
                  >
                    <option value={PER_100G_BASIS}>{t('foodDetailModal.per100g')}</option>
                    {servingUnits.map((u) => (
                      <option key={u.id} value={u.id}>
                        {t('foodDetailModal.perUnit', { label: u.label })}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <dl className="mt-3 grid grid-cols-5 gap-2 border-t border-line pt-4">
                {MACRO_KEYS.map((k) => (
                  <div key={k}>
                    <dt className="text-[10px] uppercase tracking-[0.1em] text-muted">
                      {t(`common.macro.${k}`)}
                    </dt>
                    <dd className="font-semibold text-ink tnum">
                      {displayedMacros && (k === 'Calories' ? formatNumber(displayedMacros[k]) : round(displayedMacros[k]))}
                    </dd>
                  </div>
                ))}
              </dl>
              <p className="mt-1 text-[11px] text-muted">
                {activeBasis === PER_100G_BASIS
                  ? t('foodDetailModal.per100g')
                  : t('foodDetailModal.perUnit', { label: servingUnits.find((u) => u.id === activeBasis)?.label })}
              </p>

              <div className="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted">
                {f.serving_size > 0 && (
                  <span>
                    {/* OFF's "quantity" field is package net weight, not a serving (#134) — never
                        offered as a loggable unit; labeled accordingly so it doesn't read as one. */}
                    {t(f.source === 'openfoodfacts' ? 'foodDetailModal.packageSize' : 'foodDetailModal.serving')}{' '}
                    <span className="text-ink">
                      {round(f.serving_size)}
                      {f.serving_unit}
                    </span>
                  </span>
                )}
                {f.brand && (
                  <span>
                    {t('foodDetailModal.brand')} <span className="text-ink">{f.brand}</span>
                  </span>
                )}
                {f.barcode && (
                  <span>
                    {t('foodDetailModal.barcode')} <span className="text-ink tnum">{f.barcode}</span>
                  </span>
                )}
              </div>

              <div className="mt-4">
                <p className="mb-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                  {t('foodDetailModal.manageUnitsTitle')}
                </p>
                <div className="flex flex-wrap items-center gap-1.5">
                  {servingUnits.length === 0 && (
                    <span className="text-sm text-muted">{t('foodDetailModal.noUnits')}</span>
                  )}
                  {servingUnits.map((u) => (
                    <span
                      key={u.id}
                      className="inline-flex items-center gap-1 rounded-full border border-line bg-surface-2 py-0.5 pl-2.5 pr-1 text-xs font-medium text-ink"
                    >
                      {u.label} ({round(u.grams)}g)
                      {u.custom && !demo && (
                        <button
                          onClick={() => deleteServingUnit.mutate(u.id)}
                          disabled={deleteServingUnit.isPending}
                          aria-label={t('foodDetailModal.removeUnit', { label: u.label })}
                          className="grid size-5 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
                        >
                          <CloseIcon width={12} height={12} />
                        </button>
                      )}
                    </span>
                  ))}
                </div>
                {!demo && (
                  <div className="mt-3 flex gap-2">
                    <input
                      value={unitLabel}
                      onChange={(e) => setUnitLabel(e.target.value)}
                      placeholder={t('foodDetailModal.addUnitLabelPlaceholder')}
                      aria-label={t('foodDetailModal.addUnitLabelPlaceholder')}
                      className="min-w-0 flex-1 rounded-full border border-line bg-bg px-3.5 py-2 text-sm text-ink outline-none transition focus:border-primary"
                    />
                    <input
                      type="number"
                      inputMode="decimal"
                      min="0"
                      step="any"
                      value={unitGrams}
                      onChange={(e) => setUnitGrams(e.target.value)}
                      placeholder={t('foodDetailModal.addUnitGramsPlaceholder')}
                      aria-label={t('foodDetailModal.addUnitGramsPlaceholder')}
                      className="w-24 rounded-full border border-line bg-bg px-3.5 py-2 text-sm text-ink outline-none transition focus:border-primary"
                    />
                    <Button
                      onClick={submitUnit}
                      disabled={!unitLabel.trim() || !(Number(unitGrams) > 0) || addServingUnit.isPending}
                      className="px-4 py-2 text-sm"
                    >
                      {t('foodDetailModal.addUnitSave')}
                    </Button>
                  </div>
                )}
              </div>

              {f.in_library ? (
                <div className="mt-4">
                  <p className="mb-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                    {t('foodDetailModal.manageAliasesTitle')}
                  </p>
                  <div className="flex flex-wrap items-center gap-1.5">
                    {(f.aliases ?? []).length === 0 && (
                      <span className="text-sm text-muted">{t('aliases.noAliases')}</span>
                    )}
                    {(f.aliases ?? []).map((a) => (
                      <span
                        key={a.alias}
                        className="inline-flex items-center gap-1 rounded-full border border-line bg-surface-2 py-0.5 pl-2.5 pr-1 text-xs font-medium text-ink"
                      >
                        {a.alias}
                        {!demo && (
                          <button
                            onClick={() => deleteAlias.mutate(a.alias)}
                            disabled={deleteAlias.isPending}
                            aria-label={t('aliases.removeAlias', { alias: a.alias })}
                            className="grid size-5 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
                          >
                            <CloseIcon width={12} height={12} />
                          </button>
                        )}
                      </span>
                    ))}
                  </div>

                  {!demo && (
                    <div className="mt-3 flex gap-2">
                      <input
                        value={aliasValue}
                        onChange={(e) => setAliasValue(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') submitAlias()
                        }}
                        placeholder={t('aliases.addPlaceholder')}
                        aria-label={t('aliases.addAriaLabel', { food: f.name })}
                        className="min-w-0 flex-1 rounded-full border border-line bg-bg px-3.5 py-2 text-sm text-ink outline-none transition focus:border-primary"
                      />
                      <Button
                        onClick={submitAlias}
                        disabled={!aliasValue.trim() || addAlias.isPending}
                        className="px-4 py-2 text-sm"
                      >
                        {t('aliases.add')}
                      </Button>
                    </div>
                  )}
                </div>
              ) : (
                <p className="mt-4 text-sm text-muted">{t('foodDetailModal.notInLibraryHint')}</p>
              )}

              <div className="mt-6 flex items-center justify-end gap-2">
                {f.source === 'custom' && !demo && (
                  <>
                    {confirmDelete ? (
                      <span className="flex items-center gap-2 text-sm text-muted">
                        {t('foodDetailModal.deleteConfirmTitle')}
                        <button
                          onClick={() => deleteCustom.mutate(undefined, { onSuccess: onClose })}
                          disabled={deleteCustom.isPending}
                          className="font-semibold text-accent hover:underline disabled:opacity-50"
                        >
                          {t('foodDetailModal.deleteConfirmYes')}
                        </button>
                        <button onClick={() => setConfirmDelete(false)} className="font-medium text-ink hover:underline">
                          {t('foodDetailModal.deleteConfirmNo')}
                        </button>
                      </span>
                    ) : (
                      <Button variant="ghost" onClick={() => setConfirmDelete(true)}>
                        {t('foodDetailModal.deleteCustom')}
                      </Button>
                    )}
                    <Button variant="ghost" onClick={() => onEditCustom?.(f)}>
                      {t('foodDetailModal.editCustom')}
                    </Button>
                  </>
                )}
                {!f.in_library && !demo && (
                  <Button
                    variant="ghost"
                    onClick={() => addToLibrary.mutate()}
                    disabled={addToLibrary.isPending}
                  >
                    {t('foodDetailModal.addToLibrary')}
                  </Button>
                )}
                {f.in_library && f.source !== 'custom' && !demo && (
                  <>
                    {confirmRemove ? (
                      <span className="flex items-center gap-2 text-sm text-muted">
                        {t('foodDetailModal.removeConfirmTitle')}
                        <button
                          onClick={() =>
                            removeFromLibrary.mutate(undefined, { onSuccess: onClose })
                          }
                          disabled={removeFromLibrary.isPending}
                          className="font-semibold text-accent hover:underline disabled:opacity-50"
                        >
                          {t('foodDetailModal.removeConfirmYes')}
                        </button>
                        <button
                          onClick={() => setConfirmRemove(false)}
                          className="font-medium text-ink hover:underline"
                        >
                          {t('foodDetailModal.removeConfirmNo')}
                        </button>
                      </span>
                    ) : (
                      <Button variant="ghost" onClick={() => setConfirmRemove(true)}>
                        {t('foodDetailModal.removeFromLibrary')}
                      </Button>
                    )}
                  </>
                )}
                <Button onClick={logit}>
                  <LogIcon width={16} height={16} /> {t('foodDetailModal.logThis')}
                </Button>
              </div>
            </>
          )}
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
