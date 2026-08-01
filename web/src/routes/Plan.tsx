// Diet plan builder: carb-cycling day-types, their meal slots, and the
// prescribed options for each slot. A slot option is a meal_template
// (owner_kind = "plan") under the hood, so option editing reuses the same
// catalog-search-and-pick machinery LogMeal/ComposeTemplateModal use, with
// two differences: search hits the unscoped catalog (SearchCatalog, not the
// user's personal library) since a nutritionist's prescription routinely
// names foods the user has never logged, and portions are entered in
// food_serving_units ("2 colheres") rather than grams, matching how
// prescriptions are actually written.
//
// The app never computes or suggests a target: day-type macros are typed in
// by the user from the nutritionist's numbers, never derived from the food
// list below them.

import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import {
  usePlans,
  usePlanBundle,
  useActivePlan,
  useCreatePlan,
  useUpdatePlan,
  useDeletePlan,
  useCreateDayType,
  useUpdateDayType,
  useDeleteDayType,
  useCreateSlot,
  useUpdateSlot,
  useDeleteSlot,
  useCreateSlotOption,
  useUpdateSlotOption,
  useDeleteSlotOption,
  useTemplate,
  useCatalogSearch,
} from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { ImportPlanCard } from './PlanImport'
import { PageHeader } from '@/components/PageHeader'
import { Card, Button, Pill, Spinner, EmptyState, Toggle, Field } from '@/components/ui'
import { GoalIcon, TrashIcon, CopyIcon, ChevronLeft, SearchIcon, CheckIcon } from '@/components/icons'
import { sourceLabel } from '@/components/FoodCard'
import { fadeUp } from '@/lib/motion'
import { formatNumber, scaleMacros, sumMacros } from '@/lib/format'
import {
  GRAMS_UNIT_ID,
  unitOptionsFor,
  gramsFor,
  type SelectedFood,
} from '@/lib/servingUnits'
import { MACRO_KEYS } from '@/lib/types'
import type {
  DietPlan,
  DietPlanDayTypeBundle,
  DietPlanSlotBundle,
  DietPlanSlotOption,
  DietPlanSlotOptionInput,
  FoodDetail,
  FoodMatch,
  Macros,
  ResolvedItem,
} from '@/lib/types'

const ZERO_MACROS: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

// The common case (a 7-day pattern) is anchored on a Monday; if today is
// already Monday this returns today, otherwise the coming one.
export function nextMondayISO(): string {
  const d = new Date()
  d.setDate(d.getDate() + ((1 - d.getDay() + 7) % 7))
  return d.toISOString().slice(0, 10)
}

// Companion to nextMondayISO for anchoring a schedule that starts "now"
// (e.g. an imported weekday grid, #223): the most recent Monday on or
// before today, today counting as itself when today is Monday.
export function mostRecentMondayISO(): string {
  const d = new Date()
  d.setDate(d.getDate() - ((d.getDay() + 6) % 7))
  return d.toISOString().slice(0, 10)
}

function byPosition<T extends { position: number }>(list: T[]): T[] {
  return [...list].sort((a, b) => a.position - b.position)
}

// --- FoodDetail <-> FoodMatch, so a saved item's serving units survive a
// re-open of its slot option (see the FoodMatch comment in lib/types.ts).
function toFoodMatch(food: FoodDetail): FoodMatch {
  return {
    FoodID: food.food_id,
    Name: food.name,
    Source: food.source,
    Per100g: food.per_100g,
    MatchScore: 1,
    Category: food.category,
    Brand: food.brand,
    Barcode: food.barcode,
    ImageURL: food.image_url,
    ServingSize: food.serving_size,
    ServingUnit: food.serving_unit,
    ServingUnits: food.serving_units,
  }
}

function matchToFoodDetail(m: FoodMatch): FoodDetail {
  return {
    food_id: m.FoodID,
    name: m.Name,
    source: m.Source,
    per_100g: m.Per100g,
    category: m.Category ?? '',
    brand: m.Brand ?? '',
    barcode: m.Barcode ?? '',
    image_url: m.ImageURL ?? '',
    serving_size: m.ServingSize ?? 0,
    serving_unit: m.ServingUnit ?? '',
    query_count: 0,
    last_used: '',
    in_library: false,
    serving_units: m.ServingUnits ?? [],
    volume_units_eligible: false,
  }
}

// Ad libitum ("à vontade") is purely quantity/grams === 0 — no stored flag
// (trap #11). LocalItem below tracks it as editor-only UI state and
// collapses back to a zero amount at save time. id is a client-only, never
// persisted, so item rows have a React key stable across reorders/edits.
export type LocalItem = SelectedFood & { adLibitum: boolean; id: string }

function isAdLibitum(item: ResolvedItem): boolean {
  return item.Parsed.NormalizedGrams === 0
}

function fromResolvedItem(item: ResolvedItem): LocalItem {
  const adLibitum = isAdLibitum(item)
  const food = matchToFoodDetail(item.Match)
  const matchedUnit = (food.serving_units ?? []).find((u) => u.label === item.Parsed.Unit)
  let quantity: number
  if (adLibitum) {
    quantity = 1
  } else if (matchedUnit) {
    quantity = item.Parsed.Quantity
  } else {
    quantity = item.Parsed.NormalizedGrams
  }
  return {
    food,
    unitID: matchedUnit ? matchedUnit.id : GRAMS_UNIT_ID,
    quantity,
    adLibitum,
    id: crypto.randomUUID(),
  }
}

