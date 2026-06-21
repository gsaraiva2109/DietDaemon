// Body tracking hub, weight, measurements, and progress photos. Sub-tabs are
// driven by the :tab route param so each view is linkable. Write controls are
// disabled in demo mode (reads still return sample data).

import { useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, EmptyState, Eyebrow, Pill, Spinner } from '@/components/ui'
import { WeightChart } from '@/components/WeightChart'
import { MeasurementChart } from '@/components/MeasurementChart'
import { PhotoGrid } from '@/components/PhotoGrid'
import { PhotoCompare } from '@/components/PhotoCompare'
import { TrashIcon } from '@/components/icons'
import { useDemo } from '@/lib/demo'
import { round } from '@/lib/format'
import {
  useBodySummary,
  useDeleteMeasurement,
  useDeletePhoto,
  useDeleteWeight,
  useLogMeasurements,
  useLogWeight,
  useMeasurements,
  usePhotos,
  useRange,
  useUploadPhoto,
  useWeightLog,
  useWeightTrend,
} from '@/lib/queries'
import {
  MEASUREMENT_FIELDS,
  type MeasurementEntry,
  type ProgressPhoto,
} from '@/lib/types'

const TABS = ['weight', 'measurements', 'photos'] as const
type Tab = (typeof TABS)[number]

const PHOTO_VIEWS = ['front', 'side', 'back'] as const

function today(): string {
  return new Date().toISOString().slice(0, 10)
}
function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

const DEMO_NOTE = 'Logging is disabled here.'

