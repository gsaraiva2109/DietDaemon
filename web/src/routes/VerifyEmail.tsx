// /verify-email?token=… — landed on from the verification email. POSTs the
// token, then refreshes the session so the verify banner clears, and routes on.

import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '@/lib/auth'
import { useVerifyEmail } from '@/lib/queries'
import { AuthLayout } from '@/components/AuthLayout'
import { Spinner } from '@/components/ui'

export function VerifyEmail() {
  const verify = useVerifyEmail()
  const { refresh } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const ran = useRef(false)
  const [state, setState] = useState<'verifying' | 'error'>('verifying')

  useEffect(() => {
    if (ran.current) return
    ran.current = true
    const token = params.get('token')
    if (!token) {
      setState('error')
      return
    }
    verify
      .mutateAsync(token)
      .then(async () => {
        await refresh()
        toast.success('Email verified.')
        navigate('/', { replace: true })
      })
      .catch(() => setState('error'))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (state === 'error') {
    return (
      <AuthLayout
        title="Verification failed"
        subtitle="That link is invalid or has expired."
        footer={
          <Link to="/" className="font-medium text-primary hover:underline">
            Go to dashboard
          </Link>
        }
      >
        <span />
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title="Verifying your email" subtitle="One moment…">
      <div className="grid place-items-center py-4">
        <Spinner label="Verifying" />
      </div>
    </AuthLayout>
  )
}
