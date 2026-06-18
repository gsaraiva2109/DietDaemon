// Settings — editable daily targets (PUT /targets), theme, demo, token.

import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { useToday, useSetTargets } from '@/lib/queries'
import { useAuth } from '@/lib/auth'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Pill, Spinner } from '@/components/ui'
import { MACRO_KEYS, MACRO_META, type Macros } from '@/lib/types'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function Settings() {
  const today = useToday()
  const setTargets = useSetTargets()
  const { signOut } = useAuth()
  const { demo } = useDemo()

  const [draft, setDraft] = useState<Macros>(ZERO)

  // Seed the form from the current targets once they load.
  useEffect(() => {
    if (today.data?.Targets) setDraft(today.data.Targets)
  }, [today.data?.Targets])

  function set(k: keyof Macros, v: number) {
    setDraft((d) => ({ ...d, [k]: v }))
  }

  return (
    <div>
      <PageHeader eyebrow="Settings" title="Preferences" />

      <Card className="mb-5 p-5">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-semibold text-ink">Daily targets</h2>
          {demo && <Pill tone="muted">disabled in demo</Pill>}
        </div>

        {today.isLoading ? (
          <Spinner />
        ) : (
          <>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
              {MACRO_KEYS.map((k) => (
                <label key={k} className="block">
                  <span className="mb-1 block text-xs uppercase tracking-[0.1em] text-muted">
                    {MACRO_META[k].label}
                  </span>
                  <div className="flex items-baseline gap-1">
                    <input
                      type="number"
                      min={0}
                      value={draft[k]}
                      disabled={demo}
                      onChange={(e) => set(k, Number(e.target.value))}
                      className="w-full rounded-lg border border-line bg-bg px-3 py-2 text-lg font-semibold text-ink outline-none transition focus:border-primary disabled:opacity-60 tnum"
                    />
                    <span className="text-sm text-muted">{MACRO_META[k].unit}</span>
                  </div>
                </label>
              ))}
            </div>

            <div className="mt-5 flex items-center gap-3">
              <Button
                onClick={() => setTargets.mutate(draft)}
                disabled={demo || setTargets.isPending}
              >
                {setTargets.isPending ? 'Saving…' : 'Save targets'}
              </Button>
              {setTargets.isSuccess && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                  Saved.
                </motion.span>
              )}
              {setTargets.isError && (
                <span className="text-sm font-medium text-accent" role="alert">
                  {setTargets.error instanceof Error ? setTargets.error.message : 'Failed to save'}
                </span>
              )}
            </div>
            <p className="mt-3 text-xs text-muted">
              Targets also accept the <code className="rounded bg-surface-2 px-1">/target</code> chat command.
            </p>
          </>
        )}
      </Card>

      <Card className="p-5">
        <h2 className="mb-1 font-semibold text-ink">Session</h2>
        <p className="mb-4 text-sm text-muted">Your API token is stored in this browser only.</p>
        <Button variant="ghost" onClick={signOut}>
          Sign out
        </Button>
      </Card>
    </div>
  )
}
