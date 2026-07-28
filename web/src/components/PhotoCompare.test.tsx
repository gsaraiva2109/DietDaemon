import { describe, it, expect, vi, beforeEach, afterAll } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import i18n from '@/lib/i18n'
import { PhotoCompare } from './PhotoCompare'
import type { ProgressPhoto } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      body: {
        ...actual.api.body,
        photos: {
          ...actual.api.body.photos,
          blob: vi.fn(),
        },
      },
    },
  }
})

import { api } from '@/lib/api'

const photosBlob = vi.mocked(api.body.photos.blob)

function photo(id: string, date: string, view: string): ProgressPhoto {
  return { id, user_id: 'u1', date, view, mime_type: 'image/jpeg', created_at: date }
}

// isoDaysAgo/relativeCaption in the component work off local calendar days,
// so build fixture dates relative to "now" to reliably hit each branch of
// relativeCaption (today / daysAgo / weeksAgo / monthsAgo) regardless of
// when this test happens to run.
function isoDaysAgo(daysAgo: number): string {
  const d = new Date()
  d.setDate(d.getDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

const TODAY = photo('p0', isoDaysAgo(0), 'front') // relativeCaption: today
const RECENT = photo('p1', isoDaysAgo(3), 'front') // relativeCaption: daysAgo
const WEEKS = photo('p2', isoDaysAgo(20), 'side') // relativeCaption: weeksAgo
const MONTHS = photo('p3', isoDaysAgo(200), 'back') // relativeCaption: monthsAgo

const originalCreateObjectURL = URL.createObjectURL
const originalRevokeObjectURL = URL.revokeObjectURL

beforeEach(() => {
  photosBlob.mockReset()
  photosBlob.mockResolvedValue(new Blob(['x']))
  URL.createObjectURL = vi.fn(() => 'blob:mock-url')
  URL.revokeObjectURL = vi.fn()
})

// Restore real URL object-URL methods after this file's tests, so other test
// files in the same run aren't affected.
afterAll(() => {
  URL.createObjectURL = originalCreateObjectURL
  URL.revokeObjectURL = originalRevokeObjectURL
})

function renderCompare(photos: ProgressPhoto[] = [MONTHS, WEEKS, RECENT, TODAY]) {
  const onClose = vi.fn()
  render(<PhotoCompare photos={photos} onClose={onClose} />)
  return { onClose }
}

describe('PhotoCompare', () => {
  it('renders with oldest as before and newest as after by default', () => {
    renderCompare()

    const [beforeSelect, afterSelect] = screen.getAllByRole('combobox') as HTMLSelectElement[]
    expect(beforeSelect.value).toBe(MONTHS.id)
    expect(afterSelect.value).toBe(TODAY.id)
  })

  it('shows the monthsAgo and today captions for the default before/after', () => {
    renderCompare()
    expect(screen.getByText(MONTHS.date)).toBeInTheDocument()
    expect(screen.getByText(i18n.t('photoCompare.monthsAgo', { count: 7 }))).toBeInTheDocument()
    expect(screen.getByText(i18n.t('photoCompare.today'))).toBeInTheDocument()
  })

  it('shows the daysAgo and weeksAgo captions when those photos are selected', () => {
    renderCompare()

    const [beforeSelect, afterSelect] = screen.getAllByRole('combobox')
    fireEvent.change(beforeSelect, { target: { value: RECENT.id } })
    expect(screen.getByText(i18n.t('photoCompare.daysAgo', { count: 3 }))).toBeInTheDocument()

    fireEvent.change(afterSelect, { target: { value: WEEKS.id } })
    expect(screen.getByText(i18n.t('photoCompare.weeksAgo', { count: 3 }))).toBeInTheDocument()
  })

  it('lets the user pick a different before/after photo', () => {
    renderCompare()

    const [beforeSelect, afterSelect] = screen.getAllByRole('combobox')
    fireEvent.change(beforeSelect, { target: { value: WEEKS.id } })
    expect(screen.getAllByText(WEEKS.date)).not.toHaveLength(0)

    fireEvent.change(afterSelect, { target: { value: RECENT.id } })
    expect(screen.getAllByText(RECENT.date).length).toBeGreaterThan(0)
  })

  it('renders without crashing when there are no photos to compare', () => {
    renderCompare([])

    expect(screen.getAllByRole('combobox')).toHaveLength(2)
    expect(screen.queryByText(/ago$/)).not.toBeInTheDocument()
  })

  it('closes when the close (X) button is clicked', () => {
    const { onClose } = renderCompare()
    const closeButton = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeButton)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked', () => {
    const { onClose } = renderCompare()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Enter is pressed on the backdrop (keyboard a11y)', () => {
    const { onClose } = renderCompare()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Enter' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Space is pressed on the backdrop (keyboard a11y)', () => {
    const { onClose } = renderCompare()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: ' ' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('ignores unrelated keys on the backdrop', () => {
    const { onClose } = renderCompare()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Tab' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('closes on the Escape key (global listener)', () => {
    const { onClose } = renderCompare()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