export function toResolvedItem(li: LocalItem): ResolvedItem {
  const grams = li.adLibitum ? 0 : gramsFor(li)
  const unit = unitOptionsFor(li.food).find((u) => u.id === li.unitID)
  const namedUnit = unit && unit.id !== GRAMS_UNIT_ID ? unit.label : ''
  return {
    Parsed: {
      RawPhrase: li.food.name,
      Quantity: li.adLibitum ? 0 : li.quantity,
      Unit: namedUnit,
      NormalizedGrams: grams,
      Locale: '',
    },
    Match: toFoodMatch(li.food),
    Macros: li.adLibitum ? ZERO_MACROS : scaleMacros(li.food.per_100g, grams),
  }
}

// ---------------------------------------------------------------------------
// Clone helpers: "duplicate day-type / duplicate slot / copy option" are the
// feature this whole issue exists for (a real plan is ~80 items; clone-and-
// tweak is what makes entering that survivable). Each is a short sequence of
// the same granular create calls a user would make by hand, fired in order.
// ---------------------------------------------------------------------------

function usePlanClone(planID: string) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [busy, setBusy] = useState<Set<string>>(new Set())

  async function run(id: string, fn: () => Promise<void>) {
    setBusy((s) => new Set(s).add(id))
    try {
      await fn()
      await qc.invalidateQueries({ queryKey: ['plan'] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('plan.cloneFailed'))
    } finally {
      setBusy((s) => {
        const next = new Set(s)
        next.delete(id)
        return next
      })
    }
  }

  async function cloneOptionsInto(dayTypeID: string, slotID: string, options: DietPlanSlotOption[]) {
    for (const opt of byPosition(options)) {
      const tmpl = await api.templates.get(opt.template_id)
      await api.plans.options.create(planID, dayTypeID, slotID, {
        label: opt.label,
        position: opt.position,
        items: tmpl.items,
      })
    }
  }

  async function cloneSlotInto(dayTypeID: string, slot: DietPlanSlotBundle, position?: number) {
    const newSlot = await api.plans.slots.create(planID, dayTypeID, {
      label: slot.label,
      time_of_day: slot.time_of_day,
      position: position ?? slot.position,
    })
    await cloneOptionsInto(dayTypeID, newSlot.id, slot.options)
  }

  function cloneDayType(dt: DietPlanDayTypeBundle, position: number) {
    return run(`dt:${dt.id}`, async () => {
      const newDT = await api.plans.dayTypes.create(planID, {
        name: t('plan.copyOf', { name: dt.name }),
        position,
        targets: dt.targets,
        water_goal_ml: dt.water_goal_ml,
      })
      for (const slot of byPosition(dt.slots)) {
        await cloneSlotInto(newDT.id, slot)
      }
    })
  }

  function cloneSlot(dayTypeID: string, slot: DietPlanSlotBundle, position: number) {
    return run(`slot:${slot.id}`, () => cloneSlotInto(dayTypeID, slot, position))
  }

  function copyOption(dayTypeID: string, slotID: string, opt: DietPlanSlotOption, position: number) {
    return run(`opt:${opt.id}`, async () => {
      const tmpl = await api.templates.get(opt.template_id)
      await api.plans.options.create(planID, dayTypeID, slotID, {
        label: t('plan.copyOf', { name: opt.label }),
        position,
        items: tmpl.items,
      })
    })
  }

  return { cloneDayType, cloneSlot, copyOption, isBusy: (id: string) => busy.has(id) }
}

// ---------------------------------------------------------------------------
// Top level: pick or create a plan, then edit it.
// ---------------------------------------------------------------------------

export function Plan() {
  const { t } = useTranslation()
  const { demo } = useDemo()
  const plans = usePlans()
  const [selectedID, setSelectedID] = useState<string | null>(null)

  return (
    <div>
      <PageHeader eyebrow={t('plan.eyebrow')} title={t('plan.title')}>
        {selectedID && (
          <Button variant="ghost" onClick={() => setSelectedID(null)}>
            <ChevronLeft width={16} height={16} /> {t('plan.backToList')}
          </Button>
        )}
      </PageHeader>

      {demo && (
        <p className="mb-5 rounded-lg bg-surface-2 px-4 py-2 text-sm text-muted">{t('plan.demoNotice')}</p>
      )}

      {selectedID ? (
        <PlanBuilder planID={selectedID} onDeleted={() => setSelectedID(null)} />
      ) : (
        <PlanList
          loading={plans.isLoading}
          plans={plans.data ?? []}
          demo={demo}
          onSelect={setSelectedID}
        />
      )}
    </div>
  )
}

function PlanList({
  loading,
  plans,
  demo,
  onSelect,
}: Readonly<{ loading: boolean; plans: DietPlan[]; demo: boolean; onSelect: (id: string) => void }>) {
  return (
    <motion.div variants={fadeUp} initial="hidden" animate="show" className="space-y-5">
      {!demo && <ImportPlanCard onCreated={onSelect} />}
      {!demo && <NewPlanCard onCreated={onSelect} />}
      <PlanListBody loading={loading} plans={plans} onSelect={onSelect} />
    </motion.div>
  )
}

