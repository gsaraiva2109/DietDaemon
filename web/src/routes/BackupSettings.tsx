// Scheduled backup settings: enable/disable, pick a destination (local disk
// or S3), set the interval, and a manual "run now" trigger. A settings
// sub-page, same shape as Aliases.tsx / Security.tsx (back link + PageHeader).

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useBackupConfig, useSetBackupConfig, useRunBackupNow } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Field, Spinner } from '@/components/ui'
import { ChevronLeft, ClockIcon } from '@/components/icons'
import type { BackupConfig } from '@/lib/types'

const DEFAULTS: BackupConfig = {
  UserID: '',
  Enabled: false,
  Destination: 'local',
  LocalSubdir: '',
  S3Bucket: '',
  S3Prefix: '',
  S3Region: '',
  S3Endpoint: '',
  IntervalHrs: 24,
  LastRunAt: '',
}

function formatLastRun(iso: string): string {
  if (!iso || iso.startsWith('0001-01-01')) return 'Never run yet'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return 'Never run yet'
  return d.toLocaleString()
}

export function BackupSettings() {
  const { demo } = useDemo()
  const query = useBackupConfig()
  const setConfig = useSetBackupConfig()
  const runNow = useRunBackupNow()

  const [draft, setDraft] = useState<BackupConfig | null>(null)
  const server = query.data ?? DEFAULTS
  const values = draft ?? server

  // Reset the draft whenever fresh server data lands (e.g. after a save).
  // Adjusting state during render (React's documented pattern) instead of an
  // effect, since setting state synchronously in an effect double-renders.
  const [prevData, setPrevData] = useState(query.data)
  if (query.data && query.data !== prevData) {
    setPrevData(query.data)
    setDraft(null)
  }

  function set<K extends keyof BackupConfig>(key: K, value: BackupConfig[K]) {
    setDraft({ ...values, [key]: value })
  }

  function save() {
    if (demo) return
    setConfig.mutate(values)
  }

  return (
    <div>
      <Link
        to="/settings"
        prefetch="intent"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Backup" />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          Backup settings are read only here.
        </p>
      )}

      {query.isLoading ? (
        <Spinner label="Loading backup settings" />
      ) : (
        <>
          <Card className="mb-5 p-5">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h2 className="font-semibold text-ink">Scheduled backup</h2>
                <p className="mt-0.5 text-sm text-muted">
                  Automatically export your meals and daily rollups on a recurring schedule.
                </p>
              </div>
              <label className="flex items-center gap-2 text-sm font-medium text-ink">
                <input
                  type="checkbox"
                  checked={values.Enabled}
                  disabled={demo}
                  onChange={(e) => set('Enabled', e.target.checked)}
                  className="size-4 rounded border-line accent-primary disabled:opacity-60"
                />
                Enabled
              </label>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <span className="mb-2 block text-xs font-medium text-muted">Destination</span>
                <div role="radiogroup" aria-label="Destination" className="inline-flex gap-1 rounded-full bg-surface-2 p-1">
                  {(['local', 's3'] as const).map((d) => {
                    const active = values.Destination === d
                    return (
                      <button
                        key={d}
                        role="radio"
                        aria-checked={active}
                        disabled={demo}
                        onClick={() => set('Destination', d)}
                        className={`rounded-full px-4 py-1.5 text-sm font-semibold transition disabled:opacity-60 ${
                          active ? 'bg-primary text-primary-ink' : 'text-muted hover:text-ink'
                        }`}
                      >
                        {d === 'local' ? 'Local disk' : 'S3'}
                      </button>
                    )
                  })}
                </div>
              </div>

              <Field
                label="Interval (hours)"
                type="number"
                min={1}
                value={values.IntervalHrs}
                disabled={demo}
                onChange={(e) => set('IntervalHrs', Number(e.target.value))}
              />

              {values.Destination === 'local' ? (
                <Field
                  label="Subdirectory"
                  hint="Under the server's configured backup directory."
                  value={values.LocalSubdir}
                  disabled={demo}
                  onChange={(e) => set('LocalSubdir', e.target.value)}
                  placeholder="e.g. alice"
                  className="sm:col-span-2"
                />
              ) : (
                <>
                  <Field
                    label="Bucket"
                    value={values.S3Bucket}
                    disabled={demo}
                    onChange={(e) => set('S3Bucket', e.target.value)}
                    placeholder="my-backups-bucket"
                  />
                  <Field
                    label="Prefix"
                    value={values.S3Prefix}
                    disabled={demo}
                    onChange={(e) => set('S3Prefix', e.target.value)}
                    placeholder="dietdaemon/alice"
                  />
                  <Field
                    label="Region"
                    hint="Leave blank to use the server default."
                    value={values.S3Region}
                    disabled={demo}
                    onChange={(e) => set('S3Region', e.target.value)}
                    placeholder="us-east-1"
                  />
                  <Field
                    label="Endpoint"
                    hint="For S3-compatible stores (e.g. MinIO). Leave blank for AWS."
                    value={values.S3Endpoint}
                    disabled={demo}
                    onChange={(e) => set('S3Endpoint', e.target.value)}
                    placeholder="https://minio.example.com"
                  />
                </>
              )}
            </div>

            <div className="mt-5 flex items-center gap-3">
              <Button onClick={save} disabled={demo || setConfig.isPending}>
                {setConfig.isPending ? 'Saving…' : 'Save'}
              </Button>
              {setConfig.isSuccess && !draft && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                  Saved.
                </motion.span>
              )}
              {setConfig.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setConfig.error instanceof Error ? setConfig.error.message : 'Failed to save'}
                </span>
              )}
            </div>
          </Card>

          <Card className="p-5">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-sm text-muted">
                <ClockIcon width={16} height={16} />
                Last run: {formatLastRun(server.LastRunAt)}
              </div>
              <Button
                variant="ghost"
                onClick={() => runNow.mutate()}
                disabled={demo || runNow.isPending}
              >
                {runNow.isPending ? 'Running…' : 'Run now'}
              </Button>
            </div>
            {runNow.isSuccess && (
              <p className="mt-3 text-sm font-medium text-primary">Backup completed.</p>
            )}
            {runNow.isError && (
              <p className="mt-3 text-sm font-medium text-accent" role="alert">
                {runNow.error instanceof Error ? runNow.error.message : 'Backup failed'}
              </p>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
