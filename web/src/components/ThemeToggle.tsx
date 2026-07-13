// Standalone animated light/dark toggle. Extracted from UtilityBar so any
// surface (dashboard side column, headers) can drop it in. Keeps the original
// look: a round bordered button whose sun/moon icon rotates in on theme change.

import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { useTheme } from '@/lib/theme'
import { SunIcon, MoonIcon } from './icons'

export function ThemeToggle({ className = '' }: { className?: string }) {
  const { theme, toggle } = useTheme()
  const { t } = useTranslation()

  return (
    <button
      onClick={toggle}
      aria-label={t('themeToggle.switchTheme', { theme: theme === 'dark' ? 'light' : 'dark' })}
      className={`grid size-9 place-items-center rounded-full border border-line bg-surface text-muted transition hover:text-ink ${className}`}
    >
      <motion.span key={theme} initial={{ rotate: -30, opacity: 0 }} animate={{ rotate: 0, opacity: 1 }}>
        {theme === 'dark' ? <MoonIcon width={18} height={18} /> : <SunIcon width={18} height={18} />}
      </motion.span>
    </button>
  )
}
