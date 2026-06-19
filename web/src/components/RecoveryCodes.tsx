// Recovery codes — shown ONCE after TOTP enrollment or regeneration. Each code
// is a one-time fallback if the authenticator is lost. Offer copy + download,
// then make the user acknowledge before dismissing.

import { motion } from 'framer-motion'
import { toast } from 'sonner'
import { Button } from './ui'
import { CopyIcon, DownloadIcon } from './icons'
import { scaleIn } from '@/lib/motion'

export function RecoveryCodes({
  codes,
  onDone,
}: {
  codes: string[]
  onDone: () => void
}) {
  async function copy() {
    try {
      await navigator.clipboard.writeText(codes.join('\n'))
      toast.success('Recovery codes copied.')
    } catch {
      toast.error('Could not copy — select and copy them manually.')
    }
  }

  function download() {
    const blob = new Blob([codes.join('\n') + '\n'], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'dietdaemon-recovery-codes.txt'
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }

  return (
    <motion.div
      variants={scaleIn}
      initial="hidden"
      animate="show"
      className="rounded-xl border border-primary/40 bg-primary-soft/50 p-4"
    >
      <p className="text-sm font-medium text-ink">
        Save these recovery codes somewhere safe — they won't be shown again.
      </p>
      <p className="mt-1 text-xs text-muted">
        Each code works once if you lose access to your authenticator.
      </p>
      <ul className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1.5 rounded-lg border border-line bg-surface px-4 py-3 text-sm text-ink tnum">
        {codes.map((c) => (
          <li key={c}>{c}</li>
        ))}
      </ul>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button type="button" variant="ghost" onClick={copy}>
          <CopyIcon width={16} height={16} /> Copy
        </Button>
        <Button type="button" variant="ghost" onClick={download}>
          <DownloadIcon width={16} height={16} /> Download
        </Button>
        <Button type="button" onClick={onDone} className="ml-auto">
          I've saved them
        </Button>
      </div>
    </motion.div>
  )
}
