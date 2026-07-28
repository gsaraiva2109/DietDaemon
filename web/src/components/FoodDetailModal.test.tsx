import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { FoodDetailModal } from './FoodDetailModal'
import { DemoProvider } from '@/lib/demo'
import type { FoodDetail } from '@/lib/types'

const navigateMock = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => navigateMock }
})

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      foods: {
        ...actual.api.foods,
        get: vi.fn(),
        addAlias: vi.fn(),
        deleteAlias: vi.fn(),
        removeFromLibrary: vi.fn(),
        addToLibrary: vi.fn(),
        deleteCustom: vi.fn(),
        addServingUnit: vi.fn(),
        deleteServingUnit: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const getFood = vi.mocked(api.foods.get)
const addAlias = vi.mocked(api.foods.addAlias)
const deleteAlias = vi.mocked(api.foods.deleteAlias)
const removeFromLibrary = vi.mocked(api.foods.removeFromLibrary)
const addToLibrary = vi.mocked(api.foods.addToLibrary)
const deleteCustom = vi.mocked(api.foods.deleteCustom)
const addServingUnit = vi.mocked(api.foods.addServingUnit)
const deleteServingUnit = vi.mocked(api.foods.deleteServingUnit)

const BASE_FOOD: FoodDetail = {
  food_id: 'f1',
  name: 'Greek Yogurt',
  source: 'food_library',
  per_100g: { Calories: 100, Protein: 10, Carbs: 5, Fat: 2, Fiber: 0 },
  category: 'Dairy',
  brand: 'Brandy',
  barcode: '12345',
  image_url: '',
  serving_size: 170,
  serving_unit: 'g',
  query_count: 3,
  last_used: '',
  aliases: [{ food_id: 'f1', alias: 'yogurt', normalized: 'yogurt' }],
  in_library: true,
  serving_units: [{ id: 'u1', label: '1 cup', grams: 245, custom: true }],
  volume_units_eligible: false,
}

const CUSTOM_FOOD: FoodDetail = {
  ...BASE_FOOD,
  food_id: 'f2',
  name: 'My Custom Bar',
  source: 'custom',
  serving_units: [],
  aliases: [],
}

const NOT_IN_LIBRARY_FOOD: FoodDetail = {
  ...BASE_FOOD,
  food_id: 'f3',
  name: 'Random OFF Food',
  source: 'openfoodfacts',
  in_library: false,
  serving_units: [],
  aliases: [],
}

function renderModal(foodID = 'f1', onEditCustom?: (f: FoodDetail) => void) {
  const queryClient = new QueryClient()
  const onClose = vi.fn()
  render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <DemoProvider>
          <FoodDetailModal foodID={foodID} onClose={onClose} onEditCustom={onEditCustom} />
        </DemoProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  )
  return { onClose }
}

beforeEach(() => {
  getFood.mockReset()
  addAlias.mockReset()
  deleteAlias.mockReset()
  removeFromLibrary.mockReset()
  addToLibrary.mockReset()
  deleteCustom.mockReset()
  addServingUnit.mockReset()
  deleteServingUnit.mockReset()
  navigateMock.mockReset()
  localStorage.clear()
})

