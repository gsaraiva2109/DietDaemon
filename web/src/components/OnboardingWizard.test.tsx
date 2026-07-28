import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DemoProvider } from '@/lib/demo'
import { OnboardingWizard } from './OnboardingWizard'
import type { UserProfile, TDEEResult } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      profile: { get: vi.fn(), put: vi.fn() },
      body: { ...actual.api.body, weight: { ...actual.api.body.weight, log: vi.fn() } },
      setTargets: vi.fn(),
      tdee: vi.fn(),
    },
  }
})

import { api } from '@/lib/api'

const getProfile = vi.mocked(api.profile.get)
const putProfile = vi.mocked(api.profile.put)
const logWeight = vi.mocked(api.body.weight.log)
const setTargets = vi.mocked(api.setTargets)
const tdee = vi.mocked(api.tdee)

const NOT_ONBOARDED: UserProfile = {
  user_id: 'u1', height_cm: 0, birth_date: '', gender: 'male', activity_level: 'moderate',
  goal: 'cut', target_weight_kg: 0, weekly_rate: 0.5, onboarded: false, created_at: '', updated_at: '',
}

const ONBOARDED: UserProfile = { ...NOT_ONBOARDED, onboarded: true, height_cm: 180, birth_date: '1990-01-01' }

const TDEE_RESULT: TDEEResult = {
  bmr: 1700, tdee: 2400, cut_cal: 1900, maintain_cal: 2400, bulk_cal: 2900,
  protein_g: 150, fat_g: 70, carbs_g: 250,
}

function renderWizard({ demo = false }: { demo?: boolean } = {}) {
  localStorage.setItem('dd.demo', demo ? '1' : '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <OnboardingWizard />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  getProfile.mockReset()
  putProfile.mockReset()
  logWeight.mockReset()
  setTargets.mockReset()
  tdee.mockReset()
})

