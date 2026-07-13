// Progress photos, grouped by date. Each thumbnail loads its binary via an
// authed blob fetch (an <img src> can't carry the Bearer header), so AuthedImage
// fetches the blob, object-URLs it, and revokes on unmount.

import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { EmptyState, Eyebrow } from './ui'
import { CameraIcon } from './icons'
import { api } from '@/lib/api'
import type { ProgressPhoto } from '@/lib/types'

export function AuthedImage({
  id,
  alt = '',
  className = '',
}: {
  id: string
  alt?: string
  className?: string
}) {
  const [src, setSrc] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    let url: string | null = null
    api.body.photos
      .blob(id)
      .then((blob) => {
        if (cancelled) return
        url = URL.createObjectURL(blob)
        setSrc(url)
      })
      .catch(() => {
        /* leave placeholder visible on failure */
      })
    return () => {
      cancelled = true
      if (url) URL.revokeObjectURL(url)
    }
  }, [id])

  if (!src) {
    return <div className={`animate-pulse bg-surface-2 ${className}`} aria-hidden />
  }
  return <img src={src} alt={alt} className={className} loading="lazy" />
}

function groupByDate(photos: ProgressPhoto[]): [string, ProgressPhoto[]][] {
  const map = new Map<string, ProgressPhoto[]>()
  for (const p of photos) {
    const arr = map.get(p.date) ?? []
    arr.push(p)
    map.set(p.date, arr)
  }
  // Newest date first.
  return [...map.entries()].sort((a, b) => (a[0] < b[0] ? 1 : -1))
}

export function PhotoGrid({
  photos,
  onSelect,
}: {
  photos: ProgressPhoto[]
  onSelect?: (p: ProgressPhoto) => void
}) {
  const { t } = useTranslation()
  const groups = useMemo(() => groupByDate(photos), [photos])

  if (!photos.length) {
    return (
      <EmptyState
        title={t('photoGrid.emptyTitle')}
        hint={t('photoGrid.emptyHint')}
        icon={<CameraIcon />}
      />
    )
  }

  return (
    <div className="space-y-5">
      {groups.map(([date, items]) => (
        <div key={date}>
          <div className="mb-2">
            <Eyebrow>{date}</Eyebrow>
          </div>
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
            {items.map((p) => {
              const viewLabel = t(`photoGrid.views.${p.view}`)
              return (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => onSelect?.(p)}
                  className="group relative aspect-square overflow-hidden rounded-lg border border-line bg-surface-2"
                  aria-label={t('photoGrid.photoAriaLabel', { view: viewLabel, date: p.date })}
                >
                  <AuthedImage
                    id={p.id}
                    alt={t('photoGrid.photoAlt', { view: viewLabel, date: p.date })}
                    className="size-full object-cover transition group-hover:brightness-105"
                  />
                  <span className="absolute bottom-1 left-1 rounded-full bg-ink/60 px-2 py-0.5 text-[10px] font-medium capitalize text-surface">
                    {viewLabel}
                  </span>
                </button>
              )
            })}
          </div>
        </div>
      ))}
    </div>
  )
}
