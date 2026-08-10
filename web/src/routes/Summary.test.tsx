import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Summary } from './Summary'
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

function rollup(date: string, consumedCal: number, targetCal = 2200): DailyRollup {
  return {
    UserID: 'u1',
    Date: date,
    Consumed: { Calories: consumedCal, Protein: 150, Carbs: 200, Fat: 60, Fiber: 20 },
    Targets: { Calories: targetCal, Protein: 180, Carbs: 220, Fat: 70, Fiber: 30 },
  }
}

// Three logged days plus one unlogged (Calories 0, excluded from stats) and
// deliberately unsorted so `.at(-1)` (last logged) drives the `target`.
const ROLLUPS: DailyRollup[] = [
  rollup('2026-07-23', 0), // not logged, filtered out
  rollup('2026-07-24', 2000), // dist 200, ratio 0.909 -> on target
  rollup('2026-07-25', 1500), // dist 700, ratio 0.68 -> off target
  rollup('2026-07-26', 2200), // dist 0, ratio 1.0 -> on target (best, and last logged)
]

function renderSummary() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <Summary />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  rollupRange.mockReset()
})

describe('Summary', () => {
  it('shows a spinner while loading', () => {
    rollupRange.mockReturnValue(new Promise(() => {}))
    renderSummary()

    expect(screen.getByRole('status')).toBeInTheDocument()
    expect(screen.queryByText('No data in range')).not.toBeInTheDocument()
  })

  it('shows the empty state when there is no data in range', async () => {
    rollupRange.mockResolvedValue([])
    renderSummary()

    expect(await screen.findByText('No data in range')).toBeInTheDocument()
  })

  it('shows the empty state when every day has zero calories logged', async () => {
    rollupRange.mockResolvedValue([rollup('2026-07-26', 0)])
    renderSummary()

    expect(await screen.findByText('No data in range')).toBeInTheDocument()
  })

  it('computes and renders averages, adherence, and best/worst days', async () => {
    rollupRange.mockResolvedValue(ROLLUPS)
    renderSummary()

    await screen.findByText('Avg calories / day')

    // Tile values are scoped via `within` because the same rounded numbers
    // (avg Calories/Protein) also appear inside the per-macro MacroBar rows.
    const calTile = screen.getByText('Avg calories / day').closest('div')!.parentElement as HTMLElement
    expect(within(calTile).getByText('1,900')).toBeInTheDocument() // (2000+1500+2200)/3

    const proteinTile = screen.getByText('Avg protein / day').closest('div')!.parentElement as HTMLElement
    expect(within(proteinTile).getByText('150')).toBeInTheDocument()

    const onTargetTile = screen.getByText('Days on target').closest('div')!.parentElement as HTMLElement
    expect(within(onTargetTile).getByText('2')).toBeInTheDocument() // 2 of 3 days within +/-10%
    expect(within(onTargetTile).getByText('of 3')).toBeInTheDocument()

    const adherenceTile = screen.getByText('Calorie adherence').closest('div')!.parentElement as HTMLElement
    expect(within(adherenceTile).getByText('86')).toBeInTheDocument() // round(((.909+.68+1)/3)*100)

    // Best day (dist 0) and worst day (dist 700) both render a kcal label.
    expect(screen.getByText('2,200 kcal')).toBeInTheDocument()
    expect(screen.getByText('1,500 kcal')).toBeInTheDocument()
  })

  it('requests a wider date range when a different day-count is selected', async () => {
    rollupRange.mockResolvedValue(ROLLUPS)
    renderSummary()

    await screen.findByText('Avg calories / day')
    const [firstStart] = rollupRange.mock.calls[0]

    fireEvent.click(screen.getByText('14d'))

    await waitFor(() => expect(rollupRange.mock.calls.length).toBeGreaterThan(1))
    const lastCall = rollupRange.mock.calls.at(-1)!
    const [newStart] = lastCall
    // 14 days back starts strictly earlier than 7 days back.
    expect(newStart < firstStart).toBe(true)
  })

  it('opens and closes the export modal', async () => {
    rollupRange.mockResolvedValue(ROLLUPS)
    renderSummary()

    await screen.findByText('Avg calories / day')
    fireEvent.click(screen.getByText('Export'))

    expect(await screen.findByText('Download your data')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByText('Download your data')).not.toBeInTheDocument()
  })
})
