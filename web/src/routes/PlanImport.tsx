// Diet plan import: paste-text extraction (#193). A model call best-effort
// transcribes a nutritionist's prescription into a PlanDraft; nothing here
// persists until the user resolves every food item against the real catalog
// and explicitly confirms. Reuses PlanItemRow/ItemSearchResults from Plan.tsx
// so a resolved draft item edits exactly like a hand-built option's items do,
// and fires the same create-day-types/slots/options sequence usePlanClone
// already demonstrates there.

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useExtractPlanFromText, useExtractPlanFromImage, useCatalogSearch } from '@/lib/queries'
import { pdfToImages } from '@/lib/pdfToImages'
import { Card, Button, Pill, Field } from '@/components/ui'
import { MACRO_KEYS } from '@/lib/types'
import type {
  FoodDetail,
  FoodServingUnit,
  PlanDraft,
  PlanDraftDayType,
  PlanDraftItem,
  PlanDraftOption,
  PlanDraftSlot,
} from '@/lib/types'
import { ItemSearchResults, PlanItemRow, toResolvedItem, todayISO, nextMondayISO, type LocalItem } from './Plan'
import { GRAMS_UNIT_ID } from '@/lib/servingUnits'

const MAX_PASTE_CHARS = 20_000

type Stage =
  | { kind: 'collapsed' }
  | { kind: 'paste' }
  | { kind: 'photo' }
  | { kind: 'error'; message: string }
  | { kind: 'review'; draft: PlanDraft }

export function ImportPlanCard({ onCreated }: Readonly<{ onCreated: (id: string) => void }>) {
  const { t } = useTranslation()
  const [stage, setStage] = useState<Stage>({ kind: 'collapsed' })

  function onExtracted(draft: PlanDraft) {
    setStage(
      draft.unreadable ? { kind: 'error', message: t('plan.extractUnreadable') } : { kind: 'review', draft },
    )
  }
  function onFailed(message: string) {
    setStage({ kind: 'error', message })
  }

  if (stage.kind === 'paste') {
    return <PasteTextCard onCancel={() => setStage({ kind: 'collapsed' })} onExtracted={onExtracted} onFailed={onFailed} />
  }

  if (stage.kind === 'photo') {
    return <PhotoImportCard onCancel={() => setStage({ kind: 'collapsed' })} onExtracted={onExtracted} onFailed={onFailed} />
  }

  if (stage.kind === 'error') {
    return (
      <Card className="p-5">
        <p className="mb-3 text-sm font-medium text-accent" role="alert">
          {stage.message}
        </p>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={() => setStage({ kind: 'paste' })} className="px-3 py-1.5 text-xs">
            {t('plan.tryImportAgain')}
          </Button>
          <Button variant="ghost" onClick={() => setStage({ kind: 'collapsed' })} className="px-3 py-1.5 text-xs">
            {t('plan.fallbackToManual')}
          </Button>
        </div>
      </Card>
    )
  }

  if (stage.kind === 'review') {
    return <DraftReview draft={stage.draft} onCancel={() => setStage({ kind: 'collapsed' })} onCreated={onCreated} />
  }

  return (
    <Card className="p-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="font-semibold text-ink">{t('plan.importTitle')}</p>
          <p className="text-sm text-muted">{t('plan.importHint')}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={() => setStage({ kind: 'paste' })}>
            {t('plan.importFromText')}
          </Button>
          <Button variant="ghost" onClick={() => setStage({ kind: 'photo' })}>
            {t('plan.importFromPhoto')}
          </Button>
        </div>
      </div>
    </Card>
  )
}

