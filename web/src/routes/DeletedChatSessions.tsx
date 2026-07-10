// Settings > Recently deleted: chat sessions soft-deleted within the last 30
// days (#53). Same page shape as AssistantSettings (back link + PageHeader).

import { Link } from 'react-router-dom'
import { useDeletedChatSessions, useRestoreChatSession } from '@/lib/queries'
import { PageHeader } from '@/components/PageHeader'
import { Card, EmptyState, Spinner } from '@/components/ui'
import { ChevronLeft, RestoreIcon, TrashIcon } from '@/components/icons'

export function DeletedChatSessions() {
  const deleted = useDeletedChatSessions()
  const restore = useRestoreChatSession()

  return (
    <div>
      <Link to="/settings" prefetch="intent" className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink">
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Recently deleted" />
      <p className="mb-5 text-sm text-muted">
        Deleted conversations stay here for 30 days before they're gone for good.
      </p>

      {deleted.isLoading ? (
        <Spinner label="Loading deleted conversations" />
      ) : deleted.isError ? (
        <EmptyState
          icon={<TrashIcon width={28} height={28} />}
          title="Couldn't load deleted conversations"
          hint={deleted.error instanceof Error ? deleted.error.message : 'Try again later.'}
        />
      ) : !deleted.data?.length ? (
        <EmptyState
          icon={<TrashIcon width={28} height={28} />}
          title="Nothing deleted"
          hint="Conversations you delete from Chat show up here for 30 days."
        />
      ) : (
        <Card className="divide-y divide-line p-0">
          {deleted.data.map((s) => (
            <div key={s.id} className="flex items-center gap-3 px-4 py-3">
              <span className="min-w-0 flex-1 truncate text-sm text-ink">{s.title || 'New conversation'}</span>
              <button
                onClick={() => restore.mutate(s.id)}
                disabled={restore.isPending}
                className="inline-flex items-center gap-1.5 rounded-lg border border-line px-3 py-1.5 text-xs font-medium text-ink transition hover:bg-surface-2 disabled:opacity-50"
              >
                <RestoreIcon width={14} height={14} /> Restore
              </button>
            </div>
          ))}
        </Card>
      )}

      {restore.isError && (
        <p className="mt-3 text-sm font-medium text-accent" role="alert">
          {restore.error instanceof Error ? restore.error.message : 'Failed to restore'}
        </p>
      )}
    </div>
  )
}
