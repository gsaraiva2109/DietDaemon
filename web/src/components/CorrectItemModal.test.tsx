import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DemoProvider } from '@/lib/demo'
import { CorrectItemModal } from './CorrectItemModal'
import type { Meal, FoodDetail } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      correctItem: vi.fn(),
      addItem: vi.fn(),
      foods: { ...actual.api.foods, searchCatalog: vi.fn() },
    },
  }
})

import { api } from '@/lib/api'

const correctItem = vi.mocked(api.correctItem)
const addItem = vi.mocked(api.addItem)
const searchCatalog = vi.mocked(api.foods.searchCatalog)

const MEAL: Meal = {
  ID: 'm1',
  UserID: 'u1',
  At: '2026-07-28T08:00:00Z',
  RawText: 'yogurt',
  Confidence: 0.9,
  ParserTier: 0,
  CreatedAt: '2026-07-28T08:00:00Z',
  PlanSlotID: '',
  PlanOptionID: '',
  Items: [
    {
      Parsed: { RawPhrase: 'yogurt', Quantity: 150, Unit: 'g', NormalizedGrams: 150, Locale: 'en' },
      Match: { FoodID: 'f1', Name: 'Greek yogurt', Source: 'food_library', Per100g: { Calories: 59, Protein: 10, Carbs: 3.6, Fat: 0.4, Fiber: 0 }, MatchScore: 0.9 },
      Macros: { Calories: 88.5, Protein: 15, Carbs: 5.4, Fat: 0.6, Fiber: 0 },
    },
  ],
}

const CATALOG_FOOD: FoodDetail = {
  food_id: 'f2',
  name: 'Skyr',
  source: 'food_library',
  per_100g: { Calories: 65, Protein: 11, Carbs: 4, Fat: 0.2, Fiber: 0 },
  category: '',
  brand: '',
  barcode: '',
  image_url: '',
  serving_size: 100,
  serving_unit: 'g',
  query_count: 0,
  last_used: '',
  in_library: true,
  volume_units_eligible: false,
}

function renderModal(index: number | undefined) {
  localStorage.setItem('dd.demo', '0')
  const queryClient = new QueryClient()
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <CorrectItemModal meal={MEAL} index={index} onClose={onClose} />
      </DemoProvider>
    </QueryClientProvider>,
  )
  return { onClose }
}

beforeEach(() => {
  correctItem.mockReset()
  addItem.mockReset()
  searchCatalog.mockReset()
  searchCatalog.mockResolvedValue([])
})

describe('CorrectItemModal correct mode', () => {
  it('renders prefilled from the item at index and closes via the X button', () => {
    const { onClose } = renderModal(0)
    expect(screen.getByDisplayValue('Greek yogurt')).toBeInTheDocument()
    expect(screen.getByDisplayValue('150')).toBeInTheDocument()

    const closeX = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeX)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked, and the backdrop is a real (keyboard-operable) button', () => {
    const { onClose } = renderModal(0)
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    expect(backdrop.tagName).toBe('BUTTON')
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('submits via correct.mutate (not add.mutate), with label "Save correction"', async () => {
    correctItem.mockResolvedValue({ ...MEAL })
    const { onClose } = renderModal(0)

    const saveButton = screen.getByRole('button', { name: 'Save correction' })
    fireEvent.click(saveButton)

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(correctItem).toHaveBeenCalledWith('m1', 0, expect.objectContaining({ Match: expect.objectContaining({ Name: 'Greek yogurt' }) }))
    expect(addItem).not.toHaveBeenCalled()
  })
})

describe('CorrectItemModal add mode', () => {
  it('renders blank fields and label "Add item", disabled until a name is entered', () => {
    renderModal(undefined)
    expect(screen.getByRole('button', { name: 'Add item' })).toBeDisabled()
    fireEvent.change(screen.getByLabelText('Food name'), { target: { value: 'Banana' } })
    expect(screen.getByRole('button', { name: 'Add item' })).toBeEnabled()
  })

  it('submits via add.mutate (not correct.mutate)', async () => {
    addItem.mockResolvedValue({ ...MEAL })
    const { onClose } = renderModal(undefined)

    fireEvent.change(screen.getByLabelText('Food name'), { target: { value: 'Banana' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add item' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(addItem).toHaveBeenCalledWith('m1', expect.objectContaining({ Match: expect.objectContaining({ Name: 'Banana' }) }))
    expect(correctItem).not.toHaveBeenCalled()
  })
})

describe('CorrectItemModal catalog search (nested-ternary branches)', () => {
  it('loading branch: shows a spinner while the query is in flight', async () => {
    searchCatalog.mockReturnValue(new Promise(() => {}))
    renderModal(0)

    fireEvent.click(screen.getByRole('button', { name: 'Search catalog instead' }))
    fireEvent.change(screen.getByPlaceholderText('Search the food catalog'), { target: { value: 'sky' } })

    expect(await screen.findByText('Loading', { exact: false })).toBeInTheDocument()
  })

  it('empty branch: shows the no-results message', async () => {
    renderModal(0)
    fireEvent.click(screen.getByRole('button', { name: 'Search catalog instead' }))
    fireEvent.change(screen.getByPlaceholderText('Search the food catalog'), { target: { value: 'zzz' } })

    expect(await screen.findByText('No matches in the catalog.')).toBeInTheDocument()
  })

  it('results branch: lists matches, and picking one previews scaled macros then confirms the replace', async () => {
    searchCatalog.mockResolvedValue([CATALOG_FOOD])
    renderModal(0)

    fireEvent.click(screen.getByRole('button', { name: 'Search catalog instead' }))
    fireEvent.change(screen.getByPlaceholderText('Search the food catalog'), { target: { value: 'sky' } })

    fireEvent.click(await screen.findByText('Skyr'))
    expect(await screen.findByText('Preview')).toBeInTheDocument()
    // 150g of Skyr (65 kcal/100g) -> scaled Calories = 97.5, rounds to 98.
    expect(screen.getByText('98')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Confirm replace' }))
    expect(screen.getByDisplayValue('Skyr')).toBeInTheDocument()
  })
})
