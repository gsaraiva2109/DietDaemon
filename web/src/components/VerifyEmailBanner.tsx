// A quiet banner nudging the user to verify their email. Shown in the app shell
// whenever the session reports email_verified === false. "Resend" re-sends the
// verification email (the mock logs the link/token to its console).

import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/lib/auth'
import { useResendVerify } from '@/lib/queries'

export function VerifyEmailBanner() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const resend = useResendVerify()

  // Only when we have a real session user that isn't verified yet.
  if (!user || user.email_verified) return null

  async function onResend() {
    try {
      await resend.mutateAsync()
      toast.success(t('verifyEmailBanner.emailSent'))
    } catch {
      toast.error(t('verifyEmailBanner.sendFailed'))
    }
  }

  return (
    <div
      role="status"
      className="mb-5 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-accent/30 bg-accent/10 px-4 py-3"
    >
      <p className="text-sm text-ink">
        <span className="font-semibold">{t('verifyEmailBanner.title')}</span>
        {t('verifyEmailBanner.sentToPrefix')}{' '}
        <span className="font-medium">{user.email}</span>.
      </p>
      <button
        onClick={onResend}
        disabled={resend.isPending}
        className="text-sm font-semibold text-accent underline-offset-2 hover:underline disabled:opacity-50"
      >
        {resend.isPending ? t('verifyEmailBanner.sending') : t('verifyEmailBanner.resend')}
      </button>
    </div>
  )
}
