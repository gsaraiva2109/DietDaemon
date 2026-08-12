import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { FastingCard } from './FastingCard'
import { DemoProvider } from '@/lib/demo'
import type { Fast } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      fasting: {
        ...actual.api.fasting,
        active: vi.fn(),
        history: vi.fn(),
        start: vi.fn(),
        end: vi.fn(),
      },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const active = vi.mocked(api.fasting.active)
const history = vi.mocked(api.fasting.history)
const start = vi.mocked(api.fasting.start)
const end = vi.mocked(api.fasting.end)

function fast(overrides: Partial<Fast> = {}): Fast {
  return {
    id: 'f1',
    user_id: 'u1',
    start_at: new Date().toISOString(),
    end_at: null,
    target_hours: 16,
    completed: false,
    created_at: '',
    ...overrides,
  }
}

function renderCard(queryClient = new QueryClient()) {
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <FastingCard />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  active.mockReset()
  history.mockReset().mockResolvedValue([])
  start.mockReset().mockResolvedValue(fast())
  end.mockReset().mockResolvedValue(fast())
})

describe('FastingCard idle state (no active fast)', () => {
  it('shows the no-active-fast message and starts a fast with the selected target', async () => {
    active.mockRejectedValue(new ApiError(404, 'not found'))
    renderCard()

    expect(await screen.findByText('No active fast. Pick a window and start.')).toBeInTheDocument()

    // Default target is 16h; switch to 18h before starting.
    fireEvent.click(screen.getByRole('button', { name: '18h' }))
    expect(screen.getByRole('button', { name: '18h' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: '16h' })).toHaveAttribute('aria-pressed', 'false')

    fireEvent.click(screen.getByRole('button', { name: 'Start 18h fast' }))
    await waitFor(() => expect(start).toHaveBeenCalledWith(18))
  })

  it('shows the last fast duration when history has a completed fast', async () => {
    active.mockRejectedValue(new ApiError(404, 'not found'))
    const startAt = new Date('2026-01-01T00:00:00Z')
    const endAt = new Date('2026-01-01T16:00:00Z') // 16h later
    history.mockResolvedValue([
      fast({ start_at: startAt.toISOString(), end_at: endAt.toISOString() }),
    ])
    renderCard()

    expect(await screen.findByText('16.0h')).toBeInTheDocument()
    // "Ready for another?" is a bare text node sharing a <p> with the
    // duration span, so match on the paragraph's full text instead of an
    // exact standalone string.
    expect(
      screen.getByText(
        (_, node) => node?.tagName === 'P' && (node.textContent ?? '').includes('Ready for another?'),
      ),
    ).toBeInTheDocument()
  })
})

describe('FastingCard error state', () => {
  it('shows a retry button and refetches on click', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    active.mockRejectedValue(new ApiError(500, 'boom'))
    renderCard(queryClient)

    const retryBtn = await screen.findByText("Couldn't load, retry")
    active.mockClear()
    active.mockRejectedValue(new ApiError(500, 'boom again'))
    fireEvent.click(retryBtn)

    await waitFor(() => expect(active).toHaveBeenCalled())
  })
})

describe('FastingCard active fast', () => {
  it('renders elapsed time, target-reached message, in-progress pill, and ends the fast', async () => {
    // 2h15m ago, target 2h -> already past target so "reached" is true.
    const startAt = new Date(Date.now() - (2 * 3_600_000 + 15 * 60_000))
    active.mockResolvedValue(fast({ start_at: startAt.toISOString(), target_hours: 2 }))
    renderCard()

    expect(await screen.findByText('In progress')).toBeInTheDocument()
    expect(screen.getByText('2:15')).toBeInTheDocument()
    expect(screen.getByText('of 2h')).toBeInTheDocument()
    expect(screen.getByText('Target reached, nice work.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'End fast' }))
    await waitFor(() => expect(end).toHaveBeenCalledTimes(1))
  })

  it('shows the "Ending…" label and disables the button while the end mutation is pending', async () => {
    const startAt = new Date(Date.now() - 60 * 60_000) // 1h ago
    active.mockResolvedValue(fast({ start_at: startAt.toISOString(), target_hours: 16 }))
    let resolveEnd: (v: Fast) => void = () => {}
    end.mockImplementation(() => new Promise((resolve) => { resolveEnd = resolve }))
    renderCard()

    const endBtn = await screen.findByRole('button', { name: 'End fast' })
    fireEvent.click(endBtn)

    const endingBtn = await screen.findByRole('button', { name: 'Ending…' })
    expect(endingBtn).toBeDisabled()

    resolveEnd(fast())
  })
})
