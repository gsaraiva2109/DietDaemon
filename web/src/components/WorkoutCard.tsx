// WorkoutCard, recent sessions plus an inline log form. Amber accent paired
// with the "Workout" label; intensity is shown as a labelled Pill so colour is
// never the only cue. Backend is Phase 4 (404 → empty state).

import { useState } from 'react'
import { useWorkouts, useLogWorkout } from '@/lib/queries'
import { Card, Eyebrow, Pill, Spinner } from '@/components/ui'
import { DumbbellIcon } from '@/components/icons'
import type { Workout, WorkoutIntensity } from '@/lib/types'

const AMBER = 'var(--color-carbs)'

const INTENSITY_TONE: Record<WorkoutIntensity, 'primary' | 'neutral' | 'accent'> = {
  light: 'primary',
  moderate: 'neutral',
  heavy: 'accent',
}

export function WorkoutCard() {
  const workouts = useWorkouts(5)
  const logWorkout = useLogWorkout()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [minutes, setMinutes] = useState('')
  const [intensity, setIntensity] = useState<WorkoutIntensity>('moderate')

  function submit() {
    const mins = Number(minutes)
    if (!name.trim() || !(mins > 0)) return
    logWorkout.mutate(
      { name: name.trim(), duration_min: mins, intensity },
      {
        onSuccess: () => {
          setName('')
          setMinutes('')
          setIntensity('moderate')
          setOpen(false)
        },
      },
    )
  }

  const list = workouts.data ?? []

  return (
    <Card className="flex h-full flex-col gap-4 p-5">
      <header className="flex items-center justify-between">
        <div className="flex items-center gap-2" style={{ color: AMBER }}>
          <DumbbellIcon width={18} height={18} />
          <Eyebrow>Workout</Eyebrow>
        </div>
        <button
          onClick={() => setOpen((o) => !o)}
          className="text-sm font-medium text-primary hover:underline"
        >
          {open ? 'Cancel' : 'Log'}
        </button>
      </header>

      {workouts.isLoading ? (
        <Spinner />
      ) : workouts.isError ? (
        <button
          onClick={() => workouts.refetch()}
          className="self-start text-sm font-medium text-accent hover:underline"
        >
          Couldn't load, retry
        </button>
      ) : (
        <>
          {open && (
            <div className="flex flex-col gap-2 rounded-xl border border-line bg-surface-2/50 p-3">
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Upper body"
                className="w-full rounded-lg border border-line bg-surface px-3 py-2 text-sm text-ink outline-none focus:border-primary"
              />
              <div className="flex gap-2">
                <input
                  value={minutes}
                  onChange={(e) => setMinutes(e.target.value)}
                  type="number"
                  min={1}
                  placeholder="min"
                  className="w-20 rounded-lg border border-line bg-surface px-3 py-2 text-sm text-ink outline-none focus:border-primary tnum"
                />
                <select
                  value={intensity}
                  onChange={(e) => setIntensity(e.target.value as WorkoutIntensity)}
                  className="flex-1 rounded-lg border border-line bg-surface px-3 py-2 text-sm text-ink outline-none focus:border-primary"
                >
                  <option value="light">Light</option>
                  <option value="moderate">Moderate</option>
                  <option value="heavy">Heavy</option>
                </select>
                <button
                  onClick={submit}
                  disabled={logWorkout.isPending}
                  className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-ink transition hover:brightness-105 disabled:opacity-50"
                >
                  Save
                </button>
              </div>
            </div>
          )}

          {list.length === 0 ? (
            <p className="text-sm text-muted">No workouts logged this week. Log one above.</p>
          ) : (
            <ul className="flex flex-col gap-2.5">
              {list.map((w: Workout) => (
                <li key={w.id} className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-ink">{w.name}</p>
                    <p className="text-xs text-muted tnum">{w.duration_min} min</p>
                  </div>
                  <Pill tone={INTENSITY_TONE[w.intensity]}>{w.intensity}</Pill>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </Card>
  )
}
