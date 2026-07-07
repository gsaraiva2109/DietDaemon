// Renders a tool-call message part (the assistant invoking a DietDaemon
// command mid-conversation) as a quiet inline chip, in the same visual family
// as the confidence/source Pills in MacroTrace.tsx — a labelled capsule while
// the command runs, resolving into its plain-text reply.

import { motion } from 'framer-motion'
import type { ToolCallMessagePartProps } from '@assistant-ui/react'
import { Pill } from './ui'
import { SparkleIcon } from './icons'
import { fadeUp } from '@/lib/motion'

export function ToolCallChip({ toolName, argsText, result, status }: ToolCallMessagePartProps) {
  const running = status.type === 'running' && result === undefined

  return (
    <motion.div
      variants={fadeUp}
      initial="hidden"
      animate="show"
      className="my-1.5 flex flex-col gap-2 rounded-lg border border-line bg-surface-2 px-3 py-2.5"
    >
      <div className="flex items-center gap-2 text-xs text-muted">
        <span className="text-primary">
          <SparkleIcon width={14} height={14} />
        </span>
        <span>
          {running ? 'Running' : 'Ran'} <code className="rounded bg-surface px-1 py-0.5 text-ink">/{toolName}</code>
          {argsText ? <span className="text-muted"> {argsText}</span> : null}
        </span>
        {running && <span className="size-3 animate-spin rounded-full border-2 border-line border-t-primary" />}
      </div>
      {typeof result === 'string' && result && (
        <p className="flex items-start gap-2 text-sm text-ink">
          <Pill tone="primary">via /{toolName}</Pill>
          <span className="flex-1">{result}</span>
        </p>
      )}
    </motion.div>
  )
}
