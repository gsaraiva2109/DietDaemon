import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { TargetReviewPrompt } from './TargetReviewPrompt'
import { DemoProvider } from '@/lib/demo'
import type { TargetReviewSuggestion } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      targetReview: vi.fn(),
    },
  }
})

import { api } from '@/lib/api'

const targetReview = vi.mocked(api.targetReview)

const SUGGESTION: TargetReviewSuggestion = {
  message: 'Your weight has been stable for the last 4 weeks while your goal is set to cut — want to review your target?',
  observed_trend_kg_per_week: 0,
  goal: 'cut',
  since_date: '2026-07-03',
}

function renderPrompt() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <TargetReviewPrompt />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  targetReview.mockReset()
  localStorage.clear()
})

describe('TargetReviewPrompt', () => {
  it('renders the message when the hook returns a suggestion', async () => {
    targetReview.mockResolvedValue(SUGGESTION)
    renderPrompt()

    expect(await screen.findByText(SUGGESTION.message)).toBeInTheDocument()
  })

  it('renders nothing when there is no suggestion', async () => {
    targetReview.mockResolvedValue({ message: '', observed_trend_kg_per_week: 0, goal: '', since_date: '' })
    renderPrompt()

    await new Promise((r) => setTimeout(r, 0))
    expect(screen.queryByText('Review target')).not.toBeInTheDocument()
  })

  it('dispatches dd:onboarding when "Review target" is clicked', async () => {
    targetReview.mockResolvedValue(SUGGESTION)
    const onOnboarding = vi.fn()
    window.addEventListener('dd:onboarding', onOnboarding)
    renderPrompt()

    fireEvent.click(await screen.findByText('Review target'))
    expect(onOnboarding).toHaveBeenCalledTimes(1)
    window.removeEventListener('dd:onboarding', onOnboarding)
  })

  it('dismisses and suppresses via localStorage', async () => {
    targetReview.mockResolvedValue(SUGGESTION)
    renderPrompt()

    fireEvent.click(await screen.findByText('Dismiss'))
    expect(screen.queryByText(SUGGESTION.message)).not.toBeInTheDocument()
    expect(Number(localStorage.getItem('dd:targetReviewDismissedUntil'))).toBeGreaterThan(Date.now())
  })
})
