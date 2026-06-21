// A quiet banner nudging the user to verify their email. Shown in the app shell
// whenever the session reports email_verified === false. "Resend" re-sends the
// verification email (the mock logs the link/token to its console).

import { toast } from 'sonner'
import { useAuth } from '@/lib/auth'
import { useResendVerify } from '@/lib/queries'

export function VerifyEmailBanner() {
  const { user } = useAuth()
  const resend = useResendVerify()

  // Only when we have a real session user that isn't verified yet.
  if (!user || user.email_verified !== false) return null

  async function onResend() {
    try {
      await resend.mutateAsync()
      toast.success('Verification email sent. Check your inbox.')
    } catch {
      toast.error('Could not send the email. Try again shortly.')
    }
  }

  return (
    <div
      role="status"
      className="mb-5 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-accent/30 bg-accent/10 px-4 py-3"
    >
      <p className="text-sm text-ink">
        <span className="font-semibold">Verify your email</span>, we sent a link to{' '}
        <span className="font-medium">{user.email}</span>.
      </p>
      <button
        onClick={onResend}
        disabled={resend.isPending}
        className="text-sm font-semibold text-accent underline-offset-2 hover:underline disabled:opacity-50"
      >
        {resend.isPending ? 'Sending…' : 'Resend email'}
      </button>
    </div>
  )
}
