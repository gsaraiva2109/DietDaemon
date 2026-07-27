import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Plan } from './Plan'
import { DemoProvider } from '@/lib/demo'
import type { DietPlan } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      plans: {
        ...actual.api.plans,
        list: vi.fn(),
        active: vi.fn(),
        create: vi.fn(),
        get: vi.fn(),
      },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const list = vi.mocked(api.plans.list)
const active = vi.mocked(api.plans.active)
const create = vi.mocked(api.plans.create)

function plan(overrides: Partial<DietPlan> = {}): DietPlan {
  return {
    id: 'p1',
    user_id: 'u1',
    name: 'Carb cycling',
    notes: '',
    valid_from: '2026-07-27',
    valid_to: '',
    cycle_pattern: [],
    cycle_anchor_date: '2026-07-27',
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function renderPlan() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <Plan />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  list.mockReset().mockResolvedValue([])
  active.mockReset().mockRejectedValue(new ApiError(404, 'not found'))
  create.mockReset()
})

describe('Plan list', () => {
  it('shows the empty state and a new-plan form when there are no plans', async () => {
    renderPlan()
    expect(await screen.findByText('No plans yet')).toBeInTheDocument()
    expect(screen.getByText('New plan')).toBeInTheDocument()
  })

  // The issue calls out a 7-length cycle anchored on a Monday as the offered
  // default (arbitrary lengths are still supported elsewhere) — this pins
  // that the default is an actual upcoming Monday, not just "some date".
  it('defaults the new-plan cycle anchor to the coming Monday', async () => {
    renderPlan()
    const anchorInput = (await screen.findByLabelText('Cycle anchor date')) as HTMLInputElement
    const anchor = new Date(`${anchorInput.value}T00:00:00`)
    const startOfToday = new Date(new Date().toDateString())

    expect(anchor.getDay()).toBe(1) // Monday
    expect(anchor.getTime()).toBeGreaterThanOrEqual(startOfToday.getTime())
    expect(anchor.getTime()).toBeLessThan(startOfToday.getTime() + 7 * 24 * 60 * 60 * 1000)
  })

  it('creates a plan with the form fields and switches into the builder', async () => {
    create.mockResolvedValue(plan())
    renderPlan()

    fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'Carb cycling' } })
    fireEvent.click(screen.getByText('Create plan'))

    await waitFor(() =>
      expect(create).toHaveBeenCalledWith(expect.objectContaining({ name: 'Carb cycling' })),
    )
    // Successful create hands off to the builder view (back button appears).
    expect(await screen.findByText('Back to plans')).toBeInTheDocument()
  })
})
