import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { CustomFoodModal } from './CustomFoodModal'
import type { FoodDetail, NutritionLabelDraft } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      foods: {
        ...actual.api.foods,
        ocrScan: vi.fn(),
        createCustom: vi.fn(),
        updateCustom: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const ocrScan = vi.mocked(api.foods.ocrScan)
const createCustom = vi.mocked(api.foods.createCustom)
const updateCustom = vi.mocked(api.foods.updateCustom)

const DRAFT: NutritionLabelDraft = {
  name: 'Test Cereal',
  basis_grams: 40,
  calories: 150,
  protein_g: 3,
  carbs_g: 30,
  fat_g: 2,
  fiber_g: 4,
  low_confidence_fields: ['fiber_g'],
  unreadable: false,
}

const SAVED_FOOD: FoodDetail = {
  food_id: 'f1',
  name: 'Test Cereal',
  source: 'custom',
  per_100g: { Calories: 375, Protein: 7.5, Carbs: 75, Fat: 5, Fiber: 10 },
  category: '',
  brand: '',
  barcode: '',
  image_url: '',
  serving_size: 40,
  serving_unit: 'g',
  query_count: 0,
  last_used: '',
  in_library: true,
  volume_units_eligible: false,
}

// Serving size deliberately not 100g, so prefilled values must be visibly
// scaled from per_100g rather than a passthrough copy (valuesFor's `scale`).
const EDIT_FOOD: FoodDetail = {
  food_id: 'edit-1',
  name: 'Edit Cereal',
  source: 'custom',
  per_100g: { Calories: 200, Protein: 10, Carbs: 20, Fat: 5, Fiber: 2 },
  category: '',
  brand: '',
  barcode: '',
  image_url: '',
  serving_size: 50,
  serving_unit: 'g',
  query_count: 0,
  last_used: '',
  in_library: true,
  volume_units_eligible: false,
}

const UPDATED_FOOD: FoodDetail = { ...EDIT_FOOD, name: 'Edit Cereal (updated)' }

function renderModal(food?: FoodDetail) {
  const queryClient = new QueryClient()
  const onSaved = vi.fn()
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={queryClient}>
      <CustomFoodModal food={food} onClose={onClose} onSaved={onSaved} />
    </QueryClientProvider>,
  )
  return { onSaved, onClose }
}

async function scanLabel() {
  const input = screen.getByLabelText('Scan label', { selector: 'input' })
  const file = new File(['fake'], 'label.jpg', { type: 'image/jpeg' })
  fireEvent.change(input, { target: { files: [file] } })
}

beforeEach(() => {
  ocrScan.mockReset()
  createCustom.mockReset()
  updateCustom.mockReset()
})

describe('CustomFoodModal OCR label scan', () => {
  it('prefills the form from a successful scan and flags low-confidence fields, without auto-saving', async () => {
    ocrScan.mockResolvedValue(DRAFT)
    const { onSaved } = renderModal()

    await scanLabel()

    await waitFor(() => expect(screen.getByDisplayValue('Test Cereal')).toBeInTheDocument())
    expect(screen.getByDisplayValue('40')).toBeInTheDocument()
    expect(screen.getByDisplayValue('150')).toBeInTheDocument()
    expect(screen.getByDisplayValue('4')).toBeInTheDocument()

    // Fiber was flagged low-confidence by the draft.
    expect(screen.getByText('Low confidence — double-check this value.')).toBeInTheDocument()

    // Scanning alone must never submit the form.
    expect(createCustom).not.toHaveBeenCalled()
    expect(onSaved).not.toHaveBeenCalled()

    // Save still requires its own explicit click, gated by normal validation.
    createCustom.mockResolvedValue(SAVED_FOOD)
    const saveButton = screen.getByRole('button', { name: 'Save food' })
    expect(saveButton).toBeEnabled()
    fireEvent.click(saveButton)

    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1))
    expect(onSaved.mock.calls[0][0]).toEqual(SAVED_FOOD)
  })

  it('shows the unreadable message and leaves the form untouched', async () => {
    ocrScan.mockResolvedValue({
      name: null,
      basis_grams: null,
      calories: null,
      protein_g: null,
      carbs_g: null,
      fat_g: null,
      fiber_g: null,
      low_confidence_fields: [],
      unreadable: true,
    })
    renderModal()

    await scanLabel()

    await waitFor(() =>
      expect(
        screen.getByText("Couldn't read a label in that photo. Try a clearer photo or enter the nutrients manually."),
      ).toBeInTheDocument(),
    )

    // Save is still gated by the (untouched, empty) required fields.
    expect(screen.getByRole('button', { name: 'Save food' })).toBeDisabled()
    expect(createCustom).not.toHaveBeenCalled()
  })
})

