// Renders a single tool-call message part (the assistant invoking a
// DietDaemon command mid-conversation) as a slim row. Always mounted inside
// a ToolCallGroup, which supplies the surrounding card/border — this stays
// borderless so a run of calls reads as one block instead of stacked boxes.

import { motion } from 'framer-motion'
import type { ToolCallMessagePartProps } from '@assistant-ui/react'
import { useTranslation } from 'react-i18next'
import { SparkleIcon } from './icons'
import { fadeUp } from '@/lib/motion'

export function ToolCallChip({ toolName, argsText, result, status }: ToolCallMessagePartProps) {
  const { t } = useTranslation()
  const running = status.type === 'running' && result === undefined

  return (
    <motion.div variants={fadeUp} initial="hidden" animate="show" className="flex flex-col gap-1.5 py-2 first:pt-0 last:pb-0">
      <div className="flex items-center gap-2 text-xs text-muted">
        <span className="text-primary">
          <SparkleIcon width={14} height={14} />
        </span>
        <span className={running ? 'chat-shimmer' : undefined}>
          {running ? t('toolCallChip.running') : t('toolCallChip.ran')} <code className="rounded bg-surface px-1 py-0.5 text-ink">/{toolName}</code>
          {argsText ? <span className="text-muted"> {argsText}</span> : null}
        </span>
      </div>
      {typeof result === 'string' && result && <p className="pl-5 text-sm text-ink">{result}</p>}
    </motion.div>
  )
}
