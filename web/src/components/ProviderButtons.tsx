// "Sign in with X" buttons, rendered from GET /auth/providers. Clicking begins
// a full-page OIDC redirect (the backend bounces to the provider and back to
// /auth/callback). Renders nothing when no providers are configured.

import { api } from '@/lib/api'
import { useProviders } from '@/lib/queries'
import { Button } from './ui'

export function ProviderButtons({ verb = 'Continue' }: { verb?: string }) {
  const providers = useProviders()
  const list = providers.data?.providers ?? []
  if (list.length === 0) return null

  const oidcOnly = providers.data?.registration_mode === 'oidc-only'

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-2">
        {list.map((p) => (
          <Button
            key={p.id}
            type="button"
            variant="ghost"
            onClick={() => window.location.assign(api.auth.oidcStartUrl(p.id))}
          >
            {verb} with {p.name}
          </Button>
        ))}
      </div>
      {/* Divider only when a password form also shows below/above it. */}
      {!oidcOnly && (
        <div className="flex items-center gap-3 text-xs text-muted">
          <span className="h-px flex-1 bg-line" />
          or
          <span className="h-px flex-1 bg-line" />
        </div>
      )}
    </div>
  )
}
