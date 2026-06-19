// Login — email + password + remember, inside the calm AuthLayout. Errors are
// always generic (never reveal which field). A "View demo" button drops into
// demo mode (no backend). Honors ?next= to return where the guard sent us.

import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import { useDemo } from '@/lib/demo'
import { useProviders } from '@/lib/queries'
import { AUTH_ERROR, RateLimitError } from '@/lib/api'
import { AuthLayout } from '@/components/AuthLayout'
import { Button, Field, FormError } from '@/components/ui'

export function Login() {
  const { login } = useAuth()
  const { setDemo } = useDemo()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = params.get('next') || '/'
  const providers = useProviders()
  const canRegister = providers.data?.registration_mode !== 'oidc-only'

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!email.trim() || !password) return
    setBusy(true)
    setError(null)
    try {
      await login(email, password, remember)
      navigate(next, { replace: true })
    } catch (err) {
      if (err instanceof RateLimitError) {
        const n = err.retryAfter
        setError(
          n
            ? `Too many attempts. Try again in ${n} second${n === 1 ? '' : 's'}.`
            : 'Too many attempts. Try again shortly.',
        )
      } else {
        setError(AUTH_ERROR)
      }
    } finally {
      setBusy(false)
    }
  }

  function viewDemo() {
    setDemo(true)
    navigate('/', { replace: true })
  }

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Sign in to your DietDaemon dashboard."
      footer={
        canRegister && (
          <>
            New here?{' '}
            <Link to="/register" className="font-medium text-primary hover:underline">
              Create an account
            </Link>
          </>
        )
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label="Email"
          type="email"
          autoComplete="email"
          autoFocus
          value={email}
          disabled={busy}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
        />
        <Field
          label="Password"
          type="password"
          autoComplete="current-password"
          value={password}
          disabled={busy}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
        />
        <label className="flex items-center gap-2 text-sm text-muted">
          <input
            type="checkbox"
            checked={remember}
            disabled={busy}
            onChange={(e) => setRemember(e.target.checked)}
            className="size-4 rounded border-line accent-primary"
          />
          Keep me signed in
        </label>
        <FormError>{error}</FormError>
        <Button type="submit" disabled={busy || !email.trim() || !password}>
          {busy ? 'Signing in…' : 'Sign in'}
        </Button>
        <Button type="button" variant="ghost" onClick={viewDemo} disabled={busy}>
          View demo
        </Button>
      </form>
    </AuthLayout>
  )
}
