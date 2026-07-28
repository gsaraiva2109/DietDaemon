import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { SaveTemplateModal } from './SaveTemplateModal'
import type { ResolvedItem } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      templates: { ...actual.api.templates, create: vi.fn() },
    },
  }
})

import { api } from '@/lib/api'

const createTemplate = vi.mocked(api.templates.create)

const ITEM: ResolvedItem = {
  Parsed: { RawPhrase: 'rice', Quantity: 100, Unit: 'g', NormalizedGrams: 100, Locale: 'en' },
  Match: { FoodID: 'f1', Name: 'Rice', Source: 'food_library', Per100g: { Calories: 130, Protein: 3, Carbs: 28, Fat: 0.3, Fiber: 0.4 }, MatchScore: 1 },
  Macros: { Calories: 130, Protein: 3, Carbs: 28, Fat: 0.3, Fiber: 0.4 },
}

function renderModal(items: ResolvedItem[] = [ITEM]) {
  const queryClient = new QueryClient()
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={queryClient}>
      <SaveTemplateModal items={items} onClose={onClose} />
    </QueryClientProvider>,
  )
  return { onClose }
}

beforeEach(() => {
  createTemplate.mockReset()
})

describe('SaveTemplateModal', () => {
  it('renders items and the running total', () => {
    renderModal()
    expect(screen.getByText('Rice')).toBeInTheDocument()
    expect(screen.getByText('Total')).toBeInTheDocument()
  })

  it('renders the empty state when there are no items', () => {
    renderModal([])
    expect(screen.getByText('No items to save.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save template' })).toBeDisabled()
  })

  it('closes via the X button', () => {
    const { onClose } = renderModal()
    const closeX = screen.getByRole('button', { name: 'Close' })
    fireEvent.click(closeX)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes when the backdrop is clicked, and the backdrop is a real (keyboard-operable) button', () => {
    const { onClose } = renderModal()
    const backdrop = screen.getByRole('button', { name: 'Dismiss' })
    expect(backdrop.tagName).toBe('BUTTON')
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('closes on Escape', () => {
    const { onClose } = renderModal()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('saves via create.mutate and closes on success', async () => {
    createTemplate.mockResolvedValue({ id: 't1', user_id: 'u1', name: 'My template', items: [ITEM], created_at: '', last_used: '' })
    const { onClose } = renderModal()

    fireEvent.change(screen.getByPlaceholderText('e.g. Post-workout breakfast'), { target: { value: 'My template' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save template' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(createTemplate).toHaveBeenCalledWith('My template', [ITEM])
  })

  it('submits on Enter in the name field', async () => {
    createTemplate.mockResolvedValue({ id: 't1', user_id: 'u1', name: 'Enter save', items: [ITEM], created_at: '', last_used: '' })
    renderModal()
    const input = screen.getByPlaceholderText('e.g. Post-workout breakfast')
    fireEvent.change(input, { target: { value: 'Enter save' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    await waitFor(() => expect(createTemplate).toHaveBeenCalled())
  })

  it('shows the mutation error message on failure', async () => {
    createTemplate.mockRejectedValue(new Error('Name already used'))
    renderModal()
    fireEvent.change(screen.getByPlaceholderText('e.g. Post-workout breakfast'), { target: { value: 'Dup' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save template' }))
    expect(await screen.findByText('Name already used')).toBeInTheDocument()
  })
})
