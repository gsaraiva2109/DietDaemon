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

      {templates.isLoading ? (
        <Spinner label={t('templates.loading')} />
      ) : !templates.data?.length ? (
        <EmptyState
          title={t('templates.emptyTitle')}
          hint={t('templates.emptyHint')}
          icon={<TemplateIcon />}
        />
      ) : (
        <motion.div
          variants={stagger}
          initial="hidden"
          animate="show"
          className="flex flex-col gap-2.5"
        >
          {templates.data.map((t: MealTemplate) => (
            <motion.div key={t.id} variants={fadeUp}>
              <TemplateRow template={t} demo={demo} />
            </motion.div>
          ))}
        </motion.div>
      )}
    </div>
  )
}

function TemplateRow({ template, demo }: { template: MealTemplate; demo: boolean }) {
  const { t, i18n } = useTranslation()
  const log = useLogTemplate()
  const del = useDeleteTemplate()
  const [confirming, setConfirming] = useState<null | 'log' | 'delete'>(null)
  const [logged, setLogged] = useState(false)

  const kcal = templateKcal(template)
  const itemCount = template.items.length

  function doLog() {
    setConfirming(null)
    log.mutate(template.id, {
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
            {logged ? (
              <Pill tone="primary">
                <CheckIcon width={14} height={14} /> {t('templates.logged')}
              </Pill>
            ) : confirming === 'log' ? (
              <div className="flex items-center gap-1">
                <Button onClick={doLog} disabled={log.isPending} className="px-3 py-1.5 text-xs">
                  {log.isPending ? t('templates.logging') : t('templates.confirm')}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => setConfirming(null)}
                  className="px-3 py-1.5 text-xs"
                >
                  {t('templates.cancel')}
                </Button>
              </div>
            ) : confirming === 'delete' ? (
              <div className="flex items-center gap-1">
                <Button
                  onClick={doDelete}
                  disabled={del.isPending}
                  className="bg-accent px-3 py-1.5 text-xs text-white hover:brightness-[1.05]"
                >
                  {del.isPending ? t('templates.deleting') : t('templates.delete')}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => setConfirming(null)}
                  className="px-3 py-1.5 text-xs"
                >
                  {t('templates.cancel')}
                </Button>
              </div>
            ) : (
              <>
                <Button
                  onClick={() => setConfirming('log')}
                  disabled={log.isPending}
                  className="px-3 py-1.5 text-xs"
                >
                  <LogIcon width={15} height={15} /> {t('templates.log')}
                </Button>
                <button
                  onClick={() => setConfirming('delete')}
                  disabled={del.isPending}
                  aria-label={t('templates.deleteAria', { name: template.name })}
                  className="grid size-8 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
                >
                  <TrashIcon width={16} height={16} />
                </button>
              </>
            )}
          </div>
        )}
      </div>

      {(log.isError || del.isError) && (
        <p className="mt-2 text-sm font-medium text-accent" role="alert">
          {log.error instanceof Error
            ? log.error.message
            : del.error instanceof Error
              ? del.error.message
              : t('templates.genericError')}
        </p>
      )}
    </Card>
  )
}
