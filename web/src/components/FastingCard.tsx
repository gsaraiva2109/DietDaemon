// FastingCard, backend-driven intermittent-fasting timer. Supersedes the old
// localStorage FastingTimer. Three states: idle (pick a target, start), active
// (sage progress arc + live HH:MM elapsed), and just-ended (last duration +
// start new). Sage accent paired with the "Fasting" label. The ring is static
// under reduced motion (Framer honours the OS setting via MotionConfig).

import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useActiveFast, useFastHistory, useStartFast, useEndFast } from '@/lib/queries'
import { Card, Eyebrow, Pill, Spinner } from '@/components/ui'
import { ClockIcon } from '@/components/icons'
import { cssVar } from '@/lib/format'
import { easeOut } from '@/lib/motion'
import * as React from "react";

const TARGETS = [14, 16, 18, 20] // hours
const SIZE = 168
const THICK = 12

export function FastingCard() {
  const { t } = useTranslation()
  const active = useActiveFast()
  const history = useFastHistory(1)
  const startFast = useStartFast()
  const endFast = useEndFast()
  const [target, setTarget] = useState(16)

  // Tick once a second only while a fast is in progress.
  const [now, setNow] = useState(() => Date.now())
  const fast = active.data
  useEffect(() => {
    if (!fast) return
    const t = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(t)
  }, [fast])

  return (
    <Card className="flex h-full flex-col gap-4 p-5">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-primary">
          <ClockIcon width={18} height={18} />
          <Eyebrow>{t('fastingCard.title')}</Eyebrow>
        </div>
        {fast && <Pill tone="primary">{t('fastingCard.inProgress')}</Pill>}
      </header>

      {active.isLoading ? (
        <Spinner />
      ) : active.isError ? (
        <button
          onClick={() => active.refetch()}
          className="self-start text-sm font-medium text-accent hover:underline"
        >
          {t('fastingCard.retry')}
        </button>
      ) : fast ? (
        <ActiveFast
          startMs={new Date(fast.start_at).getTime()}
          targetHours={fast.target_hours}
          now={now}
          onEnd={() => endFast.mutate()}
          ending={endFast.isPending}
        />
      ) : (
        <Idle
          target={target}
          setTarget={setTarget}
          lastDurationH={lastDuration(history.data?.[0]?.start_at, history.data?.[0]?.end_at)}
          onStart={() => startFast.mutate(target)}
          starting={startFast.isPending}
        />
      )}
    </Card>
  )
}

function ActiveFast({
  startMs,
  targetHours,
  now,
  onEnd,
  ending,
}: {
  startMs: number
  targetHours: number
  now: number
  onEnd: () => void
  ending: boolean
}) {
  const { t } = useTranslation()
  const elapsedMs = Math.max(0, now - startMs)
  const elapsedH = elapsedMs / 3_600_000
  const pct = targetHours > 0 ? Math.min(1, elapsedH / targetHours) : 0
  const hh = Math.floor(elapsedMs / 3_600_000)
  const mm = Math.floor((elapsedMs % 3_600_000) / 60_000)
  const reached = elapsedH >= targetHours

  return (
    <>
      <div className="mt-1 grid place-items-center">
        <Ring pct={pct}>
          <div className="text-center">
            <div className="text-3xl font-bold leading-none text-ink tnum">
              {hh}:{String(mm).padStart(2, '0')}
            </div>
            <div className="mt-1 text-xs font-medium uppercase tracking-[0.14em] text-muted">
              {t('fastingCard.ofTarget', { hours: targetHours })}
            </div>
          </div>
        </Ring>
      </div>
      {reached && (
        <p className="text-center text-sm font-medium text-primary">{t('fastingCard.targetReached')}</p>
      )}
      <button
        onClick={onEnd}
        disabled={ending}
        className="mt-auto self-center rounded-full border border-line bg-surface px-5 py-2 text-sm font-semibold text-ink transition hover:bg-surface-2 disabled:opacity-50"
      >
        {ending ? t('fastingCard.ending') : t('fastingCard.endFast')}
      </button>
    </>
  )
}

function Idle({
  target,
  setTarget,
  lastDurationH,
  onStart,
  starting,
}: {
  target: number
  setTarget: (h: number) => void
  lastDurationH: number | null
  onStart: () => void
  starting: boolean
}) {
  const { t } = useTranslation()
  return (
    <>
      {lastDurationH !== null ? (
        <p className="text-sm text-muted">
          {t('fastingCard.lastFast')} <span className="font-medium text-ink tnum">{lastDurationH.toFixed(1)}h</span>.{' '}
          {t('fastingCard.readyAgain')}
        </p>
      ) : (
        <p className="text-sm text-muted">{t('fastingCard.noActiveFast')}</p>
      )}

      <div className="flex flex-wrap gap-2">
        {TARGETS.map((h) => {
          const sel = h === target
          return (
            <button
              key={h}
              onClick={() => setTarget(h)}
              aria-pressed={sel}
              className={`rounded-full border px-3.5 py-1.5 text-sm font-medium transition ${
                sel
                  ? 'border-transparent bg-primary text-primary-ink'
                  : 'border-line bg-surface text-ink hover:bg-surface-2'
              }`}
            >
              {h}h
            </button>
          )
        })}
      </div>

      <button
        onClick={onStart}
        disabled={starting}
        className="mt-auto self-start rounded-full bg-primary px-5 py-2.5 text-sm font-semibold text-primary-ink transition hover:brightness-105 disabled:opacity-50"
      >
        {starting ? t('fastingCard.starting') : t('fastingCard.startFast', { hours: target })}
      </button>
    </>
  )
}

// Sage progress arc, matching MacroRing's geometry/feel at a smaller size.
function Ring({ pct, children }: { pct: number; children: React.ReactNode }) {
  const r = (SIZE - THICK) / 2
  const circ = 2 * Math.PI * r
  const color = cssVar('--color-primary')
  return (
    <div className="relative inline-grid place-items-center" style={{ width: SIZE, height: SIZE }}>
      <svg width={SIZE} height={SIZE} className="-rotate-90">
        <circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={r}
          fill="none"
          stroke="var(--color-primary-soft)"
          strokeWidth={THICK}
        />
        <motion.circle
          cx={SIZE / 2}
          cy={SIZE / 2}
          r={r}
          fill="none"
          stroke={color}
          strokeWidth={THICK}
          strokeLinecap="round"
          strokeDasharray={circ}
          initial={{ strokeDashoffset: circ }}
          animate={{ strokeDashoffset: circ * (1 - pct) }}
          transition={{ duration: 0.6, ease: easeOut }}
        />
      </svg>
      <div className="absolute inset-0 grid place-items-center">{children}</div>
    </div>
  )
}

function lastDuration(startAt?: string, endAt?: string | null): number | null {
  if (!startAt || !endAt) return null
  const ms = new Date(endAt).getTime() - new Date(startAt).getTime()
  return ms > 0 ? ms / 3_600_000 : null
}
