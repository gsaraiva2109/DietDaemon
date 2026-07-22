import { useState, type SyntheticEvent } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useCreateCustomFood, useUpdateCustomFood } from '@/lib/queries'
import type { CustomFoodInput, FoodDetail, NutritionLabelDraft } from '@/lib/types'
import { Button, Field, FormError } from './ui'
import { CloseIcon } from './icons'
import { OcrLabelUpload } from './OcrLabelUpload'

type FormValues = Record<keyof CustomFoodInput, string>

// Amber, matching the confidence-colour shade already used elsewhere (see
// lib/format.ts confidence text colour) rather than inventing a new tone.
const LOW_CONFIDENCE_CLASS = 'border-amber-600 dark:border-amber-400'

// NutritionLabelDraft field name -> the matching FormValues key.
const DRAFT_FIELD_MAP: Record<string, keyof FormValues> = {
  name: 'name',
  basis_grams: 'basis_grams',
  calories: 'calories',
  protein_g: 'protein',
  carbs_g: 'carbs',
  fat_g: 'fat',
  fiber_g: 'fiber',
}

const nutrientFields: { key: keyof Pick<CustomFoodInput, 'calories' | 'protein' | 'carbs' | 'fat' | 'fiber'>; labelKey: string; unit: string }[] = [
  { key: 'calories', labelKey: 'common.macro.Calories', unit: 'kcal' },
  { key: 'protein', labelKey: 'common.macro.Protein', unit: 'g' },
  { key: 'carbs', labelKey: 'common.macro.Carbs', unit: 'g' },
  { key: 'fat', labelKey: 'common.macro.Fat', unit: 'g' },
  { key: 'fiber', labelKey: 'common.macro.Fiber', unit: 'g' },
]

function valuesFor(food?: FoodDetail): FormValues {
  const basis = food?.serving_size || 100
  const scale = basis / 100
  return {
    name: food?.name ?? '',
    calories: food ? String(food.per_100g.Calories * scale) : '',
    protein: food ? String(food.per_100g.Protein * scale) : '',
    carbs: food ? String(food.per_100g.Carbs * scale) : '',
    fat: food ? String(food.per_100g.Fat * scale) : '',
    fiber: food ? String(food.per_100g.Fiber * scale) : '',
    basis_grams: String(basis),
  }
}

