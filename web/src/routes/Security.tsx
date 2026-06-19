// Security — machine API keys + change password. Single-level Cards, matching
// Settings. A freshly created key's raw secret is revealed exactly once in a
// scaleIn panel with a copy button; after that only metadata is ever shown.

import { useState, type FormEvent } from 'react'
import { motion } from 'framer-motion'
import { toast } from 'sonner'
import { useApiKeys, useCreateApiKey, useRevokeApiKey, useChangePassword } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Field, FormError, Input, Pill, Spinner } from '@/components/ui'
import { CopyIcon, TrashIcon } from '@/components/icons'
import { scaleIn } from '@/lib/motion'
import type { NewApiKey } from '@/lib/types'

export function Security() {
  const { demo } = useDemo()

  return (
    <div>
      <PageHeader eyebrow="Settings" title="Security" />
      <ApiKeysCard demo={demo} />
      <ChangePasswordCard demo={demo} />
    </div>
  )
}

function ApiKeysCard({ demo }: { demo: boolean }) {
  const keys = useApiKeys()
  const create = useCreateApiKey()
  const revoke = useRevokeApiKey()
  const [label, setLabel] = useState('')
  const [fresh, setFresh] = useState<NewApiKey | null>(null)

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    if (!label.trim()) return
    const key = await create.mutateAsync(label.trim())
    setFresh(key)
    setLabel('')
  }

  async function copyKey(value: string) {
    try {
      await navigator.clipboard.writeText(value)
      toast.success('API key copied to clipboard.')
    } catch {
      toast.error('Could not copy — select and copy it manually.')
    }
  }

  const list = keys.data ?? []
  const active = list.filter((k) => !k.revoked_at)

  return (
    <Card className="mb-5 p-5">
      <div className="mb-1 flex items-center justify-between">
        <h2 className="font-semibold text-ink">API keys</h2>
        {demo && <Pill tone="muted">disabled in demo</Pill>}
      </div>
      <p className="mb-4 text-sm text-muted">
        Machine keys for scripts and integrations. Send as{' '}
        <code className="rounded bg-surface-2 px-1 tnum">Authorization: Bearer ddk_…</code>
      </p>

      {fresh && (
        <motion.div
          variants={scaleIn}
          initial="hidden"
          animate="show"
          className="mb-4 rounded-xl border border-primary/40 bg-primary-soft/50 p-4"
        >
          <p className="text-sm font-medium text-ink">
            Copy your new key now — it won't be shown again.
          </p>
          <div className="mt-3 flex items-center gap-2">
            <code className="flex-1 overflow-x-auto rounded-lg border border-line bg-surface px-3 py-2 text-sm text-ink tnum">
              {fresh.key}
            </code>
            <Button type="button" onClick={() => copyKey(fresh.key)} className="shrink-0">
              <CopyIcon width={16} height={16} /> Copy
            </Button>
          </div>
          <button
            onClick={() => setFresh(null)}
            className="mt-3 text-xs font-medium text-muted hover:text-ink"
          >
            I've saved it — dismiss
          </button>
        </motion.div>
      )}

      <form onSubmit={onCreate} className="mb-5 flex flex-col gap-2 sm:flex-row">
        <Input
          value={label}
          disabled={demo || create.isPending}
          onChange={(e) => setLabel(e.target.value)}
          placeholder="Key label (e.g. “Home server”)"
          aria-label="API key label"
        />
        <Button type="submit" disabled={demo || create.isPending || !label.trim()} className="shrink-0">
          {create.isPending ? 'Creating…' : 'Create key'}
        </Button>
      </form>

      {keys.isLoading ? (
        <Spinner />
      ) : active.length === 0 ? (
        <p className="text-sm text-muted">No API keys yet.</p>
      ) : (
        <ul className="flex flex-col divide-y divide-line">
          {active.map((k) => (
            <li key={k.id} className="flex items-center justify-between gap-3 py-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-ink">{k.label}</p>
                <p className="text-xs text-muted">
                  Created {new Date(k.created_at).toLocaleDateString()}
                  {k.last_used_at && ` · Last used ${new Date(k.last_used_at).toLocaleDateString()}`}
                </p>
              </div>
              <button
                onClick={() => revoke.mutate(k.id)}
                disabled={revoke.isPending}
                className="grid size-9 shrink-0 place-items-center rounded-lg text-muted transition hover:bg-surface-2 hover:text-accent disabled:opacity-50"
                aria-label={`Revoke ${k.label}`}
              >
                <TrashIcon width={18} height={18} />
              </button>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

function ChangePasswordCard({ demo }: { demo: boolean }) {
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (next !== confirm) {
      setError('New passwords do not match.')
      return
    }
    if (next.length < 8) {
      setError('Use 8 or more characters.')
      return
    }
    try {
      await change.mutateAsync({ current, next })
      toast.success('Password changed.')
      setCurrent('')
      setNext('')
      setConfirm('')
    } catch {
      setError('Could not change password. Check your current password.')
    }
  }

  return (
    <Card className="p-5">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-semibold text-ink">Change password</h2>
        {demo && <Pill tone="muted">disabled in demo</Pill>}
      </div>
      <form onSubmit={onSubmit} className="flex max-w-sm flex-col gap-4">
        <Field
          label="Current password"
          type="password"
          autoComplete="current-password"
          value={current}
          disabled={demo || change.isPending}
          onChange={(e) => setCurrent(e.target.value)}
        />
        <Field
          label="New password"
          type="password"
          autoComplete="new-password"
          value={next}
          disabled={demo || change.isPending}
          onChange={(e) => setNext(e.target.value)}
          hint="Use 8 or more characters."
        />
        <Field
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          value={confirm}
          disabled={demo || change.isPending}
          onChange={(e) => setConfirm(e.target.value)}
        />
        <FormError>{error}</FormError>
        <Button
          type="submit"
          disabled={demo || change.isPending || !current || !next}
          className="self-start"
        >
          {change.isPending ? 'Saving…' : 'Update password'}
        </Button>
      </form>
    </Card>
  )
}
