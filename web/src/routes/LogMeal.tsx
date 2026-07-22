// Log a meal as natural text. POST is async (202); we show an accepted state
// and let the dashboard/history pick up the result on the next poll.

import { useEffect, useMemo, useState, type SyntheticEvent } from 'react'
import { motion } from 'framer-motion'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  useLogMeal,
  useTemplates,
  useLogTemplate,
  useLogStructuredMeal,
  useFoods,
  useSearchFoods,
  useCatalogSearch,
  useAddServingUnit,
} from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, EmptyState, Spinner } from '@/components/ui'
import { DuplicateMealModal } from '@/components/DuplicateMealModal'
import { FoodCard } from '@/components/FoodCard'
import { CustomFoodModal } from '@/components/CustomFoodModal'
import type { MealTemplate, FoodDetail, FoodServingUnit } from '@/lib/types'
import { GRAMS_UNIT_ID, GENERIC_VOLUME_UNITS, unitOptionsFor, gramsFor, type SelectedFood } from '@/lib/servingUnits'
import { TemplateIcon, CopyIcon, SearchIcon, FoodsIcon, TrashIcon } from '@/components/icons'
import { fadeUp, stagger } from '@/lib/motion'
import { formatNumber, scaleMacros, sumMacros } from '@/lib/format'

const EXAMPLES = ['200g grilled chicken, 2 eggs, 150g rice', '1 banana and a glass of milk', '3 slices of pizza']

