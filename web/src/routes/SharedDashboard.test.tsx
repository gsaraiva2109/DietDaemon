import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { SharedDashboard } from './SharedDashboard'
import { DEMO_MEALS, DEMO_TARGETS, DEMO_CONSUMED } from '@/lib/demoData'
import type { DailyRollup, Meal, StreakResponse, WeeklyBudgetResponse, BodyCompositionSummary } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    sharedApi: vi.fn(),
  }
})

import { sharedApi } from '@/lib/api'

const sharedApiMock = vi.mocked(sharedApi)

const TODAY: DailyRollup = { UserID: 'u1', Date: '2024-06-20', Consumed: DEMO_CONSUMED, Targets: DEMO_TARGETS }
const MEALS: Meal[] = DEMO_MEALS
const STREAK: StreakResponse = { current_days: 5 }
const BUDGET: WeeklyBudgetResponse = {
  calories: { plain: 2900, effective: 3050 },
  protein: { plain: 175, effective: 182 },
}
const BODY: BodyCompositionSummary = {
  current_weight_kg: 77.6,
  start_weight_kg: 80,
  change_kg: -2.4,
  trend_direction: 'down',
}

function mockShared(overrides: {
  rollupToday?: () => Promise<DailyRollup>
  meals?: () => Promise<Meal[]>
  targets?: () => Promise<{ UserID: string; Targets: DailyRollup['Targets'] }>
  budgetWeekly?: () => Promise<WeeklyBudgetResponse>
  bodySummary?: () => Promise<BodyCompositionSummary>
  streak?: () => Promise<StreakResponse>
}) {
  sharedApiMock.mockReturnValue({
    rollupToday: overrides.rollupToday ?? (() => Promise.resolve(TODAY)),
    meals: overrides.meals ?? (() => Promise.resolve(MEALS)),
    targets: overrides.targets ?? (() => Promise.resolve({ UserID: 'u1', Targets: DEMO_TARGETS })),
    budgetWeekly: overrides.budgetWeekly ?? (() => Promise.resolve(BUDGET)),
    bodySummary: overrides.bodySummary ?? (() => Promise.resolve(BODY)),
    streak: overrides.streak ?? (() => Promise.resolve(STREAK)),
  })
}

function renderShared(token = 'tok_123') {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/shared/${token}`]}>
        <Routes>
          <Route path="/shared/:token" element={<SharedDashboard />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  sharedApiMock.mockReset()
})

describe('SharedDashboard top-level state ternary', () => {
  it('loading branch: shows a spinner while the rollup is in flight', async () => {
    mockShared({ rollupToday: () => new Promise(() => {}) })
    renderShared()

    expect(await screen.findByText('Loading dashboard', { exact: false })).toBeInTheDocument()
  })

  it('invalid branch: shows the invalid-link empty state when the rollup errors (revoked/unknown token)', async () => {
    mockShared({ rollupToday: () => Promise.reject(new Error('404')) })
    renderShared()

    expect(await screen.findByText('This link is no longer valid')).toBeInTheDocument()
    expect(
      screen.getByText('It may have been revoked, or never existed. Ask whoever shared it for a new one.'),
    ).toBeInTheDocument()
  })

  it('invalid branch: also triggers when the rollup resolves falsy', async () => {
    // Belt-and-suspenders branch in the original ternary: `today.isError || !today.data`.
    mockShared({ rollupToday: () => Promise.resolve(undefined as unknown as DailyRollup) })
    renderShared()

    expect(await screen.findByText('This link is no longer valid')).toBeInTheDocument()
  })

  it('content branch: renders macro rings, budget/weight cards, and the streak pill', async () => {
    mockShared({})
    renderShared()

    expect(await screen.findByText('Today')).toBeInTheDocument()
    expect(await screen.findByText('5-day streak')).toBeInTheDocument()
    expect(await screen.findByText('Weekly budget')).toBeInTheDocument()
    expect(screen.getByText('3,050')).toBeInTheDocument() // effective calorie budget
    expect(await screen.findByText('Weight')).toBeInTheDocument()
    // formatNumber rounds to an integer regardless of the round() precision passed in.
    expect(screen.getByText('78')).toBeInTheDocument()
  })

  it('content branch: hides the weight card when current_weight_kg is 0', async () => {
    mockShared({ bodySummary: () => Promise.resolve({ ...BODY, current_weight_kg: 0 }) })
    renderShared()

    await screen.findByText('Today')
    expect(screen.queryByText('Weight')).not.toBeInTheDocument()
  })

  it('content branch: shows the section error note when budget/body queries fail', async () => {
    mockShared({
      budgetWeekly: () => Promise.reject(new Error('boom')),
      bodySummary: () => Promise.reject(new Error('boom')),
    })
    renderShared()

    await screen.findByText('Today')
    expect(await screen.findByText("Couldn't load this section.")).toBeInTheDocument()
  })
})

describe('SharedDashboard recent-meals ternary', () => {
  it('loading branch: shows a spinner for the meals section', async () => {
    mockShared({ meals: () => new Promise(() => {}) })
    renderShared()

    await screen.findByText('Today')
    expect(await screen.findAllByRole('status')).not.toHaveLength(0)
  })

  it('error branch: shows the section error note', async () => {
    mockShared({ meals: () => Promise.reject(new Error('boom')) })
    renderShared()

    await screen.findByText('Today')
    expect(await screen.findByText("Couldn't load this section.")).toBeInTheDocument()
  })

  it('empty branch: shows "no meals" copy', async () => {
    mockShared({ meals: () => Promise.resolve([]) })
    renderShared()

    await screen.findByText('Today')
    expect(await screen.findByText('No meals logged yet.')).toBeInTheDocument()
  })

  it('list branch: renders each meal', async () => {
    mockShared({})
    renderShared()

    expect(await screen.findByText(MEALS[0].RawText)).toBeInTheDocument()
    expect(screen.getByText(MEALS[1].RawText)).toBeInTheDocument()
  })
})