export function CustomFoodModal({ food, onClose, onSaved }: Readonly<{
  food?: FoodDetail
  onClose: () => void
  onSaved: (food: FoodDetail) => void
}>) {
  const { t } = useTranslation()
  const [values, setValues] = useState(() => valuesFor(food))
  const [lowConfidence, setLowConfidence] = useState<Set<keyof FormValues>>(new Set())
  const [unreadable, setUnreadable] = useState(false)
  const create = useCreateCustomFood()
  const update = useUpdateCustomFood(food?.food_id ?? '')
  const saving = create.isPending || update.isPending
  const error = create.error ?? update.error

  function set(key: keyof FormValues, value: string) {
    setValues((current) => ({ ...current, [key]: value }))
  }

  // Prefill-only: fills in whichever fields the scan found, flags
  // low-confidence ones, and never submits anything itself (issue #87).
  function onExtracted(draft: NutritionLabelDraft) {
    if (draft.unreadable) {
      setUnreadable(true)
      setLowConfidence(new Set())
      return
    }
    setUnreadable(false)
    setValues((current) => {
      const next = { ...current }
      for (const [draftKey, formKey] of Object.entries(DRAFT_FIELD_MAP)) {
        const value = draft[draftKey as keyof NutritionLabelDraft]
        if (value !== null && value !== undefined) next[formKey] = String(value)
      }
      return next
    })
    setLowConfidence(
      new Set(draft.low_confidence_fields.map((f) => DRAFT_FIELD_MAP[f]).filter((f): f is keyof FormValues => Boolean(f))),
    )
  }

  function submit(e: SyntheticEvent) {
    e.preventDefault()
    const input = {
      name: values.name.trim(),
      calories: Number(values.calories),
      protein: Number(values.protein),
      carbs: Number(values.carbs),
      fat: Number(values.fat),
      fiber: Number(values.fiber),
      basis_grams: Number(values.basis_grams),
    }
    const invalid = !input.name || input.basis_grams <= 0 || Object.values(input).some(
      (value) => typeof value === 'number' && (!Number.isFinite(value) || value < 0),
    )
    if (invalid) return
    const mutation = food ? update : create
    mutation.mutate(input, { onSuccess: onSaved })
  }

  const isComplete = values.name.trim() && Object.entries(values)
    .filter(([key]) => key !== 'name')
    .every(([, value]) => value !== '' && Number.isFinite(Number(value)) && Number(value) >= 0) && Number(values.basis_grams) > 0

  if (!food) {
    return (
        <motion.div
            className="fixed inset-0 grid place-items-center p-4"
            style={{zIndex: 1600}}
            initial={{opacity: 0}}
            animate={{opacity: 1}}
            exit={{opacity: 0}}
        >
          <button
              type="button"
              aria-label={t('customFood.close')}
              onClick={onClose}
              className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
          />
          <motion.form
              aria-label={t(food ? 'customFood.editAriaLabel' : 'customFood.createAriaLabel')}
              onSubmit={submit}
              initial={{opacity: 0, scale: 0.96, y: 8}}
              animate={{opacity: 1, scale: 1, y: 0}}
              className="relative w-full max-w-lg rounded-xl border border-line bg-surface p-6 shadow-lift"
          >
            <button
                type="button"
                onClick={onClose}
                aria-label={t('customFood.close')}
                className="absolute right-4 top-4 text-muted hover:text-ink"
            >
              <CloseIcon/>
            </button>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
              {t('customFood.eyebrow')}
            </p>
            <h2 className="mt-1 pr-8 text-xl font-bold text-ink">
              {t(food ? 'customFood.editTitle' : 'customFood.createTitle')}
            </h2>
            <p className="mt-1 text-sm text-muted">{t('customFood.labelHint')}</p>

            {!food && (
                <div className="mt-4">
                  <OcrLabelUpload onExtracted={onExtracted}/>
                  {unreadable && <FormError>{t('customFood.scanUnreadable')}</FormError>}
                </div>
            )}

            <div className="mt-5">
              <Field
                  label={t('customFood.nameLabel')}
                  value={values.name}
                  onChange={(e) => set('name', e.target.value)}
                  placeholder={t('customFood.namePlaceholder')}
                  inputClassName={lowConfidence.has('name') ? LOW_CONFIDENCE_CLASS : undefined}
                  hint={lowConfidence.has('name') ? t('customFood.lowConfidenceHint') : undefined}
                  autoFocus
              />
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <Field
                  label={t('customFood.basisLabel')}
                  type="number"
                  inputMode="decimal"
                  min="0.01"
                  step="any"
                  value={values.basis_grams}
                  onChange={(e) => set('basis_grams', e.target.value)}
                  inputClassName={lowConfidence.has('basis_grams') ? LOW_CONFIDENCE_CLASS : undefined}
                  hint={lowConfidence.has('basis_grams') ? t('customFood.lowConfidenceHint') : t('customFood.basisHint')}
              />
              <p className="self-end pb-3 text-sm text-muted">{t('customFood.nutrientHint')}</p>
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              {nutrientFields.map(({key, labelKey, unit}) => (
                  <Field
                      key={key}
                      label={`${t(labelKey)} (${unit})`}
                      type="number"
                      inputMode="decimal"
                      min="0"
                      step="any"
                      value={values[key]}
                      onChange={(e) => set(key, e.target.value)}
                      inputClassName={lowConfidence.has(key) ? LOW_CONFIDENCE_CLASS : undefined}
                      hint={lowConfidence.has(key) ? t('customFood.lowConfidenceHint') : undefined}
                  />
              ))}
            </div>

            {error && <p className="mt-4 text-sm text-accent">{error.message}</p>}
            <div className="mt-6 flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={onClose}>{t('customFood.cancel')}</Button>
              <Button type="submit" disabled={!isComplete || saving}>
                {saving ? t('customFood.saving') : t('customFood.save')}
              </Button>
            </div>
          </motion.form>
        </motion.div>
    )
  } else {
    return (
        <motion.div
            className="fixed inset-0 grid place-items-center p-4"
            style={{zIndex: 1600}}
            initial={{opacity: 0}}
            animate={{opacity: 1}}
            exit={{opacity: 0}}
        >
          <button
              type="button"
              aria-label={t('customFood.close')}
              onClick={onClose}
              className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
          />
          <motion.form
              aria-label={t(food ? 'customFood.editAriaLabel' : 'customFood.createAriaLabel')}
              onSubmit={submit}
              initial={{opacity: 0, scale: 0.96, y: 8}}
              animate={{opacity: 1, scale: 1, y: 0}}
              className="relative w-full max-w-lg rounded-xl border border-line bg-surface p-6 shadow-lift"
          >
            <button
                type="button"
                onClick={onClose}
                aria-label={t('customFood.close')}
                className="absolute right-4 top-4 text-muted hover:text-ink"
            >
              <CloseIcon/>
            </button>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
              {t('customFood.eyebrow')}
            </p>
            <h2 className="mt-1 pr-8 text-xl font-bold text-ink">
              {t(food ? 'customFood.editTitle' : 'customFood.createTitle')}
            </h2>
            <p className="mt-1 text-sm text-muted">{t('customFood.labelHint')}</p>

            {!food && (
                <div className="mt-4">
                  <OcrLabelUpload onExtracted={onExtracted}/>
                  {unreadable && <FormError>{t('customFood.scanUnreadable')}</FormError>}
                </div>
            )}

            <div className="mt-5">
              <Field
                  label={t('customFood.nameLabel')}
                  value={values.name}
                  onChange={(e) => set('name', e.target.value)}
                  placeholder={t('customFood.namePlaceholder')}
                  inputClassName={lowConfidence.has('name') ? LOW_CONFIDENCE_CLASS : undefined}
                  hint={lowConfidence.has('name') ? t('customFood.lowConfidenceHint') : undefined}
                  autoFocus
              />
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <Field
                  label={t('customFood.basisLabel')}
                  type="number"
                  inputMode="decimal"
                  min="0.01"
                  step="any"
                  value={values.basis_grams}
                  onChange={(e) => set('basis_grams', e.target.value)}
                  inputClassName={lowConfidence.has('basis_grams') ? LOW_CONFIDENCE_CLASS : undefined}
                  hint={lowConfidence.has('basis_grams') ? t('customFood.lowConfidenceHint') : t('customFood.basisHint')}
              />
              <p className="self-end pb-3 text-sm text-muted">{t('customFood.nutrientHint')}</p>
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              {nutrientFields.map(({key, labelKey, unit}) => (
                  <Field
                      key={key}
                      label={`${t(labelKey)} (${unit})`}
                      type="number"
                      inputMode="decimal"
                      min="0"
                      step="any"
                      value={values[key]}
                      onChange={(e) => set(key, e.target.value)}
                      inputClassName={lowConfidence.has(key) ? LOW_CONFIDENCE_CLASS : undefined}
                      hint={lowConfidence.has(key) ? t('customFood.lowConfidenceHint') : undefined}
                  />
              ))}
            </div>

            {error && <p className="mt-4 text-sm text-accent">{error.message}</p>}
            <div className="mt-6 flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={onClose}>{t('customFood.cancel')}</Button>
              <Button type="submit" disabled={!isComplete || saving}>
                {saving ? t('customFood.saving') : t('customFood.saveChanges')}
              </Button>
            </div>
          </motion.form>
        </motion.div>
    )
  }
}