describe('OnboardingWizard visibility', () => {
  it('opens automatically when the profile is not yet onboarded', async () => {
    getProfile.mockResolvedValue(NOT_ONBOARDED)
    renderWizard()
    expect(await screen.findByText('A few body stats')).toBeInTheDocument()
  })

  it('stays hidden once the profile is already onboarded', async () => {
    getProfile.mockResolvedValue(ONBOARDED)
    renderWizard()
    await waitFor(() => expect(getProfile).toHaveBeenCalled())
    expect(screen.queryByText('A few body stats')).not.toBeInTheDocument()
  })

  it('never opens in demo mode, even when not onboarded', () => {
    renderWizard({ demo: true })
    expect(screen.queryByText('A few body stats')).not.toBeInTheDocument()
  })

  it('opens in edit mode on the dd:onboarding event, prefilled, with Cancel instead of Skip', async () => {
    getProfile.mockResolvedValue(ONBOARDED)
    renderWizard()

    // The event handler closes over `profile`, which loads asynchronously; keep
    // redispatching until the query resolves and a re-registered listener picks
    // up the loaded profile (avoids a fixed sleep for the react-query fetch).
    await waitFor(() => {
      act(() => window.dispatchEvent(new CustomEvent('dd:onboarding')))
      expect(screen.getByDisplayValue('180')).toBeInTheDocument() // height_cm prefilled
    })

    expect(screen.getByText('Edit profile')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Skip' })).not.toBeInTheDocument()
  })
})

describe('OnboardingWizard step navigation', () => {
  it('gates Next on step 0 until height/weight/birth date are filled', async () => {
    getProfile.mockResolvedValue(NOT_ONBOARDED)
    renderWizard()
    await screen.findByText('A few body stats')

    const nextButton = screen.getByRole('button', { name: 'Next' })
    expect(nextButton).toBeDisabled()

    fireEvent.change(screen.getByLabelText('Height', { exact: false }), { target: { value: '180' } })
    fireEvent.change(screen.getByLabelText('Current weight', { exact: false }), { target: { value: '75' } })
    fireEvent.change(screen.getByLabelText('Date of birth', { exact: false }), { target: { value: '1990-01-01' } })

    expect(nextButton).toBeEnabled()
    fireEvent.click(nextButton)
    expect(await screen.findByText('How active are you?')).toBeInTheDocument()
  })

  it('skip marks the profile onboarded with whatever was filled', async () => {
    getProfile.mockResolvedValue(NOT_ONBOARDED)
    putProfile.mockResolvedValue(ONBOARDED)
    renderWizard()
    await screen.findByText('A few body stats')

    fireEvent.click(screen.getByRole('button', { name: 'Skip' }))
    await waitFor(() => expect(putProfile).toHaveBeenCalledWith(expect.objectContaining({ onboarded: true })))
  })
})

describe('OnboardingWizard recommended-calories ternary (cut/maintain/bulk)', () => {
  async function goToPlanStep(goal: 'cut' | 'maintain' | 'bulk') {
    getProfile.mockResolvedValue(NOT_ONBOARDED)
    tdee.mockResolvedValue(TDEE_RESULT)
    renderWizard()
    await screen.findByText('A few body stats')

    fireEvent.change(screen.getByLabelText('Height', { exact: false }), { target: { value: '180' } })
    fireEvent.change(screen.getByLabelText('Current weight', { exact: false }), { target: { value: '75' } })
    fireEvent.change(screen.getByLabelText('Date of birth', { exact: false }), { target: { value: '1990-01-01' } })
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))

    await screen.findByText('How active are you?')
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))

    const goalLabel = goal === 'cut' ? 'Cut' : goal === 'bulk' ? 'Bulk' : 'Maintain'
    // getByRole's `name` filter matches the full accessible name (label span +
    // hint span concatenated), so anchor a regex rather than the plain string.
    const goalButton = await screen.findByRole('button', { name: new RegExp(`^${goalLabel}`) })
    fireEvent.click(goalButton)
    fireEvent.change(screen.getByLabelText('Target weight', { exact: false }), { target: { value: '70' } })
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))

    await screen.findByText('Your plan')
    // Wait for the async useTDEE query (mocked) to resolve so `recommended`
    // is populated before Save is clicked -- otherwise save() sees it as null.
    await waitFor(() => expect(screen.queryByText('Crunching your numbers…')).not.toBeInTheDocument())
  }

  it('cut branch: recommended calories come from cut_cal', async () => {
    await goToPlanStep('cut')
    fireEvent.click(screen.getByRole('button', { name: 'Save plan' }))
    await waitFor(() => expect(setTargets).toHaveBeenCalledWith(expect.objectContaining({ Calories: TDEE_RESULT.cut_cal })))
  })

  it('bulk branch: recommended calories come from bulk_cal', async () => {
    await goToPlanStep('bulk')
    fireEvent.click(screen.getByRole('button', { name: 'Save plan' }))
    await waitFor(() => expect(setTargets).toHaveBeenCalledWith(expect.objectContaining({ Calories: TDEE_RESULT.bulk_cal })))
  })

  it('maintain branch: recommended calories come from maintain_cal', async () => {
    await goToPlanStep('maintain')
    fireEvent.click(screen.getByRole('button', { name: 'Save plan' }))
    await waitFor(() => expect(setTargets).toHaveBeenCalledWith(expect.objectContaining({ Calories: TDEE_RESULT.maintain_cal })))
  })
})

describe('NumberField decimal input (exercises the fixed regex)', () => {
  it('accepts partial and full decimals, rejects letters and a second dot', async () => {
    getProfile.mockResolvedValue(NOT_ONBOARDED)
    renderWizard()
    await screen.findByText('A few body stats')

    const weight = screen.getByLabelText('Current weight', { exact: false }) as HTMLInputElement

    fireEvent.change(weight, { target: { value: '82.' } })
    expect(weight.value).toBe('82.')

    fireEvent.change(weight, { target: { value: '82.5' } })
    expect(weight.value).toBe('82.5')

    // Rejected: a second '.' leaves the field unchanged.
    fireEvent.change(weight, { target: { value: '82.5.1' } })
    expect(weight.value).toBe('82.5')

    // Rejected: a letter leaves the field unchanged.
    fireEvent.change(weight, { target: { value: '82x' } })
    expect(weight.value).toBe('82.5')
  })

  it('resolves quickly for a long digit run with no dot (regex backtracking regression)', async () => {
    getProfile.mockResolvedValue(NOT_ONBOARDED)
    renderWizard()
    await screen.findByText('A few body stats')

    const weight = screen.getByLabelText('Current weight', { exact: false }) as HTMLInputElement
    const longDigits = '9'.repeat(20_000)

    const start = performance.now()
    fireEvent.change(weight, { target: { value: longDigits } })
    expect(performance.now() - start).toBeLessThan(500)
    expect(weight.value).toBe(longDigits)
  })
})
