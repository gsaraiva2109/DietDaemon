// Pending aliases, review queue for embedding near-misses. The resolver
// queues a strong-but-unconfirmed phrase -> food match here instead of
// silently writing it into the personal food library; the user confirms or
// rejects each one. A settings sub-page, mirrors Aliases.tsx.

import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useDemo } from '@/lib/demo'
import { usePendingAliases, useConfirmPendingAlias, useRejectPendingAlias } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Card, EmptyState, Spinner, Pill } from '@/components/ui'
import { ChevronLeft, CheckIcon, CloseIcon } from '@/components/icons'
import { stagger, fadeUp } from '@/lib/motion'
import type { PendingAlias } from '@/lib/types'

export function PendingAliases() {
  const { t } = useTranslation()
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
        <ChevronLeft width={18} height={18} /> {t('nav.settings')}
      </Link>

      <PageHeader eyebrow={t('nav.settings')} title={t('pendingAliases.title')} />

      <p className="mb-6 max-w-prose text-sm text-muted">
        {t('pendingAliases.description')}
      </p>

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          {t('pendingAliases.readOnly')}
        </p>
      )}

      {isLoading ? (
        <Spinner label={t('pendingAliases.loading')} />
      ) : !pending.length ? (
        <EmptyState
          title={t('pendingAliases.emptyTitle')}
          hint={t('pendingAliases.emptyHint')}
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
  const { t } = useTranslation()
  const confirm = useConfirmPendingAlias()
  const reject = useRejectPendingAlias()
  const busy = confirm.isPending || reject.isPending

  return (
    <Card className="flex items-center gap-4 p-4">
      <div className="min-w-0 flex-1">
        <p className="truncate font-semibold text-ink">"{alias.phrase}"</p>
        <p className="mt-0.5 truncate text-sm text-muted">
          {t('pendingAliases.matches', { food: alias.food_name })}
        </p>
      </div>
      <Pill tone="primary">
        {t('pendingAliases.matchPercent', { percent: Math.round(alias.match_score * 100) })}
      </Pill>
      {!demo && (
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => confirm.mutate(alias.id)}
            disabled={busy}
            aria-label={t('pendingAliases.confirmAriaLabel', { phrase: alias.phrase })}
            className="grid size-9 place-items-center rounded-full text-primary transition hover:bg-primary-soft disabled:opacity-50"
          >
            <CheckIcon width={18} height={18} />
          </button>
          <button
            onClick={() => reject.mutate(alias.id)}
            disabled={busy}
            aria-label={t('pendingAliases.rejectAriaLabel', { phrase: alias.phrase })}
            className="grid size-9 place-items-center rounded-full text-muted transition hover:bg-accent/12 hover:text-accent disabled:opacity-50"
          >
            <CloseIcon width={18} height={18} />
          </button>
        </div>
      )}
    </Card>
  )
}
