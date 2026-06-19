// Settings — editable daily targets (PUT /targets), theme, demo, token.

import { useState } from 'react'
import { motion } from 'framer-motion'
import { Link, useNavigate } from 'react-router-dom'
import { useToday, useSetTargets } from '@/lib/queries'
import { useAuth } from '@/lib/auth'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Pill, Spinner } from '@/components/ui'
import { ExportModal } from '@/components/ExportModal'
import { ChevronRight, FoodsIcon, GoalIcon, DownloadIcon, BodyIcon, SettingsIcon } from '@/components/icons'
import { MACRO_KEYS, MACRO_META, type Macros } from '@/lib/types'

const ZERO: Macros = { Calories: 0, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }

export function Settings() {
  const today = useToday()
  const setTargets = useSetTargets()
  const { logout } = useAuth()
  const { demo, setDemo } = useDemo()
  const navigate = useNavigate()
  const [exporting, setExporting] = useState(false)
  const [signingOut, setSigningOut] = useState(false)

  async function signOut() {
    setSigningOut(true)
    if (demo) setDemo(false)
    await logout()
    navigate('/login', { replace: true })
  }

  // null = not yet edited; derive values from server data.
  const [draft, setDraft] = useState<Macros | null>(null)
  const serverTargets = today.data?.Targets ?? ZERO
  const values = draft ?? serverTargets

  function set(k: keyof Macros, v: number) {
    setDraft((d) => ({ ...(d ?? values), [k]: v }))
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
                      value={values[k]}
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
                onClick={() => setTargets.mutate(values)}
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

      {/* Manage — links to the new feature surfaces. */}
      <Card className="mb-5 p-2">
        <RowLink to="/settings/security" Icon={SettingsIcon} label="Security" hint="API keys & password" />
        <RowLink to="/goals" Icon={GoalIcon} label="Body profile & goals" hint="TDEE, targets, onboarding" />
        <RowLink to="/settings/aliases" Icon={FoodsIcon} label="Food aliases" hint="Manage learned names" />
        <button
          onClick={() => window.dispatchEvent(new CustomEvent('dd:onboarding'))}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left transition hover:bg-surface-2"
        >
          <span className="text-muted"><BodyIcon width={20} height={20} /></span>
          <span className="flex-1">
            <span className="block text-sm font-medium text-ink">Edit body profile</span>
            <span className="block text-xs text-muted">Re-run the setup wizard</span>
          </span>
          <ChevronRight width={18} height={18} className="text-muted" />
        </button>
        <button
          onClick={() => setExporting(true)}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-3 text-left transition hover:bg-surface-2"
        >
          <span className="text-muted"><DownloadIcon width={20} height={20} /></span>
          <span className="flex-1">
            <span className="block text-sm font-medium text-ink">Export data</span>
            <span className="block text-xs text-muted">Download meals or rollups as CSV / JSON</span>
          </span>
          <ChevronRight width={18} height={18} className="text-muted" />
        </button>
      </Card>

      <Card className="p-5">
        <h2 className="mb-1 font-semibold text-ink">Session</h2>
        <p className="mb-4 text-sm text-muted">
          You're signed in with a secure server session. Sign out to end it on this device.
        </p>
        <Button variant="ghost" onClick={signOut} disabled={signingOut}>
          {signingOut ? 'Signing out…' : 'Sign out'}
        </Button>
      </Card>

      {exporting && <ExportModal onClose={() => setExporting(false)} />}
    </div>
  )
}

function RowLink({
  to,
  Icon,
  label,
  hint,
}: {
  to: string
  Icon: typeof FoodsIcon
  label: string
  hint: string
}) {
  return (
    <Link
      to={to}
      className="flex items-center gap-3 rounded-lg px-3 py-3 transition hover:bg-surface-2"
    >
      <span className="text-muted"><Icon width={20} height={20} /></span>
      <span className="flex-1">
        <span className="block text-sm font-medium text-ink">{label}</span>
        <span className="block text-xs text-muted">{hint}</span>
      </span>
      <ChevronRight width={18} height={18} className="text-muted" />
    </Link>
  )
}