describe('CustomFoodModal edit mode', () => {
  it('renders the edit title/button, hides the OCR uploader, and prefills fields scaled from per_100g', () => {
    renderModal(EDIT_FOOD)

    expect(screen.getByText('Edit custom food')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save food' })).not.toBeInTheDocument()

    // Edit mode never shows the scan-a-label uploader (`{!food && (...)}`).
    expect(screen.queryByLabelText('Scan label')).not.toBeInTheDocument()

    expect(screen.getByDisplayValue('Edit Cereal')).toBeInTheDocument()
    expect(screen.getByDisplayValue('50')).toBeInTheDocument() // basis_grams, from serving_size
    // per_100g scaled by 50/100 = 0.5.
    expect(screen.getByDisplayValue('100')).toBeInTheDocument() // calories
    expect(screen.getByDisplayValue('5')).toBeInTheDocument() // protein
    expect(screen.getByDisplayValue('10')).toBeInTheDocument() // carbs
    expect(screen.getByDisplayValue('2.5')).toBeInTheDocument() // fat
    expect(screen.getByDisplayValue('1')).toBeInTheDocument() // fiber
  })

  it('submits via update.mutate (not create.mutate) and calls onSaved on success', async () => {
    updateCustom.mockResolvedValue(UPDATED_FOOD)
    const { onSaved } = renderModal(EDIT_FOOD)

    const saveButton = screen.getByRole('button', { name: 'Save changes' })
    expect(saveButton).toBeEnabled()
    fireEvent.click(saveButton)

    await waitFor(() => expect(onSaved).toHaveBeenCalledTimes(1))
    expect(onSaved.mock.calls[0][0]).toEqual(UPDATED_FOOD)
    expect(updateCustom).toHaveBeenCalledWith(
      EDIT_FOOD.food_id,
      expect.objectContaining({ name: 'Edit Cereal', basis_grams: 50, calories: 100, protein: 5, carbs: 10, fat: 2.5, fiber: 1 }),
    )
    expect(createCustom).not.toHaveBeenCalled()
  })
})

describe('CustomFoodModal manual-entry validation', () => {
  it('keeps Save disabled until name, basis_grams, and every macro are valid, live as fields change', () => {
    renderModal()

    const saveButton = screen.getByRole('button', { name: 'Save food' })
    const name = screen.getByLabelText('Food name')
    const basis = screen.getByLabelText('Label basis (g)')
    const calories = screen.getByLabelText('Calories (kcal)')
    const protein = screen.getByLabelText('Protein (g)')
    const carbs = screen.getByLabelText('Carbs (g)')
    const fat = screen.getByLabelText('Fat (g)')
    const fiber = screen.getByLabelText('Fiber (g)')

    // Nothing filled in yet.
    expect(saveButton).toBeDisabled()

    fireEvent.change(basis, { target: { value: '100' } })
    fireEvent.change(calories, { target: { value: '200' } })
    fireEvent.change(protein, { target: { value: '10' } })
    fireEvent.change(carbs, { target: { value: '20' } })
    fireEvent.change(fat, { target: { value: '5' } })
    fireEvent.change(fiber, { target: { value: '2' } })

    // All macros valid, but name is still empty.
    expect(saveButton).toBeDisabled()

    fireEvent.change(name, { target: { value: 'Granola' } })
    expect(saveButton).toBeEnabled()

    // Clearing a field back out (-> NaN once parsed) disables Save again.
    fireEvent.change(fiber, { target: { value: '' } })
    expect(saveButton).toBeDisabled()
    fireEvent.change(fiber, { target: { value: '2' } })
    expect(saveButton).toBeEnabled()

    // basis_grams must be strictly positive.
    fireEvent.change(basis, { target: { value: '0' } })
    expect(saveButton).toBeDisabled()
    fireEvent.change(basis, { target: { value: '100' } })
    expect(saveButton).toBeEnabled()

    // Negative macro values are rejected too.
    fireEvent.change(protein, { target: { value: '-1' } })
    expect(saveButton).toBeDisabled()
    fireEvent.change(protein, { target: { value: '10' } })
    expect(saveButton).toBeEnabled()
  })
})

describe('CustomFoodModal mutation error', () => {
  it('renders the error from create.error when the save mutation rejects', async () => {
    createCustom.mockRejectedValue(new Error('A food with that name already exists.'))
    renderModal()

    fireEvent.change(screen.getByLabelText('Food name'), { target: { value: 'Granola' } })
    fireEvent.change(screen.getByLabelText('Label basis (g)'), { target: { value: '100' } })
    fireEvent.change(screen.getByLabelText('Calories (kcal)'), { target: { value: '200' } })
    fireEvent.change(screen.getByLabelText('Protein (g)'), { target: { value: '10' } })
    fireEvent.change(screen.getByLabelText('Carbs (g)'), { target: { value: '20' } })
    fireEvent.change(screen.getByLabelText('Fat (g)'), { target: { value: '5' } })
    fireEvent.change(screen.getByLabelText('Fiber (g)'), { target: { value: '2' } })

    fireEvent.click(screen.getByRole('button', { name: 'Save food' }))

    expect(await screen.findByText('A food with that name already exists.')).toBeInTheDocument()
  })
})

describe('CustomFoodModal cancel', () => {
  it('calls onClose when Cancel is clicked', () => {
    const { onClose } = renderModal()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
