// Register — email + password + display name, inside AuthLayout. Gated by the
// server's registration_mode: hidden entirely when 'oidc-only'. Errors stay
// generic (no per-field server detail).

import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import { useProviders } from '@/lib/queries'
import { RateLimitError } from '@/lib/api'
import { AuthLayout } from '@/components/AuthLayout'
import { Button, Field, FormError } from '@/components/ui'

const REGISTER_ERROR = 'Could not create your account. Check your details and try again.'

export function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = params.get('next') || '/'
  const providers = useProviders()
  const closed = providers.data?.registration_mode === 'oidc-only'

  const [email, setEmail] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!email.trim() || !password) return
    setBusy(true)
    setError(null)
    try {
      await register(email, password, displayName)
      navigate(next, { replace: true })
    } catch (err) {
      setError(err instanceof RateLimitError ? 'Too many attempts. Try again shortly.' : REGISTER_ERROR)
    } finally {
      setBusy(false)
    }
  }

  if (closed) {
    return (
      <AuthLayout
        title="Registration is closed"
        subtitle="This dashboard only allows sign-in with a connected provider."
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
    <AuthLayout
      title="Create your account"
      subtitle="Start tracking with DietDaemon."
      footer={
        <>
          Already have an account?{' '}
          <Link to="/login" className="font-medium text-primary hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label="Display name"
          type="text"
          autoComplete="name"
          autoFocus
          value={displayName}
          disabled={busy}
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder="Your name"
          hint="Optional — what we'll call you."
        />
        <Field
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          disabled={busy}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
        />
        <Field
          label="Password"
          type="password"
          autoComplete="new-password"
          value={password}
          disabled={busy}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="At least 8 characters"
          hint="Use 8 or more characters."
        />
        <FormError>{error}</FormError>
        <Button type="submit" disabled={busy || !email.trim() || !password}>
          {busy ? 'Creating account…' : 'Create account'}
        </Button>
      </form>
    </AuthLayout>
  )
}
