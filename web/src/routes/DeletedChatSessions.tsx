// Settings > Recently deleted: chat sessions soft-deleted within the last 30
// days (#53). Same page shape as AssistantSettings (back link + PageHeader).

import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useDeletedChatSessions, useRestoreChatSession } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Card, EmptyState, Spinner } from '@/components/ui'
import { ChevronLeft, RestoreIcon, TrashIcon } from '@/components/icons'

export function DeletedChatSessions() {
  const { t } = useTranslation()
  const deleted = useDeletedChatSessions()
  const restore = useRestoreChatSession()

  return (
    <div>
      <Link to="/settings" prefetch="intent" className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink">
        <ChevronLeft width={18} height={18} /> {t('nav.settings')}
      </Link>

      <PageHeader eyebrow={t('nav.settings')} title={t('deletedChatSessions.title')} />
      <p className="mb-5 text-sm text-muted">
        {t('deletedChatSessions.description')}
      </p>

      {deleted.isLoading ? (
        <Spinner label={t('deletedChatSessions.loading')} />
      ) : deleted.isError ? (
        <EmptyState
          icon={<TrashIcon width={28} height={28} />}
          title={t('deletedChatSessions.loadErrorTitle')}
          hint={deleted.error instanceof Error ? deleted.error.message : t('deletedChatSessions.tryAgainLater')}
        />
      ) : !deleted.data?.length ? (
        <EmptyState
          icon={<TrashIcon width={28} height={28} />}
          title={t('deletedChatSessions.emptyTitle')}
          hint={t('deletedChatSessions.emptyHint')}
        />
      ) : (
        <Card className="divide-y divide-line p-0">
          {deleted.data.map((s) => (
            <div key={s.id} className="flex items-center gap-3 px-4 py-3">
              <span className="min-w-0 flex-1 truncate text-sm text-ink">{s.title || t('deletedChatSessions.newConversation')}</span>
              <button
                onClick={() => restore.mutate(s.id)}
                disabled={restore.isPending}
                className="inline-flex items-center gap-1.5 rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-surface-2 disabled:opacity-50"
              >
                <RestoreIcon width={14} height={14} /> {t('deletedChatSessions.restore')}
              </button>
            </div>
          ))}
        </Card>
      )}

      {restore.isError && (
        <p className="mt-3 text-sm font-medium text-accent" role="alert">
          {restore.error instanceof Error ? restore.error.message : t('deletedChatSessions.restoreFailed')}
        </p>
      )}
    </div>
  )
}
