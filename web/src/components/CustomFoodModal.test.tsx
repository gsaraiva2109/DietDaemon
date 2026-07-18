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
      },
    },
  }
})

import { api } from '@/lib/api'

const ocrScan = vi.mocked(api.foods.ocrScan)
const createCustom = vi.mocked(api.foods.createCustom)

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
}

function renderModal() {
  const queryClient = new QueryClient()
  const onSaved = vi.fn()
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={queryClient}>
      <CustomFoodModal onClose={onClose} onSaved={onSaved} />
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
