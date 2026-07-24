import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { LogMeal } from './LogMeal'
import { DemoProvider } from '@/lib/demo'
import type { FoodDetail, MealTemplate } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      logMeal: vi.fn(),
      logMealStructured: vi.fn(),
      foods: {
        ...actual.api.foods,
        list: vi.fn(),
        search: vi.fn(),
        searchCatalog: vi.fn(),
        addServingUnit: vi.fn(),
      },
      templates: {
        ...actual.api.templates,
        list: vi.fn(),
        log: vi.fn(),
      },
    },
  }
})

vi.mock('@/components/DuplicateMealModal', () => ({
  DuplicateMealModal: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="duplicate-meal-modal">
      <button onClick={onClose}>close-duplicate</button>
    </div>
  ),
}))

vi.mock('@/components/CustomFoodModal', () => ({
  CustomFoodModal: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="custom-food-modal">
      <button onClick={onClose}>close-custom</button>
    </div>
  ),
}))

import { api } from '@/lib/api'

const logMeal = vi.mocked(api.logMeal)
const logMealStructured = vi.mocked(api.logMealStructured)
const list = vi.mocked(api.foods.list)
const search = vi.mocked(api.foods.search)
const searchCatalog = vi.mocked(api.foods.searchCatalog)
const templatesList = vi.mocked(api.templates.list)
const templatesLog = vi.mocked(api.templates.log)

function food(overrides: Partial<FoodDetail>): FoodDetail {
  return {
    food_id: 'f1',
    name: 'Chicken breast',
    source: 'food_library',
    per_100g: { Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0 },
    category: '',
    brand: '',
    barcode: '',
    image_url: '',
    serving_size: 100,
    serving_unit: 'g',
    query_count: 3,
    last_used: '',
    in_library: true,
    volume_units_eligible: false,
    ...overrides,
  }
}

function template(overrides: Partial<MealTemplate>): MealTemplate {
  return {
    id: 't1',
    user_id: 'u1',
    name: 'Breakfast',
    items: [],
    created_at: '',
    last_used: '',
    ...overrides,
  }
}

function renderLogMeal() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <LogMeal />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  logMeal.mockReset().mockResolvedValue({ status: 'accepted' })
  logMealStructured.mockReset()
  list.mockReset().mockResolvedValue([])
  search.mockReset().mockResolvedValue([])
  searchCatalog.mockReset().mockResolvedValue([])
  templatesList.mockReset().mockResolvedValue([])
  templatesLog.mockReset()
})