export function LogMeal() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  // Pre-fill from a deep link (e.g. "Log this" on a food / frequent-food pill).
  const [text, setText] = useState(() => params.get('text') ?? '')
  const [mode, setMode] = useState<'text' | 'picker'>('text')
  const log = useLogMeal()
  const templates = useTemplates()
  const logTemplate = useLogTemplate()
  const { demo } = useDemo()
  const [duplicating, setDuplicating] = useState(false)

  function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!text.trim()) return
    log.mutate(text.trim(), { onSuccess: () => setText('') })
  }

  const recentTemplates = (templates.data ?? []).slice(0, 6)

  return (
    <div>
      <PageHeader eyebrow={t('logMeal.eyebrow')} title={t('logMeal.title')} />

      <div className="mb-4 flex gap-2">
        {(['text', 'picker'] as const).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => setMode(m)}
            className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition ${
              mode === m
                ? 'border-transparent bg-primary text-white'
                : 'border-line bg-surface text-muted hover:text-ink'
            }`}
          >
            {t(m === 'text' ? 'logMeal.textTab' : 'logMeal.pickerTab')}
          </button>
        ))}
      </div>

      {mode === 'text' ? (
        <Card className="p-5">
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={3}
              placeholder={t('logMeal.placeholder')}
              aria-label={t('logMeal.mealDescriptionAria')}
              className="w-full resize-none rounded-lg border border-line bg-bg px-4 py-3 text-lg text-ink outline-none transition focus:border-primary"
            />
            <div className="flex items-center justify-between gap-3">
              <p className="text-xs text-muted">{t('logMeal.parserHint')}</p>
              <Button type="submit" disabled={log.isPending || !text.trim()}>
                {log.isPending ? t('logMeal.sending') : t('logMeal.logMealButton')}
              </Button>
            </div>
          </form>

          {log.isSuccess && (
            <motion.p
              variants={fadeUp}
              initial="hidden"
              animate="show"
              className="mt-4 rounded-lg bg-primary-soft px-4 py-3 text-sm font-medium text-primary"
            >
              {t('logMeal.loggedSuccess')}
            </motion.p>
          )}
          {log.isError && (
            <p className="mt-4 text-sm font-medium text-accent" role="alert">
              {log.error instanceof Error ? log.error.message : t('logMeal.logFailed')}
            </p>
          )}
        </Card>
      ) : (
        <FoodPicker />
      )}

      {/* Quick actions: log a saved template, or copy a meal from a past day. */}
      <div className="mt-6 flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('logMeal.fromTemplate')}</p>
          <Button variant="ghost" onClick={() => setDuplicating(true)} className="px-3 py-1.5 text-xs">
            <CopyIcon width={15} height={15} /> {t('logMeal.copyFromDay')}
          </Button>
        </div>
        {recentTemplates.length ? (
          <div className="flex flex-wrap gap-2">
            {recentTemplates.map((t: MealTemplate) => (
              <button
                key={t.id}
                type="button"
                disabled={demo || logTemplate.isPending}
                onClick={() => logTemplate.mutate(t.id)}
                className="inline-flex items-center gap-1.5 rounded-full border border-line bg-surface px-3 py-1.5 text-sm text-muted transition hover:text-ink disabled:opacity-50"
              >
                <TemplateIcon width={15} height={15} /> {t.name}
              </button>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted">{t('logMeal.noTemplates')}</p>
        )}
        {logTemplate.isSuccess && (
          <p className="text-sm font-medium text-primary">{t('logMeal.templateLoggedSuccess')}</p>
        )}
      </div>

      {mode === 'text' && (
        <div className="mt-6">
          <p className="mb-2 text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('logMeal.examplesHeading')}</p>
          <div className="flex flex-wrap gap-2">
            {EXAMPLES.map((ex) => (
              <button
                key={ex}
                type="button"
                onClick={() => setText(ex)}
                className="rounded-full border border-line bg-surface px-3 py-1.5 text-sm text-muted transition hover:text-ink"
              >
                {ex}
              </button>
            ))}
          </div>
        </div>
      )}

      {duplicating && <DuplicateMealModal onClose={() => setDuplicating(false)} />}
    </div>
  )
}

// Precise alternative to the free-text parser: search the library/catalog,
// pick exact foods, set grams, log synchronously via POST /meals.
function FoodPicker() {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const [tab, setTab] = useState<'library' | 'catalog'>('library')
  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<SelectedFood[]>([])
  const [customFoodOpen, setCustomFoodOpen] = useState(false)
  const logStructured = useLogStructuredMeal()

  useEffect(() => {
    const id = setTimeout(() => setQuery(rawQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawQuery])

  const searching = query.length > 0
  const search = useSearchFoods(query)
  const browse = useFoods()
  const catalog = useCatalogSearch(query, '', 30)

  const isLoading = tab === 'catalog' ? catalog.isLoading : searching ? search.isLoading : browse.isLoading
  const foods = useMemo(() => {
    if (tab === 'catalog') return catalog.data ?? []
    return (searching ? search.data : browse.data) ?? []
  }, [tab, catalog.data, searching, search.data, browse.data])

  const selectedIds = useMemo(() => new Set(selected.map((s) => s.food.food_id)), [selected])
  const total = sumMacros(selected.map((s) => scaleMacros(s.food.per_100g, gramsFor(s))))

  function addFood(food: FoodDetail) {
    if (selectedIds.has(food.food_id)) return
    // OFF's serving_size is package weight, not a serving (#134) — never
    // used as a default log amount, unlike other sources' real servings.
    const defaultGrams = food.source !== 'openfoodfacts' && food.serving_size > 0 ? food.serving_size : 100
    setSelected((cur) => [...cur, { food, unitID: GRAMS_UNIT_ID, quantity: defaultGrams }])
  }

  function removeFood(foodID: string) {
    setSelected((cur) => cur.filter((s) => s.food.food_id !== foodID))
  }

  function setQuantity(foodID: string, quantity: number) {
    setSelected((cur) => cur.map((s) => (s.food.food_id === foodID ? { ...s, quantity } : s)))
  }

  function setUnit(foodID: string, unitID: string) {
    setSelected((cur) => cur.map((s) => (s.food.food_id === foodID ? { ...s, unitID } : s)))
  }

  // A serving unit just created for foodID rides straight into that item's
  // options (and gets selected) without waiting on the food list to refetch.
  function onUnitCreated(foodID: string, unit: { id: string; label: string; grams: number; custom: boolean }) {
    setSelected((cur) =>
      cur.map((s) =>
        s.food.food_id === foodID
          ? { ...s, unitID: unit.id, food: { ...s.food, serving_units: [...(s.food.serving_units ?? []), unit] } }
          : s,
      ),
    )
  }

  function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    if (!selected.length) return
    logStructured.mutate(
      selected.map((s) => {
        const unit = unitOptionsFor(s.food).find((u) => u.id === s.unitID)
        return {
          food_id: s.food.food_id,
          grams: gramsFor(s),
          unit: unit && unit.id !== GRAMS_UNIT_ID ? unit.label : undefined,
          quantity: unit && unit.id !== GRAMS_UNIT_ID ? s.quantity : undefined,
        }
      }),
      { onSuccess: () => setSelected([]) },
    )
  }

  return (
    <Card className="p-5">
      <div className="mb-4 flex gap-2">
        {(['library', 'catalog'] as const).map((tb) => (
          <button
            key={tb}
            type="button"
            onClick={() => setTab(tb)}
            className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition ${
              tab === tb
                ? 'border-transparent bg-primary text-white'
                : 'border-line bg-surface text-muted hover:text-ink'
            }`}
          >
            {t(tb === 'library' ? 'foods.libraryTab' : 'foods.catalogTab')}
          </button>
        ))}
      </div>

      <div className="relative mb-4">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
          <SearchIcon width={18} height={18} />
        </span>
        <input
          value={rawQuery}
          onChange={(e) => setRawQuery(e.target.value)}
          placeholder={t('foods.searchPlaceholder')}
          aria-label={t('foods.searchAriaLabel')}
          className="w-full rounded-full border border-line bg-surface py-2.5 pl-10 pr-4 text-ink outline-none transition focus:border-primary"
        />
      </div>

      {isLoading ? (
        <Spinner label={t('foods.loadingLabel')} />
      ) : !foods.length ? (
        <EmptyState
          icon={<FoodsIcon />}
          title={tab === 'catalog' ? t('foods.catalogEmptyTitle') : searching ? t('foods.noMatchesTitle') : t('foods.emptyTitle')}
          hint={tab === 'catalog' ? t('foods.catalogEmptyHint') : searching ? t('foods.noMatchesHint') : t('foods.emptyHint')}
        />
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show" className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {foods.map((f: FoodDetail) => (
            <FoodCard key={f.food_id} food={f} onClick={() => addFood(f)} />
          ))}
        </motion.div>
      )}

      <div className="mt-5 flex justify-end">
        <button
          type="button"
          disabled={demo}
          onClick={() => setCustomFoodOpen(true)}
          className="rounded-full border border-line bg-surface px-4 py-2 text-sm font-semibold text-ink transition hover:border-primary disabled:opacity-50"
        >
          {t('foods.addCustom')}
        </button>
      </div>

      <form onSubmit={onSubmit} className="mt-6 border-t border-line pt-5">
        <p className="mb-3 text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('logMeal.selectedHeading')}</p>
        {selected.length ? (
          <ul className="flex flex-col gap-2">
            {selected.map((s) => (
              <SelectedFoodRow
                key={s.food.food_id}
                selected={s}
                onQuantityChange={(q) => setQuantity(s.food.food_id, q)}
                onUnitChange={(u) => setUnit(s.food.food_id, u)}
                onUnitCreated={(unit) => onUnitCreated(s.food.food_id, unit)}
                onRemove={() => removeFood(s.food.food_id)}
              />
            ))}
          </ul>
        ) : (
          <p className="text-sm text-muted">{t('logMeal.emptySelection')}</p>
        )}

        {selected.length > 0 && (
          <div className="mt-3 flex items-center justify-between rounded-lg bg-surface-2 px-3 py-2 text-sm">
            <span className="font-medium text-ink">{t('logMeal.total')}</span>
            <span className="tnum text-muted">
              {formatNumber(total.Calories)} kcal · {formatNumber(total.Protein)}P ·{' '}
              {formatNumber(total.Carbs)}C · {formatNumber(total.Fat)}F
            </span>
          </div>
        )}

        <div className="mt-4 flex items-center justify-between gap-3">
          {logStructured.isSuccess && (
            <p className="text-sm font-medium text-primary">{t('logMeal.pickerLoggedSuccess')}</p>
          )}
          {logStructured.isError && (
            <p className="text-sm font-medium text-accent" role="alert">
              {logStructured.error instanceof Error ? logStructured.error.message : t('logMeal.logFailed')}
            </p>
          )}
          <Button type="submit" className="ml-auto" disabled={!selected.length || logStructured.isPending}>
            {logStructured.isPending ? t('logMeal.sending') : t('logMeal.logMealButton')}
          </Button>
        </div>
      </form>

      {customFoodOpen && (
        <CustomFoodModal
          onClose={() => setCustomFoodOpen(false)}
          onSaved={(food) => {
            setCustomFoodOpen(false)
            addFood(food)
          }}
        />
      )}
    </Card>
  )
}

