// Flags a sustained divergence between the user's stated goal and their
// observed weight trend. Never mutates anything itself — "Review target"
// routes to the existing onboarding wizard (the same explicit-save flow
// already used to edit goals). Renders nothing when there's no suggestion,
// or when the user dismissed it within the last 7 days.

import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useTargetReviewSuggestion } from '@/lib/queries'
import { Card, Button } from './ui'
import { TrendsIcon } from './icons'

const DISMISS_KEY = 'dd:targetReviewDismissedUntil'
const DISMISS_DAYS = 7

function isDismissed(): boolean {
  const until = Number(localStorage.getItem(DISMISS_KEY))
  return Number.isFinite(until) && Date.now() < until
}

function openWizard() {
  window.dispatchEvent(new CustomEvent('dd:onboarding'))
}

export function TargetReviewPrompt() {
  const { t } = useTranslation()
  const { data } = useTargetReviewSuggestion()
  const [dismissed, setDismissed] = useState(isDismissed)

  if (dismissed || !data || !data.message) return null

  function dismiss() {
    localStorage.setItem(DISMISS_KEY, String(Date.now() + DISMISS_DAYS * 24 * 60 * 60 * 1000))
    setDismissed(true)
  }

  return (
    <Card className="p-5">
      <div className="flex items-start gap-3">
        <span className="mt-0.5 text-accent">
          <TrendsIcon />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold text-ink">{t('targetReview.title')}</h2>
          <p className="mt-1 text-sm text-muted">{data.message}</p>

          <div className="mt-4 flex items-center gap-3">
            <Button onClick={openWizard}>{t('targetReview.reviewButton')}</Button>
            <Button variant="ghost" onClick={dismiss}>
              {t('targetReview.dismiss')}
            </Button>
          </div>
        </div>
      </div>
    </Card>
  )
}
