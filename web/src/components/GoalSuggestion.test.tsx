import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { GoalSuggestion } from './GoalSuggestion'
import { DemoProvider } from '@/lib/demo'
import type { GoalSuggestion as GoalSuggestionData, Macros } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      goalSuggestions: vi.fn(),
      getTargets: vi.fn(),
      setTargets: vi.fn(),
    },
  }
})

import { api } from '@/lib/api'

const goalSuggestions = vi.mocked(api.goalSuggestions)
const getTargets = vi.mocked(api.getTargets)
const setTargets = vi.mocked(api.setTargets)

const TARGETS: Macros = { Calories: 2200, Protein: 180, Carbs: 250, Fat: 70, Fiber: 30 }

const SUGGESTION: GoalSuggestionData = {
  current_intake_kcal: 2050,
  recommended_kcal: 1900,
  current_loss_kg: 0.3,
  target_loss_kg: 0.5,
  message: "You're losing 0.3 kg/week at ~2050 kcal. To hit 0.5 kg/week, try ~1900 kcal.",
}

function renderSuggestion() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <GoalSuggestion />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  goalSuggestions.mockReset()
  getTargets.mockReset().mockResolvedValue({ UserID: 'u1', Targets: TARGETS })
  setTargets.mockReset()
  localStorage.clear()
})

describe('GoalSuggestion', () => {
  it('renders nothing when there is no suggestion data', async () => {
    goalSuggestions.mockResolvedValue({
      current_intake_kcal: 0,
      recommended_kcal: 0,
      current_loss_kg: 0,
      target_loss_kg: 0,
      message: '',
    })
    renderSuggestion()

    await new Promise((r) => setTimeout(r, 0))
    expect(screen.queryByText('Suggested adjustment')).not.toBeInTheDocument()
  })

  it('renders nothing when recommended_kcal is not positive', async () => {
    goalSuggestions.mockResolvedValue({ ...SUGGESTION, recommended_kcal: 0 })
    renderSuggestion()

    await new Promise((r) => setTimeout(r, 0))
    expect(screen.queryByText('Suggested adjustment')).not.toBeInTheDocument()
  })

  it('renders the title, stats, and static message when a suggestion is present', async () => {
    goalSuggestions.mockResolvedValue(SUGGESTION)
    renderSuggestion()

    // The card renders a fixed i18n copy string, not the API's `message`
    // field, that field only gates whether the card renders at all.
    expect(await screen.findByText('Suggested adjustment')).toBeInTheDocument()
    expect(screen.getByText('Keep going! Track your meals consistently to reach your goals.')).toBeInTheDocument()
    expect(screen.getByText('2,050')).toBeInTheDocument()
    expect(screen.getByText('1,900')).toBeInTheDocument()
  })

  it('applies the recommended target and shows the applied state on success', async () => {
    goalSuggestions.mockResolvedValue(SUGGESTION)
    setTargets.mockResolvedValue({
      UserID: 'u1',
      Targets: { ...TARGETS, Calories: SUGGESTION.recommended_kcal },
    })
    renderSuggestion()

    fireEvent.click(await screen.findByText('Apply'))

    await waitFor(() => expect(setTargets).toHaveBeenCalledWith({ ...TARGETS, Calories: SUGGESTION.recommended_kcal }))
    expect(await screen.findByText('Applied')).toBeInTheDocument()
  })

  it('shows an error message when applying the target fails', async () => {
    goalSuggestions.mockResolvedValue(SUGGESTION)
    setTargets.mockRejectedValue(new Error('network down'))
    renderSuggestion()

    fireEvent.click(await screen.findByText('Apply'))

    expect(await screen.findByText('network down')).toBeInTheDocument()
  })

  it('disables Apply and shows the unavailable hint in demo mode', async () => {
    localStorage.setItem('dd.demo', '1')
    goalSuggestions.mockResolvedValue(SUGGESTION)
    renderSuggestion()

    const button = await screen.findByText('Apply')
    expect(button.closest('button')).toBeDisabled()
    expect(screen.getByText('unavailable')).toBeInTheDocument()
  })
})