describe('LogMeal text mode', () => {
  it('submits the typed text and shows the success message', async () => {
    renderLogMeal()

    fireEvent.change(screen.getByLabelText('Meal description'), {
      target: { value: '200g grilled chicken' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Log meal' }))

    await waitFor(() => expect(logMeal).toHaveBeenCalledWith('200g grilled chicken'))
    expect(await screen.findByText("Logged, processing now. It'll appear on Today in a moment.")).toBeInTheDocument()
  })

  it('disables the submit button until there is non-whitespace text', () => {
    renderLogMeal()

    const button = screen.getByRole('button', { name: 'Log meal' })
    expect(button).toBeDisabled()

    fireEvent.change(screen.getByLabelText('Meal description'), { target: { value: '   ' } })
    expect(button).toBeDisabled()

    fireEvent.change(screen.getByLabelText('Meal description'), { target: { value: 'toast' } })
    expect(button).toBeEnabled()
  })

  it('shows up to 6 recent templates and logs one on click, and hides the list message when there are none', async () => {
    templatesList.mockResolvedValue(
      Array.from({ length: 8 }, (_, i) => template({ id: `t${i}`, name: `Template ${i}` })),
    )
    renderLogMeal()

    expect(await screen.findByText('Template 0')).toBeInTheDocument()
    expect(screen.getByText('Template 5')).toBeInTheDocument()
    expect(screen.queryByText('Template 6')).not.toBeInTheDocument()
    expect(screen.queryByText('No templates yet. Save one from a meal\'s detail page.')).not.toBeInTheDocument()

    fireEvent.click(screen.getByText('Template 0'))
    await waitFor(() => expect(templatesLog).toHaveBeenCalledWith('t0'))
  })

  it('shows the no-templates message when there are none', async () => {
    renderLogMeal()
    expect(await screen.findByText("No templates yet. Save one from a meal's detail page.")).toBeInTheDocument()
  })

  it('shows examples only in text mode and fills the textarea on click', async () => {
    renderLogMeal()

    const example = screen.getByText('1 banana and a glass of milk')
    fireEvent.click(example)
    expect(screen.getByLabelText('Meal description')).toHaveValue('1 banana and a glass of milk')

    fireEvent.click(screen.getByRole('button', { name: 'Pick foods' }))
    expect(screen.queryByText('1 banana and a glass of milk')).not.toBeInTheDocument()
  })

  it('opens the duplicate-meal modal from the copy-from-day button', async () => {
    renderLogMeal()
    fireEvent.click(screen.getByRole('button', { name: /Copy from day/ }))
    expect(await screen.findByTestId('duplicate-meal-modal')).toBeInTheDocument()
  })
})

describe('LogMeal picker mode (FoodPicker)', () => {
  async function openPicker() {
    renderLogMeal()
    fireEvent.click(screen.getByRole('button', { name: 'Pick foods' }))
    await screen.findByLabelText('Search foods')
  }

  it('shows the library empty state, then the no-matches state while searching', async () => {
    await openPicker()

    expect(await screen.findByText('No foods yet')).toBeInTheDocument()

    search.mockResolvedValue([])
    fireEvent.change(screen.getByLabelText('Search foods'), { target: { value: 'xyz' } })
    await waitFor(() => expect(search).toHaveBeenCalledWith('xyz'))

    expect(await screen.findByText('No matches')).toBeInTheDocument()
  })

  it('shows the catalog empty state on the catalog tab', async () => {
    searchCatalog.mockResolvedValue([])
    await openPicker()
    fireEvent.click(screen.getByRole('button', { name: 'Catalog' }))

    expect(await screen.findByText('No catalog matches')).toBeInTheDocument()
  })

  it('shows the loading spinner while foods are loading', async () => {
    list.mockImplementation(() => new Promise(() => {}))
    await openPicker()

    const status = await screen.findByRole('status')
    expect(status).toHaveTextContent('Loading foods')
  })

  it('adds a food on click, defaulting grams to its serving size (non-OFF source)', async () => {
    list.mockResolvedValue([food({ food_id: 'f1', name: 'Chicken breast', source: 'taco', serving_size: 120 })])
    await openPicker()

    fireEvent.click(await screen.findByText('Chicken breast'))

    const quantityInput = await screen.findByLabelText('Quantity for Chicken breast')
    expect(quantityInput).toHaveValue(120)
  })

  it('defaults grams to 100 for an OpenFoodFacts food regardless of serving_size', async () => {
    list.mockResolvedValue([
      food({ food_id: 'f2', name: 'Packaged snack', source: 'openfoodfacts', serving_size: 45 }),
    ])
    await openPicker()

    fireEvent.click(await screen.findByText('Packaged snack'))

    const quantityInput = await screen.findByLabelText('Quantity for Packaged snack')
    expect(quantityInput).toHaveValue(100)
  })

  it('does not add the same food twice and removing it clears the selection', async () => {
    list.mockResolvedValue([food({ food_id: 'f1', name: 'Chicken breast' })])
    await openPicker()

    const card = await screen.findByText('Chicken breast')
    fireEvent.click(card)
    fireEvent.click(card)
    expect(screen.getAllByLabelText('Quantity for Chicken breast')).toHaveLength(1)

    fireEvent.click(screen.getByRole('button', { name: 'Remove Chicken breast' }))
    expect(screen.queryByLabelText('Quantity for Chicken breast')).not.toBeInTheDocument()
    expect(screen.getByText('No foods selected yet. Search and tap a food to add it.')).toBeInTheDocument()
  })

  it('shows the running total and submits grams-based items on log', async () => {
    list.mockResolvedValue([food({ food_id: 'f1', name: 'Chicken breast', serving_size: 100 })])
    logMealStructured.mockResolvedValue({} as never)
    await openPicker()

    fireEvent.click(await screen.findByText('Chicken breast'))
    expect(await screen.findByText(/165 kcal · 31P · 0C · 4F/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Log meal' }))

    await waitFor(() =>
      expect(logMealStructured).toHaveBeenCalledWith([
        { food_id: 'f1', grams: 100, unit: undefined, quantity: undefined },
      ]),
    )
    expect(await screen.findByText('Logged.')).toBeInTheDocument()
  })

  it('opens the custom-food modal from the picker', async () => {
    await openPicker()
    fireEvent.click(await screen.findByRole('button', { name: 'Add custom food' }))
    expect(await screen.findByTestId('custom-food-modal')).toBeInTheDocument()
  })
})
