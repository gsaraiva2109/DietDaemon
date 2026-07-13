// A calm, self-contained manual fasting timer for the dashboard side column.
// State lives in localStorage so a refresh (or a tab reopen) never loses an
// in-progress fast: 'dd.fast.start' holds the ISO start time while running, and
// 'dd.fast.goal' the chosen goal in hours. No backend, no external deps.

import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Eyebrow, Pill, Button } from './ui'
import { ClockIcon } from './icons'

const START_KEY = 'dd.fast.start'
const GOAL_KEY = 'dd.fast.goal'
const DEFAULT_GOAL = 16
const GOAL_OPTIONS = [12, 14, 16, 18, 20, 24]

function readStart(): number | null {
  if (typeof window === 'undefined') return null
  const raw = localStorage.getItem(START_KEY)
  if (!raw) return null
  const t = new Date(raw).getTime()
  return Number.isFinite(t) ? t : null
}

function readGoal(): number {
  if (typeof window === 'undefined') return DEFAULT_GOAL
  const raw = Number(localStorage.getItem(GOAL_KEY))
  return Number.isFinite(raw) && raw > 0 ? raw : DEFAULT_GOAL
}

function formatElapsed(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(h)}:${pad(m)}:${pad(s)}`
}

export function FastingTimer() {
  const { t } = useTranslation()
  const [start, setStart] = useState<number | null>(readStart)
  const [goal, setGoal] = useState<number>(readGoal)
  const [now, setNow] = useState(() => Date.now())
  const tick = useRef<ReturnType<typeof setInterval> | null>(null)

  // Run a 1s clock only while a fast is active; always clear on unmount.
  useEffect(() => {
    if (start === null) return
    // `now` is initialized to Date.now(); the interval keeps it fresh. Avoid a
    // synchronous setState here (cascading-render lint rule).
    tick.current = setInterval(() => setNow(Date.now()), 1000)
    return () => {
      if (tick.current) clearInterval(tick.current)
    }
  }, [start])

  function handleStart() {
    const iso = new Date().toISOString()
    localStorage.setItem(START_KEY, iso)
    setStart(new Date(iso).getTime())
  }

  function handleStop() {
    localStorage.removeItem(START_KEY)
    setStart(null)
  }

  function handleGoal(hours: number) {
    localStorage.setItem(GOAL_KEY, String(hours))
    setGoal(hours)
  }

  const running = start !== null
  const elapsedMs = running ? now - start : 0
  const elapsedHours = elapsedMs / 3_600_000
  const goalMs = goal * 3_600_000
  const pct = running ? Math.min(100, (elapsedMs / goalMs) * 100) : 0
  const reached = running && elapsedMs >= goalMs

  return (
    <Card className="flex flex-col gap-4 p-5">
      <div className="flex items-center justify-between gap-3">
        <Eyebrow>{t('fastingTimer.title')}</Eyebrow>
        {running ? (
          reached ? (
            <Pill tone="primary">{t('fastingTimer.goalReached')}</Pill>
          ) : (
            <Pill tone="muted">
              {Math.floor(elapsedHours)}h / {goal}h
            </Pill>
          )
        ) : null}
      </div>

      {running ? (
        <>
          <div className="flex items-center gap-2.5">
            <span className="text-muted">
              <ClockIcon width={26} height={26} />
            </span>
            <span className="text-4xl font-extrabold tracking-tight text-ink tnum">
              {formatElapsed(elapsedMs)}
            </span>
          </div>

          <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-2">
            <div
              className="h-full rounded-full bg-primary transition-[width] duration-700 ease-out"
              style={{ width: `${pct}%` }}
            />
          </div>

          <Button variant="ghost" onClick={handleStop} className="self-start">
            {t('fastingTimer.stopReset')}
          </Button>
        </>
      ) : (
        <>
          <p className="text-sm text-muted">
            {t('fastingTimer.intro')}
          </p>

          <div className="flex flex-wrap gap-1.5">
            {GOAL_OPTIONS.map((h) => (
              <button
                key={h}
                onClick={() => handleGoal(h)}
                aria-pressed={goal === h}
                className={`rounded-full border px-3 py-1 text-xs font-medium transition ${
                  goal === h
                    ? 'border-transparent bg-primary-soft text-primary'
                    : 'border-line bg-surface text-muted hover:text-ink'
                }`}
              >
                {h}h
              </button>
            ))}
          </div>

          <Button onClick={handleStart} className="self-start">
            <ClockIcon width={16} height={16} /> {t('fastingTimer.startFast')}
          </Button>
        </>
      )}
    </Card>
  )
}
