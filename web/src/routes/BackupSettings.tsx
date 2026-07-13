// Scheduled backup settings: enable/disable, pick a destination (local disk
// or S3), set the interval, and a manual "run now" trigger. A settings
// sub-page, same shape as Aliases.tsx / Security.tsx (back link + PageHeader).

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
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

function formatLastRun(iso: string, neverLabel: string, locale: string): string {
  if (!iso || iso.startsWith('0001-01-01')) return neverLabel
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return neverLabel
  return d.toLocaleString(locale)
}

export function BackupSettings() {
  const { t, i18n } = useTranslation()
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
        <ChevronLeft width={18} height={18} /> {t('nav.settings')}
      </Link>

      <PageHeader eyebrow={t('nav.settings')} title={t('backupSettings.title')} />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          {t('backupSettings.readOnly')}
        </p>
      )}

      {query.isLoading ? (
        <Spinner label={t('backupSettings.loading')} />
      ) : (
        <>
          <Card className="mb-5 p-5">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h2 className="font-semibold text-ink">{t('backupSettings.scheduledBackup')}</h2>
                <p className="mt-0.5 text-sm text-muted">
                  {t('backupSettings.scheduledBackupDescription')}
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
                {t('backupSettings.enabled')}
              </label>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <span className="mb-2 block text-xs font-medium text-muted">{t('backupSettings.destination')}</span>
                <div role="radiogroup" aria-label={t('backupSettings.destination')} className="inline-flex gap-1 rounded-full bg-surface-2 p-1">
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
                        {d === 'local' ? t('backupSettings.local') : t('backupSettings.s3')}
                      </button>
                    )
                  })}
                </div>
              </div>

              <Field
                label={t('backupSettings.intervalHours')}
                type="number"
                min={1}
                value={values.IntervalHrs}
                disabled={demo}
                onChange={(e) => set('IntervalHrs', Number(e.target.value))}
              />

              {values.Destination === 'local' ? (
                <Field
                  label={t('backupSettings.subdirectory')}
                  hint={t('backupSettings.subdirectoryHint')}
                  value={values.LocalSubdir}
                  disabled={demo}
                  onChange={(e) => set('LocalSubdir', e.target.value)}
                  placeholder={t('backupSettings.subdirectoryPlaceholder')}
                  className="sm:col-span-2"
                />
              ) : (
                <>
                  <Field
                    label={t('backupSettings.bucket')}
                    value={values.S3Bucket}
                    disabled={demo}
                    onChange={(e) => set('S3Bucket', e.target.value)}
                    placeholder="my-backups-bucket"
                  />
                  <Field
                    label={t('backupSettings.prefix')}
                    value={values.S3Prefix}
                    disabled={demo}
                    onChange={(e) => set('S3Prefix', e.target.value)}
                    placeholder="dietdaemon/alice"
                  />
                  <Field
                    label={t('backupSettings.region')}
                    hint={t('backupSettings.regionHint')}
                    value={values.S3Region}
                    disabled={demo}
                    onChange={(e) => set('S3Region', e.target.value)}
                    placeholder="us-east-1"
                  />
                  <Field
                    label={t('backupSettings.endpoint')}
                    hint={t('backupSettings.endpointHint')}
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
                {setConfig.isPending ? t('backupSettings.saving') : t('backupSettings.save')}
              </Button>
              {setConfig.isSuccess && !draft && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                  {t('backupSettings.saved')}
                </motion.span>
              )}
              {setConfig.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setConfig.error instanceof Error ? setConfig.error.message : t('backupSettings.saveFailed')}
                </span>
              )}
            </div>
          </Card>

          <Card className="p-5">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-sm text-muted">
                <ClockIcon width={16} height={16} />
                {t('backupSettings.lastRun', { time: formatLastRun(server.LastRunAt, t('backupSettings.neverRunYet'), i18n.language) })}
              </div>
              <Button
                variant="ghost"
                onClick={() => runNow.mutate()}
                disabled={demo || runNow.isPending}
              >
                {runNow.isPending ? t('backupSettings.running') : t('backupSettings.runNow')}
              </Button>
            </div>
            {runNow.isSuccess && (
              <p className="mt-3 text-sm font-medium text-primary">{t('backupSettings.backupCompleted')}</p>
            )}
            {runNow.isError && (
              <p className="mt-3 text-sm font-medium text-accent" role="alert">
                {runNow.error instanceof Error ? runNow.error.message : t('backupSettings.backupFailed')}
              </p>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
