// First-run / auth gate. Shown when the API rejects requests with 401. For a
// no-auth localhost server this never appears (the probe succeeds with no
// token). Calm, single-purpose screen — not a marketing login.

import { useState, type FormEvent } from 'react'
import { motion } from 'framer-motion'
import { useAuth } from '@/lib/auth'
import { Button } from './ui'
import { LeafIcon } from './icons'
import { fadeUp } from '@/lib/motion'

export function TokenGate() {
  const { signIn } = useAuth()
  const [token, setToken] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!token.trim()) return
    setBusy(true)
    setError(null)
    try {
      await signIn(token)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign in failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid min-h-[100dvh] place-items-center px-6">
      <motion.div
        variants={fadeUp}
        initial="hidden"
        animate="show"
        className="w-full max-w-sm"
      >
        <div className="mb-7 flex flex-col items-center text-center">
          <span className="mb-4 grid size-14 place-items-center rounded-2xl bg-primary-soft text-primary">
            <LeafIcon width={28} height={28} />
          </span>
          <h1 className="text-2xl font-bold tracking-tight text-ink">DietDaemon</h1>
          <p className="mt-1 text-sm text-muted">
            Enter your API token to open the dashboard.
          </p>
        </div>

        <form onSubmit={onSubmit} className="flex flex-col gap-3">
          <input
            type="password"
            autoFocus
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="API token"
            aria-label="API token"
            className="w-full rounded-xl border border-line bg-surface px-4 py-3 text-ink outline-none transition focus:border-primary"
          />
          {error && (
            <p className="text-sm font-medium text-accent" role="alert">
              {error}
            </p>
          )}
          <Button type="submit" disabled={busy || !token.trim()}>
            {busy ? 'Checking…' : 'Continue'}
          </Button>
        </form>
        <p className="mt-4 text-center text-xs text-muted">
          The token is the <code className="rounded bg-surface-2 px-1">API_AUTH_TOKEN</code>{' '}
          from your server, stored only in this browser.
        </p>
      </motion.div>
    </div>
  )
}
