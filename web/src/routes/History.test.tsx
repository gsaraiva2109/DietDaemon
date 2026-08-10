import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { History } from './History'
import { DemoProvider } from '@/lib/demo'
import type { Meal, ParserTier } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      meals: vi.fn(),
    },
  }
})

import { api } from '@/lib/api'

const meals = vi.mocked(api.meals)

function meal(opts: {
  id: string
  at: string
  rawText: string
  tier: ParserTier
  foodName?: string
}): Meal {
  return {
    ID: opts.id,
    UserID: 'u1',
    At: opts.at,
    RawText: opts.rawText,
    Confidence: 0.95,
    ParserTier: opts.tier,
    CreatedAt: opts.at,
    PlanSlotID: '',
    PlanOptionID: '',
    Items: [
      {
        Parsed: { RawPhrase: opts.rawText, Quantity: 1, Unit: '', NormalizedGrams: 100, Locale: '' },
        Match: {
          FoodID: 'f1',
          Name: opts.foodName ?? 'Food',
          Source: 'taco',
          Per100g: { Calories: 200, Protein: 10, Carbs: 20, Fat: 5, Fiber: 2 },
          MatchScore: 1,
        },
        Macros: { Calories: 200, Protein: 10, Carbs: 20, Fat: 5, Fiber: 2 },
      },
    ],
  }
}

const now = new Date()
const yesterday = new Date()
yesterday.setDate(now.getDate() - 1)

const CHICKEN_TODAY = meal({ id: 'm1', at: now.toISOString(), rawText: 'grilled chicken', tier: 0, foodName: 'Chicken breast' })
const RICE_TODAY = meal({ id: 'm2', at: now.toISOString(), rawText: 'rice bowl', tier: 2, foodName: 'Rice' })
const OATS_YESTERDAY = meal({ id: 'm3', at: yesterday.toISOString(), rawText: 'oats and banana', tier: 1, foodName: 'Oats' })

function renderHistory() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <History />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  meals.mockReset()
})

describe('History', () => {
  it('shows a spinner while loading', () => {
    meals.mockReturnValue(new Promise(() => {}))
    renderHistory()

    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('shows the empty state when there are no meals at all', async () => {
    meals.mockResolvedValue([])
    renderHistory()

    expect(await screen.findByText('No meals logged yet')).toBeInTheDocument()
  })

  it('groups meals by day, with Today/Yesterday relative labels', async () => {
    meals.mockResolvedValue([CHICKEN_TODAY, RICE_TODAY, OATS_YESTERDAY])
    renderHistory()

    expect(await screen.findByText('Today')).toBeInTheDocument()
    expect(screen.getByText('Yesterday')).toBeInTheDocument()
    expect(screen.getByText('grilled chicken')).toBeInTheDocument()
    expect(screen.getByText('rice bowl')).toBeInTheDocument()
    expect(screen.getByText('oats and banana')).toBeInTheDocument()
  })

  it('filters meals by search text across raw text and item names', async () => {
    meals.mockResolvedValue([CHICKEN_TODAY, RICE_TODAY, OATS_YESTERDAY])
    renderHistory()

    await screen.findByText('grilled chicken')
    fireEvent.change(screen.getByLabelText('Search meals'), { target: { value: 'chicken' } })

    expect(screen.getByText('grilled chicken')).toBeInTheDocument()
    expect(screen.queryByText('rice bowl')).not.toBeInTheDocument()
    expect(screen.queryByText('oats and banana')).not.toBeInTheDocument()
  })

  it('filters meals by parser tier', async () => {
    meals.mockResolvedValue([CHICKEN_TODAY, RICE_TODAY, OATS_YESTERDAY])
    renderHistory()

    await screen.findByText('grilled chicken')
    fireEvent.click(screen.getByRole('button', { name: 'AI' }))

    expect(screen.getByText('rice bowl')).toBeInTheDocument()
    expect(screen.queryByText('grilled chicken')).not.toBeInTheDocument()
    expect(screen.queryByText('oats and banana')).not.toBeInTheDocument()
  })

  it('shows a no-matches empty state when the filter excludes every meal', async () => {
    meals.mockResolvedValue([CHICKEN_TODAY])
    renderHistory()

    await screen.findByText('grilled chicken')
    fireEvent.change(screen.getByLabelText('Search meals'), { target: { value: 'nothing matches this' } })

    expect(await screen.findByText('No matches')).toBeInTheDocument()
    expect(screen.queryByText('grilled chicken')).not.toBeInTheDocument()
  })
})
