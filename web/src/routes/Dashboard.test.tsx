import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Dashboard } from './Dashboard'
import { DemoProvider } from '@/lib/demo'
import type {
  DailyRollup,
  Meal,
  DietPlan,
  PlanBundle,
  PlanDayView,
  StreakResponse,
  WeeklyBudgetResponse,
  BodyCompositionSummary,
} from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      rollupToday: vi.fn(),
      rollupRange: vi.fn(),
      meals: vi.fn(),
      streak: vi.fn(),
      body: { ...actual.api.body, summary: vi.fn() },
      budget: { ...actual.api.budget, weekly: vi.fn() },
      plans: {
        ...actual.api.plans,
        active: vi.fn(),
        day: vi.fn(),
        get: vi.fn(),
        setOverride: vi.fn(),
      },
      templates: { ...actual.api.templates, log: vi.fn() },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const rollupToday = vi.mocked(api.rollupToday)
const rollupRange = vi.mocked(api.rollupRange)
const meals = vi.mocked(api.meals)
const streak = vi.mocked(api.streak)
const bodySummary = vi.mocked(api.body.summary)
const budgetWeekly = vi.mocked(api.budget.weekly)
const plansActive = vi.mocked(api.plans.active)
const plansDay = vi.mocked(api.plans.day)
const plansGet = vi.mocked(api.plans.get)
const plansSetOverride = vi.mocked(api.plans.setOverride)
const templatesLog = vi.mocked(api.templates.log)

const TARGETS = { Calories: 2600, Protein: 180, Carbs: 320, Fat: 70, Fiber: 30 }
const CONSUMED = { Calories: 1840, Protein: 120, Carbs: 210, Fat: 48, Fiber: 12 }
const TODAY_ROLLUP: DailyRollup = { UserID: 'u1', Date: '2026-07-27', Consumed: CONSUMED, Targets: TARGETS }
const BODY: BodyCompositionSummary = {
  current_weight_kg: 0,
  start_weight_kg: 0,
  change_kg: 0,
  trend_direction: 'stable',
}
const STREAK: StreakResponse = { current_days: 0 }
const BUDGET: WeeklyBudgetResponse = {
  calories: { plain: 2600, effective: 2600 },
  protein: { plain: 180, effective: 180 },
}

function noPlanView(): PlanDayView {
  return { date: '2026-07-27', plan_active: false, overridden: false, day_type: null, slots: [], targets: { UserID: 'u1', Targets: TARGETS, WaterGoalMl: 0 } }
}

function plan(overrides: Partial<DietPlan> = {}): DietPlan {
  return {
    id: 'p1',
    user_id: 'u1',
    name: 'Carb cycling',
    notes: '',
    valid_from: '2026-01-01',
    valid_to: '',
    cycle_pattern: ['dt-high', 'dt-low'],
    cycle_anchor_date: '2026-01-01',
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

function meal(at: string, kcal: number): Meal {
  return {
    ID: `m-${at}`,
    UserID: 'u1',
    At: at,
    RawText: 'test meal',
    Confidence: 1,
    ParserTier: 0,
    CreatedAt: at,
    PlanSlotID: '',
    PlanOptionID: '',
    Items: [
      {
        Parsed: { RawPhrase: 'food', Quantity: 1, Unit: '', NormalizedGrams: 100, Locale: '' },
        Match: { FoodID: 'f1', Name: 'Food', Source: 'taco', Per100g: { Calories: kcal, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 }, MatchScore: 1 },
        Macros: { Calories: kcal, Protein: 0, Carbs: 0, Fat: 0, Fiber: 0 },
      },
    ],
  }
}

// Builds a PlanDayView with two slots (07:00 Café / 12:30 Almoço) so slot
// inference has two well-separated windows to place a meal into.
function planDayWithSlots(): PlanDayView {
  return {
    date: '2026-07-27',
    plan_active: true,
    overridden: false,
    day_type: { id: 'dt-high', plan_id: 'p1', name: 'ALTO CARBO', position: 0, targets: TARGETS, water_goal_ml: 2000 },
    slots: [
      { id: 'slot-breakfast', day_type_id: 'dt-high', position: 0, time_of_day: '07:00', label: 'Café', options: [{ id: 'opt-1', slot_id: 'slot-breakfast', position: 0, label: 'Opção 1', template_id: 'tmpl-1' }] },
      { id: 'slot-lunch', day_type_id: 'dt-high', position: 1, time_of_day: '12:30', label: 'Almoço', options: [{ id: 'opt-2', slot_id: 'slot-lunch', position: 0, label: 'Opção 1', template_id: 'tmpl-2' }] },
    ],
    targets: { UserID: 'u1', Targets: TARGETS, WaterGoalMl: 2000 },
  }
}

function bundleWith(dayTypes: PlanBundle['day_types']): PlanBundle {
  return { plan: plan(), day_types: dayTypes }
}

function renderDashboard() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <Dashboard />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  rollupToday.mockReset().mockResolvedValue(TODAY_ROLLUP)
  rollupRange.mockReset().mockResolvedValue([])
  meals.mockReset().mockResolvedValue([])
  streak.mockReset().mockResolvedValue(STREAK)
  bodySummary.mockReset().mockResolvedValue(BODY)
  budgetWeekly.mockReset().mockResolvedValue(BUDGET)
  plansActive.mockReset().mockRejectedValue(new ApiError(404, 'not found'))
  plansDay.mockReset().mockResolvedValue(noPlanView())
  plansGet.mockReset().mockResolvedValue(bundleWith([]))
  plansSetOverride.mockReset().mockResolvedValue({ status: 'ok' })
  templatesLog.mockReset().mockResolvedValue({ status: 'queued' } as never)
})