function PasteTextCard({
  onExtracted,
  onFailed,
  onCancel,
}: Readonly<{
  onExtracted: (draft: PlanDraft) => void
  onFailed: (message: string) => void
  onCancel: () => void
}>) {
  const { t } = useTranslation()
  const extract = useExtractPlanFromText()
  const [text, setText] = useState('')

  function submit() {
    if (!text.trim() || extract.isPending) return
    extract.mutate(text, {
      onSuccess: onExtracted,
      onError: (err) => onFailed(err instanceof Error ? err.message : t('plan.extractFailed')),
    })
  }

  return (
    <Card className="p-5">
      <p className="mb-3 font-semibold text-ink">{t('plan.importFromText')}</p>
      <label htmlFor="plan-import-text" className="mb-1.5 block text-sm font-medium text-ink">
        {t('plan.pastePlaceholder')}
      </label>
      <textarea
        id="plan-import-text"
        value={text}
        onChange={(e) => setText(e.target.value.slice(0, MAX_PASTE_CHARS))}
        maxLength={MAX_PASTE_CHARS}
        placeholder={t('plan.pastePlaceholder')}
        rows={10}
        className="w-full rounded-xl border border-line bg-surface px-4 py-3 text-sm text-ink outline-none transition placeholder:text-muted/70 focus:border-primary"
      />
      <p className="mb-3 text-right text-xs text-muted tnum">
        {text.length}/{MAX_PASTE_CHARS}
      </p>
      <div className="flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} className="px-3 py-1.5 text-sm">
          {t('plan.cancel')}
        </Button>
        <Button onClick={submit} disabled={!text.trim() || extract.isPending} className="px-3 py-1.5 text-sm">
          {extract.isPending ? t('plan.extracting') : t('plan.extractButton')}
        </Button>
      </div>
    </Card>
  )
}

function isPdfFile(file: File): boolean {
  return file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf')
}

function PhotoImportCard({
  onExtracted,
  onFailed,
  onCancel,
}: Readonly<{
  onExtracted: (draft: PlanDraft) => void
  onFailed: (message: string) => void
  onCancel: () => void
}>) {
  const { t } = useTranslation()
  const extract = useExtractPlanFromImage()
  const [rendering, setRendering] = useState(false)
  const [multiPage, setMultiPage] = useState(false)

  async function handleFile(file: File) {
    setMultiPage(false)
    let imageFile = file
    if (isPdfFile(file)) {
      setRendering(true)
      let pages: Blob[]
      try {
        pages = await pdfToImages(file)
      } catch (err) {
        setRendering(false)
        onFailed(err instanceof Error ? err.message : t('plan.extractFailed'))
        return
      }
      setRendering(false)
      if (pages.length === 0) {
        onFailed(t('plan.extractFailed'))
        return
      }
      // Multi-page merging into one extraction call is out of scope: use the
      // first page and tell the user, rather than silently dropping the rest.
      if (pages.length > 1) setMultiPage(true)
      imageFile = new File([pages[0]], 'plan-page.png', { type: 'image/png' })
    }
    extract.mutate(imageFile, {
      onSuccess: onExtracted,
      onError: (err) => onFailed(err instanceof Error ? err.message : t('plan.extractFailed')),
    })
  }

  const busy = rendering || extract.isPending

  return (
    <Card className="p-5">
      <p className="mb-1 font-semibold text-ink">{t('plan.importFromPhoto')}</p>
      <p className="mb-3 text-sm text-muted">{t('plan.importPhotoHint')}</p>
      {multiPage && (
        <p className="mb-3 text-sm font-medium text-accent" role="status">
          {t('plan.pdfMultiPageNotice')}
        </p>
      )}
      <label htmlFor="plan-import-photo" className="mb-1.5 block text-sm font-medium text-ink">
        {t('plan.choosePhotoFile')}
      </label>
      <input
        id="plan-import-photo"
        type="file"
        accept="image/jpeg,image/png,application/pdf"
        disabled={busy}
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) void handleFile(file)
        }}
        className="block w-full text-sm text-ink"
      />
      {busy && <p className="mt-2 text-sm text-muted">{rendering ? t('plan.renderingPdf') : t('plan.extracting')}</p>}
      <div className="mt-4 flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} disabled={busy} className="px-3 py-1.5 text-sm">
          {t('plan.cancel')}
        </Button>
      </div>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Draft review: day-types -> slots -> options -> items. Resolution state is
// a flat map keyed by position in the tree (draft items have no server id
// yet), so every level below just forwards its own index down.
// ---------------------------------------------------------------------------

function itemKey(dtIdx: number, slotIdx: number, optIdx: number, itemIdx: number): string {
  return `${dtIdx}:${slotIdx}:${optIdx}:${itemIdx}`
}

