import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DuplicateMealModal } from './DuplicateMealModal'
import { DemoProvider } from '@/lib/demo'
import type { Meal } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      meals: vi.fn(),
      duplicateMeal: vi.fn(),
    },
  }
})

import { api } from '@/lib/api'

const meals = vi.mocked(api.meals)
const duplicateMeal = vi.mocked(api.duplicateMeal)

function meal(id: string, at: string, rawText: string, kcal: number): Meal {
  return {
    ID: id,
    UserID: 'u1',
    At: at,
    RawText: rawText,
    Items: [
      {
        Parsed: { RawPhrase: rawText, Quantity: 1, Unit: 'serving', NormalizedGrams: 100, Locale: 'en' },
        Match: { FoodID: 'f1', Name: rawText, Source: 'food_library', Per100g: { Calories: kcal, Protein: 10, Carbs: 10, Fat: 5, Fiber: 2 }, MatchScore: 1 },
        Macros: { Calories: kcal, Protein: 10, Carbs: 10, Fat: 5, Fiber: 2 },
      },
    ],
    Confidence: 1,
    ParserTier: 0,
    CreatedAt: at,
    PlanSlotID: '',
    PlanOptionID: '',
  }
}

const TODAY_MEAL = meal('m1', new Date().toISOString(), 'Chicken salad', 400)

const yesterday = new Date()
yesterday.setDate(yesterday.getDate() - 1)
const YESTERDAY_MEAL = meal('m2', yesterday.toISOString(), 'Oatmeal', 250)

const lastWeek = new Date()
lastWeek.setDate(lastWeek.getDate() - 8)
const OLD_MEAL = meal('m3', lastWeek.toISOString(), 'Steak dinner', 700)

function renderModal() {
  const queryClient = new QueryClient()
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <DuplicateMealModal onClose={onClose} />
      </DemoProvider>
    </QueryClientProvider>,
  )
  return { onClose }
}

beforeEach(() => {
  meals.mockReset()
  duplicateMeal.mockReset()
})

describe('DuplicateMealModal', () => {
  it('renders the day-picker step once meals load', async () => {
    meals.mockResolvedValue([TODAY_MEAL])
    renderModal()

    expect(screen.getByText('Pick a day')).toBeInTheDocument()
    expect(await screen.findByText('Today')).toBeInTheDocument()
    expect(screen.getByText('1 meal')).toBeInTheDocument()
  })

  it('labels a day-group as Yesterday or by weekday/month/day, per dayLabel branch', async () => {
    meals.mockResolvedValue([TODAY_MEAL, YESTERDAY_MEAL, OLD_MEAL])
    renderModal()

    expect(await screen.findByText('Today')).toBeInTheDocument()
    expect(screen.getByText('Yesterday')).toBeInTheDocument()
    expect(
      screen.getByText(lastWeek.toLocaleDateString('en', { weekday: 'long', month: 'long', day: 'numeric' })),
    ).toBeInTheDocument()
  })

  it('shows the empty state when there are no meals to duplicate', async () => {
    meals.mockResolvedValue([])
    renderModal()

    expect(await screen.findByText('No meals to duplicate')).toBeInTheDocument()
  })

  it('drills into a day to show its meals, then back to days', async () => {
    meals.mockResolvedValue([TODAY_MEAL])
    renderModal()

    fireEvent.click(await screen.findByText('Today'))
    expect(await screen.findByText('Pick a meal')).toBeInTheDocument()
    expect(screen.getByText('Chicken salad')).toBeInTheDocument()
    expect(screen.getByText('400')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Back to days'))
    expect(await screen.findByText('Pick a day')).toBeInTheDocument()
  })

  it('duplicates the selected meal and closes on success', async () => {
    meals.mockResolvedValue([TODAY_MEAL])
    duplicateMeal.mockResolvedValue(undefined as never)
    const { onClose } = renderModal()

    fireEvent.click(await screen.findByText('Today'))
    fireEvent.click(await screen.findByText('Chicken salad'))

    await waitFor(() => expect(duplicateMeal).toHaveBeenCalledWith('m1'))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('shows an error message when duplicating fails', async () => {
    meals.mockResolvedValue([TODAY_MEAL])
    duplicateMeal.mockRejectedValue(new Error('boom'))
    renderModal()

    fireEvent.click(await screen.findByText('Today'))
    fireEvent.click(await screen.findByText('Chicken salad'))

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })

  it('closes when the close (X) button is clicked', async () => {
    meals.mockResolvedValue([])
    const { onClose } = renderModal()

    await screen.findByText('No meals to duplicate')
    const closeButton = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeButton)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked', async () => {
    meals.mockResolvedValue([])
    const { onClose } = renderModal()

    await screen.findByText('No meals to duplicate')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Enter is pressed on the backdrop (keyboard a11y)', async () => {
    meals.mockResolvedValue([])
    const { onClose } = renderModal()

    await screen.findByText('No meals to duplicate')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Enter' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Space is pressed on the backdrop (keyboard a11y)', async () => {
    meals.mockResolvedValue([])
    const { onClose } = renderModal()

    await screen.findByText('No meals to duplicate')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: ' ' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('ignores unrelated keys on the backdrop', async () => {
    meals.mockResolvedValue([])
    const { onClose } = renderModal()

    await screen.findByText('No meals to duplicate')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Tab' })
    expect(onClose).not.toHaveBeenCalled()
  })
})
