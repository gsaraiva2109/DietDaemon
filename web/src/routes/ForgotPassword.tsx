// /forgot-password, request a reset link. The response is ALWAYS generic
// (never reveals whether an account exists), so we show the same confirmation
// regardless of outcome.

import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useForgotPassword } from '@/lib/queries'
import { AuthLayout } from '@/components/AuthLayout'
import { Button, Field } from '@/components/ui'

export function ForgotPassword() {
  const forgot = useForgotPassword()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!email.trim()) return
    setBusy(true)
    try {
      await forgot.mutateAsync(email.trim().toLowerCase())
    } catch {
      // Deliberately ignore failures, never leak account existence.
    } finally {
      setBusy(false)
      setSent(true)
    }
  }

  if (sent) {
    return (
      <AuthLayout
        title="Check your email"
        subtitle="If an account exists for that address, we've sent a link to reset your password."
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
      title="Reset your password"
      subtitle="Enter your email and we'll send a reset link."
      footer={
        <Link to="/login" className="font-medium text-primary hover:underline">
          Back to sign in
        </Link>
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
        <Button type="submit" disabled={busy || !email.trim()}>
          {busy ? 'Sending…' : 'Send reset link'}
        </Button>
      </form>
    </AuthLayout>
  )
}