function allItemsResolved(draft: PlanDraft, resolved: Record<string, LocalItem>): boolean {
  for (const [dtIdx, dt] of draft.day_types.entries()) {
    for (const [slotIdx, slot] of dt.slots.entries()) {
      for (const [optIdx, opt] of slot.options.entries()) {
        for (let itemIdx = 0; itemIdx < opt.items.length; itemIdx++) {
          if (!resolved[itemKey(dtIdx, slotIdx, optIdx, itemIdx)]) return false
        }
      }
    }
  }
  return true
}

// A resolved catalog food seeds quantity/unit from the draft's guess: the
// matched serving unit's label if the model named one the food actually has,
// grams otherwise. ad_libitum always wins (mirrors the manual builder's
// quantity===0 convention, applied at save time by toResolvedItem).
function draftItemQuantity(draftItem: PlanDraftItem, matchedUnit: FoodServingUnit | undefined): number {
  if (draftItem.ad_libitum) return 1
  if (draftItem.quantity != null) return draftItem.quantity
  return matchedUnit ? 1 : 100
}

function fromDraftItem(draftItem: PlanDraftItem, food: FoodDetail): LocalItem {
  const matchedUnit = draftItem.unit
    ? (food.serving_units ?? []).find((u) => u.label === draftItem.unit)
    : undefined
  const quantity = draftItemQuantity(draftItem, matchedUnit)
  return {
    food,
    unitID: matchedUnit ? matchedUnit.id : GRAMS_UNIT_ID,
    quantity,
    adLibitum: draftItem.ad_libitum,
    id: crypto.randomUUID(),
  }
}

function LowConfidenceBadge({ fields }: Readonly<{ fields?: string[] }>) {
  const { t } = useTranslation()
  if (!fields?.length) return null
  return <Pill tone="accent">{t('plan.lowConfidenceBadge', { fields: fields.join(', ') })}</Pill>
}

function DraftItemRow({
  draftItem,
  resolvedItem,
  onResolve,
  onUnresolve,
  onChange,
}: Readonly<{
  draftItem: PlanDraftItem
  resolvedItem: LocalItem | undefined
  onResolve: (food: FoodDetail) => void
  onUnresolve: () => void
  onChange: (patch: Partial<LocalItem>) => void
}>) {
  const { t } = useTranslation()
  const [rawQuery, setRawQuery] = useState(draftItem.raw_name)
  const [query, setQuery] = useState('')
  useEffect(() => {
    const id = setTimeout(() => setQuery(rawQuery.trim()), 250)
    return () => clearTimeout(id)
  }, [rawQuery])
  const search = useCatalogSearch(query)

  if (resolvedItem) {
    // onRemove here re-opens the picker rather than deleting: a draft
    // item's count is fixed by extraction, "remove" means "pick again".
    return <PlanItemRow item={resolvedItem} onChange={onChange} onRemove={onUnresolve} />
  }

  return (
    <li className="rounded-lg border border-dashed border-line bg-surface px-3 py-2">
      <p className="mb-1.5 text-sm font-medium text-ink">{draftItem.raw_name}</p>
      <input
        value={rawQuery}
        onChange={(e) => setRawQuery(e.target.value)}
        aria-label={t('plan.itemSearchAria', { name: draftItem.raw_name })}
        placeholder={t('plan.itemSearchPlaceholder')}
        className="w-full rounded-lg border border-line bg-bg px-3 py-1.5 text-sm text-ink outline-none focus:border-primary"
      />
      {query.length > 0 && (
        <ul className="mt-1 max-h-40 divide-y divide-line overflow-y-auto rounded-lg border border-line">
          <ItemSearchResults search={search} onPick={onResolve} />
        </ul>
      )}
    </li>
  )
}

