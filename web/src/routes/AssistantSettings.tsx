// Chat assistant settings: per-user custom instructions appended to the
// localized base system prompt (same shape as Claude/ChatGPT "custom
// instructions" — an addition, not a replacement). Same page shape as
// AIKeySettings (back link + PageHeader), plus the Ollama tool-calling note.

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAssistantSettings, useSetAssistantSettings } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Spinner } from '@/components/ui'
import { ChevronLeft } from '@/components/icons'

const MAX_LEN = 2000

export function AssistantSettings() {
  const { demo } = useDemo()
  const query = useAssistantSettings()
  const setSettings = useSetAssistantSettings()

  // null = not yet edited; derive the value from server data (same pattern as
  // Settings.tsx's daily-targets draft).
  const [draft, setDraft] = useState<string | null>(null)
  const serverValue = query.data?.custom_instructions ?? ''
  const instructions = draft ?? serverValue
  const dirty = draft !== null && draft !== serverValue

  return (
    <div>
      <Link to="/settings" prefetch="intent" className="inline-flex items-center gap-1 text-sm text-muted hover:text-ink">
        <ChevronLeft width={18} height={18} /> Settings
      </Link>

      <PageHeader eyebrow="Settings" title="Assistant" />

      {demo && (
        <p className="mb-5 rounded-xl border border-line bg-surface-2 px-4 py-2.5 text-sm text-muted">
          Assistant settings are read only here.
        </p>
      )}

      {query.isLoading ? (
        <Spinner label="Loading assistant settings" />
      ) : (
        <Card className="mb-5 p-5">
          <h2 className="mb-1 font-semibold text-ink">Custom instructions</h2>
          <p className="mb-4 text-sm text-muted">
            Added to the assistant's base instructions for every conversation — tone, dietary
            preferences, units, anything you want it to always keep in mind.
          </p>
          <textarea
            value={instructions}
            disabled={demo}
            maxLength={MAX_LEN}
            onChange={(e) => setDraft(e.target.value)}
            rows={6}
            placeholder="e.g. Keep replies short. I count macros in metric. I'm vegetarian."
            className="w-full resize-none rounded-lg border border-line bg-bg px-4 py-3 text-sm text-ink outline-none transition focus:border-primary disabled:opacity-60"
          />
          <div className="mt-2 flex items-center justify-between text-xs text-muted">
            <span>{instructions.length}/{MAX_LEN}</span>
          </div>
          <div className="mt-4 flex items-center gap-3">
            <Button
              onClick={() => setSettings.mutate({ custom_instructions: instructions })}
              disabled={demo || !dirty || setSettings.isPending}
            >
              {setSettings.isPending ? 'Saving…' : 'Save'}
            </Button>
            {setSettings.isSuccess && !dirty && (
              <motion.span initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="text-sm font-medium text-primary">
                Saved.
              </motion.span>
            )}
            {setSettings.isError && (
              <span className="text-sm font-medium text-accent" role="alert">
                {setSettings.error instanceof Error ? setSettings.error.message : 'Failed to save'}
              </span>
            )}
          </div>
        </Card>
      )}

      <Card className="p-5">
        <h2 className="mb-1 font-semibold text-ink">Using a local model (Ollama)</h2>
        <p className="text-sm text-muted">
          The chat assistant calls DietDaemon commands (like <code className="rounded bg-surface-2 px-1">/suggest</code>{' '}
          or logging a meal) as tools mid-conversation. Tool-calling is model-dependent — if your
          self-hosted Ollama model doesn't support it, the assistant still chats normally, it just
          won't be able to act on your data. Known-good models: <code className="rounded bg-surface-2 px-1">llama3.1</code>,{' '}
          <code className="rounded bg-surface-2 px-1">qwen2.5</code>,{' '}
          <code className="rounded bg-surface-2 px-1">mistral-nemo</code>,{' '}
          <code className="rounded bg-surface-2 px-1">firefunction-v2</code>. See{' '}
          <code className="rounded bg-surface-2 px-1">docs/CHAT_ASSISTANT.md</code> for details.
        </p>
      </Card>
    </div>
  )
}
