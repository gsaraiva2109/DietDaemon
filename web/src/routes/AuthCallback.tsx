// /auth/callback — where the backend lands the browser after an OIDC round
// trip. On success the session cookie is already set by the redirect, so we
// just re-probe the session and route into the app; on ?error= we bounce back
// to /login with a generic message. ?link=1 means an account-link flow.

import { useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '@/lib/auth'
import { AuthLayout } from '@/components/AuthLayout'
import { Spinner } from '@/components/ui'

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

    if (error) {
      toast.error('Sign-in was cancelled or failed. Please try again.')
      navigate('/login', { replace: true })
      return
    }

    refresh().then(() => {
      if (linking) {
        toast.success('Account linked.')
        navigate('/settings/security', { replace: true })
      } else {
        toast.success('Signed in.')
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
