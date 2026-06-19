// Passkey management for the Security page: list registered passkeys, add a new
// one (runs the WebAuthn registration ceremony), rename, and delete. The native
// prompt is real; against the dev mock the verify step is stubbed.

import { useState, type SubmitEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { usePasskeys, useRenamePasskey, useDeletePasskey } from '@/lib/queries'
import { registerPasskey, browserSupportsWebAuthn, isWebAuthnCancel } from '@/lib/webauthn'
import { Button, Field, Input, Spinner } from './ui'
import { TrashIcon, CheckIcon } from './icons'
import type { Passkey } from '@/lib/types'

export function PasskeyManager() {
  const passkeys = usePasskeys()
  const qc = useQueryClient()
  const [label, setLabel] = useState('')
  const [registering, setRegistering] = useState(false)
  const supported = browserSupportsWebAuthn()

  async function onAdd(e: SubmitEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!label.trim()) return
    setRegistering(true)
    try {
      await registerPasskey(label.trim())
      await qc.invalidateQueries({ queryKey: ['auth', 'passkeys'] })
      setLabel('')
      toast.success('Passkey added.')
    } catch (err) {
      if (!isWebAuthnCancel(err)) toast.error('Could not add passkey. Try again.')
    } finally {
      setRegistering(false)
    }
  }

  if (!supported) {
    return <p className="text-sm text-muted">This browser does not support passkeys.</p>
  }

  const list = passkeys.data ?? []

  return (
    <div className="flex flex-col gap-5">
      <form onSubmit={onAdd} className="flex flex-col gap-2 sm:flex-row sm:items-end">
        <Field
          label="New passkey name"
          value={label}
          disabled={registering}
          onChange={(e) => setLabel(e.target.value)}
          placeholder="e.g. “MacBook Touch ID”"
          className="flex-1"
        />
        <Button type="submit" disabled={registering || !label.trim()} className="shrink-0">
          {registering ? 'Waiting for device…' : 'Add passkey'}
        </Button>
      </form>

      {passkeys.isLoading ? (
        <Spinner />
      ) : list.length === 0 ? (
        <p className="text-sm text-muted">No passkeys yet.</p>
      ) : (
        <ul className="flex flex-col divide-y divide-line">
          {list.map((k) => (
            <PasskeyRow key={k.id} passkey={k} />
          ))}
        </ul>
      )}
    </div>
  )
}

function PasskeyRow({ passkey }: { passkey: Passkey }) {
  const rename = useRenamePasskey()
  const remove = useDeletePasskey()
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(passkey.label)

  async function save() {
    if (draft.trim() && draft.trim() !== passkey.label) {
      await rename.mutateAsync({ id: passkey.id, label: draft.trim() })
    }
    setEditing(false)
  }

  return (
    <li className="flex items-center justify-between gap-3 py-3">
      {editing ? (
        <div className="flex flex-1 items-center gap-2">
          <Input
            value={draft}
            autoFocus
            onChange={(e) => setDraft(e.target.value)}
            aria-label="Passkey name"
          />
          <button
            onClick={save}
            disabled={rename.isPending}
            className="grid size-9 shrink-0 place-items-center rounded-lg text-primary transition hover:bg-surface-2"
            aria-label="Save name"
          >
            <CheckIcon width={18} height={18} />
          </button>
        </div>
      ) : (
        <button onClick={() => setEditing(true)} className="min-w-0 text-left">
          <p className="truncate text-sm font-medium text-ink">{passkey.label}</p>
          <p className="text-xs text-muted">
            Added {new Date(passkey.created_at).toLocaleDateString()}
            {passkey.last_used_at && ` · Last used ${new Date(passkey.last_used_at).toLocaleDateString()}`}
          </p>
        </button>
      )}
      <button
        onClick={() => remove.mutate(passkey.id)}
        disabled={remove.isPending}
        className="grid size-9 shrink-0 place-items-center rounded-lg text-muted transition hover:bg-surface-2 hover:text-accent disabled:opacity-50"
        aria-label={`Delete ${passkey.label}`}
      >
        <TrashIcon width={18} height={18} />
      </button>
    </li>
  )
}
