// Source precedence, per-user order for the external nutrition sources the
// resolver falls back to when a food isn't in the personal library. Few
// sources exist, so plain up/down buttons cover reordering without pulling in
// a drag-and-drop dependency. A settings sub-page, mirrors Settings.tsx's
// local-draft-then-save pattern for targets.

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useDemo } from '@/lib/demo'
import { usePrecedence, useSetPrecedence, useFoodImportStatus } from '@/lib/queries'
import { relativeTime } from '@/lib/format'
import { PageHeader } from '@/components/PageHeader'
import { Card, Button, Spinner, Pill, EmptyState } from '@/components/ui'
import { ChevronLeft, ChevronDown } from '@/components/icons'
import { NUTRITION_SOURCES, SOURCE_LABELS } from '@/lib/types'

// Tone + label per FoodImportStatus.last_result. Pill only ships
// neutral/primary/accent/muted tones, so "changed_during_import" (a soft
// warning) shares neutral rather than inventing an amber tone.
const RESULT_TONE: Record<string, 'primary' | 'muted' | 'accent' | 'neutral'> = {
  imported: 'primary',
  skipped: 'muted',
  failed: 'accent',
  changed_during_import: 'neutral',
}

const RESULT_LABEL_KEY: Record<string, string> = {
  imported: 'sourcePrecedence.resultImported',
  skipped: 'sourcePrecedence.resultSkipped',
  failed: 'sourcePrecedence.resultFailed',
  changed_during_import: 'sourcePrecedence.resultChanged',
}

export function SourcePrecedence() {
  const { t, i18n } = useTranslation()
  const { demo } = useDemo()
  const { data, isLoading } = usePrecedence()
  const setPrecedence = useSetPrecedence()
  const importStatus = useFoodImportStatus()

  const serverOrder = data?.order.length ? data.order : [...NUTRITION_SOURCES]
  const [draft, setDraft] = useState<string[] | null>(null)
  const order = draft ?? serverOrder

  // Drop the local draft whenever the server value moves underneath us (after
  // a successful save, or a demo-mode toggle), so "dirty" reflects reality.
  // Adjusted during render (React's documented pattern) rather than an
  // effect, since setting state synchronously in an effect double-renders.
  const [prevData, setPrevData] = useState(data)
  if (data !== prevData) {
    setPrevData(data)
    setDraft(null)
  }

  function move(index: number, dir: -1 | 1) {
    const j = index + dir
    if (j < 0 || j >= order.length) return
    const next = [...order]
    ;[next[index], next[j]] = [next[j], next[index]]
    setDraft(next)
  }

  const dirty = draft !== null

  return (
    <div>
      <Link
        to="/settings"
        prefetch="intent"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> {t('nav.settings')}
      </Link>

      <PageHeader eyebrow={t('nav.settings')} title={t('sourcePrecedence.title')} />

      <p className="mb-6 max-w-prose text-sm text-muted">
        {t('sourcePrecedence.description')}
      </p>

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          {t('sourcePrecedence.readOnly')}
        </p>
      )}

      {isLoading ? (
        <Spinner label={t('sourcePrecedence.loading')} />
      ) : (
        <>
          <Card className="p-2">
            {order.map((source, i) => (
              <div key={source} className="flex items-center gap-3 rounded-lg px-3 py-3">
                <span className="flex-1 text-sm font-medium text-ink">
                  {SOURCE_LABELS[source] ?? source}
                </span>
                {!demo && (
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => move(i, -1)}
                      disabled={i === 0}
                      aria-label={t('sourcePrecedence.moveUp', { source: SOURCE_LABELS[source] ?? source })}
                      className="grid size-8 place-items-center rounded-full text-muted transition hover:bg-surface-2 hover:text-ink disabled:opacity-30"
                    >
                      <ChevronDown width={16} height={16} className="rotate-180" />
                    </button>
                    <button
                      onClick={() => move(i, 1)}
                      disabled={i === order.length - 1}
                      aria-label={t('sourcePrecedence.moveDown', { source: SOURCE_LABELS[source] ?? source })}
                      className="grid size-8 place-items-center rounded-full text-muted transition hover:bg-surface-2 hover:text-ink disabled:opacity-30"
                    >
                      <ChevronDown width={16} height={16} />
                    </button>
                  </div>
                )}
              </div>
            ))}
          </Card>

          {!demo && (
            <div className="mt-5 flex items-center gap-3">
              <Button
                onClick={() => setPrecedence.mutate(order)}
                disabled={!dirty || setPrecedence.isPending}
              >
                {setPrecedence.isPending ? t('sourcePrecedence.saving') : t('sourcePrecedence.saveOrder')}
              </Button>
              {setPrecedence.isSuccess && !dirty && (
                <motion.span
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="text-sm font-medium text-primary"
                >
                  {t('sourcePrecedence.saved')}
                </motion.span>
              )}
              {setPrecedence.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setPrecedence.error instanceof Error ? setPrecedence.error.message : t('sourcePrecedence.saveFailed')}
                </span>
              )}
            </div>
          )}

          <h2 className="mt-8 mb-2 font-semibold text-ink">{t('sourcePrecedence.importStatusTitle')}</h2>
          <p className="mb-4 max-w-prose text-sm text-muted">
            {t('sourcePrecedence.importStatusDescription')}
          </p>

          {importStatus.isLoading ? (
            <Spinner label={t('sourcePrecedence.importStatusLoading')} />
          ) : !importStatus.data?.length ? (
            <EmptyState
              title={t('sourcePrecedence.importStatusEmptyTitle')}
              hint={t('sourcePrecedence.importStatusEmptyHint')}
            />
          ) : (
            <Card className="p-2">
              {importStatus.data.map((s) => (
                <div key={s.source} className="border-t border-line px-3 py-3 first:border-t-0">
                  <div className="flex items-center gap-3">
                    <span className="flex-1 text-sm font-medium text-ink">
                      {SOURCE_LABELS[s.source] ?? s.source}
                    </span>
                    <Pill tone={RESULT_TONE[s.last_result] ?? 'neutral'}>
                      {t(RESULT_LABEL_KEY[s.last_result] ?? s.last_result)}
                    </Pill>
                    <span className="text-xs text-muted">{relativeTime(s.last_run_at, t, i18n.language)}</span>
                  </div>
                  {s.last_error && (
                    <p className="mt-1 truncate text-xs text-accent" title={s.last_error}>
                      {s.last_error}
                    </p>
                  )}
                </div>
              ))}
            </Card>
          )}
        </>
      )}
    </div>
  )
}
