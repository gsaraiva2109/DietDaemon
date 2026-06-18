// Templates — saved meals you can re-log with one tap. List, log, and delete.

import { useState } from 'react'
import { motion } from 'framer-motion'
import type { MealTemplate } from '@/lib/types'
import { useTemplates, useLogTemplate, useDeleteTemplate } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Card, Button, Pill, Spinner, EmptyState } from '@/components/ui'
import { TemplateIcon, TrashIcon, LogIcon, CheckIcon } from '@/components/icons'
import { stagger, fadeUp } from '@/lib/motion'
import { formatNumber, relativeTime } from '@/lib/format'

function templateKcal(t: MealTemplate): number {
  return t.items.reduce((s, it) => s + (it.Macros?.Calories ?? 0), 0)
}

export function Templates() {
  const templates = useTemplates()
  const { demo } = useDemo()

  return (
    <div>
      <PageHeader eyebrow="Templates" title="Saved meals" />

      {templates.isLoading ? (
        <Spinner label="Loading templates" />
      ) : !templates.data?.length ? (
        <EmptyState
          title="No templates yet"
          hint="Save a meal as a template from any meal's detail page."
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
            {itemCount} item{itemCount === 1 ? '' : 's'} · {formatNumber(kcal)} kcal · used{' '}
            {relativeTime(template.last_used)}
          </p>
        </div>

        {!demo && (
          <div className="flex shrink-0 items-center gap-1.5">
            {logged ? (
              <Pill tone="primary">
                <CheckIcon width={14} height={14} /> Logged
              </Pill>
            ) : confirming === 'log' ? (
              <div className="flex items-center gap-1">
                <Button onClick={doLog} disabled={log.isPending} className="px-3 py-1.5 text-xs">
                  {log.isPending ? 'Logging…' : 'Confirm'}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => setConfirming(null)}
                  className="px-3 py-1.5 text-xs"
                >
                  Cancel
                </Button>
              </div>
            ) : confirming === 'delete' ? (
              <div className="flex items-center gap-1">
                <Button
                  onClick={doDelete}
                  disabled={del.isPending}
                  className="bg-accent px-3 py-1.5 text-xs text-white hover:brightness-[1.05]"
                >
                  {del.isPending ? 'Deleting…' : 'Delete'}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => setConfirming(null)}
                  className="px-3 py-1.5 text-xs"
                >
                  Cancel
                </Button>
              </div>
            ) : (
              <>
                <Button
                  onClick={() => setConfirming('log')}
                  disabled={log.isPending}
                  className="px-3 py-1.5 text-xs"
                >
                  <LogIcon width={15} height={15} /> Log
                </Button>
                <button
                  onClick={() => setConfirming('delete')}
                  disabled={del.isPending}
                  aria-label={`Delete ${template.name}`}
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
              : 'Something went wrong'}
        </p>
      )}
    </Card>
  )
}
