// /magic?token=…, the one-click passwordless sign-in link. Verifies the token
// (which sets the session cookie), refreshes, and routes into the app. When the
// account has TOTP enabled, the verify returns an MFA challenge instead of a
// session; we hand off to <MfaChallenge>.

import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { api } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { isMfaChallenge } from '@/lib/types'
import { MfaChallenge } from '@/routes/Login'
import { AuthLayout } from '@/components/AuthLayout'
import { Spinner } from '@/components/ui'

export function MagicLink() {
  const { t } = useTranslation()
  const { refresh } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const ran = useRef(false)
  const token = params.get('token')
  // Missing token is a render-time fact, not an effect side-effect, avoid
  // synchronous setState in the effect (react-hooks/set-state-in-effect).
  const [state, setState] = useState<'verifying' | 'error' | 'mfa'>(token ? 'verifying' : 'error')
  const [challengeToken, setChallengeToken] = useState<string | null>(null)

  useEffect(() => {
    if (ran.current) return
    ran.current = true
    if (!token) return
    api.auth.magic
      .verifyToken(token)
      .then(async (res) => {
        if (isMfaChallenge(res)) {
          setChallengeToken(res.challenge_token)
          setState('mfa')
          return
        }
        await refresh()
        toast.success(t('magicLink.signedIn'))
        navigate('/', { replace: true })
      })
      .catch(() => setState('error'))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (state === 'error') {
    return (
      <AuthLayout
        title={t('magicLink.expiredTitle')}
        subtitle={t('magicLink.expiredSubtitle')}
        footer={
          <Link to="/login" className="font-medium text-primary hover:underline">
            {t('magicLink.backToSignIn')}
          </Link>
        }
      >
        <span />
      </AuthLayout>
    )
  }

  if (state === 'mfa' && challengeToken) {
    return (
      <MfaChallenge
        challengeToken={challengeToken}
        onVerified={async () => {
          await refresh()
          toast.success(t('magicLink.signedIn'))
          navigate('/', { replace: true })
        }}
        onBack={() => {
          setState('error')
        }}
      />
    )
  }

  return (
    <AuthLayout title={t('magicLink.signingInTitle')} subtitle={t('magicLink.oneMoment')}>
      <div className="grid place-items-center py-4">
        <Spinner label={t('magicLink.completingSignIn')} />
      </div>
    </AuthLayout>
  )
}
