// Link Bot, generate a one-time code to connect a chat platform (Telegram /
// Discord / Matrix) to this account. Consumes the existing bot link-code
// backend (POST /bot/link-code). The code is single-use and expires after 10
// minutes; the user pastes `/link CODE` into the bot to complete the link.
//
// After generating a code the page opens an SSE stream that listens for the
// bot to consume the code. When linked the page transitions to a success
// state automatically — no manual polling or refresh needed.

import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '@/components/PageHeader'
import { Button, Card, Pill } from '@/components/ui'
import { CopyIcon, LinkIcon } from '@/components/icons'
import { useCreateLinkCode } from '@/lib/queries'
import { api } from '@/lib/api'

const PLATFORMS = [
  { id: 'telegram', label: 'Telegram' },
  { id: 'discord', label: 'Discord' },
  { id: 'matrix', label: 'Matrix' },
] as const

type Platform = (typeof PLATFORMS)[number]['id']

const CODE_TTL_S = 10 * 60 // codes expire after 10 minutes

export function LinkBot() {
  const { t } = useTranslation()
  const [platform, setPlatform] = useState<Platform>('telegram')
  const [code, setCode] = useState<string | null>(null)
  const [remaining, setRemaining] = useState(0)
  const [linked, setLinked] = useState(false)
  const createCode = useCreateLinkCode()
  const sseRef = useRef<EventSource | null>(null)

  // Close any open SSE stream.
  function closeStream() {
    if (sseRef.current) {
      sseRef.current.close()
      sseRef.current = null
    }
  }

  // Reset any live code when the target platform changes — a code is bound to
  // the platform it was minted for.
  function pick(p: Platform) {
    if (p === platform) return
    setPlatform(p)
    setCode(null)
    setRemaining(0)
    setLinked(false)
    closeStream()
  }

  async function generate() {
    closeStream()
    setLinked(false)
    const res = await createCode.mutateAsync(platform)
    setCode(res.code)
    setRemaining(CODE_TTL_S)

    // Subscribe to SSE so we know when the bot consumes the code.
    const es = api.bot.streamLinkCode(res.code)
    sseRef.current = es

    es.addEventListener('linked', () => {
      setLinked(true)
      closeStream()
      toast.success(t('linkBot.botConnectedToast'))
    })

    es.addEventListener('expired', () => {
      closeStream()
    })

    es.onerror = () => {
      // EventSource errors on close — ignore if we already closed it.
      closeStream()
    }
  }

  // Clean up SSE on unmount.
  useEffect(() => {
    return () => closeStream()
  }, [])

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
      <PageHeader eyebrow={t('nav.settings')} title={t('linkBot.title')} />

      <Card className="mb-5 p-5">
        <h2 className="mb-1 font-semibold text-ink">{t('linkBot.connectHeading')}</h2>
        <p className="mb-5 text-sm text-muted">
          {t('linkBot.introText', { platform: platformLabel })}
        </p>

        <p className="mb-2 text-xs font-medium uppercase tracking-[0.1em] text-muted">{t('linkBot.platformLabel')}</p>
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

        {linked ? (
          <SuccessPanel platformLabel={platformLabel} onLinkAnother={() => { setCode(null); setLinked(false); closeStream() }} />
        ) : code === null ? (
          <Button onClick={generate} disabled={createCode.isPending}>
            {createCode.isPending ? t('linkBot.generating') : t('linkBot.generateCode')}
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
            {createCode.error instanceof Error ? createCode.error.message : t('linkBot.generateCodeFailed')}
          </p>
        )}
      </Card>

      <Card className="p-5">
        <h2 className="mb-1 font-semibold text-ink">{t('linkBot.howItWorks')}</h2>
        <ol className="ml-4 list-decimal space-y-1 text-sm text-muted">
          <li>{t('linkBot.step1')}</li>
          <li>
            {t('linkBot.step2', { platform: platformLabel })}{' '}
            <code className="rounded bg-surface-2 px-1">/link CODE</code>.
          </li>
          <li>{t('linkBot.step3')}</li>
        </ol>
      </Card>
    </div>
  )
}

function SuccessPanel({
  platformLabel,
  onLinkAnother,
}: {
  platformLabel: string
  onLinkAnother: () => void
}) {
  const { t } = useTranslation()
  return (
    <div>
      <div className="mb-4 flex items-center gap-3 rounded-xl border border-green-500/30 bg-green-500/10 px-5 py-4">
        <div>
          <p className="font-semibold text-ink">{t('linkBot.connectedHeading')}</p>
          <p className="text-sm text-muted">
            {t('linkBot.connectedDesc', { platform: platformLabel })}
          </p>
        </div>
      </div>
      <Button variant="ghost" onClick={onLinkAnother}>
        {t('linkBot.linkAnother')}
      </Button>
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
  const { t } = useTranslation()
  const mm = Math.floor(remaining / 60)
  const ss = String(remaining % 60).padStart(2, '0')

  async function copy() {
    try {
      await navigator.clipboard.writeText(code)
      toast.success(t('linkBot.codeCopiedToast'))
    } catch {
      toast.error(t('linkBot.codeCopyFailed'))
    }
  }

  if (expired) {
    return (
      <div>
        <p className="mb-3 text-sm font-medium text-accent" role="alert">
          {t('linkBot.codeExpired')}
        </p>
        <Button onClick={onRegenerate} disabled={regenerating}>
          {regenerating ? t('linkBot.generating') : t('linkBot.generateNewCode')}
        </Button>
      </div>
    )
  }

  return (
    <div>
      <button
        onClick={copy}
        title={t('linkBot.clickToCopy')}
        className="group flex w-full items-center justify-between gap-4 rounded-xl border border-line bg-surface-2 px-5 py-4 text-left transition hover:border-primary"
      >
        <span className="font-mono text-3xl font-bold tracking-[0.3em] text-ink tnum">{code}</span>
        <span className="flex items-center gap-1.5 text-sm text-muted transition group-hover:text-primary">
          <CopyIcon width={18} height={18} />
          {t('linkBot.copy')}
        </span>
      </button>

      <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted">
        <span>
          {t('linkBot.sendCodePrefix')} <code className="rounded bg-surface-2 px-1 font-mono">/link {code}</code>{' '}
          {t('linkBot.sendCodeSuffix', { platform: platformLabel })}
        </span>
        <Pill tone="muted">
          <LinkIcon width={13} height={13} />
          {t('linkBot.expiresIn', { time: `${mm}:${ss}` })}
        </Pill>
      </div>
    </div>
  )
}
