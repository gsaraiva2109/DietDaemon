// /reset-password?token=…, set a new password from the emailed link. On
// success, send the user to /login to sign in with the new password.

import { useState, type SyntheticEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { useResetPassword } from '@/lib/queries'
import { AuthLayout } from '@/components/AuthLayout'
import { Button, Field, FormError } from '@/components/ui'

export function ResetPassword() {
  const { t } = useTranslation()
  const reset = useResetPassword()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get('token')

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: SyntheticEvent) {
    e.preventDefault()
    setError(null)
    if (!token) {
      setError(t('resetPassword.invalidLink'))
      return
    }
    if (password.length < 8) {
      setError(t('resetPassword.passwordHint'))
      return
    }
    if (password !== confirm) {
      setError(t('resetPassword.passwordMismatch'))
      return
    }
    setBusy(true)
    try {
      await reset.mutateAsync({ token, password })
      toast.success(t('resetPassword.successToast'))
      navigate('/login', { replace: true })
    } catch {
      setError(t('resetPassword.resetFailed'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout
      title={t('resetPassword.title')}
      subtitle={t('resetPassword.subtitle')}
      footer={
        <Link to="/login" className="font-medium text-primary hover:underline">
          {t('resetPassword.backToSignIn')}
        </Link>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label={t('resetPassword.newPasswordLabel')}
          type="password"
          autoComplete="new-password"
          autoFocus
          value={password}
          disabled={busy}
          onChange={(e) => setPassword(e.target.value)}
          hint={t('resetPassword.passwordHint')}
        />
        <Field
          label={t('resetPassword.confirmPasswordLabel')}
          type="password"
          autoComplete="new-password"
          value={confirm}
          disabled={busy}
          onChange={(e) => setConfirm(e.target.value)}
        />
        <FormError>{error}</FormError>
        <Button type="submit" disabled={busy || !password || !confirm}>
          {busy ? t('resetPassword.updating') : t('resetPassword.updatePassword')}
        </Button>
      </form>
    </AuthLayout>
  )
}
