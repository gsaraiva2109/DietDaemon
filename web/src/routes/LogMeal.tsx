// Log a meal as natural text. POST is async (202); we show an accepted state
// and let the dashboard/history pick up the result on the next poll.

import { useState, type FormEvent } from 'react'
import { motion } from 'framer-motion'
import { useLogMeal } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card } from '@/components/ui'
import { fadeUp } from '@/lib/motion'

const EXAMPLES = ['200g frango grelhado, 2 ovos, 150g arroz', '1 banana e um copo de leite', '3 slices of pizza']

export function LogMeal() {
  const [text, setText] = useState('')
  const log = useLogMeal()

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!text.trim()) return
    log.mutate(text.trim(), { onSuccess: () => setText('') })
  }

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
    </div>
  )
}
