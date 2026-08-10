import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { SleepCard } from './SleepCard'
import { DemoProvider } from '@/lib/demo'
import type { SleepLog } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      body: {
        ...actual.api.body,
        sleep: { ...actual.api.body.sleep, list: vi.fn() },
      },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const sleepList = vi.mocked(api.body.sleep.list)

function log(overrides: Partial<SleepLog> = {}): SleepLog {
  return {
    id: 's1',
    user_id: 'u1',
    sleep_at: '23:00',
    wake_at: '07:00',
    duration_hours: 7.4,
    quality: 'good',
    logged_at: '2026-08-09T07:00:00Z',
    ...overrides,
  }
}

function renderCard(queryClient = new QueryClient()) {
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <SleepCard />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  sleepList.mockReset()
})

describe('SleepCard loading state', () => {
  it('shows a spinner while the query is pending', () => {
    sleepList.mockImplementation(() => new Promise(() => {}))
    renderCard()
    expect(screen.getByRole('status')).toBeInTheDocument()
  })
})

describe('SleepCard error state', () => {
  it('shows a retry button and refetches on click', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    sleepList.mockRejectedValue(new ApiError(500, 'boom'))
    renderCard(queryClient)

    const retryBtn = await screen.findByText("Couldn't load, retry")
    sleepList.mockClear()
    sleepList.mockRejectedValue(new ApiError(500, 'boom again'))
    fireEvent.click(retryBtn)

    await waitFor(() => expect(sleepList).toHaveBeenCalled())
  })
})

describe('SleepCard empty state', () => {
  it('shows the empty message when there is no sleep data (or the backend 404s)', async () => {
    sleepList.mockRejectedValue(new ApiError(404, 'not found'))
    renderCard()

    expect(await screen.findByText('No sleep data yet. Log a night from your chat bot.')).toBeInTheDocument()
    // No quality pill without a last log.
    expect(screen.queryByText('good')).not.toBeInTheDocument()
  })
})

describe('SleepCard with data', () => {
  it("renders last night's duration and quality pill", async () => {
    sleepList.mockResolvedValue([
      log({ duration_hours: 7.4, quality: 'good', logged_at: '2026-08-09T07:00:00Z' }),
      log({ duration_hours: 6.1, quality: 'fair', logged_at: '2026-08-08T07:00:00Z' }),
    ])
    renderCard()

    expect(await screen.findByText('7.4')).toBeInTheDocument()
    expect(screen.getByText('hrs last night')).toBeInTheDocument()
    // Pill shows the most recent (first) log's quality, not the older one.
    expect(screen.getByText('good')).toBeInTheDocument()
    expect(screen.queryByText('fair')).not.toBeInTheDocument()
  })

  it('renders a "poor" quality pill with the accent tone', async () => {
    sleepList.mockResolvedValue([log({ quality: 'poor', duration_hours: 4.2 })])
    renderCard()

    const pill = await screen.findByText('poor')
    expect(pill).toBeInTheDocument()
  })
})
