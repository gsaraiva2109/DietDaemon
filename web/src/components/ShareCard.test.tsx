import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { ShareCard } from './ShareCard'
import type { Macros } from '@/lib/types'

vi.mock('html-to-image', () => ({ toPng: vi.fn() }))
vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return { ...actual, triggerDownload: vi.fn() }
})

import { toPng } from 'html-to-image'
import { triggerDownload } from '@/lib/api'

const toPngMock = vi.mocked(toPng)
const triggerDownloadMock = vi.mocked(triggerDownload)

const CONSUMED: Macros = { Calories: 850, Protein: 120, Carbs: 200, Fat: 60, Fiber: 25 }

function renderCard() {
  const onClose = vi.fn()
  render(<ShareCard heading="Today" subtitle="Great job" consumed={CONSUMED} onClose={onClose} />)
  return { onClose }
}

beforeEach(() => {
  toPngMock.mockReset()
  triggerDownloadMock.mockReset()
})

describe('ShareCard rendering', () => {
  it('renders the calorie total and macro chips', () => {
    renderCard()
    expect(screen.getByText('850')).toBeInTheDocument()
    expect(screen.getByText('Today')).toBeInTheDocument()
    expect(screen.getByText('Great job')).toBeInTheDocument()
  })
})

describe('ShareCard close', () => {
  it('closes via the X button', () => {
    const { onClose } = renderCard()
    const closeX = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeX)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked, and the backdrop is a real (keyboard-operable) button', () => {
    const { onClose } = renderCard()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    expect(backdrop.tagName).toBe('BUTTON')
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes on Escape', () => {
    const { onClose } = renderCard()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

describe('ShareCard downloadPng (exercises the fixed mime regex)', () => {
  it('downloads a PNG blob extracted from a normal data URL', async () => {
    toPngMock.mockResolvedValue('data:image/png;base64,QUFB')
    renderCard()

    fireEvent.click(screen.getByRole('button', { name: 'Download PNG' }))

    await waitFor(() => expect(triggerDownloadMock).toHaveBeenCalledTimes(1))
    const [blob, filename] = triggerDownloadMock.mock.calls[0]
    expect(filename).toBe('dietdaemon-share.png')
    expect((blob as Blob).type).toBe('image/png')
  })

  it('extracts a non-default mime type correctly', async () => {
    toPngMock.mockResolvedValue('data:image/jpeg;base64,QUFB')
    renderCard()

    fireEvent.click(screen.getByRole('button', { name: 'Download PNG' }))

    await waitFor(() => expect(triggerDownloadMock).toHaveBeenCalledTimes(1))
    const [blob] = triggerDownloadMock.mock.calls[0]
    expect((blob as Blob).type).toBe('image/jpeg')
  })

  it('resolves quickly and falls back to image/png for a pathological meta string with no ";"', async () => {
    // Shaped to stress the old lazy `.*?` scan: a long run of non-';' chars
    // after the colon, with nothing for it to lazily stop at. The [^;]*
    // rewrite has one linear pass regardless, so this must stay fast.
    const meta = `data:${'a'.repeat(50_000)}`
    toPngMock.mockResolvedValue(`${meta},QUFB`)
    renderCard()

    const start = performance.now()
    fireEvent.click(screen.getByRole('button', { name: 'Download PNG' }))
    await waitFor(() => expect(triggerDownloadMock).toHaveBeenCalledTimes(1))
    expect(performance.now() - start).toBeLessThan(500)

    const [blob] = triggerDownloadMock.mock.calls[0]
    // No ';' in meta at all -> exec() finds no match -> falls back to default.
    expect((blob as Blob).type).toBe('image/png')
  })

  it('shows the render error when capture fails', async () => {
    toPngMock.mockRejectedValue(new Error('canvas tainted'))
    renderCard()

    fireEvent.click(screen.getByRole('button', { name: 'Download PNG' }))

    expect(await screen.findByText('canvas tainted')).toBeInTheDocument()
    expect(triggerDownloadMock).not.toHaveBeenCalled()
  })
})

describe('ShareCard copyPng (no Clipboard API in jsdom -> falls back to download)', () => {
  it('falls back to downloading when clipboard image writes are unsupported', async () => {
    toPngMock.mockResolvedValue('data:image/png;base64,QUFB')
    renderCard()

    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))

    await waitFor(() => expect(triggerDownloadMock).toHaveBeenCalledTimes(1))
    expect(screen.queryByText('Copied')).not.toBeInTheDocument()
  })
})

describe('ShareCard copyPng (Clipboard API available)', () => {
  it('writes the PNG via navigator.clipboard.write and shows the Copied state', async () => {
    const write = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('ClipboardItem', function ClipboardItem(items: unknown) { return items })
    Object.defineProperty(navigator, 'clipboard', { value: { write }, configurable: true })
    toPngMock.mockResolvedValue('data:image/png;base64,QUFB')
    renderCard()

    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))

    expect(await screen.findByText('Copied')).toBeInTheDocument()
    expect(write).toHaveBeenCalledTimes(1)
    expect(triggerDownloadMock).not.toHaveBeenCalled()

    vi.unstubAllGlobals()
  })

  it('falls back to the generic copy-error message for a non-Error rejection', async () => {
    const write = vi.fn().mockRejectedValue('nope')
    vi.stubGlobal('ClipboardItem', function ClipboardItem(items: unknown) { return items })
    Object.defineProperty(navigator, 'clipboard', { value: { write }, configurable: true })
    toPngMock.mockResolvedValue('data:image/png;base64,QUFB')
    renderCard()

    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))

    expect(await screen.findByText('Could not copy image')).toBeInTheDocument()

    vi.unstubAllGlobals()
  })
})
