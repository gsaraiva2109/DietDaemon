// Linked accounts, connect/disconnect OIDC providers for the current user.
// Lists configured providers; each row shows the linked identity (with Unlink)
// or a Link button that begins the OIDC flow in link mode.

import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { api } from '@/lib/api'
import { useProviders, useIdentities, useUnlinkIdentity } from '@/lib/queries'
import { Button, Spinner } from './ui'

export function LinkedAccounts() {
  const { t } = useTranslation()
  const providers = useProviders()
  const identities = useIdentities()
  const unlink = useUnlinkIdentity()

  const list = providers.data?.providers ?? []
  if (providers.isLoading || identities.isLoading) return <Spinner />
  if (list.length === 0) {
    return <p className="text-sm text-muted">{t('linkedAccounts.noProviders')}</p>
  }

  const linkedBy = new Map((identities.data ?? []).map((i) => [i.provider, i]))

  async function onUnlink(id: string) {
    try {
      await unlink.mutateAsync(id)
      toast.success(t('linkedAccounts.unlinkedToast'))
    } catch {
      toast.error(t('linkedAccounts.unlinkFailed'))
    }
  }

  return (
    <ul className="flex flex-col divide-y divide-line">
      {list.map((p) => {
        const linked = linkedBy.get(p.id)
        return (
          <li key={p.id} className="flex items-center justify-between gap-3 py-3">
            <div className="min-w-0">
              <p className="text-sm font-medium text-ink">{p.name}</p>
              <p className="truncate text-xs text-muted">
                {linked ? linked.email || t('linkedAccounts.connected') : t('linkedAccounts.notConnected')}
              </p>
            </div>
            {linked ? (
              <Button
                variant="ghost"
                onClick={() => onUnlink(linked.id)}
                disabled={unlink.isPending}
                className="shrink-0"
              >
                {t('linkedAccounts.unlink')}
              </Button>
            ) : (
              <Button
                variant="ghost"
                onClick={() => window.location.assign(api.auth.oidcStartUrl(p.id, true))}
                className="shrink-0"
              >
                {t('linkedAccounts.link')}
              </Button>
            )}
          </li>
        )
      })}
    </ul>
  )
}
