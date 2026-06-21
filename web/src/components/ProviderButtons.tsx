// "Sign in with X" buttons, rendered from GET /auth/providers. Clicking begins
// a full-page OIDC redirect (the backend bounces to the provider and back to
// /auth/callback). Renders nothing when no providers are configured.
//
// Sits below the password form, so the "or" divider renders on top (form → or
// → providers). Each button carries the provider's brand icon, falling back to
// a generic key for providers we don't have a mark for.

import type { ReactElement, SVGProps } from 'react'
import { api } from '@/lib/api'
import { useProviders } from '@/lib/queries'
import { Button } from './ui'
import {
  AppleIcon,
  Auth0Icon,
  AuthentikIcon,
  DiscordIcon,
  GitHubIcon,
  GitLabIcon,
  GoogleIcon,
  KeyIcon,
  KeycloakIcon,
  MicrosoftIcon,
  OktaIcon,
} from './icons'

type Icon = (p: SVGProps<SVGSVGElement>) => ReactElement

// Brand mark per provider, keyed by the provider id (the OIDC_PROVIDERS key).
// A few aliases map vendor naming variants onto the same logo. Anything missing
// gets KeyIcon, so a newly-configured provider always renders something sane.
const PROVIDER_ICONS: Record<string, Icon> = {
  google: GoogleIcon,
  github: GitHubIcon,
  gitlab: GitLabIcon,
  microsoft: MicrosoftIcon,
  azure: MicrosoftIcon,
  azuread: MicrosoftIcon,
  entra: MicrosoftIcon,
  entraid: MicrosoftIcon,
  discord: DiscordIcon,
  auth0: Auth0Icon,
  keycloak: KeycloakIcon,
  okta: OktaIcon,
  apple: AppleIcon,
  authentik: AuthentikIcon,
}

function providerIcon(id: string): Icon {
  return PROVIDER_ICONS[id.toLowerCase()] ?? KeyIcon
}

// The OIDC callback is shared by sign-in and sign-up, so stash which verb the
// user clicked. sessionStorage survives the provider round trip (same tab,
// same origin) and AuthCallback reads it to word its toast correctly.
export const OIDC_INTENT_KEY = 'dd.oidc_intent'

function startOIDC(id: string, verb: string) {
  sessionStorage.setItem(OIDC_INTENT_KEY, verb === 'Sign up' ? 'signup' : 'signin')
  window.location.assign(api.auth.oidcStartUrl(id))
}

export function ProviderButtons({ verb = 'Continue' }: { verb?: string }) {
  const providers = useProviders()
  const list = providers.data?.providers ?? []
  if (list.length === 0) return null

  const oidcOnly = providers.data?.registration_mode === 'oidc-only'

  return (
    <div className="flex flex-col gap-3">
      {/* Divider only when a password form also shows above it. */}
      {!oidcOnly && (
        <div className="flex items-center gap-3 text-xs text-muted">
          <span className="h-px flex-1 bg-line" />
          or
          <span className="h-px flex-1 bg-line" />
        </div>
      )}
      <div className="flex flex-col gap-2">
        {list.map((p) => {
          const Icon = providerIcon(p.id)
          return (
            <Button
              key={p.id}
              type="button"
              variant="ghost"
              onClick={() => startOIDC(p.id, verb)}
            >
              <Icon width={18} height={18} aria-hidden />
              {verb} with {p.name}
            </Button>
          )
        })}
      </div>
    </div>
  )
}