function PlanListBody({
  loading,
  plans,
  onSelect,
}: Readonly<{ loading: boolean; plans: DietPlan[]; onSelect: (id: string) => void }>) {
  const { t } = useTranslation()
  const active = useActivePlan()

  if (loading) return <Spinner label={t('plan.loading')} />
  if (plans.length === 0) {
    return <EmptyState icon={<GoalIcon width={28} height={28} />} title={t('plan.noPlansTitle')} hint={t('plan.noPlansHint')} />
  }
  return (
    <div className="flex flex-col gap-2.5">
      {plans.map((p) => (
        <Card key={p.id} className="p-4">
          <button
            type="button"
            onClick={() => onSelect(p.id)}
            className="flex w-full items-center justify-between gap-3 text-left"
          >
            <div className="min-w-0">
              <p className="flex items-center gap-2 truncate font-semibold text-ink">
                {p.name}
                {active.data?.id === p.id && <Pill tone="primary">{t('plan.activePill')}</Pill>}
              </p>
              <p className="mt-0.5 truncate text-sm text-muted">
                {p.valid_from} → {p.valid_to || t('plan.openEnded')}
              </p>
            </div>
          </button>
        </Card>
      ))}
    </div>
  )
}

function NewPlanCard({ onCreated }: Readonly<{ onCreated: (id: string) => void }>) {
  const { t } = useTranslation()
  const create = useCreatePlan()
  const [name, setName] = useState('')
  const [notes, setNotes] = useState('')
  const [validFrom, setValidFrom] = useState(todayISO())
  const [validTo, setValidTo] = useState('')
  const [anchor, setAnchor] = useState(nextMondayISO())

  function submit() {
    if (!name.trim()) return
    create.mutate(
      { name: name.trim(), notes, valid_from: validFrom, valid_to: validTo, cycle_anchor_date: anchor },
      { onSuccess: (created) => onCreated(created.id) },
    )
  }

  return (
    <Card className="p-5">
      <p className="mb-4 font-semibold text-ink">{t('plan.newPlanTitle')}</p>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t('plan.nameLabel')} value={name} onChange={(e) => setName(e.target.value)} placeholder={t('plan.namePlaceholder')} />
        <Field label={t('plan.notesLabel')} value={notes} onChange={(e) => setNotes(e.target.value)} placeholder={t('plan.notesPlaceholder')} />
        <Field label={t('plan.validFromLabel')} type="date" value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
        <Field label={t('plan.validToLabel')} type="date" value={validTo} onChange={(e) => setValidTo(e.target.value)} hint={t('plan.validToHint')} />
        <Field label={t('plan.anchorLabel')} type="date" value={anchor} onChange={(e) => setAnchor(e.target.value)} hint={t('plan.anchorHint')} />
      </div>
      <div className="mt-4 flex justify-end">
        <Button onClick={submit} disabled={!name.trim() || create.isPending}>
          {create.isPending ? t('plan.creating') : t('plan.createButton')}
        </Button>
      </div>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// One plan: meta, cycle, day-types.
// ---------------------------------------------------------------------------

function PlanBuilder({ planID, onDeleted }: Readonly<{ planID: string; onDeleted: () => void }>) {
  const { t } = useTranslation()
  const bundle = usePlanBundle(planID)
  const del = useDeletePlan()
  const [confirmDelete, setConfirmDelete] = useState(false)

  if (bundle.isLoading) return <Spinner label={t('plan.loading')} />
  if (!bundle.data) return <EmptyState title={t('plan.loadFailed')} />

  const { plan, day_types: dayTypes } = bundle.data
  const sortedDayTypes = byPosition(dayTypes)

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-ink">{plan.name}</h2>
        {confirmDelete ? (
          <div className="flex gap-2">
            <Button
              onClick={() => del.mutate(planID, { onSuccess: onDeleted })}
              disabled={del.isPending}
              className="bg-accent text-white hover:brightness-105"
            >
              {del.isPending ? t('plan.deleting') : t('plan.deleteConfirm')}
            </Button>
            <Button variant="ghost" onClick={() => setConfirmDelete(false)}>
              {t('plan.cancel')}
            </Button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setConfirmDelete(true)}
            aria-label={t('plan.delete')}
            className="grid size-9 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
          >
            <TrashIcon width={17} height={17} />
          </button>
        )}
      </div>

      <PlanMetaForm key={plan.id} plan={plan} />
      <CycleEditor plan={plan} dayTypes={sortedDayTypes} />
      <DayTypesSection planID={planID} dayTypes={sortedDayTypes} />
    </div>
  )
}

function PlanMetaForm({ plan }: Readonly<{ plan: DietPlan }>) {
  const { t } = useTranslation()
  const update = useUpdatePlan()
  const [name, setName] = useState(plan.name)
  const [notes, setNotes] = useState(plan.notes)
  const [validFrom, setValidFrom] = useState(plan.valid_from)
  const [validTo, setValidTo] = useState(plan.valid_to)

  function save() {
    if (!name.trim()) return
    update.mutate({ planID: plan.id, input: { ...plan, name: name.trim(), notes, valid_from: validFrom, valid_to: validTo } })
  }

  return (
    <Card className="p-5">
      <p className="mb-4 text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.planMetaTitle')}</p>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={t('plan.nameLabel')} value={name} onChange={(e) => setName(e.target.value)} />
        <Field label={t('plan.notesLabel')} value={notes} onChange={(e) => setNotes(e.target.value)} />
        <Field label={t('plan.validFromLabel')} type="date" value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
        <Field label={t('plan.validToLabel')} type="date" value={validTo} onChange={(e) => setValidTo(e.target.value)} hint={t('plan.validToHint')} />
      </div>
      <div className="mt-4 flex justify-end">
        <Button onClick={save} disabled={!name.trim() || update.isPending}>
          {update.isPending ? t('plan.saving') : t('plan.saveButton')}
        </Button>
      </div>
    </Card>
  )
}

