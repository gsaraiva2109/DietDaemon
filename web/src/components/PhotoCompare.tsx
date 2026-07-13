// Side-by-side before/after comparison modal. Two selects pick which photos to
// compare (default oldest vs newest). Images load via the shared AuthedImage.

import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { AuthedImage } from './PhotoGrid'
import { CloseIcon } from './icons'
import { scaleIn } from '@/lib/motion'
import type { ProgressPhoto } from '@/lib/types'

function relativeCaption(t: (key: string, opts?: Record<string, unknown>) => string, date: string): string {
  const then = new Date(date + 'T00:00:00').getTime()
  const days = Math.round((Date.now() - then) / 86_400_000)
  if (days <= 0) return t('photoCompare.today')
  if (days < 7) return t('photoCompare.daysAgo', { count: days })
  const weeks = Math.round(days / 7)
  if (weeks < 9) return t('photoCompare.weeksAgo', { count: weeks })
  const months = Math.round(days / 30)
  return t('photoCompare.monthsAgo', { count: months })
}

function PhotoPane({
  photos,
  value,
  onChange,
  label,
}: {
  photos: ProgressPhoto[]
  value: string
  onChange: (id: string) => void
  label: string
}) {
  const { t } = useTranslation()
  const photo = photos.find((p) => p.id === value)
  return (
    <div className="flex flex-1 flex-col gap-3">
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-label={label}
        className="w-full rounded-full border border-line bg-bg px-4 py-2 text-sm text-ink outline-none focus:border-primary"
      >
        {photos.map((p) => (
          <option key={p.id} value={p.id}>
            {p.date} · {p.view}
          </option>
        ))}
      </select>
      <div className="relative aspect-[3/4] overflow-hidden rounded-xl border border-line bg-surface-2">
        {photo && (
          <AuthedImage
            id={photo.id}
            alt={`${label}: ${photo.view} ${photo.date}`}
            className="size-full object-cover"
          />
        )}
      </div>
      {photo && (
        <div className="text-center">
          <p className="text-sm font-semibold text-ink">{photo.date}</p>
          <p className="text-xs text-muted">{relativeCaption(t, photo.date)}</p>
        </div>
      )}
    </div>
  )
}

export function PhotoCompare({
  photos,
  onClose,
}: {
  photos: ProgressPhoto[]
  onClose: () => void
}) {
  const { t } = useTranslation()
  // Oldest -> newest for sensible "before" / "after" defaults.
  const ordered = useMemo(
    () => [...photos].sort((a, b) => (a.date < b.date ? -1 : 1)),
    [photos],
  )
  const [beforeId, setBeforeId] = useState(() => ordered[0]?.id ?? '')
  const [afterId, setAfterId] = useState(() => ordered[ordered.length - 1]?.id ?? '')

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <AnimatePresence>
      <motion.div
        className="fixed inset-0 grid place-items-center p-4"
        style={{ zIndex: 1300 }}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
      >
        <div
          className="absolute inset-0 bg-ink/30 backdrop-blur-sm"
          style={{ zIndex: 1200 }}
          onClick={onClose}
        />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label={t('photoCompare.ariaLabel')}
          variants={scaleIn}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative w-full max-w-2xl rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1300 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                {t('photoCompare.compare')}
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">{t('photoCompare.title')}</h2>
            </div>
            <button onClick={onClose} aria-label={t('photoCompare.close')} className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          <div className="flex gap-4">
            <PhotoPane
              photos={ordered}
              value={beforeId}
              onChange={setBeforeId}
              label={t('photoCompare.beforePhoto')}
            />
            <PhotoPane
              photos={ordered}
              value={afterId}
              onChange={setAfterId}
              label={t('photoCompare.afterPhoto')}
            />
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