// One row in the "selected foods" list: name, unit select, quantity input,
// a computed-grams readout when the unit isn't grams, an inline "add unit"
// form (the TACO/no-portion-data escape hatch, #134), and remove.
function SelectedFoodRow({
  selected: s,
  onQuantityChange,
  onUnitChange,
  onUnitCreated,
  onRemove,
}: Readonly<{
  selected: SelectedFood
  onQuantityChange: (quantity: number) => void
  onUnitChange: (unitID: string) => void
  onUnitCreated: (unit: FoodServingUnit) => void
  onRemove: () => void
}>) {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const [addingUnit, setAddingUnit] = useState(false)
  const [unitLabel, setUnitLabel] = useState('')
  const [unitGrams, setUnitGrams] = useState('')
  const addServingUnit = useAddServingUnit(s.food.food_id)
  const options = unitOptionsFor(s.food)

  function submitUnit() {
    const grams = Number(unitGrams)
    if (!unitLabel.trim() || grams <= 0 || demo) return
    addServingUnit.mutate(
      { label: unitLabel.trim(), grams },
      {
        onSuccess: (unit) => {
          onUnitCreated(unit)
          setAddingUnit(false)
          setUnitLabel('')
          setUnitGrams('')
        },
      },
    )
  }

  const itemMacros = scaleMacros(s.food.per_100g, gramsFor(s))

  return (
    <li className="flex flex-col gap-2 border-b border-line pb-2 last:border-0">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          <p className="truncate font-medium text-ink">{s.food.name}</p>
          <p className="tnum text-xs text-muted">{formatNumber(itemMacros.Calories)} kcal</p>
        </div>
        <input
          type="number"
          inputMode="decimal"
          min="0"
          step="any"
          value={s.quantity}
          onChange={(e) => onQuantityChange(Number(e.target.value))}
          aria-label={t('logMeal.quantityAria', { name: s.food.name })}
          className="w-16 rounded-lg border border-line bg-bg px-2 py-1 text-right text-ink tnum outline-none focus:border-primary"
        />
        <select
          value={s.unitID}
          onChange={(e) => onUnitChange(e.target.value)}
          aria-label={t('logMeal.unitAria', { name: s.food.name })}
          className="rounded-lg border border-line bg-bg px-2 py-1 text-sm text-ink outline-none focus:border-primary"
        >
          {options.map((o) => (
            <option key={o.id} value={o.id}>
              {o.id === GRAMS_UNIT_ID || GENERIC_VOLUME_UNITS.some((g) => g.id === o.id)
                ? t(`logMeal.unit.${o.id}`)
                : o.label}
            </option>
          ))}
        </select>
        {s.unitID !== GRAMS_UNIT_ID && (
          <span className="whitespace-nowrap text-xs text-muted tnum">≈ {Math.round(gramsFor(s))}g</span>
        )}
        <button
          type="button"
          onClick={() => setAddingUnit((v) => !v)}
          disabled={demo}
          aria-label={t('logMeal.addUnit')}
          className="text-lg leading-none text-muted transition hover:text-primary disabled:opacity-50"
        >
          +
        </button>
        <button
          type="button"
          onClick={onRemove}
          aria-label={t('mealDetail.removeItem', { name: s.food.name })}
          className="text-muted transition hover:text-accent"
        >
          <TrashIcon width={16} height={16} />
        </button>
      </div>
      {addingUnit && (
        <div className="flex items-center gap-2 pl-1">
          <input
            value={unitLabel}
            onChange={(e) => setUnitLabel(e.target.value)}
            placeholder={t('logMeal.addUnitLabelPlaceholder')}
            className="min-w-0 flex-1 rounded-lg border border-line bg-bg px-2 py-1 text-sm text-ink outline-none focus:border-primary"
          />
          <input
            type="number"
            inputMode="decimal"
            min="0"
            step="any"
            value={unitGrams}
            onChange={(e) => setUnitGrams(e.target.value)}
            placeholder={t('logMeal.addUnitGramsPlaceholder')}
            className="w-24 rounded-lg border border-line bg-bg px-2 py-1 text-sm text-ink outline-none focus:border-primary"
          />
          <button
            type="button"
            onClick={submitUnit}
            disabled={addServingUnit.isPending || !unitLabel.trim() || Number(unitGrams) <= 0}
            className="rounded-lg bg-primary px-3 py-1 text-xs font-semibold text-white disabled:opacity-50"
          >
            {t('logMeal.addUnitSave')}
          </button>
          <button
            type="button"
            onClick={() => setAddingUnit(false)}
            className="text-xs font-medium text-muted hover:text-ink"
          >
            {t('logMeal.addUnitCancel')}
          </button>
        </div>
      )}
    </li>
  )
}
