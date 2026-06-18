// Settings — read-only targets (the REST API has no targets-write endpoint;
// targets are set via the chat `/target` command), plus token management.

import { useToday } from '@/lib/queries'
import { useAuth } from '@/lib/auth'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Pill, Spinner } from '@/components/ui'
import { MACRO_KEYS, MACRO_META, type Macros } from '@/lib/types'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function Settings() {
  const today = useToday()
  const { signOut } = useAuth()
  const targets = today.data?.Targets ?? ZERO
  const hasTargets = MACRO_KEYS.some((k) => targets[k] > 0)

  return (
    <div>
      <PageHeader eyebrow="Settings" title="Preferences" />

      <Card className="mb-5 p-5">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-semibold text-ink">Daily targets</h2>
          <Pill tone="muted">read-only</Pill>
        </div>
        {today.isLoading ? (
          <Spinner />
        ) : !hasTargets ? (
          <p className="text-sm text-muted">
            No targets set. Set them with the <code className="rounded bg-surface-2 px-1">/target</code> command in
            your chat bot.
          </p>
        ) : (
          <dl className="grid grid-cols-2 gap-4 sm:grid-cols-5">
            {MACRO_KEYS.map((k) => (
              <div key={k}>
                <dt className="text-xs uppercase tracking-[0.1em] text-muted">{MACRO_META[k].label}</dt>
                <dd className="mt-1 text-xl font-bold text-ink tnum">
                  {Math.round(targets[k])}
                  <span className="ml-1 text-sm font-normal text-muted">{MACRO_META[k].unit}</span>
                </dd>
              </div>
            ))}
          </dl>
        )}
        <p className="mt-4 text-xs text-muted">
          Editing targets from the dashboard needs a backend endpoint that doesn't exist yet — they're managed
          through chat for now.
        </p>
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
