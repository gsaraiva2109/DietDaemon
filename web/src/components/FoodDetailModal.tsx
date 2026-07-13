// Full food detail in a modal, fetched fresh by id. Shows the complete
// per-100g breakdown, serving info, provenance, aliases, and a shortcut to log.

import { useEffect } from 'react'
import { AnimatePresence, motion, type Variants } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useFood } from '@/lib/queries'
import { Button, Pill, Spinner } from './ui'
import { CloseIcon, LogIcon } from './icons'
import { sourceLabel } from './FoodCard'
import { MACRO_KEYS, type FoodAlias } from '@/lib/types'
import { formatNumber, round } from '@/lib/format'
import { easeOut } from '@/lib/motion'

const scaleInDialog: Variants = {
  hidden: { opacity: 0, scale: 0.96, y: 8 },
  show: { opacity: 1, scale: 1, y: 0, transition: { duration: 0.4, ease: easeOut } },
}

export function FoodDetailModal({ foodID, onClose }: { foodID: string; onClose: () => void }) {
  const { t } = useTranslation()
  const food = useFood(foodID)
  const navigate = useNavigate()

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const f = food.data

  function logit() {
    if (!f) return
    navigate(`/log?text=${encodeURIComponent(f.name)}`)
    onClose()
  }

  return (
    <AnimatePresence>
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
          onClick={onClose}
        />
        <motion.div
          role="dialog"
          aria-modal="true"
          aria-label={f ? f.name : t('foodDetailModal.ariaLabelFallback')}
          variants={scaleInDialog}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative w-full max-w-md rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <button
            onClick={onClose}
            aria-label={t('foodDetailModal.close')}
            className="absolute right-4 top-4 text-muted hover:text-ink"
          >
            <CloseIcon />
          </button>

          {food.isLoading || !f ? (
            <div className="py-10">
              <Spinner label={t('foodDetailModal.loading')} />
            </div>
          ) : (
            <>
              <h2 className="pr-8 text-xl font-bold text-ink">{f.name}</h2>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <Pill tone={f.source === 'food_library' ? 'primary' : 'neutral'}>
                  {sourceLabel(f.source, t)}
                </Pill>
                {f.category && <Pill tone="muted">{f.category}</Pill>}
              </div>

              <dl className="mt-5 grid grid-cols-5 gap-2 border-t border-line pt-4">
                {MACRO_KEYS.map((k) => (
                  <div key={k}>
                    <dt className="text-[10px] uppercase tracking-[0.1em] text-muted">
                      {t(`common.macro.${k}`)}
                    </dt>
                    <dd className="font-semibold text-ink tnum">
                      {k === 'Calories' ? formatNumber(f.per_100g[k]) : round(f.per_100g[k])}
                    </dd>
                  </div>
                ))}
              </dl>
              <p className="mt-1 text-[11px] text-muted">{t('foodDetailModal.per100g')}</p>

              <div className="mt-4 flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted">
                {f.serving_size > 0 && (
                  <span>
                    {t('foodDetailModal.serving')}{' '}
                    <span className="text-ink">
                      {round(f.serving_size)}
                      {f.serving_unit}
                    </span>
                  </span>
                )}
                {f.brand && (
                  <span>
                    {t('foodDetailModal.brand')} <span className="text-ink">{f.brand}</span>
                  </span>
                )}
                {f.barcode && (
                  <span>
                    {t('foodDetailModal.barcode')} <span className="text-ink tnum">{f.barcode}</span>
                  </span>
                )}
              </div>

              {f.aliases && f.aliases.length > 0 && (
                <div className="mt-4">
                  <p className="mb-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                    {t('foodDetailModal.aliases')}
                  </p>
                  <div className="flex flex-wrap gap-1.5">
                    {f.aliases.map((a: FoodAlias) => (
                      <Pill key={a.alias} tone="neutral">
                        {a.alias}
                      </Pill>
                    ))}
                  </div>
                </div>
              )}

              <div className="mt-6 flex justify-end">
                <Button onClick={logit}>
                  <LogIcon width={16} height={16} /> {t('foodDetailModal.logThis')}
                </Button>
              </div>
            </>
          )}
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
