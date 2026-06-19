// Login — email + password + remember, inside the calm AuthLayout. Errors are
// always generic (never reveal which field). A "View demo" button drops into
// demo mode (no backend). Honors ?next= to return where the guard sent us.

import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import { useDemo, demoAvailable } from '@/lib/demo'
import { useProviders, useMagicRequest } from '@/lib/queries'
import { AUTH_ERROR, RateLimitError } from '@/lib/api'
import { AuthLayout } from '@/components/AuthLayout'
import { ProviderButtons } from '@/components/ProviderButtons'
import { MagicCodeEntry } from '@/components/MagicCodeEntry'
import { Button, Field, FormError } from '@/components/ui'

export function Login() {
  const { login } = useAuth()
  const { setDemo } = useDemo()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = params.get('next') || '/'
  const providers = useProviders()
  const oidcOnly = providers.data?.registration_mode === 'oidc-only'
  const canRegister = !oidcOnly

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  // Set when login defers to a second factor (TOTP).
  const [challengeToken, setChallengeToken] = useState<string | null>(null)
  // Set when the user requested a passwordless sign-in code.
  const [magicEmail, setMagicEmail] = useState<string | null>(null)
  const magicRequest = useMagicRequest()

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!email.trim() || !password) return
    setBusy(true)
    setError(null)
    try {
      const res = await login(email, password, remember)
      if (res.status === 'mfa_required') {
        setChallengeToken(res.challengeToken)
        return
      }
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

  async function emailMeCode() {
    const addr = email.trim().toLowerCase()
    if (!addr) {
      setError('Enter your email first.')
      return
    }
    setError(null)
    try {
      await magicRequest.mutateAsync(addr)
    } catch {
      // Generic — never leak whether the address exists.
    }
    setMagicEmail(addr)
  }

  if (magicEmail) {
    return (
      <AuthLayout
        title="Check your email"
        subtitle="We sent a sign-in code. Enter it below, or use the link in the email."
      >
        <MagicCodeEntry
          email={magicEmail}
          onVerified={() => navigate(next, { replace: true })}
          onBack={() => setMagicEmail(null)}
        />
      </AuthLayout>
    )
  }

  if (challengeToken) {
    return (
      <MfaChallenge
        challengeToken={challengeToken}
        onVerified={() => navigate(next, { replace: true })}
        onBack={() => {
          setChallengeToken(null)
          setPassword('')
        }}
      />
    )
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
      <div className="flex flex-col gap-4">
        {!oidcOnly && (
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
            <div className="flex items-center justify-between">
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
              <Link
                to="/forgot-password"
                className="text-sm font-medium text-primary hover:underline"
              >
                Forgot password?
              </Link>
            </div>
            <FormError>{error}</FormError>
            <Button type="submit" disabled={busy || !email.trim() || !password}>
              {busy ? 'Signing in…' : 'Sign in'}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={emailMeCode}
              disabled={busy || magicRequest.isPending}
            >
              {magicRequest.isPending ? 'Sending…' : 'Email me a sign-in code'}
            </Button>
          </form>
        )}
        <ProviderButtons verb="Sign in" />
        {demoAvailable() && (
          <Button type="button" variant="ghost" onClick={viewDemo} disabled={busy}>
            View demo
          </Button>
        )}
      </div>
    </AuthLayout>
  )
}

// Second step of a 2FA login: a TOTP code, or a recovery code as fallback.
function MfaChallenge({
  challengeToken,
  onVerified,
  onBack,
}: {
  challengeToken: string
  onVerified: () => void
  onBack: () => void
}) {
  const { verifyTotp } = useAuth()
  const [code, setCode] = useState('')
  const [recovery, setRecovery] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    setBusy(true)
    setError(null)
    try {
      await verifyTotp(challengeToken, code.trim(), recovery)
      onVerified()
    } catch (err) {
      setError(
        err instanceof RateLimitError
          ? 'Too many attempts. Try again shortly.'
          : 'That code did not match. Try again.',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <AuthLayout
      title="Two-factor verification"
      subtitle={
        recovery
          ? 'Enter one of your recovery codes.'
          : 'Enter the 6-digit code from your authenticator app.'
      }
      footer={
        <button type="button" onClick={onBack} className="font-medium text-primary hover:underline">
          Back to sign in
        </button>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <Field
          label={recovery ? 'Recovery code' : 'Authentication code'}
          inputMode={recovery ? 'text' : 'numeric'}
          autoComplete="one-time-code"
          autoFocus
          maxLength={recovery ? 20 : 6}
          value={code}
          disabled={busy}
          onChange={(e) =>
            setCode(recovery ? e.target.value : e.target.value.replace(/\D/g, ''))
          }
          placeholder={recovery ? 'xxxxx-xxxxx' : '000000'}
        />
        <FormError>{error}</FormError>
        <Button type="submit" disabled={busy || !code.trim()}>
          {busy ? 'Verifying…' : 'Verify'}
        </Button>
        <button
          type="button"
          onClick={() => {
            setRecovery((v) => !v)
            setCode('')
            setError(null)
          }}
          className="text-sm font-medium text-muted hover:text-ink"
        >
          {recovery ? 'Use authenticator code instead' : 'Use a recovery code instead'}
        </button>
      </form>
    </AuthLayout>
  )
}
