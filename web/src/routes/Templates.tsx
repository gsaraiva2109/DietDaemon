// Templates, saved meals you can re-log with one tap. List, log, and delete.

import { useState } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import type { MealTemplate } from '@/lib/types'
import { useTemplates, useLogTemplate, useDeleteTemplate } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Card, Button, Pill, Spinner, EmptyState } from '@/components/ui'
import { TemplateIcon, TrashIcon, LogIcon, CheckIcon } from '@/components/icons'
import { ComposeTemplateModal } from '@/components/ComposeTemplateModal'
import { stagger, fadeUp } from '@/lib/motion'
import { formatNumber, relativeTime } from '@/lib/format'

function templateKcal(t: MealTemplate): number {
  return t.items.reduce((s, it) => s + (it.Macros?.Calories ?? 0), 0)
}

// Error message is either the log or the delete mutation's Error, whichever
// fired, or a generic fallback. Extracted so the nested `instanceof` ternary
// isn't inline in the JSX.
function templateErrorMessage(logError: unknown, delError: unknown, fallback: string): string {
  if (logError instanceof Error) return logError.message
  if (delError instanceof Error) return delError.message
  return fallback
}

// Loading/empty/list ladder for the templates list, split out of Templates()
// so the ternary there becomes a plain if/else chain.
function TemplatesBody({
  loading,
  templates,
  demo,
}: Readonly<{ loading: boolean; templates: MealTemplate[]; demo: boolean }>) {
  const { t } = useTranslation()

  if (loading) return <Spinner label={t('templates.loading')} />
  if (!templates.length) {
    return <EmptyState title={t('templates.emptyTitle')} hint={t('templates.emptyHint')} icon={<TemplateIcon />} />
  }
  return (
    <motion.div variants={stagger} initial="hidden" animate="show" className="flex flex-col gap-2.5">
      {templates.map((tpl) => (
        <motion.div key={tpl.id} variants={fadeUp}>
          <TemplateRow template={tpl} demo={demo} />
        </motion.div>
      ))}
    </motion.div>
  )
}

export function Templates() {
  const { t } = useTranslation()
  const templates = useTemplates()
  const { demo } = useDemo()
  const [composing, setComposing] = useState(false)

  return (
    <div>
      <PageHeader eyebrow={t('templates.eyebrow')} title={t('templates.title')}>
        {!demo && (
          <Button onClick={() => setComposing(true)} className="px-4 py-2 text-sm">
            {t('templates.newFromScratch')}
          </Button>
        )}
      </PageHeader>

      {composing && <ComposeTemplateModal onClose={() => setComposing(false)} />}

      <TemplatesBody loading={templates.isLoading} templates={templates.data ?? []} demo={demo} />
    </div>
  )
}

// The action area (Log/Delete vs. their confirm steps vs. the post-log
// "Logged" pill) is a 4-way mutually exclusive state, same treatment as
// TemplatesBody: if/else instead of a nested ternary.
function TemplateActions({
  logged,
  confirming,
  logPending,
  delPending,
  templateName,
  onLog,
  onDelete,
  onStartLog,
  onStartDelete,
  onCancel,
}: Readonly<{
  logged: boolean
  confirming: null | 'log' | 'delete'
  logPending: boolean
  delPending: boolean
  templateName: string
  onLog: () => void
  onDelete: () => void
  onStartLog: () => void
  onStartDelete: () => void
  onCancel: () => void
}>) {
  const { t } = useTranslation()

  if (logged) {
    return (
      <Pill tone="primary">
        <CheckIcon width={14} height={14} /> {t('templates.logged')}
      </Pill>
    )
  }
  if (confirming === 'log') {
    return (
      <div className="flex items-center gap-1">
        <Button onClick={onLog} disabled={logPending} className="px-3 py-1.5 text-xs">
          {logPending ? t('templates.logging') : t('templates.confirm')}
        </Button>
        <Button variant="ghost" onClick={onCancel} className="px-3 py-1.5 text-xs">
          {t('templates.cancel')}
        </Button>
      </div>
    )
  }
  if (confirming === 'delete') {
    return (
      <div className="flex items-center gap-1">
        <Button
          onClick={onDelete}
          disabled={delPending}
          className="bg-accent px-3 py-1.5 text-xs text-white hover:brightness-[1.05]"
        >
          {delPending ? t('templates.deleting') : t('templates.delete')}
        </Button>
        <Button variant="ghost" onClick={onCancel} className="px-3 py-1.5 text-xs">
          {t('templates.cancel')}
        </Button>
      </div>
    )
  }
  return (
    <>
      <Button onClick={onStartLog} disabled={logPending} className="px-3 py-1.5 text-xs">
        <LogIcon width={15} height={15} /> {t('templates.log')}
      </Button>
      <button
        onClick={onStartDelete}
        disabled={delPending}
        aria-label={t('templates.deleteAria', { name: templateName })}
        className="grid size-8 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
      >
        <TrashIcon width={16} height={16} />
      </button>
    </>
  )
}

function TemplateRow({ template, demo }: Readonly<{ template: MealTemplate; demo: boolean }>) {
  const { t, i18n } = useTranslation()
  const log = useLogTemplate()
  const del = useDeleteTemplate()
  const [confirming, setConfirming] = useState<null | 'log' | 'delete'>(null)
  const [logged, setLogged] = useState(false)

  const kcal = templateKcal(template)
  const itemCount = template.items.length

  function doLog() {
    setConfirming(null)
    log.mutate({ id: template.id }, {
      onSuccess: () => {
        setLogged(true)
        window.setTimeout(() => setLogged(false), 2200)
      },
    })
  }

  function doDelete() {
    setConfirming(null)
    del.mutate(template.id)
  }

  return (
    <Card className="p-4">
      <div className="flex items-center gap-4">
        <div className="min-w-0 flex-1">
          <p className="truncate font-semibold text-ink">{template.name}</p>
          <p className="mt-0.5 text-sm text-muted">
            {itemCount} {itemCount === 1 ? t('templates.item') : t('templates.items')} ·{' '}
            {formatNumber(kcal)} kcal ·{' '}
            {t('templates.usedAt', { time: relativeTime(template.last_used, t, i18n.language) })}
          </p>
        </div>

        {!demo && (
          <div className="flex shrink-0 items-center gap-1.5">
            <TemplateActions
              logged={logged}
              confirming={confirming}
              logPending={log.isPending}
              delPending={del.isPending}
              templateName={template.name}
              onLog={doLog}
              onDelete={doDelete}
              onStartLog={() => setConfirming('log')}
              onStartDelete={() => setConfirming('delete')}
              onCancel={() => setConfirming(null)}
            />
          </div>
        )}
      </div>

      {(log.isError || del.isError) && (
        <p className="mt-2 text-sm font-medium text-accent" role="alert">
          {templateErrorMessage(log.error, del.error, t('templates.genericError'))}
        </p>
      )}
    </Card>
  )
}
