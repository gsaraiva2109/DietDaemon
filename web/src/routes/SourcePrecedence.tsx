// Source precedence, per-user order for the external nutrition sources the
// resolver falls back to when a food isn't in the personal library. Few
// sources exist, so plain up/down buttons cover reordering without pulling in
// a drag-and-drop dependency. A settings sub-page, mirrors Settings.tsx's
// local-draft-then-save pattern for targets.

import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useDemo } from '@/lib/demo'
import { usePrecedence, useSetPrecedence } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Card, Button, Spinner } from '@/components/ui'
import { ChevronLeft, ChevronDown } from '@/components/icons'
import { NUTRITION_SOURCES, SOURCE_LABELS } from '@/lib/types'

export function SourcePrecedence() {
  const { demo } = useDemo()
  const { data, isLoading } = usePrecedence()
  const setPrecedence = useSetPrecedence()

  const serverOrder = data?.order.length ? data.order : [...NUTRITION_SOURCES]
  const [draft, setDraft] = useState<string[] | null>(null)
  const order = draft ?? serverOrder

  // Drop the local draft whenever the server value moves underneath us (after
  // a successful save, or a demo-mode toggle), so "dirty" reflects reality.
  useEffect(() => {
    setDraft(null)
  }, [data])

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
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Nutrition source order" />

      <p className="mb-6 max-w-prose text-sm text-muted">
        When a food isn't already in your library, these sources are tried in order until one has
        a match. Move your preferred source to the top.
      </p>

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          Source order is read only here.
        </p>
      )}

      {isLoading ? (
        <Spinner label="Loading source order" />
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
                      aria-label={`Move ${SOURCE_LABELS[source] ?? source} up`}
                      className="grid size-8 place-items-center rounded-full text-muted transition hover:bg-surface-2 hover:text-ink disabled:opacity-30"
                    >
                      <ChevronDown width={16} height={16} className="rotate-180" />
                    </button>
                    <button
                      onClick={() => move(i, 1)}
                      disabled={i === order.length - 1}
                      aria-label={`Move ${SOURCE_LABELS[source] ?? source} down`}
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
                {setPrecedence.isPending ? 'Saving…' : 'Save order'}
              </Button>
              {setPrecedence.isSuccess && !dirty && (
                <motion.span
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="text-sm font-medium text-primary"
                >
                  Saved.
                </motion.span>
              )}
              {setPrecedence.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setPrecedence.error instanceof Error ? setPrecedence.error.message : 'Failed to save'}
                </span>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}
