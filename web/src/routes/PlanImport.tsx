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
import { pdfToText, type PdfTextResult } from '@/lib/pdfToText'
import { Card, Button, Pill, Field } from '@/components/ui'
import { MACRO_KEYS } from '@/lib/types'
import type {
  DietPlan,
  FoodDetail,
  FoodServingUnit,
  PlanDraft,
  PlanDraftDayType,
  PlanDraftItem,
  PlanDraftOption,
  PlanDraftSlot,
} from '@/lib/types'
import {
  ItemSearchResults,
  PlanItemRow,
  toResolvedItem,
  todayISO,
  nextMondayISO,
  mostRecentMondayISO,
  type LocalItem,
} from './Plan'
import { GRAMS_UNIT_ID } from '@/lib/servingUnits'

const MAX_PASTE_CHARS = 20_000

// Mirrors the backend's maxPlanPages / maxPlanTotalBytes (#220/#222),
// enforced here too so an over-limit upload fails fast, before any network
// call.
const MAX_PLAN_PAGES = 10
const MAX_PLAN_TOTAL_BYTES = 25 * 1024 * 1024

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

// Progress through a photo/PDF import: idle -> (rendering ->) extracting.
// "rendering" covers client-side PDF work (reading the text layer, or
// rendering pages to PNGs for the scanned fallback); "extracting" covers the
// network call, labeled with which path is running and how many pages.
type PhotoStage =
  | { kind: 'idle' }
  | { kind: 'rendering' }
  | { kind: 'extracting'; mode: 'native' | 'scan'; pages: number }

// Set only when native extraction looked garbled (#220): shows a text
// preview and an explicit opt-in to retry as a scan, instead of silently
// switching modes per requirement 6 of the epic.
interface MalformedState {
  file: File
  preview: string
  pageCount: number
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
  const extractText = useExtractPlanFromText()
  const extractImage = useExtractPlanFromImage()
  const [stage, setStage] = useState<PhotoStage>({ kind: 'idle' })
  const [malformed, setMalformed] = useState<MalformedState | null>(null)

  function fail(err: unknown) {
    setStage({ kind: 'idle' })
    onFailed(err instanceof Error ? err.message : t('plan.extractFailed'))
  }

  async function extractAsScan(pages: Blob[]) {
    if (pages.length === 0) {
      setStage({ kind: 'idle' })
      onFailed(t('plan.extractFailed'))
      return
    }
    if (pages.length > MAX_PLAN_PAGES) {
      setStage({ kind: 'idle' })
      onFailed(t('plan.pdfTooManyPages', { count: pages.length, max: MAX_PLAN_PAGES }))
      return
    }
    const files = pages.map((blob, i) => new File([blob], `plan-page-${i + 1}.png`, { type: 'image/png' }))
    setStage({ kind: 'extracting', mode: 'scan', pages: files.length })
    extractImage.mutate(files, { onSuccess: onExtracted, onError: fail })
  }

  async function retryAsScan(file: File) {
    setMalformed(null)
    setStage({ kind: 'rendering' })
    let pages: Blob[]
    try {
      pages = await pdfToImages(file)
    } catch (err) {
      fail(err)
      return
    }
    await extractAsScan(pages)
  }

  async function handlePdf(file: File) {
    setStage({ kind: 'rendering' })
    let result: PdfTextResult
    try {
      result = await pdfToText(file)
    } catch (err) {
      fail(err)
      return
    }
    if (result.pageCount > MAX_PLAN_PAGES) {
      setStage({ kind: 'idle' })
      onFailed(t('plan.pdfTooManyPages', { count: result.pageCount, max: MAX_PLAN_PAGES }))
      return
    }
    if (result.status === 'malformed') {
      setStage({ kind: 'idle' })
      setMalformed({ file, preview: result.text, pageCount: result.pageCount })
      return
    }
    if (result.status === 'empty') {
      let pages: Blob[]
      try {
        pages = await pdfToImages(file)
      } catch (err) {
        fail(err)
        return
      }
      await extractAsScan(pages)
      return
    }
    setStage({ kind: 'extracting', mode: 'native', pages: result.pageCount })
    extractText.mutate(result.text, { onSuccess: onExtracted, onError: fail })
  }