function DraftOptionCard({
  option,
  dtIdx,
  slotIdx,
  optIdx,
  resolved,
  onResolve,
  onUnresolve,
  onChange,
}: Readonly<{
  option: PlanDraftOption
  dtIdx: number
  slotIdx: number
  optIdx: number
  resolved: Record<string, LocalItem>
  onResolve: (key: string, draftItem: PlanDraftItem, food: FoodDetail) => void
  onUnresolve: (key: string) => void
  onChange: (key: string, patch: Partial<LocalItem>) => void
}>) {
  return (
    <div className="rounded-lg border border-line bg-surface-2/50 p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <p className="text-sm font-semibold text-ink">{option.label}</p>
        <LowConfidenceBadge fields={option.low_confidence_fields} />
      </div>
      <ul className="flex flex-col gap-2">
        {option.items.map((item, itemIdx) => {
          const key = itemKey(dtIdx, slotIdx, optIdx, itemIdx)
          return (
            <DraftItemRow
              key={key}
              draftItem={item}
              resolvedItem={resolved[key]}
              onResolve={(food) => onResolve(key, item, food)}
              onUnresolve={() => onUnresolve(key)}
              onChange={(patch) => onChange(key, patch)}
            />
          )
        })}
      </ul>
    </div>
  )
}

function DraftSlotCard({
  slot,
  dtIdx,
  slotIdx,
  resolved,
  onResolve,
  onUnresolve,
  onChange,
}: Readonly<{
  slot: PlanDraftSlot
  dtIdx: number
  slotIdx: number
  resolved: Record<string, LocalItem>
  onResolve: (key: string, draftItem: PlanDraftItem, food: FoodDetail) => void
  onUnresolve: (key: string) => void
  onChange: (key: string, patch: Partial<LocalItem>) => void
}>) {
  return (
    <div className="rounded-lg border border-line bg-bg p-3">
      <p className="mb-2 text-sm font-semibold text-ink">
        {slot.label}
        {slot.time_of_day && <span className="ml-2 text-xs font-normal text-muted tnum">{slot.time_of_day}</span>}
      </p>
      <div className="space-y-2">
        {slot.options.map((opt, optIdx) => (
          <DraftOptionCard
            key={opt.label}
            option={opt}
            dtIdx={dtIdx}
            slotIdx={slotIdx}
            optIdx={optIdx}
            resolved={resolved}
            onResolve={onResolve}
            onUnresolve={onUnresolve}
            onChange={onChange}
          />
        ))}
      </div>
    </div>
  )
}

function DraftDayTypeCard({
  dayType,
  dtIdx,
  resolved,
  onResolve,
  onUnresolve,
  onChange,
}: Readonly<{
  dayType: PlanDraftDayType
  dtIdx: number
  resolved: Record<string, LocalItem>
  onResolve: (key: string, draftItem: PlanDraftItem, food: FoodDetail) => void
  onUnresolve: (key: string) => void
  onChange: (key: string, patch: Partial<LocalItem>) => void
}>) {
  const { t } = useTranslation()
  return (
    <div className="rounded-lg border border-line p-3">
      <div className="mb-1 flex items-center justify-between gap-2">
        <p className="font-semibold text-ink">{dayType.name}</p>
        <LowConfidenceBadge fields={dayType.low_confidence_fields} />
      </div>
      <p className="mb-3 text-xs text-muted tnum">
        {MACRO_KEYS.map((k) => {
          const macroLabel = t(`common.macro.${k}`)
          return `${dayType.targets[k]} ${macroLabel}`
        }).join(' · ')}
        {dayType.water_goal_ml != null && ` · ${dayType.water_goal_ml} ml`}
      </p>
      <div className="space-y-2">
        {dayType.slots.map((slot, slotIdx) => (
          <DraftSlotCard
            key={slot.label}
            slot={slot}
            dtIdx={dtIdx}
            slotIdx={slotIdx}
            resolved={resolved}
            onResolve={onResolve}
            onUnresolve={onUnresolve}
            onChange={onChange}
          />
        ))}
      </div>
    </div>
  )
}

// Fires the same create-plan -> day-types -> slots -> options sequence
// usePlanClone demonstrates for duplication, walking the draft in order.
// Stops and toasts on the first failure; partial creation is left in place
// rather than attempting rollback, matching usePlanClone's own behaviour.
async function createPlanFromDraft(
  draft: PlanDraft,
  resolved: Record<string, LocalItem>,
  meta: { name: string; validFrom: string; validTo: string; anchor: string },
): Promise<string> {
  const plan = await api.plans.create({
    name: meta.name,
    valid_from: meta.validFrom,
    valid_to: meta.validTo,
    cycle_anchor_date: meta.anchor,
  })
  for (const [dtIdx, dt] of draft.day_types.entries()) {
    const newDayType = await api.plans.dayTypes.create(plan.id, {
      name: dt.name,
      position: dtIdx,
      targets: dt.targets,
      water_goal_ml: dt.water_goal_ml ?? 0,
    })
    for (const [slotIdx, slot] of dt.slots.entries()) {
      const newSlot = await api.plans.slots.create(plan.id, newDayType.id, {
        label: slot.label,
        position: slotIdx,
        time_of_day: slot.time_of_day ?? '',
      })
      for (const [optIdx, opt] of slot.options.entries()) {
        const items = opt.items.map((_, itemIdx) => toResolvedItem(resolved[itemKey(dtIdx, slotIdx, optIdx, itemIdx)]))
        await api.plans.options.create(plan.id, newDayType.id, newSlot.id, {
          label: opt.label,
          position: optIdx,
          items,
        })
      }
    }
  }
  return plan.id
}

