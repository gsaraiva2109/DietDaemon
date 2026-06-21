// Link Bot, generate a one-time code to connect a chat platform (Telegram /
// Discord / Matrix) to this account. Consumes the existing bot link-code
// backend (POST /bot/link-code). The code is single-use and expires after 10
// minutes; the user pastes `/link CODE` into the bot to complete the link.

import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Pill } from '@/components/ui'
import { CopyIcon, LinkIcon } from '@/components/icons'
import { useCreateLinkCode } from '@/lib/queries'

const PLATFORMS = [
  { id: 'telegram', label: 'Telegram' },
  { id: 'discord', label: 'Discord' },
  { id: 'matrix', label: 'Matrix' },
] as const

type Platform = (typeof PLATFORMS)[number]['id']

const CODE_TTL_S = 10 * 60 // codes expire after 10 minutes

export function LinkBot() {
  const [platform, setPlatform] = useState<Platform>('telegram')
  const [code, setCode] = useState<string | null>(null)
  const [remaining, setRemaining] = useState(0)
  const createCode = useCreateLinkCode()

  // Reset any live code when the target platform changes, a code is bound to
  // the platform it was minted for.
  function pick(p: Platform) {
    if (p === platform) return
    setPlatform(p)
    setCode(null)
    setRemaining(0)
  }

  async function generate() {
    const res = await createCode.mutateAsync(platform)
    setCode(res.code)
    setRemaining(CODE_TTL_S)
  }

  // Countdown tick. Text-only, so reduced-motion needs no special-casing.
  useEffect(() => {
    if (remaining <= 0) return
    const t = setInterval(() => setRemaining((s) => Math.max(0, s - 1)), 1000)
    return () => clearInterval(t)
  }, [remaining])

  const expired = code !== null && remaining <= 0
  const platformLabel = PLATFORMS.find((p) => p.id === platform)?.label ?? ''

  return (
    <div>
      <PageHeader eyebrow="Settings" title="Link Bot" />

      <Card className="mb-5 p-5">
        <h2 className="mb-1 font-semibold text-ink">Connect a chat bot</h2>
        <p className="mb-5 text-sm text-muted">
          Log meals and check your day from {platformLabel}. Pick a platform, generate a
          one-time code, then send it to the DietDaemon bot.
        </p>

        <p className="mb-2 text-xs font-medium uppercase tracking-[0.1em] text-muted">Platform</p>
        <div className="mb-6 flex flex-wrap gap-2">
          {PLATFORMS.map((p) => {
            const active = p.id === platform
            return (
              <button
                key={p.id}
                onClick={() => pick(p.id)}
                aria-pressed={active}
                className={`rounded-full border px-4 py-1.5 text-sm font-medium transition ${
                  active
                    ? 'border-transparent bg-primary text-primary-ink'
                    : 'border-line bg-surface text-ink hover:bg-surface-2'
                }`}
              >
                {p.label}
              </button>
            )
          })}
        </div>

        {code === null ? (
          <Button onClick={generate} disabled={createCode.isPending}>
            {createCode.isPending ? 'Generating…' : 'Generate code'}
          </Button>
        ) : (
          <CodePanel
            code={code}
            expired={expired}
            remaining={remaining}
            platformLabel={platformLabel}
            onRegenerate={generate}
            regenerating={createCode.isPending}
          />
        )}

        {createCode.isError && (
          <p className="mt-3 text-sm font-medium text-accent" role="alert">
            {createCode.error instanceof Error ? createCode.error.message : 'Could not generate a code'}
          </p>
        )}
      </Card>

      <Card className="p-5">
        <h2 className="mb-1 font-semibold text-ink">How it works</h2>
        <ol className="ml-4 list-decimal space-y-1 text-sm text-muted">
          <li>Generate a code above.</li>
          <li>
            Open the DietDaemon bot on {platformLabel} and send{' '}
            <code className="rounded bg-surface-2 px-1">/link CODE</code>.
          </li>
          <li>The bot confirms and your account is linked. The code works once.</li>
        </ol>
      </Card>
    </div>
  )
}

function CodePanel({
  code,
  expired,
  remaining,
  platformLabel,
  onRegenerate,
  regenerating,
}: {
  code: string
  expired: boolean
  remaining: number
  platformLabel: string
  onRegenerate: () => void
  regenerating: boolean
}) {
  const mm = Math.floor(remaining / 60)
  const ss = String(remaining % 60).padStart(2, '0')

  async function copy() {
    try {
      await navigator.clipboard.writeText(code)
      toast.success('Code copied to clipboard.')
    } catch {
      toast.error('Could not copy, select the code manually.')
    }
  }

  if (expired) {
    return (
      <div>
        <p className="mb-3 text-sm font-medium text-accent" role="alert">
          Code expired, generate a new one.
        </p>
        <Button onClick={onRegenerate} disabled={regenerating}>
          {regenerating ? 'Generating…' : 'Generate new code'}
        </Button>
      </div>
    )
  }

  return (
    <div>
      <button
        onClick={copy}
        title="Click to copy"
        className="group flex w-full items-center justify-between gap-4 rounded-xl border border-line bg-surface-2 px-5 py-4 text-left transition hover:border-primary"
      >
        <span className="font-mono text-3xl font-bold tracking-[0.3em] text-ink tnum">{code}</span>
        <span className="flex items-center gap-1.5 text-sm text-muted transition group-hover:text-primary">
          <CopyIcon width={18} height={18} />
          Copy
        </span>
      </button>

      <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted">
        <span>
          Send <code className="rounded bg-surface-2 px-1 font-mono">/link {code}</code> to the
          bot on {platformLabel}.
        </span>
        <Pill tone="muted">
          <LinkIcon width={13} height={13} />
          expires in {mm}:{ss}
        </Pill>
      </div>
    </div>
  )
}
