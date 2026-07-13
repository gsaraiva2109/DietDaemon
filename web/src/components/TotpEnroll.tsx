// TOTP enrollment flow. Begin → render the QR from the otpauth_url (plus the
// base32 secret for manual entry), confirm a 6-digit code, then reveal recovery
// codes once. Lives inside the Security "Two-factor" section.

import { useEffect, useState, type FormEvent } from 'react'
import QRCode from 'qrcode'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useTotpEnroll, useTotpVerify } from '@/lib/queries'
import { Button, Field, FormError, Spinner } from './ui'
import { RecoveryCodes } from './RecoveryCodes'
import { fadeUp } from '@/lib/motion'

export function TotpEnroll({ onComplete, onCancel }: { onComplete: () => void; onCancel: () => void }) {
  const { t } = useTranslation()
  const enroll = useTotpEnroll()
  const verify = useTotpVerify()
  const [secret, setSecret] = useState('')
  const [qr, setQr] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [recovery, setRecovery] = useState<string[] | null>(null)

  // Begin enrollment once on mount.
  useEffect(() => {
    let alive = true
    enroll
      .mutateAsync()
      .then(async (res) => {
        if (!alive) return
        setSecret(res.secret)
        try {
          setQr(await QRCode.toDataURL(res.otpauth_url, { width: 200, margin: 1 }))
        } catch {
          /* QR render failed, manual secret entry still works. */
        }
      })
      .catch(() => alive && setError(t('totpEnroll.startFailed')))
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function onVerify(e: FormEvent) {
    e.preventDefault()
    if (code.trim().length < 6) return
    setError(null)
    try {
      const res = await verify.mutateAsync(code.trim())
      setRecovery(res.recovery_codes)
    } catch {
      setError(t('totpEnroll.verifyFailed'))
    }
  }

  if (recovery) {
    return <RecoveryCodes codes={recovery} onDone={onComplete} />
  }

  if (enroll.isPending && !secret) {
    return <Spinner label={t('totpEnroll.preparing')} />
  }

  if (error && !secret) {
    return (
      <div className="flex flex-col gap-3">
        <FormError>{error}</FormError>
        <Button variant="ghost" onClick={onCancel} className="self-start">
          {t('totpEnroll.close')}
        </Button>
      </div>
    )
  }

  return (
    <motion.div variants={fadeUp} initial="hidden" animate="show" className="flex flex-col gap-4">
      <div className="flex flex-col items-center gap-3 sm:flex-row sm:items-start">
        {qr && (
          <img
            src={qr}
            alt={t('totpEnroll.qrAlt')}
            width={160}
            height={160}
            className="rounded-lg border border-line bg-white p-2"
          />
        )}
        <div className="text-sm text-muted">
          <p>{t('totpEnroll.scanInstructions')}</p>
          <p className="mt-2">{t('totpEnroll.manualEntry')}</p>
          <code className="mt-1 block break-all rounded bg-surface-2 px-2 py-1 text-ink tnum">
            {secret}
          </code>
        </div>
      </div>

      <form onSubmit={onVerify} className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <Field
          label={t('totpEnroll.codeLabel')}
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          value={code}
          disabled={verify.isPending}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
          placeholder="000000"
          className="sm:max-w-[10rem]"
        />
        <Button type="submit" disabled={verify.isPending || code.length < 6} className="shrink-0">
          {verify.isPending ? t('totpEnroll.verifying') : t('totpEnroll.verifyAndEnable')}
        </Button>
        <Button type="button" variant="ghost" onClick={onCancel} className="shrink-0">
          {t('totpEnroll.cancel')}
        </Button>
      </form>
      <FormError>{error}</FormError>
    </motion.div>
  )
}