export function Body() {
  const { tab } = useParams<{ tab?: string }>()
  const navigate = useNavigate()
  const active: Tab = TABS.includes(tab as Tab) ? (tab as Tab) : 'weight'

  return (
    <div>
      <PageHeader eyebrow="Body" title="Composition" />

      <div className="mb-6 inline-flex gap-1 rounded-full border border-line bg-surface p-1">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => navigate(`/body/${t}`)}
            className={`rounded-full px-4 py-1.5 text-sm font-medium capitalize transition ${
              active === t ? 'bg-primary-soft text-primary' : 'text-muted hover:text-ink'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {active === 'weight' && <WeightTab />}
      {active === 'measurements' && <MeasurementsTab />}
      {active === 'photos' && <PhotosTab />}
    </div>
  )
}

// --- Weight ----------------------------------------------------------------

const RANGES = [
  { label: '30d', days: 30 },
  { label: '90d', days: 90 },
  { label: '180d', days: 180 },
  { label: 'All', days: 365 },
] as const

function trendArrow(dir: string): string {
  if (dir === 'down') return '↓'
  if (dir === 'up') return '↑'
  return '→'
}

function WeightTab() {
  const { demo } = useDemo()
  const [days, setDays] = useState(90)
  const [date, setDate] = useState(today())
  const [weight, setWeight] = useState('')
  const [note, setNote] = useState('')

  const trend = useWeightTrend(days)
  const log = useWeightLog(days)
  const summary = useBodySummary()
  const range = useRange(isoDaysAgo(days - 1), isoDaysAgo(0))
  const logWeight = useLogWeight()
  const deleteWeight = useDeleteWeight()

  const intake = useMemo(
    () => (range.data ?? []).map((r) => ({ date: r.Date, calories: Math.round(r.Consumed.Calories) })),
    [range.data],
  )

  const s = summary.data

  function submit() {
    const kg = Number(weight)
    if (!kg) return
    logWeight.mutate(
      { date, weightKg: kg, note: note.trim() || undefined },
      { onSuccess: () => { setWeight(''); setNote('') } },
    )
  }

  return (
    <div className="space-y-6">
      {s && (
        <Card className="flex flex-wrap items-center gap-6 p-5">
          <div>
            <Eyebrow>Current</Eyebrow>
            <p className="mt-1 text-2xl font-bold text-ink tnum">{round(s.current_weight_kg, 1)} kg</p>
          </div>
          <div>
            <Eyebrow>Start</Eyebrow>
            <p className="mt-1 text-2xl font-bold text-ink tnum">{round(s.start_weight_kg, 1)} kg</p>
          </div>
          <div>
            <Eyebrow>Change</Eyebrow>
            <p className="mt-1 flex items-center gap-1.5 text-2xl font-bold text-ink tnum">
              <span className="text-primary">{trendArrow(s.trend_direction)}</span>
              {s.change_kg > 0 ? '+' : ''}{round(s.change_kg, 1)} kg
            </p>
          </div>
        </Card>
      )}

      <Card className="p-5">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <Eyebrow>Trend</Eyebrow>
          <div className="inline-flex gap-1 rounded-full border border-line bg-surface p-1">
            {RANGES.map((r) => (
              <button
                key={r.label}
                onClick={() => setDays(r.days)}
                className={`rounded-full px-3 py-1 text-sm font-medium transition ${
                  days === r.days ? 'bg-primary-soft text-primary' : 'text-muted hover:text-ink'
                }`}
              >
                {r.label}
              </button>
            ))}
          </div>
        </div>
        {trend.isLoading ? <Spinner /> : <WeightChart trend={trend.data ?? []} intake={intake} />}
      </Card>

      <Card className="p-5">
        <Eyebrow>Log a weigh-in</Eyebrow>
        <div className="mt-3 flex flex-wrap items-end gap-3">
          <label className="block">
            <span className="mb-1 block text-xs font-medium text-muted">Date</span>
            <input
              type="date"
              value={date}
              max={today()}
              onChange={(e) => setDate(e.target.value)}
              disabled={demo}
              className="rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary disabled:opacity-50"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-xs font-medium text-muted">Weight (kg)</span>
            <input
              type="number"
              step="0.1"
              value={weight}
              placeholder="82.0"
              onChange={(e) => setWeight(e.target.value)}
              disabled={demo}
              className="w-28 rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary tnum disabled:opacity-50"
            />
          </label>
          <label className="block flex-1">
            <span className="mb-1 block text-xs font-medium text-muted">Note (optional)</span>
            <input
              value={note}
              placeholder="e.g. after morning run"
              onChange={(e) => setNote(e.target.value)}
              disabled={demo}
              className="w-full rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary disabled:opacity-50"
            />
          </label>
          <Button onClick={submit} disabled={demo || logWeight.isPending || !weight}>
            {logWeight.isPending ? 'Saving…' : 'Log'}
          </Button>
        </div>
        {demo && <p className="mt-3 text-sm text-muted">{DEMO_NOTE}</p>}
      </Card>

      <Card className="p-5">
        <Eyebrow>History</Eyebrow>
        {log.isLoading ? (
          <div className="mt-3"><Spinner /></div>
        ) : !(log.data ?? []).length ? (
          <div className="mt-3"><EmptyState title="No weigh-ins yet" /></div>
        ) : (
          <ul className="mt-3 divide-y divide-line">
            {[...(log.data ?? [])].reverse().map((e) => (
              <li key={e.id} className="flex items-center justify-between gap-3 py-2.5">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-semibold text-ink tnum">{round(e.weight_kg, 1)} kg</span>
                  <span className="text-sm text-muted">{e.date}</span>
                  {e.note && <span className="text-sm text-muted">· {e.note}</span>}
                </div>
                <button
                  onClick={() => deleteWeight.mutate(e.id)}
                  disabled={demo || deleteWeight.isPending}
                  aria-label="Delete weigh-in"
                  className="text-muted transition hover:text-accent disabled:opacity-30"
                >
                  <TrashIcon width={18} height={18} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  )
}

// --- Measurements ----------------------------------------------------------

type CmFields = Record<string, string>

function MeasurementsTab() {
  const { demo } = useDemo()
  const [date, setDate] = useState(today())
  const [fields, setFields] = useState<CmFields>({})

  const measurements = useMeasurements()
  const logMeasurements = useLogMeasurements()
  const deleteMeasurement = useDeleteMeasurement()

  function setField(key: string, v: string) {
    setFields((f) => ({ ...f, [key]: v }))
  }

  function submit() {
    const entry: Partial<MeasurementEntry> = { date }
    let any = false
    for (const f of MEASUREMENT_FIELDS) {
      const v = Number(fields[f.key])
      if (v > 0) {
        entry[f.key] = v
        any = true
      }
    }
    if (!any) return
    logMeasurements.mutate(entry, { onSuccess: () => setFields({}) })
  }

  return (
    <div className="space-y-6">
      <Card className="p-5">
        <Eyebrow>Trend</Eyebrow>
        <div className="mt-3">
          {measurements.isLoading ? <Spinner /> : <MeasurementChart data={measurements.data ?? []} />}
        </div>
      </Card>

      <Card className="p-5">
        <Eyebrow>Log measurements</Eyebrow>
        <label className="mt-3 block w-48">
          <span className="mb-1 block text-xs font-medium text-muted">Date</span>
          <input
            type="date"
            value={date}
            max={today()}
            onChange={(e) => setDate(e.target.value)}
            disabled={demo}
            className="w-full rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary disabled:opacity-50"
          />
        </label>
        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {MEASUREMENT_FIELDS.map((f) => (
            <label key={f.key} className="block">
              <span className="mb-1 block text-xs font-medium text-muted">{f.label} (cm)</span>
              <input
                type="number"
                step="0.1"
                value={fields[f.key] ?? ''}
                onChange={(e) => setField(f.key, e.target.value)}
                disabled={demo}
                className="w-full rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary tnum disabled:opacity-50"
              />
            </label>
          ))}
        </div>
        <div className="mt-4 flex items-center gap-3">
          <Button onClick={submit} disabled={demo || logMeasurements.isPending}>
            {logMeasurements.isPending ? 'Saving…' : 'Log measurements'}
          </Button>
          {demo && <p className="text-sm text-muted">{DEMO_NOTE}</p>}
        </div>
      </Card>

      <Card className="p-5">
        <Eyebrow>History</Eyebrow>
        {measurements.isLoading ? (
          <div className="mt-3"><Spinner /></div>
        ) : !(measurements.data ?? []).length ? (
          <div className="mt-3"><EmptyState title="No measurements yet" /></div>
        ) : (
          <ul className="mt-3 divide-y divide-line">
            {[...(measurements.data ?? [])].reverse().map((e) => (
              <li key={e.id} className="flex items-center justify-between gap-3 py-2.5">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-semibold text-ink">{e.date}</span>
                  {MEASUREMENT_FIELDS.filter((f) => e[f.key] > 0).map((f) => (
                    <Pill key={f.key} tone="muted">
                      {f.label} {round(e[f.key], 1)}
                    </Pill>
                  ))}
                </div>
                <button
                  onClick={() => deleteMeasurement.mutate(e.id)}
                  disabled={demo || deleteMeasurement.isPending}
                  aria-label="Delete measurements"
                  className="text-muted transition hover:text-accent disabled:opacity-30"
                >
                  <TrashIcon width={18} height={18} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  )
}

// --- Photos ----------------------------------------------------------------

function PhotosTab() {
  const { demo } = useDemo()
  const [view, setView] = useState<string>('front')
  const [date, setDate] = useState(today())
  const [file, setFile] = useState<File | null>(null)
  const [comparing, setComparing] = useState(false)

  const photos = usePhotos()
  const uploadPhoto = useUploadPhoto()
  const deletePhoto = useDeletePhoto()

  const list = photos.data ?? []

  function submit() {
    if (!file) return
    uploadPhoto.mutate(
      { file, view, date },
      { onSuccess: () => setFile(null) },
    )
  }

  function onSelect(p: ProgressPhoto) {
    // Selecting a thumbnail offers a quick delete; comparison is its own button.
    if (demo) return
    if (window.confirm(`Delete the ${p.view} photo from ${p.date}?`)) {
      deletePhoto.mutate(p.id)
    }
  }

  return (
    <div className="space-y-6">
      <Card className="p-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <Eyebrow>Upload a photo</Eyebrow>
          <Button
            variant="ghost"
            onClick={() => setComparing(true)}
            disabled={list.length < 2}
          >
            Compare
          </Button>
        </div>
        <div className="mt-3 flex flex-wrap items-end gap-3">
          <label className="block">
            <span className="mb-1 block text-xs font-medium text-muted">View</span>
            <select
              value={view}
              onChange={(e) => setView(e.target.value)}
              disabled={demo}
              className="rounded-full border border-line bg-bg px-4 py-2 text-sm capitalize text-ink outline-none focus:border-primary disabled:opacity-50"
            >
              {PHOTO_VIEWS.map((v) => (
                <option key={v} value={v}>{v}</option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="mb-1 block text-xs font-medium text-muted">Date</span>
            <input
              type="date"
              value={date}
              max={today()}
              onChange={(e) => setDate(e.target.value)}
              disabled={demo}
              className="rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary disabled:opacity-50"
            />
          </label>
          <label className="block flex-1">
            <span className="mb-1 block text-xs font-medium text-muted">Image</span>
            <input
              type="file"
              accept="image/*"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              disabled={demo}
              className="block w-full text-sm text-muted file:mr-3 file:rounded-full file:border file:border-line file:bg-surface-2 file:px-4 file:py-2 file:text-sm file:font-semibold file:text-ink disabled:opacity-50"
            />
          </label>
          <Button onClick={submit} disabled={demo || uploadPhoto.isPending || !file}>
            {uploadPhoto.isPending ? 'Uploading…' : 'Upload'}
          </Button>
        </div>
        {demo && <p className="mt-3 text-sm text-muted">{DEMO_NOTE}</p>}
        {!demo && list.length > 0 && (
          <p className="mt-3 text-sm text-muted">Tap a photo to delete it.</p>
        )}
      </Card>

      <Card className="p-5">
        <Eyebrow>Timeline</Eyebrow>
        <div className="mt-4">
          {photos.isLoading ? <Spinner /> : <PhotoGrid photos={list} onSelect={onSelect} />}
        </div>
      </Card>

      {comparing && list.length >= 2 && (
        <PhotoCompare photos={list} onClose={() => setComparing(false)} />
      )}
    </div>
  )
}
