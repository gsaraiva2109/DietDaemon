// Inline quick-log so the primary action is one keystroke away on the
// dashboard, no need to navigate to the Log tab.

import { useState, type FormEvent } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useLogMeal } from '@/lib/queries'
import { useDemo } from '@/lib/demo'
import { Card } from './ui'
import { LogIcon } from './icons'

export function QuickLogCard() {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const log = useLogMeal()
  const { demo } = useDemo()

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!text.trim() || demo) return
    log.mutate(text.trim(), { onSuccess: () => setText('') })
  }

  return (
    <Card className="p-5">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-ink">
        <span className="grid size-7 place-items-center rounded-lg bg-primary-soft text-primary">
          <LogIcon width={16} height={16} />
        </span>
        {t('quickLogCard.title')}
      </div>
      <form onSubmit={onSubmit} className="flex gap-2">
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={demo ? t('quickLogCard.demoPlaceholder') : t('quickLogCard.placeholder')}
          disabled={demo}
          aria-label={t('quickLogCard.ariaLabel')}
          className="min-w-0 flex-1 rounded-lg border border-line bg-bg px-3 py-2.5 text-ink outline-none transition focus:border-primary disabled:opacity-60"
        />
        <button
          type="submit"
          disabled={demo || log.isPending || !text.trim()}
          className="shrink-0 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-primary-ink transition hover:brightness-105 disabled:opacity-50"
        >
          {log.isPending ? '…' : t('quickLogCard.log')}
        </button>
      </form>
      <AnimatePresence>
        {log.isSuccess && !demo && (
          <motion.p
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="mt-2 text-xs font-medium text-primary"
          >
            {t('quickLogCard.logged')}
          </motion.p>
        )}
      </AnimatePresence>
    </Card>
  )
}
