// Code-entry step for passwordless sign-in. The user requested a code by email
// (the mock logs it to its console); here they enter it to complete sign-in.
// Includes a resend button with a short cooldown.

import { useEffect, useState, type FormEvent } from 'react'
import { useAuth } from '@/lib/auth'
import { useMagicRequest, useMagicVerifyCode } from '@/lib/queries'
import { Button, Field, FormError } from './ui'

const RESEND_COOLDOWN = 30

export function MagicCodeEntry({
  email,
  onVerified,
  onBack,
}: {
  email: string
  onVerified: () => void
  onBack: () => void
}) {
  const { refresh } = useAuth()
  const verify = useMagicVerifyCode()
  const resend = useMagicRequest()
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [cooldown, setCooldown] = useState(RESEND_COOLDOWN)

  useEffect(() => {
    if (cooldown <= 0) return
    const t = setTimeout(() => setCooldown((c) => c - 1), 1000)
    return () => clearTimeout(t)
  }, [cooldown])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    setError(null)
    try {
      await verify.mutateAsync({ email, code: code.trim() })
      await refresh()
      onVerified()
    } catch {
      setError('That code is invalid or expired. Try again.')
    }
  }

  async function onResend() {
    try {
      await resend.mutateAsync(email)
    } finally {
      setCooldown(RESEND_COOLDOWN)
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      <Field
        label="Sign-in code"
        inputMode="numeric"
        autoComplete="one-time-code"
        autoFocus
        maxLength={6}
        value={code}
        disabled={verify.isPending}
        onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
        placeholder="000000"
        hint={`Sent to ${email}.`}
      />
      <FormError>{error}</FormError>
      <Button type="submit" disabled={verify.isPending || code.length < 6}>
        {verify.isPending ? 'Verifying…' : 'Sign in'}
      </Button>
      <div className="flex items-center justify-between text-sm">
        <button type="button" onClick={onBack} className="font-medium text-muted hover:text-ink">
          Use a different email
        </button>
        <button
          type="button"
          onClick={onResend}
          disabled={cooldown > 0 || resend.isPending}
          className="font-medium text-primary hover:underline disabled:text-muted disabled:no-underline"
        >
          {cooldown > 0 ? `Resend in ${cooldown}s` : 'Resend code'}
        </button>
      </div>
    </form>
  )
}