  async function handleFile(file: File) {
    setMalformed(null)
    if (file.size > MAX_PLAN_TOTAL_BYTES) {
      onFailed(t('plan.fileTooLarge', { maxMB: MAX_PLAN_TOTAL_BYTES / (1024 * 1024) }))
      return
    }
    if (isPdfFile(file)) {
      await handlePdf(file)
      return
    }
    setStage({ kind: 'extracting', mode: 'scan', pages: 1 })
    extractImage.mutate([file], { onSuccess: onExtracted, onError: fail })
  }

  const busy = stage.kind !== 'idle'
  let progressLabel: string | null = null
  if (stage.kind === 'rendering') {
    progressLabel = t('plan.renderingPdf')
  } else if (stage.kind === 'extracting') {
    const modeKey = stage.mode === 'native' ? 'plan.extractModeNative' : 'plan.extractModeScan'
    progressLabel = t(modeKey, { count: stage.pages })
  }

  return (
    <Card className="p-5">
      <p className="mb-1 font-semibold text-ink">{t('plan.importFromPhoto')}</p>
      <p className="mb-3 text-sm text-muted">{t('plan.importPhotoHint')}</p>

      {malformed && (
        <div className="mb-3 rounded-lg border border-line bg-surface-2/50 p-3">
          <p className="mb-2 text-sm font-medium text-accent" role="alert">
            {t('plan.malformedNotice')}
          </p>
          <p className="mb-1 text-xs font-medium text-muted">{t('plan.textPreviewLabel')}</p>
          <pre className="mb-2 max-h-32 overflow-y-auto whitespace-pre-wrap rounded-lg border border-line bg-bg p-2 text-xs text-ink">
            {malformed.preview}
          </pre>
          <Button
            variant="ghost"
            onClick={() => void retryAsScan(malformed.file)}
            disabled={busy}
            className="px-3 py-1.5 text-xs"
          >
            {t('plan.retryAsScan')}
          </Button>
        </div>
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
      {progressLabel && <p className="mt-2 text-sm text-muted">{progressLabel}</p>}
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

// draft.notes plus, if any, a labeled block of standalone substitution notes
// (#223) — substitutions have no DB column of their own, they ride along in
// the plan's existing free-text notes. undefined (not '') when there's
// nothing to say, so api.plans.create omits the field rather than sending an
// empty string.
function buildNotesWithSubstitutions(draft: PlanDraft): string | undefined {
  const substitutions = draft.substitutions ?? []
  const parts: string[] = []
  if (draft.notes) parts.push(draft.notes)
  if (substitutions.length > 0) {
    parts.push(['Substitutions:', ...substitutions.map((s) => `- ${s}`)].join('\n'))
  }
  return parts.length > 0 ? parts.join('\n\n') : undefined
}

// The edited weekday grid names a day-type per weekday; nameToID resolves
// those to the ids createPlanFromDraft just minted. Only returns a pattern
// once all 7 slots resolve — a null slot has no valid id to place, and
// inventing one would violate the never-invent rule, so an incomplete grid
// is left for the user to finish later in the normal CycleEditor instead.
function buildCyclePatternFromWeekdaySchedule(
  schedule: (string | null)[],
  nameToID: Record<string, string>,
): string[] | null {
  const ids = schedule.map((name) => (name ? nameToID[name] : undefined))
  return ids.every((id): id is string => !!id) ? ids : null
}

// Fires the same create-plan -> day-types -> slots -> options sequence
// usePlanClone demonstrates for duplication, walking the draft in order.
// Stops and toasts on the first failure; partial creation is left in place
// rather than attempting rollback, matching usePlanClone's own behaviour.
// Returns the day-type name -> id map alongside the plan so the caller can
// resolve the weekday grid into a cycle_pattern once everything exists.
async function createPlanFromDraft(
  draft: PlanDraft,
  resolved: Record<string, LocalItem>,
  meta: { name: string; validFrom: string; validTo: string; anchor: string; notes?: string },
): Promise<{ plan: DietPlan; dayTypeIDs: Record<string, string> }> {
  const plan = await api.plans.create({
    name: meta.name,
    notes: meta.notes,
    valid_from: meta.validFrom,
    valid_to: meta.validTo,
    cycle_anchor_date: meta.anchor,
  })
  const dayTypeIDs: Record<string, string> = {}
  for (const [dtIdx, dt] of draft.day_types.entries()) {
    const newDayType = await api.plans.dayTypes.create(plan.id, {
      name: dt.name,
      position: dtIdx,
      targets: dt.targets,
      water_goal_ml: dt.water_goal_ml ?? 0,
    })
    dayTypeIDs[dt.name] = newDayType.id
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
  return { plan, dayTypeIDs }
}

const WEEKDAY_KEYS = [
  'plan.weekdayMonday',
  'plan.weekdayTuesday',
  'plan.weekdayWednesday',
  'plan.weekdayThursday',
  'plan.weekdayFriday',
  'plan.weekdaySaturday',
  'plan.weekdaySunday',
] as const

function WeekdayScheduleSection({
  dayTypeNames,
  schedule,
  onChange,
}: Readonly<{
  dayTypeNames: string[]
  schedule: (string | null)[]
  onChange: (i: number, value: string | null) => void
}>) {
  const { t } = useTranslation()
  const incomplete = schedule.some((v) => v == null)
  return (
    <div className="mb-5 rounded-lg border border-line p-3">
      <p className="mb-1 text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.weekdayScheduleTitle')}</p>
      <p className="mb-3 text-sm text-muted">{t('plan.weekdayScheduleHint')}</p>
      <div className="grid gap-2 sm:grid-cols-2">
        {WEEKDAY_KEYS.map((key, i) => (
          <label key={key} className="flex items-center justify-between gap-2 text-sm">
            <span className="text-ink">{t(key)}</span>
            <select
              value={schedule[i] ?? ''}
              onChange={(e) => onChange(i, e.target.value || null)}
              aria-label={t(key)}
              className="rounded-lg border border-line bg-bg px-2 py-1.5 text-sm text-ink outline-none focus:border-primary"
            >
              <option value="">{t('plan.weekdayNotSpecified')}</option>
              {dayTypeNames.map((dtName) => (
                <option key={dtName} value={dtName}>
                  {dtName}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>
      {incomplete && <p className="mt-2 text-xs text-muted">{t('plan.weekdayIncompleteHint')}</p>}
    </div>
  )
}

function SubstitutionsSection({ substitutions }: Readonly<{ substitutions: string[] }>) {
  const { t } = useTranslation()
  if (substitutions.length === 0) return null
  return (
    <div className="mb-5 rounded-lg border border-dashed border-line p-3">
      <p className="mb-2 text-xs font-semibold uppercase tracking-[0.14em] text-muted">{t('plan.substitutionsTitle')}</p>
      <ul className="list-disc space-y-1 pl-5 text-sm text-ink">
        {substitutions.map((s) => (
          <li key={s}>{s}</li>
        ))}
      </ul>
    </div>
  )
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
  // Pre-#223 drafts have no weekday_schedule at all; default every slot to
  // null rather than assume anything about the extraction.
  const [weekday, setWeekday] = useState<(string | null)[]>(() => draft.weekday_schedule ?? new Array(7).fill(null))

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
      const notes = buildNotesWithSubstitutions(draft)
      const { plan, dayTypeIDs } = await createPlanFromDraft(draft, resolved, {
        name: name.trim(),
        validFrom,
        validTo,
        anchor,
        notes,
      })
      const cyclePattern = buildCyclePatternFromWeekdaySchedule(weekday, dayTypeIDs)
      if (cyclePattern) {
        await api.plans.update(plan.id, { ...plan, cycle_pattern: cyclePattern, cycle_anchor_date: mostRecentMondayISO() })
      }
      await qc.invalidateQueries({ queryKey: ['plan'] })
      onCreated(plan.id)
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

      <WeekdayScheduleSection
        dayTypeNames={draft.day_types.map((dt) => dt.name)}
        schedule={weekday}
        onChange={(i, value) => setWeekday((cur) => cur.map((v, idx) => (idx === i ? value : v)))}
      />
      <SubstitutionsSection substitutions={draft.substitutions ?? []} />

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
