import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { MacroRing } from './MacroRing'

describe('MacroRing', () => {
  it('shows remaining-to-target by default, with an aria-label summarizing progress', () => {
    render(<MacroRing consumed={800} target={2000} label="Calories" unit="kcal" color="red" />)

    expect(screen.getByText('1,200')).toBeInTheDocument()
    expect(screen.getByText('kcal left')).toBeInTheDocument()
    expect(
      screen.getByRole('img', { name: 'Calories: 800 of 2000 kcal, 40 percent' }),
    ).toBeInTheDocument()
  })

  it('switches to "over" and clamps the center value at zero once consumed exceeds target', () => {
    render(<MacroRing consumed={2500} target={2000} label="Calories" unit="kcal" color="red" />)

    expect(screen.getByText('0')).toBeInTheDocument()
    expect(screen.getByText('kcal over')).toBeInTheDocument()
  })

  it('shows consumed (not remaining) in the center when center="consumed"', () => {
    render(
      <MacroRing consumed={800} target={2000} label="Calories" unit="kcal" color="red" center="consumed" />,
    )

    expect(screen.getByText('800')).toBeInTheDocument()
    expect(screen.getByText('kcal eaten')).toBeInTheDocument()
  })
})
