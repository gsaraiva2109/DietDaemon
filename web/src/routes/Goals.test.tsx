import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Goals } from './Goals'
import { DemoProvider } from '@/lib/demo'
import type { Macros, TDEEResult, UserProfile } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getTargets: vi.fn(),
      setTargets: vi.fn(),
      tdee: vi.fn(),
      goalSuggestions: vi.fn(),
      targetReview: vi.fn(),
      profile: { ...actual.api.profile, get: vi.fn() },
      body: {
        ...actual.api.body,
        weight: { ...actual.api.body.weight, list: vi.fn() },
      },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const getTargets = vi.mocked(api.getTargets)
const setTargets = vi.mocked(api.setTargets)
const tdee = vi.mocked(api.tdee)
const goalSuggestions = vi.mocked(api.goalSuggestions)
const targetReview = vi.mocked(api.targetReview)
const profileGet = vi.mocked(api.profile.get)
const weightList = vi.mocked(api.body.weight.list)

const TARGETS: Macros = { Calories: 2200, Protein: 160, Carbs: 220, Fat: 60, Fiber: 30 }

const TDEE_RESULT: TDEEResult = {
  bmr: 1600,
  tdee: 2400,
  cut_cal: 1900,
  maintain_cal: 2400,
  bulk_cal: 2800,
  protein_g: 160,
  fat_g: 70,
  carbs_g: 250,
}

function profile(overrides: Partial<UserProfile> = {}): UserProfile {
  return {
    user_id: 'u1',
    height_cm: 178,
    birth_date: '1990-01-01',
    gender: 'male',
    activity_level: 'moderate',
    goal: 'cut',
    target_weight_kg: 75,
    weekly_rate: 0.5,
    onboarded: true,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function renderGoals(queryClient = new QueryClient()) {
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <Goals />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  profileGet.mockReset()
  getTargets.mockReset()
  setTargets.mockReset().mockResolvedValue({ UserID: 'u1', Targets: TARGETS })
  tdee.mockReset().mockResolvedValue(TDEE_RESULT)
  goalSuggestions.mockReset().mockResolvedValue({
    current_intake_kcal: 0,
    recommended_kcal: 0,
    current_loss_kg: 0,
    target_loss_kg: 0,
    message: '',
  })
  targetReview.mockReset().mockResolvedValue({ message: '', observed_trend_kg_per_week: 0, goal: '', since_date: '' })
  weightList.mockReset().mockResolvedValue([])
})

describe('Goals loading state', () => {
  it('shows a spinner while the profile query is pending', () => {
    profileGet.mockImplementation(() => new Promise(() => {}))
    getTargets.mockImplementation(() => new Promise(() => {}))
    renderGoals()

    expect(screen.getByText('Your plan')).toBeInTheDocument()
    expect(screen.getByRole('status')).toBeInTheDocument()
  })
})

describe('Goals with no profile and no targets', () => {
  it('prompts to set up a profile in both the targets and TDEE cards', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    profileGet.mockRejectedValue(new ApiError(404, 'no profile'))
    getTargets.mockRejectedValue(new ApiError(404, 'no targets'))
    renderGoals(queryClient)

    expect(await screen.findByText('No targets set yet')).toBeInTheDocument()
    expect(screen.getByText('Tell us about yourself')).toBeInTheDocument()
    // No "Edit profile" header button without a profile.
    expect(screen.queryByText('Edit profile')).not.toBeInTheDocument()

    const setupButtons = screen.getAllByText('Set up your profile')
    expect(setupButtons.length).toBeGreaterThanOrEqual(1)

    const onOnboarding = vi.fn()
    window.addEventListener('dd:onboarding', onOnboarding)
    fireEvent.click(setupButtons[0])
    expect(onOnboarding).toHaveBeenCalledTimes(1)
    window.removeEventListener('dd:onboarding', onOnboarding)
  })
})

describe('Goals with a full profile and targets', () => {
  it('renders daily targets, the TDEE card, and recalculates targets on demand', async () => {
    profileGet.mockResolvedValue(profile({ goal: 'cut' }))
    getTargets.mockResolvedValue({ UserID: 'u1', Targets: TARGETS })
    weightList.mockResolvedValue([
      { id: 'w1', user_id: 'u1', date: '2026-08-01', weight_kg: 80, note: '', created_at: '' },
    ])
    renderGoals()

    // Daily targets grid renders each macro value (formatNumber adds thousands separators).
    expect(await screen.findByText('2,200')).toBeInTheDocument()
    // TDEE card renders once tdee params are gated in (weight/height/age all > 0).
    expect(await screen.findByText('Energy budget')).toBeInTheDocument()

    // Edit profile button is present with a loaded profile, and dispatches the wizard event.
    const onOnboarding = vi.fn()
    window.addEventListener('dd:onboarding', onOnboarding)
    fireEvent.click(screen.getByText('Edit profile'))
    expect(onOnboarding).toHaveBeenCalledTimes(1)
    window.removeEventListener('dd:onboarding', onOnboarding)

    // Recalculate uses the cut_cal figure since profile.goal === 'cut'.
    const recalcBtn = await screen.findByText('Recalculate targets')
    fireEvent.click(recalcBtn)

    await waitFor(() =>
      expect(setTargets).toHaveBeenCalledWith({
        Calories: TDEE_RESULT.cut_cal,
        Protein: TDEE_RESULT.protein_g,
        Carbs: TDEE_RESULT.carbs_g,
        Fat: TDEE_RESULT.fat_g,
        Fiber: 30,
      }),
    )
  })
})
