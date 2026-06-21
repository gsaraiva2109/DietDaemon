// /auth/callback, where the backend lands the browser after an OIDC round
// trip. On success the session cookie is already set by the redirect, so we
// just re-probe the session and route into the app; on ?error= we bounce back
// to /login with a generic message. ?link=1 means an account-link flow.

import { useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '@/lib/auth'
import { AuthLayout } from '@/components/AuthLayout'
import { OIDC_INTENT_KEY } from '@/components/ProviderButtons'
import { Spinner } from '@/components/ui'

// Maps the backend's ?error= codes (handler_oidc.go) to a human reason. The
// callback is shared by sign-in and sign-up, so we surface *why* it failed
// rather than guessing the verb, far more useful than a generic "cancelled".
const OIDC_ERRORS: Record<string, string> = {
  registration_closed:
    'No account is linked to this provider yet, and registration is closed. Link it from Settings → Security, or ask for an invite.',
  email_unverified:
    "Your provider account doesn't have a verified email, so an account couldn't be created.",
  already_linked: 'That provider account is already linked to a different user.',
  provider_error: 'The identity provider rejected the sign-in. Please try again.',
  invalid_state: 'The sign-in session expired or was interrupted. Please try again.',
  unknown_provider: 'Unknown sign-in provider.',
}

export function AuthCallback() {
  const { refresh } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const ran = useRef(false)

  useEffect(() => {
    if (ran.current) return
    ran.current = true

    const error = params.get('error')
    const linking = params.get('link') === '1'
    const next = params.get('next') || '/'

    // Which verb the user clicked (set in ProviderButtons); words the toast.
    const signingUp = sessionStorage.getItem(OIDC_INTENT_KEY) === 'signup'
    sessionStorage.removeItem(OIDC_INTENT_KEY)
    const verb = signingUp ? 'sign-up' : 'sign-in'

    if (error) {
      toast.error(OIDC_ERRORS[error] ?? `Could not complete ${verb}. Please try again.`)
      navigate('/login', { replace: true })
      return
    }

    refresh().then(() => {
      if (linking) {
        toast.success('Account linked.')
        navigate('/settings/security', { replace: true })
      } else {
        toast.success(signingUp ? 'Account created. Welcome!' : 'Signed in.')
        navigate(next, { replace: true })
      }
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <AuthLayout title="Signing you in" subtitle="One moment…">
      <div className="grid place-items-center py-4">
        <Spinner label="Completing sign-in" />
      </div>
    </AuthLayout>
  )
}
