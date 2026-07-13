// /verify-email?token=…, landed on from the verification email. POSTs the
// token, then refreshes the session so the verify banner clears, and routes on.

import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/lib/auth'
import { useVerifyEmail } from '@/lib/queries'
import { AuthLayout } from '@/components/AuthLayout'
import { Spinner } from '@/components/ui'

export function VerifyEmail() {
  const { t } = useTranslation()
  const verify = useVerifyEmail()
  const { refresh } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const ran = useRef(false)
  const token = params.get('token')
  const [state, setState] = useState<'verifying' | 'error'>(
    token ? 'verifying' : 'error',
  )

  useEffect(() => {
    if (ran.current || !token) return
    ran.current = true
    verify
      .mutateAsync(token)
      .then(async () => {
        await refresh()
        toast.success(t('verifyEmail.verified'))
        navigate('/', { replace: true })
      })
      .catch(() => setState('error'))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (state === 'error') {
    return (
      <AuthLayout
        title={t('verifyEmail.failedTitle')}
        subtitle={t('verifyEmail.failedSubtitle')}
        footer={
          <Link to="/" className="font-medium text-primary hover:underline">
            {t('verifyEmail.goToDashboard')}
          </Link>
        }
      >
        <span />
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title={t('verifyEmail.verifyingTitle')} subtitle={t('verifyEmail.oneMoment')}>
      <div className="grid place-items-center py-4">
        <Spinner label={t('verifyEmail.verifying')} />
      </div>
    </AuthLayout>
  )
}