// Every change here applies immediately against the live `plan` prop (no
// local dirty buffer), so there's no staleness risk from PUT /plans/{id}
// replacing cycle_pattern wholesale — see the comment on DietPlan in
// lib/types.ts and the PUT semantics in handler_plan.go.
function CycleEditor({
  plan,
  dayTypes,
}: Readonly<{ plan: DietPlan; dayTypes: DietPlanDayTypeBundle[] }>) {
  const { t } = useTranslation()
  const update = useUpdatePlan()
  // cycle_pattern is a plain string[] that can repeat the same day-type id at
  // multiple positions, so it has no natural per-row id; rowKeys tracks a
  // synthetic one per position, kept in lockstep with every mutation below,
  // so list rows get a React key stable across reorders instead of the
  // array index.
  const [rowKeys, setRowKeys] = useState<string[]>(() => plan.cycle_pattern.map(() => crypto.randomUUID()))

  function apply(patch: Partial<DietPlan>) {
    update.mutate({ planID: plan.id, input: { ...plan, ...patch } })
  }

  function setAt(i: number, dayTypeID: string) {
    const next = [...plan.cycle_pattern]
    next[i] = dayTypeID
    apply({ cycle_pattern: next })
  }

  function addPosition() {
    apply({ cycle_pattern: [...plan.cycle_pattern, dayTypes[0]?.id ?? ''] })
    setRowKeys((keys) => [...keys, crypto.randomUUID()])
  }

  function removeAt(i: number) {
    apply({ cycle_pattern: plan.cycle_pattern.filter((_, idx) => idx !== i) })
    setRowKeys((keys) => keys.filter((_, idx) => idx !== i))
  }

  function seedWeek() {
    const pattern = new Array(7).fill(dayTypes[0]?.id ?? '')
    apply({ cycle_pattern: pattern, cycle_anchor_date: nextMondayISO() })
    setRowKeys(pattern.map(() => crypto.randomUUID()))
  }

  return (
    <Card className="p-5">
      <div className="mb-1 flex items-center justify-between">
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.cycleTitle')}</p>
        {dayTypes.length > 0 && (
          <Button variant="ghost" onClick={seedWeek} className="px-3 py-1.5 text-xs">
            {t('plan.seedWeek')}
          </Button>
        )}
      </div>
      <p className="mb-4 text-sm text-muted">{t('plan.cycleHint')}</p>

      {dayTypes.length === 0 ? (
        <p className="text-sm text-muted">{t('plan.cycleNeedsDayTypes')}</p>
      ) : (
        <>
          <Field
            label={t('plan.anchorLabel')}
            type="date"
            value={plan.cycle_anchor_date}
            onChange={(e) => apply({ cycle_anchor_date: e.target.value })}
            hint={t('plan.anchorHint')}
            className="mb-4 max-w-xs"
          />
          <ol className="flex flex-col gap-2">
            {plan.cycle_pattern.map((dayTypeID, i) => (
              // Position in the array is the identity here (cycle_pattern
              // can repeat the same day-type id at multiple positions), so
              // the key is the synthetic per-position id in rowKeys, not i.
              <li key={rowKeys[i]} className="flex items-center gap-2">
                <span className="w-6 shrink-0 text-xs font-semibold text-muted tnum">{i + 1}</span>
                <select
                  value={dayTypeID}
                  onChange={(e) => setAt(i, e.target.value)}
                  className="flex-1 rounded-lg border border-line bg-bg px-3 py-2 text-sm text-ink outline-none focus:border-primary"
                >
                  {dayTypes.map((dt) => (
                    <option key={dt.id} value={dt.id}>
                      {dt.name}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={() => removeAt(i)}
                  aria-label={t('plan.removePositionAria', { position: i + 1 })}
                  className="grid size-8 shrink-0 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
                >
                  <TrashIcon width={14} height={14} />
                </button>
              </li>
            ))}
          </ol>
          <Button variant="ghost" onClick={addPosition} className="mt-3 px-3 py-1.5 text-xs">
            + {t('plan.addPosition')}
          </Button>
        </>
      )}
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Day-types
// ---------------------------------------------------------------------------

function DayTypesSection({
  planID,
  dayTypes,
}: Readonly<{ planID: string; dayTypes: DietPlanDayTypeBundle[] }>) {
  const { t } = useTranslation()
  const create = useCreateDayType(planID)
  const clone = usePlanClone(planID)
  const nextPosition = dayTypes.length ? Math.max(...dayTypes.map((d) => d.position)) + 1 : 0

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.dayTypesTitle')}</p>
        <Button
          variant="ghost"
          onClick={() =>
            create.mutate({ name: t('plan.newDayTypeName'), position: nextPosition, targets: ZERO_MACROS, water_goal_ml: 0 })
          }
          disabled={create.isPending}
          className="px-3 py-1.5 text-xs"
        >
          + {t('plan.addDayType')}
        </Button>
      </div>

      {dayTypes.length === 0 ? (
        <EmptyState title={t('plan.noDayTypesTitle')} hint={t('plan.noDayTypesHint')} />
      ) : (
        dayTypes.map((dt) => (
          <DayTypeCard
            key={dt.id}
            planID={planID}
            dayType={dt}
            onDuplicate={() => clone.cloneDayType(dt, nextPosition)}
            duplicating={clone.isBusy(`dt:${dt.id}`)}
          />
        ))
      )}
    </div>
  )
}

function DayTypeCard({
  planID,
  dayType,
  onDuplicate,
  duplicating,
}: Readonly<{
  planID: string
  dayType: DietPlanDayTypeBundle
  onDuplicate: () => void
  duplicating: boolean
}>) {
  const { t } = useTranslation()
  const update = useUpdateDayType(planID)
  const del = useDeleteDayType(planID)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [name, setName] = useState(dayType.name)
  const [waterGoal, setWaterGoal] = useState(dayType.water_goal_ml)
  const [targets, setTargets] = useState<Macros>(dayType.targets)

  function save() {
    if (!name.trim()) return
    update.mutate({ ...dayType, name: name.trim(), water_goal_ml: waterGoal, targets })
  }

  return (
    <Card className="p-5">
      <div className="mb-4 flex items-start justify-between gap-3">
        <Field
          label={t('plan.dayTypeNameLabel')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="max-w-xs flex-1"
        />
        <div className="flex shrink-0 items-center gap-1 pt-6">
          <button
            type="button"
            onClick={onDuplicate}
            disabled={duplicating}
            aria-label={t('plan.duplicateDayType')}
            title={t('plan.duplicateDayType')}
            className="grid size-9 place-items-center rounded-full text-muted transition hover:bg-primary-soft hover:text-primary disabled:opacity-50"
          >
            <CopyIcon width={16} height={16} />
          </button>
          {confirmDelete ? (
            <>
              <Button
                onClick={() => del.mutate(dayType.id)}
                disabled={del.isPending}
                className="bg-accent px-3 py-1.5 text-xs text-white hover:brightness-105"
              >
                {t('plan.deleteConfirm')}
              </Button>
              <Button variant="ghost" onClick={() => setConfirmDelete(false)} className="px-3 py-1.5 text-xs">
                {t('plan.cancel')}
              </Button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => setConfirmDelete(true)}
              aria-label={t('plan.delete')}
              className="grid size-9 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
            >
              <TrashIcon width={16} height={16} />
            </button>
          )}
        </div>
      </div>

      <p className="mb-2 text-xs font-medium text-muted">{t('plan.dayTypeTargetsHint')}</p>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        {MACRO_KEYS.map((k) => (
          <label key={k} className="block">
            <span className="mb-1 block text-xs font-medium text-muted">{t(`common.macro.${k}`)}</span>
            <input
              type="number"
              inputMode="decimal"
              min="0"
              step="any"
              value={targets[k]}
              onChange={(e) => setTargets((m) => ({ ...m, [k]: Number(e.target.value) || 0 }))}
              className="w-full rounded-lg border border-line bg-bg px-2 py-1.5 text-sm text-ink outline-none focus:border-primary tnum"
            />
          </label>
        ))}
      </div>

      <Field
        label={t('plan.waterGoalLabel')}
        type="number"
        min={0}
        value={waterGoal}
        onChange={(e) => setWaterGoal(Number(e.target.value) || 0)}
        className="mt-3 max-w-[10rem]"
      />

      <div className="mt-4 flex justify-end">
        <Button onClick={save} disabled={!name.trim() || update.isPending} className="px-4 py-2 text-sm">
          {update.isPending ? t('plan.saving') : t('plan.saveButton')}
        </Button>
      </div>

      <div className="mt-5 border-t border-line pt-4">
        <SlotsSection planID={planID} dayTypeID={dayType.id} slots={byPosition(dayType.slots)} />
      </div>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Slots
// ---------------------------------------------------------------------------

function SlotsSection({
  planID,
  dayTypeID,
  slots,
}: Readonly<{ planID: string; dayTypeID: string; slots: DietPlanSlotBundle[] }>) {
  const { t } = useTranslation()
  const create = useCreateSlot(planID, dayTypeID)
  const clone = usePlanClone(planID)
  const nextPosition = slots.length ? Math.max(...slots.map((s) => s.position)) + 1 : 0

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.slotsTitle')}</p>
        <Button
          variant="ghost"
          onClick={() => create.mutate({ label: t('plan.newSlotLabel'), position: nextPosition, time_of_day: '' })}
          disabled={create.isPending}
          className="px-3 py-1.5 text-xs"
        >
          + {t('plan.addSlot')}
        </Button>
      </div>

      {slots.length === 0 ? (
        <p className="text-sm text-muted">{t('plan.noSlots')}</p>
      ) : (
        slots.map((slot) => (
          <SlotCard
            key={slot.id}
            planID={planID}
            dayTypeID={dayTypeID}
            slot={slot}
            onDuplicate={() => clone.cloneSlot(dayTypeID, slot, nextPosition)}
            duplicating={clone.isBusy(`slot:${slot.id}`)}
          />
        ))
      )}
    </div>
  )
}

function SlotCard({
  planID,
  dayTypeID,
  slot,
  onDuplicate,
  duplicating,
}: Readonly<{
  planID: string
  dayTypeID: string
  slot: DietPlanSlotBundle
  onDuplicate: () => void
  duplicating: boolean
}>) {
  const { t } = useTranslation()
  const update = useUpdateSlot(planID, dayTypeID)
  const del = useDeleteSlot(planID, dayTypeID)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [label, setLabel] = useState(slot.label)
  const [timeOfDay, setTimeOfDay] = useState(slot.time_of_day)

  function save() {
    if (!label.trim()) return
    update.mutate({ ...slot, label: label.trim(), time_of_day: timeOfDay })
  }

  return (
    <div className="rounded-lg border border-line bg-surface-2/50 p-4">
      <div className="mb-3 flex items-end gap-2">
        <Field label={t('plan.slotLabelLabel')} value={label} onChange={(e) => setLabel(e.target.value)} className="flex-1" />
        <Field label={t('plan.timeOfDayLabel')} type="time" value={timeOfDay} onChange={(e) => setTimeOfDay(e.target.value)} className="w-32 shrink-0" />
        <Button onClick={save} disabled={!label.trim() || update.isPending} className="px-3 py-2 text-xs">
          {t('plan.saveButton')}
        </Button>
        <button
          type="button"
          onClick={onDuplicate}
          disabled={duplicating}
          aria-label={t('plan.duplicateSlot')}
          title={t('plan.duplicateSlot')}
          className="grid size-9 shrink-0 place-items-center rounded-full text-muted transition hover:bg-primary-soft hover:text-primary disabled:opacity-50"
        >
          <CopyIcon width={15} height={15} />
        </button>
        {confirmDelete ? (
          <>
            <Button onClick={() => del.mutate(slot.id)} disabled={del.isPending} className="bg-accent px-3 py-2 text-xs text-white hover:brightness-105">
              {t('plan.deleteConfirm')}
            </Button>
            <Button variant="ghost" onClick={() => setConfirmDelete(false)} className="px-3 py-2 text-xs">
              {t('plan.cancel')}
            </Button>
          </>
        ) : (
          <button
            type="button"
            onClick={() => setConfirmDelete(true)}
            aria-label={t('plan.delete')}
            className="grid size-9 shrink-0 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
          >
            <TrashIcon width={15} height={15} />
          </button>
        )}
      </div>

      <OptionsSection planID={planID} dayTypeID={dayTypeID} slotID={slot.id} options={byPosition(slot.options)} />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Slot options (each backs a plan-owned meal_template)
// ---------------------------------------------------------------------------

function OptionsSection({
  planID,
  dayTypeID,
  slotID,
  options,
}: Readonly<{ planID: string; dayTypeID: string; slotID: string; options: DietPlanSlotOption[] }>) {
  const { t } = useTranslation()
  const clone = usePlanClone(planID)
  const [adding, setAdding] = useState(false)
  const nextPosition = options.length ? Math.max(...options.map((o) => o.position)) + 1 : 0

  return (
    <div className="space-y-2 border-t border-line pt-3">
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.optionsTitle')}</p>

      {options.map((opt) => (
        <OptionRow
          key={opt.id}
          planID={planID}
          dayTypeID={dayTypeID}
          slotID={slotID}
          option={opt}
          onCopy={() => clone.copyOption(dayTypeID, slotID, opt, nextPosition)}
          copying={clone.isBusy(`opt:${opt.id}`)}
        />
      ))}

      {adding ? (
        <OptionEditor
          planID={planID}
          dayTypeID={dayTypeID}
          slotID={slotID}
          option={null}
          position={nextPosition}
          onClose={() => setAdding(false)}
        />
      ) : (
        <Button variant="ghost" onClick={() => setAdding(true)} className="px-3 py-1.5 text-xs">
          + {t('plan.addOption')}
        </Button>
      )}
    </div>
  )
}

// optionSummaryText builds the "3 items · 420 kcal · ad libitum" line below
// an option's label, flattened into named intermediates so neither ternary
// nor template literal ends up nested inside another.
function optionSummaryText(t: (key: string) => string, itemCount: number, kcal: number, hasAdLibitum: boolean): string {
  const itemWord = itemCount === 1 ? t('templates.item') : t('templates.items')
  const adLibitumSuffix = hasAdLibitum ? ` · ${t('plan.adLibitum')}` : ''
  return `${itemCount} ${itemWord} · ${formatNumber(kcal)} kcal${adLibitumSuffix}`
}

function OptionRow({
  planID,
  dayTypeID,
  slotID,
  option,
  onCopy,
  copying,
}: Readonly<{
  planID: string
  dayTypeID: string
  slotID: string
  option: DietPlanSlotOption
  onCopy: () => void
  copying: boolean
}>) {
  const { t } = useTranslation()
  const tmpl = useTemplate(option.template_id)
  const del = useDeleteSlotOption(planID, dayTypeID, slotID)
  const [editing, setEditing] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const items = tmpl.data?.items ?? []
  const named = items.filter((it) => !isAdLibitum(it))
  const kcal = sumMacros(named.map((it) => it.Macros)).Calories
  const hasAdLibitum = items.some(isAdLibitum)

  if (editing) {
    return (
      <OptionEditor
        planID={planID}
        dayTypeID={dayTypeID}
        slotID={slotID}
        option={option}
        position={option.position}
        onClose={() => setEditing(false)}
      />
    )
  }

  return (
    <div className="flex items-center gap-2 rounded-lg border border-line bg-surface px-3 py-2">
      <button type="button" onClick={() => setEditing(true)} className="min-w-0 flex-1 text-left">
        <p className="truncate text-sm font-medium text-ink">{option.label}</p>
        <p className="text-xs text-muted">
          {tmpl.isLoading ? t('plan.loading') : optionSummaryText(t, items.length, kcal, hasAdLibitum)}
        </p>
      </button>
      <button
        type="button"
        onClick={onCopy}
        disabled={copying}
        aria-label={t('plan.copyOption')}
        title={t('plan.copyOption')}
        className="grid size-8 shrink-0 place-items-center rounded-full text-muted transition hover:bg-primary-soft hover:text-primary disabled:opacity-50"
      >
        <CopyIcon width={14} height={14} />
      </button>
      {confirmDelete ? (
        <>
          <Button onClick={() => del.mutate(option.id)} disabled={del.isPending} className="bg-accent px-3 py-1.5 text-xs text-white hover:brightness-105">
            {t('plan.deleteConfirm')}
          </Button>
          <Button variant="ghost" onClick={() => setConfirmDelete(false)} className="px-3 py-1.5 text-xs">
            {t('plan.cancel')}
          </Button>
        </>
      ) : (
        <button
          type="button"
          onClick={() => setConfirmDelete(true)}
          aria-label={t('plan.delete')}
          className="grid size-8 shrink-0 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
        >
          <TrashIcon width={14} height={14} />
        </button>
      )}
    </div>
  )
}

// Creates (option=null) or edits an existing slot option: a label plus an
// item list built from catalog search, portions entered in serving units
// (falls back to grams for foods with none), and an ad libitum toggle per
// item.
// ItemSearchResults renders the catalog-search dropdown for OptionEditor:
// loading, empty, or the match list, as plain if/else rather than a chain of
// ternaries.
export function ItemSearchResults({
  search,
  onPick,
}: Readonly<{ search: ReturnType<typeof useCatalogSearch>; onPick: (food: FoodDetail) => void }>) {
  const { t } = useTranslation()

  if (search.isLoading) return <li className="px-3 py-2 text-sm text-muted">{t('plan.itemSearching')}</li>
  if (!search.data?.length) return <li className="px-3 py-2 text-sm text-muted">{t('plan.itemNoMatches')}</li>

  return (
    <>
      {search.data.map((f) => (
        <li key={f.food_id}>
          <button
            type="button"
            onClick={() => onPick(f)}
            className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm text-ink hover:bg-surface-2"
          >
            <span className="truncate">{f.name}</span>
            <span className="shrink-0 text-xs text-muted">
              {sourceLabel(f.source, t)} · {formatNumber(f.per_100g.Calories)} kcal/100g
            </span>
          </button>
        </li>
      ))}
    </>
  )
}

function OptionEditor({
  planID,
  dayTypeID,
  slotID,
  option,
  position,
  onClose,
}: Readonly<{
  planID: string
  dayTypeID: string
  slotID: string
  option: DietPlanSlotOption | null
  position: number
  onClose: () => void
}>) {
  const { t } = useTranslation()
  const isCreate = option === null
  const tmpl = useTemplate(option?.template_id)
  const create = useCreateSlotOption(planID, dayTypeID, slotID)
  const update = useUpdateSlotOption(planID, dayTypeID, slotID)

  const [label, setLabel] = useState(option?.label ?? '')
  const [items, setItems] = useState<LocalItem[]>([])
  const [seeded, setSeeded] = useState(isCreate) // create mode starts "seeded" (empty on purpose)

  // Adjust state once the existing option's template loads — the documented
  // "adjust state during render" pattern, not an effect, so the item list
  // never flashes empty-then-populated after the request resolves.
  if (!seeded && tmpl.data) {
    setItems(tmpl.data.items.map(fromResolvedItem))
    setSeeded(true)
  }

  const [rawQuery, setRawQuery] = useState('')
  const [query, setQuery] = useState('')
  useEffect(() => {
    const id = setTimeout(() => setQuery(rawQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawQuery])
  const search = useCatalogSearch(query)

  const pending = create.isPending || update.isPending
  const error = create.error ?? update.error
  const total = sumMacros(items.filter((it) => !it.adLibitum).map((it) => scaleMacros(it.food.per_100g, gramsFor(it))))

  function addFood(food: FoodDetail) {
    setItems((cur) => [...cur, { food, unitID: GRAMS_UNIT_ID, quantity: 100, adLibitum: false, id: crypto.randomUUID() }])
    setRawQuery('')
    setQuery('')
  }

  function updateItem(i: number, patch: Partial<LocalItem>) {
    setItems((cur) => cur.map((it, idx) => (idx === i ? { ...it, ...patch } : it)))
  }

  function removeItem(i: number) {
    setItems((cur) => cur.filter((_, idx) => idx !== i))
  }

  function submit() {
    if (!label.trim() || !items.length) return
    const input: DietPlanSlotOptionInput = { label: label.trim(), position, items: items.map(toResolvedItem) }
    if (option) update.mutate({ optionID: option.id, input }, { onSuccess: onClose })
    else create.mutate(input, { onSuccess: onClose })
  }

  if (!isCreate && tmpl.isLoading) return <Spinner label={t('plan.loading')} />

  return (
    <div className="rounded-lg border border-primary/30 bg-bg p-4">
      <Field
        label={t('plan.optionLabelLabel')}
        value={label}
        onChange={(e) => setLabel(e.target.value)}
        placeholder={t('plan.optionLabelPlaceholder')}
        className="mb-3"
      />

      <div className="relative mb-2">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
          <SearchIcon width={15} height={15} />
        </span>
        <input
          value={rawQuery}
          onChange={(e) => setRawQuery(e.target.value)}
          placeholder={t('plan.itemSearchPlaceholder')}
          aria-label={t('plan.itemSearchPlaceholder')}
          className="w-full rounded-lg border border-line bg-surface py-2 pl-9 pr-3 text-sm text-ink outline-none focus:border-primary"
        />
      </div>

      {query.length > 0 && (
        <ul className="mb-3 max-h-40 divide-y divide-line overflow-y-auto rounded-lg border border-line">
          <ItemSearchResults search={search} onPick={addFood} />
        </ul>
      )}

      <ul className="mb-3 flex flex-col gap-2">
        {items.map((it, i) => (
          <PlanItemRow
            // id is a client-only UUID assigned once per item (see LocalItem),
            // stable across edits/reorders unlike the array index.
            key={it.id}
            item={it}
            onChange={(patch) => updateItem(i, patch)}
            onRemove={() => removeItem(i)}
          />
        ))}
        {!items.length && <li className="px-1 py-3 text-center text-sm text-muted">{t('plan.itemsEmpty')}</li>}
      </ul>

      {items.some((it) => !it.adLibitum) && (
        <div className="mb-3 flex items-center justify-between rounded-lg bg-surface-2 px-3 py-2 text-sm">
          <span className="font-medium text-ink">{t('plan.totalMacros')}</span>
          <span className="tnum text-muted">
            {formatNumber(total.Calories)} kcal · {formatNumber(total.Protein)}P · {formatNumber(total.Carbs)}C ·{' '}
            {formatNumber(total.Fat)}F
          </span>
        </div>
      )}

      {error && (
        <p className="mb-2 text-sm font-medium text-accent" role="alert">
          {error instanceof Error ? error.message : t('plan.saveFailed')}
        </p>
      )}

      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onClose} className="px-3 py-1.5 text-sm">
          {t('plan.cancel')}
        </Button>
        <Button onClick={submit} disabled={!label.trim() || !items.length || pending} className="px-3 py-1.5 text-sm">
          {pending ? t('plan.saving') : t('plan.saveOption')}
        </Button>
      </div>
    </div>
  )
}

export function PlanItemRow({
  item,
  onChange,
  onRemove,
}: Readonly<{
  item: LocalItem
  onChange: (patch: Partial<LocalItem>) => void
  onRemove: () => void
}>) {
  const { t } = useTranslation()
  const options = unitOptionsFor(item.food)
  const grams = item.adLibitum ? 0 : gramsFor(item)
  const macros = item.adLibitum ? ZERO_MACROS : scaleMacros(item.food.per_100g, grams)

  return (
    <li className="rounded-lg border border-line bg-surface px-3 py-2">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-ink">{item.food.name}</p>
          {!item.adLibitum && <p className="tnum text-xs text-muted">{formatNumber(macros.Calories)} kcal</p>}
        </div>
        {!item.adLibitum && (
          <>
            <input
              type="number"
              inputMode="decimal"
              min={0}
              step="any"
              value={item.quantity}
              onChange={(e) => onChange({ quantity: Number(e.target.value) || 0 })}
              aria-label={t('plan.quantityAria', { name: item.food.name })}
              className="w-16 shrink-0 rounded-full border border-line bg-bg px-2 py-1 text-right text-sm text-ink outline-none focus:border-primary tnum"
            />
            <select
              value={item.unitID}
              onChange={(e) => onChange({ unitID: e.target.value })}
              aria-label={t('plan.unitAria', { name: item.food.name })}
              className="shrink-0 rounded-full border border-line bg-bg px-2 py-1 text-sm text-ink outline-none focus:border-primary"
            >
              {options.map((u) => (
                <option key={u.id} value={u.id}>
                  {u.label}
                </option>
              ))}
            </select>
          </>
        )}
        <button
          type="button"
          onClick={onRemove}
          aria-label={t('plan.removeItemAria', { name: item.food.name })}
          className="grid size-7 shrink-0 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent"
        >
          <TrashIcon width={13} height={13} />
        </button>
      </div>
      <div className="mt-2 flex items-center gap-2 text-xs text-muted">
        <Toggle checked={item.adLibitum} onChange={(v) => onChange({ adLibitum: v })} label={t('plan.adLibitumToggleAria', { name: item.food.name })} />
        {t('plan.adLibitum')}
        {item.adLibitum && <CheckIcon width={12} height={12} className="text-primary" />}
      </div>
    </li>
  )
}
