// Export & Share — Phase 6. Pulls meals or daily rollups as CSV/JSON over a
// date range and saves them to disk. Hits the real API, so it's disabled in
// demo mode (the demo dataset is synthetic and has no backing export route).

import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { api, triggerDownload } from '@/lib/api'
import { useDemo } from '@/lib/demo'
import { scaleIn } from '@/lib/motion'
import { Button } from './ui'
import { CloseIcon, DownloadIcon } from './icons'

type DataType = 'meals' | 'rollups'
type Format = 'csv' | 'json'

/** Returns an ISO YYYY-MM-DD string for `now` shifted back by `daysAgo` days. */
function isoDaysAgo(daysAgo: number): string {
  const d = new Date()
  d.setDate(d.getDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

function SegmentedPills<T extends string>({
  options,
  value,
  onChange,
  label,
}: {
  options: { value: T; label: string }[]
  value: T
  onChange: (v: T) => void
  label: string
}) {
  return (
    <div role="radiogroup" aria-label={label} className="inline-flex gap-1 rounded-full bg-surface-2 p-1">
      {options.map((o) => {
        const active = o.value === value
        return (
          <button
            key={o.value}
            role="radio"
            aria-checked={active}
            onClick={() => onChange(o.value)}
            className={`rounded-full px-4 py-1.5 text-sm font-semibold transition ${
              active ? 'bg-primary text-primary-ink' : 'text-muted hover:text-ink'
            }`}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}

export function ExportModal({ onClose }: { onClose: () => void }) {
  const { demo } = useDemo()
  const [dataType, setDataType] = useState<DataType>('meals')
  const [format, setFormat] = useState<Format>('csv')
  const [start, setStart] = useState(() => isoDaysAgo(29))
  const [end, setEnd] = useState(() => isoDaysAgo(0))
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  async function download() {
    if (demo || pending) return
    setError(null)
    setPending(true)
    try {
      const blob =
        dataType === 'meals'
          ? await api.export.meals(format, start, end)
          : await api.export.rollups(format, start, end)
      triggerDownload(blob, `dietdaemon-${dataType}-${start}_${end}.${format}`)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Export failed')
    } finally {
      setPending(false)
    }
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
          aria-label="Export data"
          variants={scaleIn}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative w-full max-w-md rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                Export
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">Download your data</h2>
            </div>
            <button onClick={onClose} aria-label="Close" className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          <div className="space-y-5">
            <div>
              <span className="mb-2 block text-xs font-medium text-muted">Data</span>
              <SegmentedPills<DataType>
                label="Data type"
                value={dataType}
                onChange={setDataType}
                options={[
                  { value: 'meals', label: 'Meals' },
                  { value: 'rollups', label: 'Rollups' },
                ]}
              />
            </div>

            <div>
              <span className="mb-2 block text-xs font-medium text-muted">Format</span>
              <SegmentedPills<Format>
                label="Format"
                value={format}
                onChange={setFormat}
                options={[
                  { value: 'csv', label: 'CSV' },
                  { value: 'json', label: 'JSON' },
                ]}
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-muted">Start</span>
                <input
                  type="date"
                  value={start}
                  max={end}
                  onChange={(e) => setStart(e.target.value)}
                  className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none focus:border-primary tnum"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-xs font-medium text-muted">End</span>
                <input
                  type="date"
                  value={end}
                  min={start}
                  onChange={(e) => setEnd(e.target.value)}
                  className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-ink outline-none focus:border-primary tnum"
                />
              </label>
            </div>
          </div>

          {error && (
            <p className="mt-4 text-sm font-medium text-accent" role="alert">
              {error}
            </p>
          )}

          {demo && (
            <p className="mt-4 text-sm text-muted">
              Export is unavailable in demo mode — connect a real account to download your data.
            </p>
          )}

          <div className="mt-6 flex justify-end gap-2">
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={download} disabled={demo || pending}>
              {pending ? (
                <>
                  <span className="size-4 animate-spin rounded-full border-2 border-primary-ink/40 border-t-primary-ink" />
                  Exporting…
                </>
              ) : (
                <>
                  <DownloadIcon width={18} height={18} />
                  Download
                </>
              )}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
