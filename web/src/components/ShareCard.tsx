// Share card. Renders a screenshot-ready summary card (calories +
// macro row) and lets the user save it as a PNG or copy it to the clipboard.
// Pure client-side: it captures a styled DOM node with html-to-image, so it
// works in demo mode too (it only reads the macros passed in).
//
// html-to-image does not reliably resolve CSS custom properties expressed in
// oklch(). To keep the captured node faithful we resolve every macro color via
// cssVar() into explicit inline style strings and pass an explicit
// backgroundColor to toPng().

import { useEffect, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { toPng } from 'html-to-image'
import type { Macros } from '@/lib/types'
import { triggerDownload } from '@/lib/api'
import { cssVar, formatNumber } from '@/lib/format'
import { scaleIn } from '@/lib/motion'
import { Button } from './ui'
import { CloseIcon, DownloadIcon, CopyIcon, CheckIcon, LeafIcon } from './icons'

interface Props {
  heading: string
  subtitle?: string
  consumed: Macros
  onClose: () => void
}

interface MacroChip {
  label: string
  value: number
  color: string
}

/** dataURL -> Blob, so we can both download and copy a real PNG. */
function dataUrlToBlob(dataUrl: string): Blob {
  const [meta, b64] = dataUrl.split(',')
  const mime = /:(.*?);/.exec(meta)?.[1] ?? 'image/png'
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new Blob([bytes], { type: mime })
}

export function ShareCard({ heading, subtitle, consumed, onClose }: Props) {
  const captureRef = useRef<HTMLDivElement>(null)
  const [copied, setCopied] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  // Resolve tokens to concrete colors once per render (capture-safe).
  const surface = cssVar('--color-surface') || '#ffffff'
  const ink = cssVar('--color-ink') || '#1a1a1a'
  const muted = cssVar('--color-muted') || '#6b7280'
  const primarySoft = cssVar('--color-primary-soft') || '#e6efe6'
  const calColor = cssVar('--color-cal') || ink

  const chips: MacroChip[] = [
    { label: 'Protein', value: consumed.Protein, color: cssVar('--color-protein') || ink },
    { label: 'Carbs', value: consumed.Carbs, color: cssVar('--color-carbs') || ink },
    { label: 'Fat', value: consumed.Fat, color: cssVar('--color-fat') || ink },
    { label: 'Fiber', value: consumed.Fiber, color: cssVar('--color-fiber') || ink },
  ]

  async function render(): Promise<Blob> {
    const node = captureRef.current
    if (!node) throw new Error('Nothing to capture')
    const dataUrl = await toPng(node, { pixelRatio: 2, backgroundColor: surface })
    return dataUrlToBlob(dataUrl)
  }

  async function downloadPng() {
    if (busy) return
    setError(null)
    setBusy(true)
    try {
      const blob = await render()
      triggerDownload(blob, 'dietdaemon-share.png')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not render image')
    } finally {
      setBusy(false)
    }
  }

  async function copyPng() {
    if (busy) return
    setError(null)
    setBusy(true)
    try {
      const blob = await render()
      const canClip =
        typeof navigator !== 'undefined' &&
        'clipboard' in navigator &&
        typeof window.ClipboardItem !== 'undefined' &&
        typeof navigator.clipboard.write === 'function'
      if (canClip) {
        await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob })])
        setCopied(true)
        setTimeout(() => setCopied(false), 1800)
      } else {
        // Graceful fallback: clipboard image writes unsupported -> download.
        triggerDownload(blob, 'dietdaemon-share.png')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not copy image')
    } finally {
      setBusy(false)
    }
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
          aria-label="Share card"
          variants={scaleIn}
          initial="hidden"
          animate="show"
          exit="hidden"
          className="relative w-full max-w-[460px] rounded-xl border border-line bg-surface p-6 shadow-lift"
          style={{ zIndex: 1500 }}
        >
          <div className="mb-5 flex items-start justify-between">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted">
                Share
              </p>
              <h2 className="mt-1 text-xl font-bold text-ink">Share your day</h2>
            </div>
            <button onClick={onClose} aria-label="Close" className="text-muted hover:text-ink">
              <CloseIcon />
            </button>
          </div>

          {/* Captured node, explicit inline colors only (no Tailwind tokens). */}
          <div className="grid place-items-center">
            <div
              ref={captureRef}
              style={{
                width: 420,
                padding: 28,
                borderRadius: 24,
                background: `linear-gradient(155deg, ${primarySoft} 0%, ${surface} 60%)`,
                color: ink,
                fontFamily: 'inherit',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: calColor }}>
                <LeafIcon width={20} height={20} />
                <span style={{ fontSize: 13, fontWeight: 700, letterSpacing: '0.12em', textTransform: 'uppercase', color: ink }}>
                  DietDaemon
                </span>
              </div>

              <div style={{ marginTop: 22 }}>
                <div style={{ fontSize: 22, fontWeight: 700, lineHeight: 1.1, color: ink }}>
                  {heading}
                </div>
                {subtitle && (
                  <div style={{ marginTop: 4, fontSize: 13, fontWeight: 500, color: muted }}>
                    {subtitle}
                  </div>
                )}
              </div>

              <div style={{ marginTop: 20, display: 'flex', alignItems: 'baseline', gap: 8 }}>
                <span style={{ fontSize: 52, fontWeight: 800, lineHeight: 1, color: calColor }}>
                  {formatNumber(consumed.Calories)}
                </span>
                <span style={{ fontSize: 16, fontWeight: 600, color: muted }}>kcal</span>
              </div>

              <div style={{ marginTop: 22, display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 10 }}>
                {chips.map((c) => (
                  <div
                    key={c.label}
                    style={{
                      borderRadius: 14,
                      padding: '10px 8px',
                      background: surface,
                      textAlign: 'center',
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 6 }}>
                      <span style={{ display: 'block', width: 8, height: 8, borderRadius: 999, background: c.color }} />
                    </div>
                    <div style={{ fontSize: 17, fontWeight: 700, color: ink }}>
                      {Math.round(c.value)}
                      <span style={{ fontSize: 11, fontWeight: 600, color: muted }}>g</span>
                    </div>
                    <div style={{ marginTop: 2, fontSize: 10, fontWeight: 600, letterSpacing: '0.06em', textTransform: 'uppercase', color: muted }}>
                      {c.label}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {error && (
            <p className="mt-4 text-sm font-medium text-accent" role="alert">
              {error}
            </p>
          )}

          <div className="mt-6 flex justify-end gap-2">
            <Button variant="ghost" onClick={copyPng} disabled={busy}>
              {copied ? (
                <>
                  <CheckIcon width={18} height={18} />
                  Copied
                </>
              ) : (
                <>
                  <CopyIcon width={18} height={18} />
                  Copy
                </>
              )}
            </Button>
            <Button onClick={downloadPng} disabled={busy}>
              <DownloadIcon width={18} height={18} />
              Download PNG
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
