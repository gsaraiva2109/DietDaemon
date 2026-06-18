import type { ReactNode } from 'react'
import { Eyebrow } from './ui'

export function PageHeader({
  eyebrow,
  title,
  children,
}: {
  eyebrow?: string
  title: string
  children?: ReactNode
}) {
  return (
    <header className="mb-7 flex flex-wrap items-end justify-between gap-4">
      <div>
        {eyebrow && <Eyebrow>{eyebrow}</Eyebrow>}
        <h1 className="mt-1 text-3xl font-bold tracking-tight text-ink">{title}</h1>
      </div>
      {children}
    </header>
  )
}
