// Shared primitives. Single-level cards only (DESIGN.md: never nested cards).

import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from 'react'
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

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

// Toggle, an accessible on/off switch (role="switch"). No native <input
// type="checkbox"> equivalent looks like a switch, so this is a small custom
// control rather than a dependency.
export function Toggle({
  checked,
  onChange,
  disabled,
  label,
}: {
  checked: boolean
  onChange: (next: boolean) => void
  disabled?: boolean
  label: string
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative h-6 w-11 shrink-0 rounded-full border transition disabled:opacity-50 ${
        checked ? 'border-transparent bg-primary' : 'border-line bg-surface-2'
      }`}
    >
      <span
        className={`absolute top-0.5 left-0.5 size-5 rounded-full bg-white shadow-soft transition-transform ${
          checked ? 'translate-x-[20px]' : 'translate-x-0'
        }`}
      />
    </button>
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

export function Spinner({ label }: { label?: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-3 text-sm text-muted" role="status">
      <span className="size-4 animate-spin rounded-full border-2 border-line border-t-primary" />
      {label ?? t('common.loading')}…
    </div>
  )
}

// Input, the calm bordered field, extracted from the old TokenGate so every
// auth form shares one focus/disabled treatment. No parallel styling.
export function Input({ className = '', ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={`w-full rounded-xl border border-line bg-surface px-4 py-3 text-ink outline-none transition placeholder:text-muted/70 focus:border-primary disabled:opacity-60 ${className}`}
      {...rest}
    />
  )
}

// Field, label + input + inline error, wired for a11y (label htmlFor,
// aria-invalid, aria-describedby). The visible error pairs colour with text.
export function Field({
  label,
  error,
  hint,
  className = '',
  inputClassName,
  id,
  ...rest
}: InputHTMLAttributes<HTMLInputElement> & {
  label: string
  error?: string
  hint?: string
  // Extra classes for the underlying <input> only (e.g. flagging low
  // OCR-confidence fields), independent of `className` on the wrapping div.
  inputClassName?: string
}) {
  const autoId = useId()
  const fieldId = id ?? autoId
  const errId = `${fieldId}-err`
  return (
    <div className={`flex flex-col gap-1.5 ${className}`}>
      <label htmlFor={fieldId} className="text-sm font-medium text-ink">
        {label}
      </label>
      <Input
        id={fieldId}
        className={inputClassName}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errId : undefined}
        {...rest}
      />
      {hint && !error && <p className="text-xs text-muted">{hint}</p>}
      {error && (
        <p id={errId} className="text-sm font-medium text-accent" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

// FormError, a single form-level error line (generic auth copy). role=alert so
// it's announced; colour is never the only signal (it always carries text).
export function FormError({ children }: { children: ReactNode }) {
  if (!children) return null
  return (
    <p className="text-sm font-medium text-accent" role="alert">
      {children}
    </p>
  )
}
