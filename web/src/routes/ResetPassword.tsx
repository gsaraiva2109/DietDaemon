// /reset-password?token=… — set a new password from the emailed link. On
// success, send the user to /login to sign in with the new password.

import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useResetPassword } from '@/lib/queries'
import { AuthLayout } from '@/components/AuthLayout'
import { Button, Field, FormError } from '@/components/ui'

export function ResetPassword() {
  const reset = useResetPassword()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get('token')

  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (!token) {
      setError('This reset link is invalid or has expired.')
      return
    }
    if (password.length < 8) {
      setError('Use 8 or more characters.')
      return
    }
    if (password !== confirm) {
      setError('Passwords do not match.')
      return
    }
    setBusy(true)
    try {
      await reset.mutateAsync({ token, password })
      toast.success('Password updated. Please sign in.')
      navigate('/login', { replace: true })
    } catch {
      setError('Could not reset your password. The link may have expired.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout
      title="Choose a new password"
      subtitle="Enter a new password for your account."
      footer={
        <Link to="/login" className="font-medium text-primary hover:underline">
          Back to sign in
        </Link>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label="New password"
          type="password"
          autoComplete="new-password"
          autoFocus
          value={password}
          disabled={busy}
          onChange={(e) => setPassword(e.target.value)}
          hint="Use 8 or more characters."
        />
        <Field
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          value={confirm}
          disabled={busy}
          onChange={(e) => setConfirm(e.target.value)}
        />
        <FormError>{error}</FormError>
        <Button type="submit" disabled={busy || !password || !confirm}>
          {busy ? 'Updating…' : 'Update password'}
        </Button>
      </form>
    </AuthLayout>
  )
}
