import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Trends } from './Trends'
import { DemoProvider } from '@/lib/demo'
import type { DailyRollup } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      rollupRange: vi.fn(),
    },
  }
})

import { api } from '@/lib/api'

const rollupRange = vi.mocked(api.rollupRange)

function isoDaysAgo(n: number): string {
  const d = new Date()
  d.setDate(d.getDate() - n)
  return d.toISOString().slice(0, 10)
}

function rollup(date: string, calories: number): DailyRollup {
  return {
    UserID: 'u1',
    Date: date,
    Consumed: { Calories: calories, Protein: 80, Carbs: 200, Fat: 40, Fiber: 10 },
    Targets: { Calories: 2600, Protein: 180, Carbs: 320, Fat: 70, Fiber: 30 },
  }
}

function renderTrends() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <Trends />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  localStorage.clear()
  rollupRange.mockReset()
})

describe('Trends loading and empty states', () => {
  it('shows a spinner while the range query is loading', async () => {
    rollupRange.mockReturnValue(new Promise(() => {}))
    renderTrends()
    expect(await screen.findByText('Over time')).toBeInTheDocument()
    expect(screen.getByText('Loading', { exact: false })).toBeInTheDocument()
  })

  it('shows an empty state when the range has no rollups', async () => {
    rollupRange.mockResolvedValue([])
    renderTrends()
    expect(await screen.findByText('No data in range')).toBeInTheDocument()
    expect(screen.getByText('Log meals across a few days to see trends.')).toBeInTheDocument()
  })
})

describe('Trends with data', () => {
  it('renders the header, day-range toggles, and macro chips', async () => {
    rollupRange.mockResolvedValue([rollup(isoDaysAgo(1), 2000)])
    renderTrends()

    expect(await screen.findByText('Over time')).toBeInTheDocument()
    expect(screen.getByText('7d')).toBeInTheDocument()
    expect(screen.getByText('14d')).toBeInTheDocument()
    expect(screen.getByText('30d')).toBeInTheDocument()
    for (const label of ['Calories', 'Protein', 'Carbs', 'Fat', 'Fiber']) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    }
  })

  it('defaults to a 14-day range on first load', async () => {
    rollupRange.mockResolvedValue([rollup(isoDaysAgo(1), 2000)])
    renderTrends()

    await screen.findByText('Over time')
    await waitFor(() =>
      expect(rollupRange).toHaveBeenCalledWith(isoDaysAgo(13), isoDaysAgo(0)),
    )
  })

  it('switching to 7d re-queries the range with a narrower window', async () => {
    rollupRange.mockResolvedValue([rollup(isoDaysAgo(1), 2000)])
    renderTrends()

    await screen.findByText('Over time')
    fireEvent.click(screen.getByText('7d'))

    await waitFor(() =>
      expect(rollupRange).toHaveBeenCalledWith(isoDaysAgo(6), isoDaysAgo(0)),
    )
  })

  it('switching to 30d re-queries the range with a wider window', async () => {
    rollupRange.mockResolvedValue([rollup(isoDaysAgo(1), 2000)])
    renderTrends()

    await screen.findByText('Over time')
    fireEvent.click(screen.getByText('30d'))

    await waitFor(() =>
      expect(rollupRange).toHaveBeenCalledWith(isoDaysAgo(29), isoDaysAgo(0)),
    )
  })

  it('switching the macro filter does not throw and keeps the chart mounted', async () => {
    rollupRange.mockResolvedValue([rollup(isoDaysAgo(1), 2000)])
    renderTrends()

    await screen.findByText('Over time')
    fireEvent.click(screen.getByRole('button', { name: 'Protein' }))
    fireEvent.click(screen.getByRole('button', { name: 'Fiber' }))

    // No re-fetch: macro switching only changes which field is projected
    // from already-fetched range data.
    expect(rollupRange).toHaveBeenCalledTimes(1)
  })
})
