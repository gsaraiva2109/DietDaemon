import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DemoProvider } from '@/lib/demo'
import { ComposeTemplateModal } from './ComposeTemplateModal'
import type { FoodDetail } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      foods: {
        ...actual.api.foods,
        search: vi.fn(),
      },
      templates: {
        ...actual.api.templates,
        compose: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const search = vi.mocked(api.foods.search)
const compose = vi.mocked(api.templates.compose)

const APPLE: FoodDetail = {
  food_id: 'apple-1',
  name: 'Apple',
  source: 'usda',
  per_100g: { Calories: 52, Protein: 0.3, Carbs: 14, Fat: 0.2, Fiber: 2.4 },
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

const BANANA: FoodDetail = { ...APPLE, food_id: 'banana-1', name: 'Banana' }

async function addFood(food: FoodDetail) {
  await typeSearch(food.name.toLowerCase())
  fireEvent.click(await screen.findByText(food.name))
}

beforeEach(() => {
  search.mockReset()
  compose.mockReset()
})

function renderModal() {
  const queryClient = new QueryClient()
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <ComposeTemplateModal onClose={onClose} />
      </DemoProvider>
    </QueryClientProvider>,
  )
  return { onClose }
}

function getBackdropButton() {
  return screen.getByRole('button', { name: 'Dismiss' })
}

async function typeSearch(value: string) {
  fireEvent.change(screen.getByLabelText('Search foods to add'), { target: { value } })
  await waitFor(() => expect(screen.queryByText('Searching…')).not.toBeInTheDocument(), { timeout: 2000 })
}

describe('ComposeTemplateModal', () => {
  it('renders the dialog', () => {
    renderModal()
    expect(screen.getByRole('dialog', { name: 'Compose a template from your food library' })).toBeInTheDocument()
  })

  it('calls onClose when Cancel is clicked', () => {
    const { onClose } = renderModal()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose when the close icon button is clicked', () => {
    const { onClose } = renderModal()
    const closeButtons = screen.getAllByRole('button', { name: 'Close' })
    const iconClose = closeButtons.find((b) => !b.className.includes('inset-0'))!
    fireEvent.click(iconClose)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('backdrop is a native, focusable button that closes on click (a11y regression test)', () => {
    const { onClose } = renderModal()
    const overlay = getBackdropButton()
    expect(overlay.tagName).toBe('BUTTON')
    expect(overlay).toHaveAttribute('type', 'button')
    overlay.focus()
    expect(overlay).toHaveFocus()
    fireEvent.click(overlay)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('shows "no matches" when the search resolves empty (nested-ternary branch)', async () => {
    search.mockResolvedValue([])
    renderModal()
    await typeSearch('kumquat')
    expect(await screen.findByText('No matching foods.')).toBeInTheDocument()
  })

  it('lists results and adds a food to the picked list on click (nested-ternary branch)', async () => {
    search.mockResolvedValue([APPLE])
    renderModal()
    await typeSearch('appl')
    fireEvent.click(await screen.findByText('Apple'))

    // Picked list now shows the food; the search box is cleared.
    expect(screen.getByText('1 item')).toBeInTheDocument()
    expect(screen.getByDisplayValue('100')).toBeInTheDocument() // default grams
  })

  it('submits via compose.mutate with name and picked items, then calls onClose on success', async () => {
    search.mockResolvedValue([APPLE])
    compose.mockResolvedValue({
      id: 'tmpl-1',
      name: 'My Template',
      items: [],
      created_at: '',
    } as never)
    const { onClose } = renderModal()

    fireEvent.change(screen.getByLabelText('Template name'), { target: { value: 'My Template' } })
    await typeSearch('appl')
    fireEvent.click(await screen.findByText('Apple'))

    const saveButton = screen.getByRole('button', { name: 'Save template' })
    expect(saveButton).toBeEnabled()
    fireEvent.click(saveButton)

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(compose).toHaveBeenCalledWith('My Template', [{ food_id: 'apple-1', grams: 100 }])
  })

  it('calls onClose on Escape keydown', () => {
    const { onClose } = renderModal()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('ignores non-Escape keydowns', () => {
    const { onClose } = renderModal()
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(onClose).not.toHaveBeenCalled()
  })

  it('updates grams for the matching picked item only, and falls back to 0 for a non-numeric value', async () => {
    search.mockResolvedValue([APPLE, BANANA])
    renderModal()
    await addFood(APPLE)
    await addFood(BANANA)

    const appleGrams = screen.getByLabelText('Grams of Apple')
    const bananaGrams = screen.getByLabelText('Grams of Banana')

    fireEvent.change(appleGrams, { target: { value: '150' } })
    expect(appleGrams).toHaveValue(150)
    // The other picked item is untouched (setGrams only updates the matching id).
    expect(bananaGrams).toHaveValue(100)

    // Number('') is NaN -> falls back to 0 via `|| 0`.
    fireEvent.change(bananaGrams, { target: { value: '' } })
    expect(bananaGrams).toHaveValue(0)
  })

  it('removes only the targeted picked item', async () => {
    search.mockResolvedValue([APPLE, BANANA])
    renderModal()
    await addFood(APPLE)
    await addFood(BANANA)
    expect(screen.getByText('2 items')).toBeInTheDocument()

    fireEvent.click(screen.getByLabelText('Remove Apple'))

    expect(screen.getByText('1 item')).toBeInTheDocument()
    expect(screen.queryByText('Apple')).not.toBeInTheDocument()
    expect(screen.getByText('Banana')).toBeInTheDocument()
  })

  it('shows the pending "Saving…" state while the mutation is in flight', async () => {
    search.mockResolvedValue([APPLE])
    let resolveCompose!: (v: unknown) => void
    compose.mockReturnValue(
      new Promise((resolve) => {
        resolveCompose = resolve
      }) as never,
    )
    renderModal()

    fireEvent.change(screen.getByLabelText('Template name'), { target: { value: 'My Template' } })
    await addFood(APPLE)
    fireEvent.click(screen.getByRole('button', { name: 'Save template' }))

    expect(await screen.findByRole('button', { name: 'Saving…' })).toBeDisabled()

    resolveCompose({ id: 't1', name: 'My Template', items: [], created_at: '' })
    await waitFor(() => expect(screen.queryByText('Saving…')).not.toBeInTheDocument())
  })

  it('renders the error message when the compose mutation rejects', async () => {
    search.mockResolvedValue([APPLE])
    compose.mockRejectedValue(new Error('A template with that name already exists.'))
    renderModal()

    fireEvent.change(screen.getByLabelText('Template name'), { target: { value: 'My Template' } })
    await addFood(APPLE)
    fireEvent.click(screen.getByRole('button', { name: 'Save template' }))

    expect(await screen.findByText('A template with that name already exists.')).toBeInTheDocument()
  })
})
