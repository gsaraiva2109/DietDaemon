// Calm centered shell for every auth route (Login/Register/Forgot/Reset/…).
// Extracted from the old TokenGate: LeafIcon badge, max-w-sm column, fadeUp
// reveal, heading + muted subtitle, ThemeToggle in the corner. Auth screens are
// chrome — quiet, single-purpose, no marketing energy.

import type { ReactNode } from 'react'
import { motion } from 'framer-motion'
import { LeafIcon } from './icons'
import { ThemeToggle } from './ThemeToggle'
import { fadeUp } from '@/lib/motion'

export function AuthLayout({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string
  subtitle?: string
  children: ReactNode
  footer?: ReactNode
}) {
  return (
    <div className="relative grid min-h-[100dvh] place-items-center px-6">
      <div className="absolute right-5 top-5">
        <ThemeToggle />
      </div>
      <motion.div variants={fadeUp} initial="hidden" animate="show" className="w-full max-w-sm">
        <div className="mb-7 flex flex-col items-center text-center">
          <span className="mb-4 grid size-14 place-items-center rounded-2xl bg-primary-soft text-primary">
            <LeafIcon width={28} height={28} />
          </span>
          <h1 className="text-2xl font-bold tracking-tight text-ink">{title}</h1>
          {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
        </div>
        {children}
        {footer && <div className="mt-6 text-center text-sm text-muted">{footer}</div>}
      </motion.div>
    </div>
  )
}
