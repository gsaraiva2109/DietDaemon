import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { MacroBar } from './MacroBar'

// The "left"/"over" caption sits in a `.mt-1` div wrapping a single span with
// identical textContent, so text queries match both nodes. Read the div's
// textContent directly instead of asserting via a *ByText query.
function captionText(container: HTMLElement): string | null {
  return container.querySelector('.mt-1.text-xs.text-muted')?.textContent ?? null
}

describe('MacroBar under target', () => {
  it('renders the label, consumed/target, and remaining amount', () => {
    const { container } = render(
      <MacroBar consumed={120.4} target={200} label="Calories" unit="kcal" color="#f43f5e" />,
    )

    expect(screen.getByText('Calories')).toBeInTheDocument()
    // "120 / 200 kcal" is split across a nested span + bare text nodes.
    expect(screen.getByText((_, el) => el?.textContent === '120 / 200 kcal')).toBeInTheDocument()
    expect(captionText(container)).toBe('80 kcal left')
  })

  it('sets progressbar aria attributes from rounded progress', () => {
    render(<MacroBar consumed={120} target={200} label="Protein" unit="g" color="#22c55e" />)

    const bar = screen.getByRole('progressbar', { name: 'Protein progress' })
    expect(bar).toHaveAttribute('aria-valuenow', '60')
    expect(bar).toHaveAttribute('aria-valuemin', '0')
    expect(bar).toHaveAttribute('aria-valuemax', '100')
  })
})

describe('MacroBar over target', () => {
  it('shows the "over" copy instead of "left" and clamps progress at 100', () => {
    const { container } = render(<MacroBar consumed={250} target={200} label="Fat" unit="g" color="#eab308" />)

    expect(captionText(container)).toBe('50 g over')
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100')
  })
})

describe('MacroBar target <= 0 edge case', () => {
  it('reports zero progress and zero remaining without dividing by zero', () => {
    const { container } = render(<MacroBar consumed={50} target={0} label="Fiber" unit="g" color="#0ea5e9" />)

    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '0')
    expect(captionText(container)).toBe('0 g left')
  })
})

describe('MacroBar confidence tiers', () => {
  function fillOpacityClass(): string {
    const bar = screen.getByRole('progressbar')
    const fill = bar.querySelector('div')
    return fill?.className ?? ''
  }

  it('applies no opacity dimming at high confidence (>= 0.85)', () => {
    render(<MacroBar consumed={10} target={100} label="Calories" unit="kcal" color="#000" confidence={0.9} />)
    expect(fillOpacityClass()).not.toMatch(/opacity-/)
  })

  it('applies opacity-75 at medium confidence (0.6-0.85)', () => {
    render(<MacroBar consumed={10} target={100} label="Calories" unit="kcal" color="#000" confidence={0.7} />)
    expect(fillOpacityClass()).toContain('opacity-75')
  })

  it('applies opacity-50 at low confidence (< 0.6)', () => {
    render(<MacroBar consumed={10} target={100} label="Calories" unit="kcal" color="#000" confidence={0.3} />)
    expect(fillOpacityClass()).toContain('opacity-50')
  })

  it('defaults to high confidence (no dimming) when confidence is omitted', () => {
    render(<MacroBar consumed={10} target={100} label="Calories" unit="kcal" color="#000" />)
    expect(fillOpacityClass()).not.toMatch(/opacity-/)
  })
})
