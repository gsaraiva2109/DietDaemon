// Pending aliases, review queue for embedding near-misses. The resolver
// queues a strong-but-unconfirmed phrase -> food match here instead of
// silently writing it into the personal food library; the user confirms or
// rejects each one. A settings sub-page, mirrors Aliases.tsx.

import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useDemo } from '@/lib/demo'
import { usePendingAliases, useConfirmPendingAlias, useRejectPendingAlias } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Card, EmptyState, Spinner, Pill } from '@/components/ui'
import { ChevronLeft, CheckIcon, CloseIcon } from '@/components/icons'
import { stagger, fadeUp } from '@/lib/motion'
import type { PendingAlias } from '@/lib/types'

export function PendingAliases() {
  const { demo } = useDemo()
  const { data, isLoading } = usePendingAliases()
  const pending = data ?? []

  return (
    <div>
      <Link
        to="/settings"
        prefetch="intent"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Pending aliases" />

      <p className="mb-6 max-w-prose text-sm text-muted">
        When a new phrase closely matches a food you already have, it waits here for your
        confirmation instead of being learned automatically.
      </p>

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          Pending aliases are read only here.
        </p>
      )}

      {isLoading ? (
        <Spinner label="Loading pending aliases" />
      ) : !pending.length ? (
        <EmptyState
          title="Nothing waiting for review"
          hint="New near-miss matches will show up here for confirmation."
        />
      ) : (
        <motion.div variants={stagger} initial="hidden" animate="show" className="flex flex-col gap-3">
          {pending.map((pa: PendingAlias) => (
            <motion.div key={pa.id} variants={fadeUp}>
              <PendingAliasRow alias={pa} demo={demo} />
            </motion.div>
          ))}
        </motion.div>
      )}
    </div>
  )
}

function PendingAliasRow({ alias, demo }: { alias: PendingAlias; demo: boolean }) {
  const confirm = useConfirmPendingAlias()
  const reject = useRejectPendingAlias()
  const busy = confirm.isPending || reject.isPending

  return (
    <Card className="flex items-center gap-4 p-4">
      <div className="min-w-0 flex-1">
        <p className="truncate font-semibold text-ink">"{alias.phrase}"</p>
        <p className="mt-0.5 truncate text-sm text-muted">matches {alias.food_name}</p>
      </div>
      <Pill tone="primary">{Math.round(alias.match_score * 100)}% match</Pill>
      {!demo && (
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => confirm.mutate(alias.id)}
            disabled={busy}
            aria-label={`Confirm alias for ${alias.phrase}`}
            className="grid size-9 place-items-center rounded-full text-primary transition hover:bg-primary-soft disabled:opacity-50"
          >
            <CheckIcon width={18} height={18} />
          </button>
          <button
            onClick={() => reject.mutate(alias.id)}
            disabled={busy}
            aria-label={`Reject alias for ${alias.phrase}`}
            className="grid size-9 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
          >
            <CloseIcon width={18} height={18} />
          </button>
        </div>
      )}
    </Card>
  )
}
