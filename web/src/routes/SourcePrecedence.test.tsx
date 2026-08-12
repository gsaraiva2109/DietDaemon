import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { SourcePrecedence } from './SourcePrecedence'
import { DemoProvider } from '@/lib/demo'
import type { FoodImportStatus } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      precedence: {
        get: vi.fn(),
        set: vi.fn(),
      },
      foodImport: {
        status: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const precedenceGet = vi.mocked(api.precedence.get)
const precedenceSet = vi.mocked(api.precedence.set)
const foodImportStatus = vi.mocked(api.foodImport.status)

function renderPage() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <SourcePrecedence />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  precedenceGet.mockReset()
  precedenceSet.mockReset()
  foodImportStatus.mockReset().mockResolvedValue([])
  localStorage.clear()
})

describe('SourcePrecedence', () => {
  it('shows a spinner while the order is loading', () => {
    precedenceGet.mockReturnValue(new Promise(() => {}))
    renderPage()

    // Spinner appends an ellipsis after the label.
    expect(screen.getByText('Loading source order…')).toBeInTheDocument()
  })

  it('renders the server order with labeled sources', async () => {
    precedenceGet.mockResolvedValue({ order: ['taco', 'openfoodfacts'] })
    renderPage()

    const rows = await screen.findAllByText(/Open Food Facts|TACO/)
    expect(rows.map((r) => r.textContent)).toEqual(['TACO (Brazilian food database)', 'Open Food Facts'])
  })

  it('falls back to the fixed source universe when the server order is empty', async () => {
    precedenceGet.mockResolvedValue({ order: [] })
    renderPage()

    expect(await screen.findByText('Open Food Facts')).toBeInTheDocument()
    expect(screen.getByText('TACO (Brazilian food database)')).toBeInTheDocument()
  })

  it('reorders on move up/down, enabling Save only while dirty', async () => {
    precedenceGet.mockResolvedValue({ order: ['openfoodfacts', 'taco'] })
    renderPage()

    await screen.findByText('Open Food Facts')
    const saveButton = screen.getByRole('button', { name: 'Save order' })
    expect(saveButton).toBeDisabled()

    fireEvent.click(screen.getByLabelText('Move Open Food Facts down'))

    expect(saveButton).toBeEnabled()
    const rows = screen.getAllByText(/Open Food Facts|TACO/)
    expect(rows.map((r) => r.textContent)).toEqual(['TACO (Brazilian food database)', 'Open Food Facts'])
  })

  it('cannot move the first item up or the last item down', async () => {
    precedenceGet.mockResolvedValue({ order: ['openfoodfacts', 'taco'] })
    renderPage()

    await screen.findByText('Open Food Facts')
    expect(screen.getByLabelText('Move Open Food Facts up')).toBeDisabled()
    expect(screen.getByLabelText('Move TACO (Brazilian food database) down')).toBeDisabled()
  })

  it('saves the reordered list and shows a success message', async () => {
    // A tiny fake backend: precedenceGet reflects whatever precedenceSet last
    // wrote. TanStack Query's structural sharing keeps the same `data`
    // reference across a refetch when the content is deep-equal, so a static
    // mock would never flip the `data !== prevData` check that drops the
    // local draft after a save -- this needs the content to actually change.
    let serverOrder = ['openfoodfacts', 'taco']
    precedenceGet.mockImplementation(() => Promise.resolve({ order: [...serverOrder] }))
    precedenceSet.mockImplementation(async (order) => {
      serverOrder = order
      return { status: 'ok' }
    })
    renderPage()

    await screen.findByText('Open Food Facts')
    fireEvent.click(screen.getByLabelText('Move Open Food Facts down'))
    fireEvent.click(screen.getByRole('button', { name: 'Save order' }))

    await waitFor(() => expect(precedenceSet).toHaveBeenCalledWith(['taco', 'openfoodfacts']))
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('shows an error message when saving fails', async () => {
    precedenceGet.mockResolvedValue({ order: ['openfoodfacts', 'taco'] })
    precedenceSet.mockRejectedValue(new Error('network down'))
    renderPage()

    await screen.findByText('Open Food Facts')
    fireEvent.click(screen.getByLabelText('Move Open Food Facts down'))
    fireEvent.click(screen.getByRole('button', { name: 'Save order' }))

    expect(await screen.findByText('network down')).toBeInTheDocument()
  })

  it('hides reorder controls and shows the read-only hint in demo mode', async () => {
    localStorage.setItem('dd.demo', '1')
    precedenceGet.mockResolvedValue({ order: ['openfoodfacts', 'taco'] })
    renderPage()

    expect(await screen.findByText('Source order is read only here.')).toBeInTheDocument()
    expect(screen.queryByLabelText(/Move .* up/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save order' })).not.toBeInTheDocument()
  })

  it('shows the import status empty state when nothing has run yet', async () => {
    precedenceGet.mockResolvedValue({ order: [] })
    foodImportStatus.mockResolvedValue([])
    renderPage()

    expect(await screen.findByText('No imports yet')).toBeInTheDocument()
  })

  it('renders import status rows with result pill and error text', async () => {
    precedenceGet.mockResolvedValue({ order: [] })
    const status: FoodImportStatus[] = [
      { source: 'taco', last_result: 'failed', last_run_at: new Date().toISOString(), last_error: 'boom' },
    ]
    foodImportStatus.mockResolvedValue(status)
    renderPage()

    expect(await screen.findByText('Failed')).toBeInTheDocument()
    expect(screen.getByText('boom')).toBeInTheDocument()
    expect(screen.getByText('just now')).toBeInTheDocument()
  })
})