function DraftReview({
  draft,
  onCancel,
  onCreated,
}: Readonly<{ draft: PlanDraft; onCancel: () => void; onCreated: (id: string) => void }>) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [resolved, setResolved] = useState<Record<string, LocalItem>>({})
  // Plan-level metadata (name/dates/anchor) is collected here, after the
  // draft review rather than before: cycle_pattern needs real day-type ids
  // that only exist once the create sequence below has run, so it stays out
  // of this screen entirely — the user sets it in the normal builder's
  // CycleEditor once the plan lands there.
  const [name, setName] = useState(draft.plan_name ?? '')
  const [validFrom, setValidFrom] = useState(todayISO())
  const [validTo, setValidTo] = useState('')
  const [anchor, setAnchor] = useState(nextMondayISO())
  const [confirming, setConfirming] = useState(false)

  function resolveItem(key: string, draftItem: PlanDraftItem, food: FoodDetail) {
    setResolved((cur) => ({ ...cur, [key]: fromDraftItem(draftItem, food) }))
  }
  function unresolveItem(key: string) {
    setResolved((cur) => {
      const next = { ...cur }
      delete next[key]
      return next
    })
  }
  function updateItem(key: string, patch: Partial<LocalItem>) {
    setResolved((cur) => (cur[key] ? { ...cur, [key]: { ...cur[key], ...patch } } : cur))
  }

  const ready = name.trim().length > 0 && allItemsResolved(draft, resolved)

  async function confirm() {
    if (!ready || confirming) return
    setConfirming(true)
    try {
      const planID = await createPlanFromDraft(draft, resolved, { name: name.trim(), validFrom, validTo, anchor })
      await qc.invalidateQueries({ queryKey: ['plan'] })
      onCreated(planID)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('plan.saveFailed'))
    } finally {
      setConfirming(false)
    }
  }

  return (
    <Card className="p-5">
      <p className="mb-1 font-semibold text-ink">{t('plan.reviewDraftTitle')}</p>
      <p className="mb-4 text-sm text-muted">{t('plan.unmatchedItemsHint')}</p>

      <div className="mb-5 grid gap-3 sm:grid-cols-2">
        <Field label={t('plan.nameLabel')} value={name} onChange={(e) => setName(e.target.value)} placeholder={t('plan.namePlaceholder')} />
        <Field label={t('plan.validFromLabel')} type="date" value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
        <Field label={t('plan.validToLabel')} type="date" value={validTo} onChange={(e) => setValidTo(e.target.value)} hint={t('plan.validToHint')} />
        <Field label={t('plan.anchorLabel')} type="date" value={anchor} onChange={(e) => setAnchor(e.target.value)} hint={t('plan.anchorHint')} />
      </div>

      <div className="space-y-3">
        {draft.day_types.map((dt, dtIdx) => (
          <DraftDayTypeCard
            key={dt.name}
            dayType={dt}
            dtIdx={dtIdx}
            resolved={resolved}
            onResolve={resolveItem}
            onUnresolve={unresolveItem}
            onChange={updateItem}
          />
        ))}
      </div>

      <div className="mt-5 flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel} className="px-3 py-1.5 text-sm">
          {t('plan.cancel')}
        </Button>
        <Button onClick={confirm} disabled={!ready || confirming} className="px-3 py-1.5 text-sm">
          {confirming ? t('plan.saving') : t('plan.confirmImport')}
        </Button>
      </div>
    </Card>
  )
}
