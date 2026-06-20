// /magic?token=… — the one-click passwordless sign-in link. Verifies the token
// (which sets the session cookie), refreshes, and routes into the app.

import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { AuthLayout } from '@/components/AuthLayout'
import { Spinner } from '@/components/ui'

export function MagicLink() {
  const { refresh } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const ran = useRef(false)
  const token = params.get('token')
  // Missing token is a render-time fact, not an effect side-effect — avoid
  // synchronous setState in the effect (react-hooks/set-state-in-effect).
  const [state, setState] = useState<'verifying' | 'error'>(token ? 'verifying' : 'error')

  useEffect(() => {
    if (ran.current) return
    ran.current = true
    if (!token) return
    api.auth.magic
      .verifyToken(token)
      .then(async () => {
        await refresh()
        toast.success('Signed in.')
        navigate('/', { replace: true })
      })
      .catch(() => setState('error'))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (state === 'error') {
    return (
      <AuthLayout
        title="Sign-in link expired"
        subtitle="That link is invalid or has already been used."
        footer={
          <Link to="/login" className="font-medium text-primary hover:underline">
            Back to sign in
          </Link>
        }
      >
        <span />
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title="Signing you in" subtitle="One moment…">
      <div className="grid place-items-center py-4">
        <Spinner label="Completing sign-in" />
      </div>
    </AuthLayout>
  )
}
