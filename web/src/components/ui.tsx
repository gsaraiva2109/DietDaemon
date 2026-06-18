// Shared primitives. Single-level cards only (DESIGN.md: never nested cards).

import type { ButtonHTMLAttributes, ReactNode } from 'react'

export function Card({
  children,
  className = '',
  as: Tag = 'div',
}: {
  children: ReactNode
  className?: string
  as?: 'div' | 'section' | 'article' | 'li'
}) {
  return (
    <Tag
      className={`rounded-xl border border-line bg-surface shadow-soft ${className}`}
    >
      {children}
    </Tag>
  )
}

export function Eyebrow({ children }: { children: ReactNode }) {
  return (
    <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
      {children}
    </p>
  )
}

export function Pill({
  children,
  tone = 'neutral',
}: {
  children: ReactNode
  tone?: 'neutral' | 'primary' | 'accent' | 'muted'
}) {
  const tones: Record<string, string> = {
    neutral: 'bg-surface-2 text-ink border-line',
    primary: 'bg-primary-soft text-primary border-transparent',
    accent: 'bg-accent/12 text-accent border-transparent',
    muted: 'bg-surface-2 text-muted border-line',
  }
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium ${tones[tone]}`}
    >
      {children}
    </span>
  )
}

type BtnProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'ghost'
}
export function Button({ variant = 'primary', className = '', ...rest }: BtnProps) {
  const styles =
    variant === 'primary'
      ? 'bg-primary text-primary-ink hover:brightness-[1.05]'
      : 'bg-transparent text-ink hover:bg-surface-2 border border-line'
  return (
    <button
      className={`inline-flex items-center justify-center gap-2 rounded-full px-5 py-2.5 text-sm font-semibold transition disabled:opacity-50 ${styles} ${className}`}
      {...rest}
    />
  )
}

export function EmptyState({
  title,
  hint,
  icon,
}: {
  title: string
  hint?: string
  icon?: ReactNode
}) {
  return (
    <div className="grid place-items-center rounded-xl border border-dashed border-line bg-surface/50 px-6 py-16 text-center">
      {icon && <div className="mb-3 text-muted">{icon}</div>}
      <p className="font-semibold text-ink">{title}</p>
      {hint && <p className="mt-1 max-w-sm text-sm text-muted">{hint}</p>}
    </div>
  )
}

export function Spinner({ label = 'Loading' }: { label?: string }) {
  return (
    <div className="flex items-center gap-3 text-sm text-muted" role="status">
      <span className="size-4 animate-spin rounded-full border-2 border-line border-t-primary" />
      {label}…
    </div>
  )
}
