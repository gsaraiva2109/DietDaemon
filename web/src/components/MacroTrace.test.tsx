import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { MacroTrace } from './MacroTrace'
import type { ResolvedItem } from '@/lib/types'

function item(overrides: Partial<ResolvedItem['Match']> & { rawPhrase?: string }): ResolvedItem {
  const { rawPhrase, ...match } = overrides
  return {
    Parsed: {
      RawPhrase: rawPhrase ?? 'apple',
      Quantity: 1,
      Unit: 'unit',
      NormalizedGrams: 100,
      Locale: 'en',
    },
    Match: {
      FoodID: 'food-1',
      Name: 'Apple',
      Source: 'usda',
      Per100g: { Calories: 52, Protein: 0.3, Carbs: 14, Fat: 0.2, Fiber: 2.4 },
      MatchScore: 0.9,
      ...match,
    },
    Macros: { Calories: 52, Protein: 0.3, Carbs: 14, Fat: 0.2, Fiber: 2.4 },
  }
}

function renderModal(items: ResolvedItem[]) {
  const onClose = vi.fn()
  render(<MacroTrace items={items} onClose={onClose} />)
  return { onClose }
}

function getBackdropButton() {
  return screen.getByRole('button', { name: 'Dismiss' })
}

describe('MacroTrace', () => {
  it('renders the dialog and the empty state when there are no items', () => {
    renderModal([])
    expect(screen.getByRole('dialog', { name: 'Macro trace' })).toBeInTheDocument()
    expect(screen.getByText('No items to trace.')).toBeInTheDocument()
  })

  it('renders one row per item with name, source, confidence, and per-macro breakdown', () => {
    renderModal([
      item({ FoodID: 'f1', Name: 'Apple', rawPhrase: 'an apple' }),
      item({ FoodID: 'f2', Name: 'Banana', rawPhrase: 'a banana', MatchScore: 0.5 }),
    ])
    expect(screen.getByText('Apple')).toBeInTheDocument()
    expect(screen.getByText('Banana')).toBeInTheDocument()
    expect(screen.getByText('90%')).toBeInTheDocument()
    expect(screen.getByText('50%')).toBeInTheDocument()
    // Macro breakdown (rounded) renders for each item via MACRO_KEYS.
    expect(screen.getAllByText('52').length).toBeGreaterThan(0)
  })

  it('uses a stable identity key derived from item data, not array index (regression for S6479)', () => {
    // Two items sharing the same FoodID but distinct raw phrases must both
    // render distinctly, proving the key isn't collapsing on duplicate index-free data.
    renderModal([
      item({ FoodID: 'dup', Name: 'Apple', rawPhrase: 'one apple' }),
      item({ FoodID: 'dup', Name: 'Apple', rawPhrase: 'two apples' }),
    ])
    expect(screen.getAllByText('Apple')).toHaveLength(2)
  })

  it('calls onClose when the close icon button is clicked', () => {
    const { onClose } = renderModal([])
    const closeButtons = screen.getAllByRole('button', { name: 'Close' })
    const iconClose = closeButtons.find((b) => !b.className.includes('inset-0'))!
    fireEvent.click(iconClose)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('backdrop is a native, focusable button that closes on click (a11y regression test)', () => {
    const { onClose } = renderModal([])
    const overlay = getBackdropButton()
    expect(overlay.tagName).toBe('BUTTON')
    expect(overlay).toHaveAttribute('type', 'button')
    overlay.focus()
    expect(overlay).toHaveFocus()
    fireEvent.click(overlay)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('calls onClose on Escape keydown', () => {
    const { onClose } = renderModal([])
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