describe('Dashboard with no active plan', () => {
  it('renders byte-identical to the no-plan baseline: no plan surfaces, existing sections intact', async () => {
    renderDashboard()

    // Existing dashboard sections still render.
    expect(await screen.findByText("Today's meals")).toBeInTheDocument()
    expect(await screen.findByText('Streak')).toBeInTheDocument()
    expect(screen.getByText('Last 7 days · calories')).toBeInTheDocument()

    // No plan surfaces anywhere: no week strip, no day-type switcher, and
    // GET /plans/day, /plans/{id} are never even requested (enabled-gated).
    expect(screen.queryByText('Week cycle')).not.toBeInTheDocument()
    expect(screen.queryByLabelText("Switch today's day-type", { exact: false })).not.toBeInTheDocument()
    await waitFor(() => expect(plansActive).toHaveBeenCalledTimes(1))
    expect(plansDay).not.toHaveBeenCalled()
    expect(plansGet).not.toHaveBeenCalled()
  })
})

describe('Dashboard with an active plan', () => {
  it('shows the day-type badge above the ring and the week cycle strip', async () => {
    plansActive.mockResolvedValue(plan())
    plansDay.mockResolvedValue(planDayWithSlots())
    plansGet.mockResolvedValue(
      bundleWith([
        { id: 'dt-high', plan_id: 'p1', name: 'ALTO CARBO', position: 0, targets: TARGETS, water_goal_ml: 2000, slots: [] },
        { id: 'dt-low', plan_id: 'p1', name: 'BAIXO CARBO', position: 1, targets: TARGETS, water_goal_ml: 1500, slots: [] },
      ]),
    )

    renderDashboard()

    // Appears twice: the badge above the ring and the checklist card title.
    const badges = await screen.findAllByText((_, el) => el?.textContent === 'Today · ALTO CARBO')
    expect(badges.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Week cycle')).toBeInTheDocument()
    // 7 week-strip day-type pickers, one per column.
    expect(screen.getAllByLabelText(/Set .* to a different day-type/)).toHaveLength(7)
  })

  it('ticks the right slot for a bot-logged meal (no plan_slot_id/plan_option_id written) and offers "log" for the empty one', async () => {
    plansActive.mockResolvedValue(plan())
    plansDay.mockResolvedValue(planDayWithSlots())
    // A meal at 12:35 -- closer to the 12:30 lunch slot than the 07:00
    // breakfast slot -- arriving with no plan_slot_id/plan_option_id, exactly
    // how a bot-logged meal looks. Slot completion must be inferred purely
    // for display; nothing here writes those columns.
    const today = new Date()
    today.setHours(12, 35, 0, 0)
    meals.mockResolvedValue([meal(today.toISOString(), 820)])

    renderDashboard()

    await screen.findByText('Almoço')
    // Lunch slot shows a checkmark + logged kcal, no "log" buttons left.
    expect(screen.getByText('820 kcal')).toBeInTheDocument()
    // Breakfast slot is still pending: its "log opção" button is offered.
    expect(await screen.findByText('Log Opção 1')).toBeInTheDocument()
  })

  it('logs the prescribed option via the existing log-template path when tapped', async () => {
    plansActive.mockResolvedValue(plan())
    plansDay.mockResolvedValue(planDayWithSlots())
    meals.mockResolvedValue([])

    renderDashboard()

    const buttons = await screen.findAllByText('Log Opção 1')
    fireEvent.click(buttons[0])

    await waitFor(() =>
      expect(templatesLog).toHaveBeenCalledWith('tmpl-1', { plan_slot_id: 'slot-breakfast', plan_option_id: 'opt-1' }),
    )
  })

  it('tapping a week-strip day writes an override for that date', async () => {
    plansActive.mockResolvedValue(plan())
    plansDay.mockResolvedValue(planDayWithSlots())
    plansGet.mockResolvedValue(
      bundleWith([
        { id: 'dt-high', plan_id: 'p1', name: 'ALTO CARBO', position: 0, targets: TARGETS, water_goal_ml: 2000, slots: [] },
        { id: 'dt-low', plan_id: 'p1', name: 'BAIXO CARBO', position: 1, targets: TARGETS, water_goal_ml: 1500, slots: [] },
      ]),
    )

    renderDashboard()

    // Wait for the plan bundle to load (the "BAIXO CARBO" option appearing
    // means every week-strip <select> now has both day-types to pick from),
    // not just for the <select> elements to exist.
    await screen.findAllByText('BAIXO CARBO')
    const pickers = screen.getAllByLabelText(/Set .* to a different day-type/)
    fireEvent.change(pickers[0], { target: { value: 'dt-low' } })

    await waitFor(() => expect(plansSetOverride).toHaveBeenCalledWith(expect.any(String), 'dt-low'))
  })
})
