// Confirm-before-delete for a chat session (#53). Single-purpose, styled like
// DuplicateMealModal's overlay/panel — no picker step, just Cancel/Delete.
// The actual delete is a soft-delete (30-day retention, restorable from
// Settings > Recently deleted), done by the caller via
// ThreadListItemPrimitive.Archive.

import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { CloseIcon, TrashIcon } from './icons'
import { scaleIn } from '@/lib/motion'

interface Props {
  onCancel: () => void
  onConfirm: () => void
}

export function DeleteChatSessionModal({ onCancel, onConfirm }: Props) {
  const { t } = useTranslation()
  return (
    <motion.div
      className="fixed inset-0 grid place-items-center p-4"
      style={{ zIndex: 1500 }}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <div
        className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
        style={{ zIndex: 1400 }}
        onClick={onCancel}
      />
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-label={t('deleteChatSessionModal.ariaLabel')}
        variants={scaleIn}
        initial="hidden"
        animate="show"
        exit="hidden"
        className="relative w-full max-w-sm rounded-xl border border-line bg-surface p-6 shadow-lift"
        style={{ zIndex: 1500 }}
      >
        <div className="mb-4 flex items-start justify-between">
          <div className="flex items-center gap-2 text-accent">
            <TrashIcon width={20} height={20} />
            <h2 className="text-lg font-bold text-ink">{t('deleteChatSessionModal.title')}</h2>
          </div>
          <button onClick={onCancel} aria-label={t('deleteChatSessionModal.close')} className="text-muted hover:text-ink">
            <CloseIcon />
          </button>
        </div>
        <p className="mb-5 text-sm text-muted">
          {t('deleteChatSessionModal.body')}
        </p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="rounded-lg border border-line px-4 py-2 text-sm font-medium text-ink transition hover:bg-surface-2"
          >
            {t('deleteChatSessionModal.cancel')}
          </button>
          <button
            onClick={onConfirm}
            className="rounded-lg bg-ink px-4 py-2 text-sm font-semibold text-surface transition hover:brightness-110"
          >
            {t('deleteChatSessionModal.confirm')}
          </button>
        </div>
      </motion.div>
    </motion.div>
  )
}
