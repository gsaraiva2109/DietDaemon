import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { ExportModal } from './ExportModal'
import { DemoProvider } from '@/lib/demo'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      export: {
        meals: vi.fn(),
        rollups: vi.fn(),
      },
    },
    triggerDownload: vi.fn(),
  }
})

import { api, triggerDownload } from '@/lib/api'

const exportMeals = vi.mocked(api.export.meals)
const exportRollups = vi.mocked(api.export.rollups)
const triggerDownloadMock = vi.mocked(triggerDownload)

function renderModal() {
  const onClose = vi.fn()
  render(
    <DemoProvider>
      <ExportModal onClose={onClose} />
    </DemoProvider>,
  )
  return { onClose }
}

beforeEach(() => {
  exportMeals.mockReset()
  exportRollups.mockReset()
  triggerDownloadMock.mockReset()
  localStorage.clear()
})

describe('ExportModal', () => {
  it('renders the dialog with meals/CSV as the default selection', () => {
    renderModal()

    expect(screen.getByText('Download your data')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Meals' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: 'CSV' })).toHaveAttribute('aria-checked', 'true')
  })

  it('downloads meals as CSV by default and triggers the file save', async () => {
    const blob = new Blob(['a,b'])
    exportMeals.mockResolvedValue(blob)
    renderModal()

    fireEvent.click(screen.getByRole('button', { name: /Download/ }))

    await waitFor(() => expect(exportMeals).toHaveBeenCalledTimes(1))
    expect(exportMeals).toHaveBeenCalledWith('csv', expect.any(String), expect.any(String))
    expect(triggerDownloadMock).toHaveBeenCalledWith(blob, expect.stringContaining('dietdaemon-meals'))
    expect(exportRollups).not.toHaveBeenCalled()
  })

  it('switches to rollups + JSON and downloads with those params', async () => {
    const blob = new Blob(['{}'])
    exportRollups.mockResolvedValue(blob)
    renderModal()

    fireEvent.click(screen.getByRole('radio', { name: 'Rollups' }))
    fireEvent.click(screen.getByRole('radio', { name: 'JSON' }))
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))

    await waitFor(() => expect(exportRollups).toHaveBeenCalledTimes(1))
    expect(exportRollups).toHaveBeenCalledWith('json', expect.any(String), expect.any(String))
    expect(triggerDownloadMock).toHaveBeenCalledWith(blob, expect.stringContaining('dietdaemon-rollups'))
    expect(exportMeals).not.toHaveBeenCalled()
  })

  it('updates the date range from the start/end inputs', async () => {
    exportMeals.mockResolvedValue(new Blob())
    renderModal()

    fireEvent.change(screen.getByLabelText('Start'), { target: { value: '2026-01-01' } })
    fireEvent.change(screen.getByLabelText('End'), { target: { value: '2026-01-15' } })
    fireEvent.click(screen.getByRole('button', { name: /Download/ }))

    await waitFor(() => expect(exportMeals).toHaveBeenCalledWith('csv', '2026-01-01', '2026-01-15'))
  })

  it('shows an error message when the export fails', async () => {
    exportMeals.mockRejectedValue(new Error('network down'))
    renderModal()

    fireEvent.click(screen.getByRole('button', { name: /Download/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent('network down')
  })

  it('closes when Cancel is clicked', () => {
    const { onClose } = renderModal()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the close (X) button is clicked', () => {
    const { onClose } = renderModal()
    const closeButton = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeButton)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked', () => {
    const { onClose } = renderModal()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Enter is pressed on the backdrop (keyboard a11y)', () => {
    const { onClose } = renderModal()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Enter' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Space is pressed on the backdrop (keyboard a11y)', () => {
    const { onClose } = renderModal()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: ' ' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('ignores unrelated keys on the backdrop', () => {
    const { onClose } = renderModal()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Tab' })
    expect(onClose).not.toHaveBeenCalled()
  })
})