describe('FoodDetailModal', () => {
  it('shows a loading state, then the food detail once fetched', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    renderModal()

    expect(screen.getByText(/Loading food/)).toBeInTheDocument()
    expect(await screen.findByText('Greek Yogurt')).toBeInTheDocument()
    expect(screen.getByText('Library')).toBeInTheDocument()
    expect(screen.getByText('Dairy')).toBeInTheDocument()
  })

  it('scales displayed macros when switching the serving-unit basis', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    renderModal()

    await screen.findByText('Greek Yogurt')
    // per_100g basis: 100 kcal shown.
    expect(screen.getByText('100')).toBeInTheDocument()

    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'u1' } })
    // 245g of 100kcal/100g = 245 kcal.
    expect(await screen.findByText('245')).toBeInTheDocument()
  })

  it('logs the food: navigates to /log with the name and closes', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    const { onClose } = renderModal()

    fireEvent.click(await screen.findByRole('button', { name: 'Log this' }))

    expect(navigateMock).toHaveBeenCalledWith('/log?text=Greek%20Yogurt')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('adds an alias', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    addAlias.mockResolvedValue(undefined as never)
    renderModal()

    const input = await screen.findByLabelText('Add alias for Greek Yogurt')
    fireEvent.change(input, { target: { value: 'gy' } })
    const addButtons = screen.getAllByRole('button', { name: 'Add' })
    fireEvent.click(addButtons[addButtons.length - 1])

    await waitFor(() => expect(addAlias).toHaveBeenCalledWith('f1', 'gy'))
  })

  it('removes an existing alias', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    deleteAlias.mockResolvedValue(undefined as never)
    renderModal()

    await screen.findByText('Greek Yogurt')
    fireEvent.click(screen.getByRole('button', { name: 'Remove alias yogurt' }))

    await waitFor(() => expect(deleteAlias).toHaveBeenCalledWith('f1', 'yogurt'))
  })

  it('adds a serving unit', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    addServingUnit.mockResolvedValue(undefined as never)
    renderModal()

    await screen.findByText('Greek Yogurt')
    fireEvent.change(screen.getByPlaceholderText('Label (e.g. "1 slice")'), { target: { value: '1 tub' } })
    fireEvent.change(screen.getByPlaceholderText('Grams'), { target: { value: '500' } })
    fireEvent.click(screen.getAllByRole('button', { name: 'Add' })[0])

    await waitFor(() => expect(addServingUnit).toHaveBeenCalledWith('f1', '1 tub', 500))
  })

  it('removes a serving unit', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    deleteServingUnit.mockResolvedValue(undefined as never)
    renderModal()

    await screen.findByText('Greek Yogurt')
    fireEvent.click(screen.getByRole('button', { name: 'Remove 1 cup' }))

    await waitFor(() => expect(deleteServingUnit).toHaveBeenCalledWith('f1', 'u1'))
  })

  it('removes from library after confirming', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    removeFromLibrary.mockResolvedValue(undefined as never)
    const { onClose } = renderModal()

    fireEvent.click(await screen.findByRole('button', { name: 'Remove from my library' }))
    expect(screen.getByText('Remove this food from your library?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Yes, remove' }))

    await waitFor(() => expect(removeFromLibrary).toHaveBeenCalledWith('f1'))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('cancels the remove-from-library confirmation', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    renderModal()

    fireEvent.click(await screen.findByRole('button', { name: 'Remove from my library' }))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByText('Remove this food from your library?')).not.toBeInTheDocument()
    expect(removeFromLibrary).not.toHaveBeenCalled()
  })

  it('adds a food not yet in the library', async () => {
    getFood.mockResolvedValue(NOT_IN_LIBRARY_FOOD)
    addToLibrary.mockResolvedValue(undefined as never)
    renderModal('f3')

    expect(await screen.findByText('Not in your library yet — add it to search/log it quickly later, even as part of a combo you haven\'t logged as its own meal.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Add to my library' }))

    await waitFor(() => expect(addToLibrary).toHaveBeenCalledWith('f3'))
  })

  it('for custom foods: deletes after confirming, and supports edit', async () => {
    getFood.mockResolvedValue(CUSTOM_FOOD)
    deleteCustom.mockResolvedValue(undefined as never)
    const onEditCustom = vi.fn()
    const { onClose } = renderModal('f2', onEditCustom)

    fireEvent.click(await screen.findByRole('button', { name: 'Edit' }))
    expect(onEditCustom).toHaveBeenCalledWith(CUSTOM_FOOD)

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(screen.getByText('Delete this custom food?')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Yes, delete' }))

    await waitFor(() => expect(deleteCustom).toHaveBeenCalledWith('f2'))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('closes when the close (X) button is clicked', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    const { onClose } = renderModal()

    await screen.findByText('Greek Yogurt')
    const closeButton = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeButton)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    const { onClose } = renderModal()

    await screen.findByText('Greek Yogurt')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Enter is pressed on the backdrop (keyboard a11y)', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    const { onClose } = renderModal()

    await screen.findByText('Greek Yogurt')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Enter' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when Space is pressed on the backdrop (keyboard a11y)', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    const { onClose } = renderModal()

    await screen.findByText('Greek Yogurt')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: ' ' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('ignores unrelated keys on the backdrop', async () => {
    getFood.mockResolvedValue(BASE_FOOD)
    const { onClose } = renderModal()

    await screen.findByText('Greek Yogurt')
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    fireEvent.keyDown(backdrop, { key: 'Tab' })
    expect(onClose).not.toHaveBeenCalled()
  })
})
