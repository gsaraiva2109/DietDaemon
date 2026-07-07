// AI API key settings: view/set/delete the per-user AI provider key. A settings
// sub-page, same shape as BackupSettings (back link + PageHeader).

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAIKey, useSetAIKey, useDeleteAIKey } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Field, Spinner } from '@/components/ui'
import { ChevronLeft } from '@/components/icons'

const PROVIDERS = ['anthropic', 'openai'] as const

export function AIKeySettings() {
  const { demo } = useDemo()
  const query = useAIKey()
  const setKey = useSetAIKey()
  const deleteKey = useDeleteAIKey()

  const [provider, setProvider] = useState('anthropic')
  const [keyValue, setKeyValue] = useState('')

  const encKeyMissing =
    setKey.isError &&
    setKey.error instanceof Error &&
    setKey.error.message.includes('AI_KEY_ENC_KEY not configured')

  return (
    <div>
      <Link
        to="/settings"
        prefetch="intent"
        className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink"
      >
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="AI API Key" />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          AI key settings are read only here.
        </p>
      )}

      {encKeyMissing && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted" role="alert">
          The server does not have an encryption key configured for storing AI
          keys. Contact your administrator to set AI_KEY_ENC_KEY.
        </p>
      )}

      {query.isLoading ? (
        <Spinner label="Loading AI key settings" />
      ) : (
        <Card className="mb-5 p-5">
          {query.data?.has_key && (
            <p className="mb-4 text-sm text-muted">
              Provider: <span className="font-medium text-ink">{query.data.provider}</span> &mdash; key is set.
            </p>
          )}

          <div className="grid gap-4 sm:grid-cols-1">
            <div>
              <span className="mb-2 block text-xs font-medium text-muted">Provider</span>
              <div role="radiogroup" aria-label="Provider" className="inline-flex gap-1 rounded-full bg-surface-2 p-1">
                {PROVIDERS.map((p) => {
                  const active = provider === p
                  return (
                    <button
                      key={p}
                      role="radio"
                      aria-checked={active}
                      disabled={demo}
                      onClick={() => setProvider(p)}
                      className={`rounded-full px-4 py-1.5 text-sm font-semibold transition disabled:opacity-60 ${
                        active ? 'bg-primary text-primary-ink' : 'text-muted hover:text-ink'
                      }`}
                    >
                      {p === 'anthropic' ? 'Anthropic' : 'OpenAI'}
                    </button>
                  )
                })}
              </div>
            </div>

            <Field
              label="API Key"
              type="password"
              value={keyValue}
              disabled={demo}
              onChange={(e) => setKeyValue(e.target.value)}
              placeholder="sk-ant-..."
            />
          </div>

          <div className="mt-5 flex items-center gap-3">
            <Button onClick={() => setKey.mutate({ provider, key: keyValue })} disabled={demo || setKey.isPending || !keyValue}>
              {setKey.isPending ? 'Saving…' : 'Save'}
            </Button>
            {setKey.isSuccess && (
              <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                Saved.
              </motion.span>
            )}
            {setKey.isError && !encKeyMissing && (
              <span className="text-sm font-medium text-accent" role="alert">
                {setKey.error instanceof Error ? setKey.error.message : 'Failed to save'}
              </span>
            )}
          </div>

          {query.data?.has_key && (
            <div className="mt-6 border-t border-line pt-4">
              <p className="mb-3 text-sm text-muted">
                Remove the stored API key. Meal logging will fall back to the
                server-level key if one is configured.
              </p>
              <Button
                variant="ghost"
                onClick={() => deleteKey.mutate()}
                disabled={demo || deleteKey.isPending}
              >
                {deleteKey.isPending ? 'Deleting…' : 'Delete key'}
              </Button>
              {deleteKey.isSuccess && (
                <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="ml-3 text-sm font-medium text-primary">
                  Key deleted.
                </motion.span>
              )}
              {deleteKey.isError && (
                <span className="ml-3 text-sm font-medium text-accent" role="alert">
                  {deleteKey.error instanceof Error ? deleteKey.error.message : 'Failed to delete'}
                </span>
              )}
            </div>
          )}
        </Card>
      )}
    </div>
  )
}
