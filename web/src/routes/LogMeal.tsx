// Log a meal as natural text. POST is async (202); we show an accepted state
// and let the dashboard/history pick up the result on the next poll.

import { useState, type FormEvent } from 'react'
import { motion } from 'framer-motion'
import { useSearchParams } from 'react-router-dom'
import { useLogMeal, useTemplates, useLogTemplate } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card } from '@/components/ui'
import { DuplicateMealModal } from '@/components/DuplicateMealModal'
import type { MealTemplate } from '@/lib/types'
import { TemplateIcon, CopyIcon } from '@/components/icons'
import { fadeUp } from '@/lib/motion'

const EXAMPLES = ['200g frango grelhado, 2 ovos, 150g arroz', '1 banana e um copo de leite', '3 slices of pizza']

export function LogMeal() {
  const [params] = useSearchParams()
  // Pre-fill from a deep link (e.g. "Log this" on a food / frequent-food pill).
  const [text, setText] = useState(() => params.get('text') ?? '')
  const log = useLogMeal()
  const templates = useTemplates()
  const logTemplate = useLogTemplate()
  const { demo } = useDemo()
  const [duplicating, setDuplicating] = useState(false)

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!text.trim()) return
    log.mutate(text.trim(), { onSuccess: () => setText('') })
  }

  const recentTemplates = (templates.data ?? []).slice(0, 6)

  return (
    <div>
      <PageHeader eyebrow="Log" title="What did you eat?" />
      <Card className="p-5">
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            rows={3}
            placeholder="e.g. 200g chicken, 2 eggs, 150g rice"
            aria-label="Meal description"
            className="w-full resize-none rounded-lg border border-line bg-bg px-4 py-3 text-lg text-ink outline-none transition focus:border-primary"
          />
          <div className="flex items-center justify-between gap-3">
            <p className="text-xs text-muted">Plain language. The parser handles quantities &amp; units.</p>
            <Button type="submit" disabled={log.isPending || !text.trim()}>
              {log.isPending ? 'Sending…' : 'Log meal'}
            </Button>
          </div>
        </form>

        {log.isSuccess && (
          <motion.p
            variants={fadeUp}
            initial="hidden"
            animate="show"
            className="mt-4 rounded-lg bg-primary-soft px-4 py-3 text-sm font-medium text-primary"
          >
            Logged — processing now. It'll appear on Today in a moment.
          </motion.p>
        )}
        {log.isError && (
          <p className="mt-4 text-sm font-medium text-accent" role="alert">
            {log.error instanceof Error ? log.error.message : 'Failed to log meal'}
          </p>
        )}
      </Card>

      {/* Quick actions: log a saved template, or copy a meal from a past day. */}
      <div className="mt-6 flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">From template</p>
          <Button variant="ghost" onClick={() => setDuplicating(true)} className="px-3 py-1.5 text-xs">
            <CopyIcon width={15} height={15} /> Copy from day
          </Button>
        </div>
        {recentTemplates.length ? (
          <div className="flex flex-wrap gap-2">
            {recentTemplates.map((t: MealTemplate) => (
              <button
                key={t.id}
                type="button"
                disabled={demo || logTemplate.isPending}
                onClick={() => logTemplate.mutate(t.id)}
                className="inline-flex items-center gap-1.5 rounded-full border border-line bg-surface px-3 py-1.5 text-sm text-muted transition hover:text-ink disabled:opacity-50"
              >
                <TemplateIcon width={15} height={15} /> {t.name}
              </button>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted">No templates yet. Save one from a meal's detail page.</p>
        )}
        {logTemplate.isSuccess && (
          <p className="text-sm font-medium text-primary">Template logged — appearing on Today shortly.</p>
        )}
      </div>

      <div className="mt-6">
        <p className="mb-2 text-xs font-semibold uppercase tracking-[0.14em] text-muted">Examples</p>
        <div className="flex flex-wrap gap-2">
          {EXAMPLES.map((ex) => (
            <button
              key={ex}
              type="button"
              onClick={() => setText(ex)}
              className="rounded-full border border-line bg-surface px-3 py-1.5 text-sm text-muted transition hover:text-ink"
            >
              {ex}
            </button>
          ))}
        </div>
      </div>

      {duplicating && <DuplicateMealModal onClose={() => setDuplicating(false)} />}
    </div>
  )
}
