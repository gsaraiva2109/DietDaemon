import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Plan } from './Plan'
import { DemoProvider } from '@/lib/demo'
import type {
  DietPlan,
  DietPlanDayTypeBundle,
  DietPlanSlotBundle,
  DietPlanSlotOption,
  FoodDetail,
  MealTemplate,
  PlanBundle,
  ResolvedItem,
} from '@/lib/types'

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
        update: vi.fn(),
        delete: vi.fn(),
        dayTypes: {
          create: vi.fn(),
          update: vi.fn(),
          delete: vi.fn(),
        },
        slots: {
          create: vi.fn(),
          update: vi.fn(),
          delete: vi.fn(),
        },
        options: {
          create: vi.fn(),
          update: vi.fn(),
          delete: vi.fn(),
        },
      },
      templates: {
        ...actual.api.templates,
        get: vi.fn(),
      },
      foods: {
        ...actual.api.foods,
        searchCatalog: vi.fn(),
      },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const list = vi.mocked(api.plans.list)
const active = vi.mocked(api.plans.active)
const create = vi.mocked(api.plans.create)
const getBundle = vi.mocked(api.plans.get)
const updatePlan = vi.mocked(api.plans.update)
const deletePlan = vi.mocked(api.plans.delete)
const createDayType = vi.mocked(api.plans.dayTypes.create)
const updateDayType = vi.mocked(api.plans.dayTypes.update)
const deleteDayType = vi.mocked(api.plans.dayTypes.delete)
const createSlot = vi.mocked(api.plans.slots.create)
const updateSlot = vi.mocked(api.plans.slots.update)
const deleteSlot = vi.mocked(api.plans.slots.delete)
const createOption = vi.mocked(api.plans.options.create)
const updateOption = vi.mocked(api.plans.options.update)
const deleteOption = vi.mocked(api.plans.options.delete)
const templatesGet = vi.mocked(api.templates.get)
const searchCatalog = vi.mocked(api.foods.searchCatalog)

const TARGETS = { Calories: 2000, Protein: 150, Carbs: 200, Fat: 60, Fiber: 25 }

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

function option(overrides: Partial<DietPlanSlotOption> = {}): DietPlanSlotOption {
  return { id: 'opt-1', slot_id: 'slot-1', position: 0, label: 'Option 1', template_id: 'tmpl-1', ...overrides }
}

function slot(overrides: Partial<DietPlanSlotBundle> = {}): DietPlanSlotBundle {
  return { id: 'slot-1', day_type_id: 'dt-1', position: 0, time_of_day: '07:00', label: 'Breakfast', options: [], ...overrides }
}

function dayType(overrides: Partial<DietPlanDayTypeBundle> = {}): DietPlanDayTypeBundle {
  return { id: 'dt-1', plan_id: 'p1', name: 'High carb', position: 0, targets: TARGETS, water_goal_ml: 2000, slots: [], ...overrides }
}

function bundle(p: DietPlan, dayTypes: DietPlanDayTypeBundle[]): PlanBundle {
  return { plan: p, day_types: dayTypes }
}

function foodDetail(overrides: Partial<FoodDetail> = {}): FoodDetail {
  return {
    food_id: 'food-1',
    name: 'Chicken breast',
    source: 'taco',
    per_100g: { Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0 },
    category: '',
    brand: '',
    barcode: '',
    image_url: '',
    serving_size: 0,
    serving_unit: '',
    query_count: 0,
    last_used: '',
    in_library: false,
    serving_units: [],
    volume_units_eligible: false,
    ...overrides,
  }
}

function resolvedItem(overrides: Partial<ResolvedItem> = {}): ResolvedItem {
  return {
    Parsed: { RawPhrase: 'Chicken breast', Quantity: 100, Unit: '', NormalizedGrams: 100, Locale: '' },
    Match: {
      FoodID: 'food-1', Name: 'Chicken breast', Source: 'taco',
      Per100g: { Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0 }, MatchScore: 1,
    },
    Macros: { Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0 },
    ...overrides,
  }
}

function mealTemplate(overrides: Partial<MealTemplate> = {}): MealTemplate {
  return { id: 'tmpl-1', user_id: 'u1', name: 'tmpl', items: [resolvedItem()], created_at: '', last_used: '', ...overrides }
}

function renderPlan(queryClient = new QueryClient()) {
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

// Selects the (only) plan from the list to enter the builder view.
async function openBuilder() {
  renderPlan()
  fireEvent.click(await screen.findByText('Carb cycling'))
  await screen.findByRole('heading', { name: 'Carb cycling' })
}

beforeEach(() => {
  list.mockReset().mockResolvedValue([plan()])
  active.mockReset().mockRejectedValue(new ApiError(404, 'not found'))
  create.mockReset()
  getBundle.mockReset().mockResolvedValue(bundle(plan(), []))
  updatePlan.mockReset().mockResolvedValue(plan())
  deletePlan.mockReset().mockResolvedValue(undefined)
  createDayType.mockReset().mockResolvedValue(dayType())
  updateDayType.mockReset().mockResolvedValue(dayType())
  deleteDayType.mockReset().mockResolvedValue(undefined)
  createSlot.mockReset().mockResolvedValue(slot())
  updateSlot.mockReset().mockResolvedValue(slot())
  deleteSlot.mockReset().mockResolvedValue(undefined)
  createOption.mockReset().mockResolvedValue(option())
  updateOption.mockReset().mockResolvedValue(option())
  deleteOption.mockReset().mockResolvedValue(undefined)
  templatesGet.mockReset().mockResolvedValue(mealTemplate())
  searchCatalog.mockReset().mockResolvedValue([])
})

describe('Plan list', () => {
  it('shows the empty state and a new-plan form when there are no plans', async () => {
    list.mockResolvedValue([])
    renderPlan()
    expect(await screen.findByText('No plans yet')).toBeInTheDocument()
    expect(screen.getByText('New plan')).toBeInTheDocument()
  })

  // The issue calls out a 7-length cycle anchored on a Monday as the offered
  // default (arbitrary lengths are still supported elsewhere) — this pins
  // that the default is an actual upcoming Monday, not just "some date".
  it('defaults the new-plan cycle anchor to the coming Monday', async () => {
    list.mockResolvedValue([])
    renderPlan()
    const anchorInput = (await screen.findByLabelText('Cycle anchor date')) as HTMLInputElement
    const anchor = new Date(`${anchorInput.value}T00:00:00`)
    const startOfToday = new Date(new Date().toDateString())

    expect(anchor.getDay()).toBe(1) // Monday
    expect(anchor.getTime()).toBeGreaterThanOrEqual(startOfToday.getTime())
    expect(anchor.getTime()).toBeLessThan(startOfToday.getTime() + 7 * 24 * 60 * 60 * 1000)
  })

  it('creates a plan with the form fields and switches into the builder', async () => {
    list.mockResolvedValue([])
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

  it('shows the active plan pill and selecting a plan opens its builder', async () => {
    active.mockResolvedValue(plan())
    renderPlan()

    expect(await screen.findByText('Active')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Carb cycling'))
    expect(await screen.findByRole('heading', { name: 'Carb cycling' })).toBeInTheDocument()

    // Back button returns to the list.
    fireEvent.click(screen.getByText('Back to plans'))
    expect(await screen.findByText('Carb cycling')).toBeInTheDocument()
  })
})

describe('PlanBuilder shell', () => {
  it('shows a spinner while the bundle loads', async () => {
    getBundle.mockImplementation(() => new Promise(() => {})) // never resolves
    renderPlan()
    fireEvent.click(await screen.findByText('Carb cycling'))
    expect(await screen.findByText(/Loading/)).toBeInTheDocument()
  })

  it('shows a failure state when the bundle fails to load', async () => {
    getBundle.mockRejectedValue(new Error('boom'))
    // Disable retries so the query settles into 'error' within the test's
    // lifetime instead of react-query's default 3-retry backoff.
    renderPlan(new QueryClient({ defaultOptions: { queries: { retry: false } } }))
    fireEvent.click(await screen.findByText('Carb cycling'))
    expect(await screen.findByText('Could not load this plan.')).toBeInTheDocument()
  })

  it('requires a confirm step before deleting the plan', async () => {
    await openBuilder()

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(screen.getByText('Confirm delete')).toBeInTheDocument()

    // Cancel backs out without deleting.
    fireEvent.click(screen.getByText('Cancel'))
    expect(deletePlan).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    fireEvent.click(screen.getByText('Confirm delete'))
    await waitFor(() => expect(deletePlan).toHaveBeenCalledWith('p1'))

    // onDeleted() sends us back to the (now re-fetched) list.
    expect(await screen.findByText('Carb cycling')).toBeInTheDocument()
  })

  it('saves edited plan metadata', async () => {
    await openBuilder()

    const nameInputs = screen.getAllByLabelText('Name')
    fireEvent.change(nameInputs[0], { target: { value: 'Renamed plan' } })
    fireEvent.click(screen.getAllByText('Save')[0])

    await waitFor(() =>
      expect(updatePlan).toHaveBeenCalledWith('p1', expect.objectContaining({ name: 'Renamed plan' })),
    )
  })
})

describe('CycleEditor', () => {
  it('hints that day-types are needed before the cycle can be built', async () => {
    await openBuilder()
    expect(screen.getByText('Add at least one day-type below before building the cycle.')).toBeInTheDocument()
  })

  it('seeds a 7-day week once a day-type exists, and supports add/select/remove position', async () => {
    getBundle.mockResolvedValue(bundle(plan({ cycle_pattern: ['dt-1'] }), [dayType()]))
    await openBuilder()

    fireEvent.click(screen.getByText('Set up a 7-day week'))
    await waitFor(() =>
      expect(updatePlan).toHaveBeenCalledWith('p1', expect.objectContaining({ cycle_pattern: Array(7).fill('dt-1') })),
    )

    fireEvent.click(screen.getByText('+ Add position'))
    await waitFor(() =>
      expect(updatePlan).toHaveBeenCalledWith('p1', expect.objectContaining({ cycle_pattern: ['dt-1', 'dt-1'] })),
    )

    fireEvent.click(screen.getByLabelText('Remove position 1'))
    await waitFor(() =>
      expect(updatePlan).toHaveBeenCalledWith('p1', expect.objectContaining({ cycle_pattern: [] })),
    )
  })

  it('changes the anchor date', async () => {
    getBundle.mockResolvedValue(bundle(plan({ cycle_pattern: ['dt-1'] }), [dayType()]))
    await openBuilder()

    const anchorInputs = screen.getAllByLabelText('Cycle anchor date')
    fireEvent.change(anchorInputs[anchorInputs.length - 1], { target: { value: '2026-08-03' } })
    await waitFor(() =>
      expect(updatePlan).toHaveBeenCalledWith('p1', expect.objectContaining({ cycle_anchor_date: '2026-08-03' })),
    )
  })
})

describe('Day-types', () => {
  it('shows the empty state and adds a day-type', async () => {
    await openBuilder()
    expect(screen.getByText('No day-types yet')).toBeInTheDocument()

    fireEvent.click(screen.getByText('+ Add day-type'))
    await waitFor(() =>
      expect(createDayType).toHaveBeenCalledWith('p1', expect.objectContaining({ name: 'New day-type', position: 0 })),
    )
  })

  it('edits a day-type name, macros, and water goal', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType()]))
    await openBuilder()

    fireEvent.change(screen.getByLabelText('Day-type name'), { target: { value: 'Low carb' } })
    fireEvent.change(screen.getByLabelText('Carbs'), { target: { value: '80' } })
    fireEvent.change(screen.getByLabelText('Water goal (ml)'), { target: { value: '1500' } })
    // Index 0 is PlanMetaForm's own "Save" button; index 1 is the day-type's.
    fireEvent.click(screen.getAllByRole('button', { name: 'Save' })[1])

    await waitFor(() =>
      expect(updateDayType).toHaveBeenCalledWith(
        'p1',
        'dt-1',
        expect.objectContaining({ name: 'Low carb', water_goal_ml: 1500, targets: expect.objectContaining({ Carbs: 80 }) }),
      ),
    )
  })

  it('duplicates a day-type by cloning its slots and options via direct api calls', async () => {
    getBundle.mockResolvedValue(
      bundle(plan(), [dayType({ slots: [slot({ options: [option()] })] })]),
    )
    await openBuilder()

    fireEvent.click(screen.getByRole('button', { name: 'Duplicate day-type' }))

    await waitFor(() => expect(createDayType).toHaveBeenCalledWith('p1', expect.objectContaining({ name: 'High carb (copy)' })))
    await waitFor(() => expect(createSlot).toHaveBeenCalled())
    await waitFor(() => expect(templatesGet).toHaveBeenCalledWith('tmpl-1'))
    await waitFor(() => expect(createOption).toHaveBeenCalled())
  })

  it('deletes a day-type after confirming', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType()]))
    await openBuilder()

    // Index 0 is the plan's own delete button; index 1 is the day-type's.
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[1])
    fireEvent.click(screen.getByText('Confirm delete'))
    await waitFor(() => expect(deleteDayType).toHaveBeenCalledWith('p1', 'dt-1'))
  })
})

describe('Slots', () => {
  function bundleWithSlot() {
    return bundle(plan(), [dayType({ slots: [slot()] })])
  }

  it('shows "no meal slots" and adds one', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType()]))
    await openBuilder()
    expect(screen.getByText('No meal slots yet.')).toBeInTheDocument()

    fireEvent.click(screen.getByText('+ Add slot'))
    await waitFor(() =>
      expect(createSlot).toHaveBeenCalledWith('p1', 'dt-1', expect.objectContaining({ label: 'New slot' })),
    )
  })

  it('edits a slot label and time, then saves', async () => {
    getBundle.mockResolvedValue(bundleWithSlot())
    await openBuilder()

    fireEvent.change(screen.getByLabelText('Slot label'), { target: { value: 'Lunch' } })
    fireEvent.change(screen.getByLabelText('Time'), { target: { value: '12:30' } })
    // Index 0 = PlanMetaForm, 1 = day-type, 2 = this slot.
    fireEvent.click(screen.getAllByRole('button', { name: 'Save' })[2])

    await waitFor(() =>
      expect(updateSlot).toHaveBeenCalledWith('p1', 'dt-1', 'slot-1', expect.objectContaining({ label: 'Lunch', time_of_day: '12:30' })),
    )
  })

  it('duplicates a slot', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType({ slots: [slot({ options: [option()] })] })]))
    await openBuilder()

    fireEvent.click(screen.getByRole('button', { name: 'Duplicate slot' }))
    await waitFor(() => expect(createSlot).toHaveBeenCalledWith('p1', 'dt-1', expect.objectContaining({ label: 'Breakfast' })))
    await waitFor(() => expect(createOption).toHaveBeenCalled())
  })

  it('deletes a slot after confirming', async () => {
    getBundle.mockResolvedValue(bundleWithSlot())
    await openBuilder()

    // Index 0 = plan, 1 = day-type, 2 = this slot.
    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[2])
    fireEvent.click(screen.getByText('Confirm delete'))
    await waitFor(() => expect(deleteSlot).toHaveBeenCalledWith('p1', 'dt-1', 'slot-1'))
  })
})

describe('Slot options', () => {
  function bundleWithOption() {
    return bundle(plan(), [dayType({ slots: [slot({ options: [option()] })] })])
  }

  it('shows option summary once its template loads, including à vontade', async () => {
    templatesGet.mockResolvedValue(mealTemplate({ items: [resolvedItem(), resolvedItem({ Parsed: { ...resolvedItem().Parsed, NormalizedGrams: 0 } })] }))
    getBundle.mockResolvedValue(bundleWithOption())
    await openBuilder()

    expect(await screen.findByText(/kcal · à vontade \(unrestricted\)/)).toBeInTheDocument()
  })

  it('opens the add-option editor, searches the catalog, adds a food, and saves', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType({ slots: [slot()] })]))
    searchCatalog.mockResolvedValue([foodDetail()])
    createOption.mockResolvedValue(option())
    await openBuilder()

    fireEvent.click(screen.getByText('+ Add option'))
    fireEvent.change(screen.getByLabelText('Option label'), { target: { value: 'Opção 1' } })
    fireEvent.change(screen.getByPlaceholderText('Search the food catalog'), { target: { value: 'chicken' } })

    await waitFor(() => expect(searchCatalog).toHaveBeenCalledWith('chicken', '', 20), { timeout: 1000 })
    fireEvent.click(await screen.findByText('Chicken breast'))

    // Item added; adjust quantity and toggle ad libitum then back off.
    const qty = screen.getByLabelText('Quantity of Chicken breast')
    fireEvent.change(qty, { target: { value: '150' } })
    const toggle = screen.getByLabelText('Mark Chicken breast as à vontade')
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(toggle)

    fireEvent.click(screen.getByText('Save option'))
    await waitFor(() =>
      expect(createOption).toHaveBeenCalledWith(
        'p1', 'dt-1', 'slot-1',
        expect.objectContaining({
          label: 'Opção 1',
          items: [expect.objectContaining({ Parsed: expect.objectContaining({ Quantity: 150 }) })],
        }),
      ),
    )
  })

  it('shows "no matches" when the catalog search returns nothing', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType({ slots: [slot()] })]))
    searchCatalog.mockResolvedValue([])
    await openBuilder()

    fireEvent.click(screen.getByText('+ Add option'))
    fireEvent.change(screen.getByPlaceholderText('Search the food catalog'), { target: { value: 'zzz' } })

    expect(await screen.findByText('No matching foods.')).toBeInTheDocument()
  })

  it('edits an existing option: seeds items from its template, removes one, and saves', async () => {
    getBundle.mockResolvedValue(bundleWithOption())
    templatesGet.mockResolvedValue(mealTemplate({ items: [resolvedItem()] }))
    updateOption.mockResolvedValue(option())
    await openBuilder()

    fireEvent.click(await screen.findByText('Option 1'))
    expect(await screen.findByLabelText('Option label')).toHaveValue('Option 1')
    expect(screen.getByText('Chicken breast')).toBeInTheDocument()

    fireEvent.click(screen.getByLabelText('Remove Chicken breast'))
    expect(screen.getByText('Search above to add foods.')).toBeInTheDocument()

    // Save is disabled once the item list is empty.
    expect(screen.getByText('Save option')).toBeDisabled()
  })

  it('cancels the option editor without saving', async () => {
    getBundle.mockResolvedValue(bundle(plan(), [dayType({ slots: [slot()] })]))
    await openBuilder()

    fireEvent.click(screen.getByText('+ Add option'))
    fireEvent.click(screen.getByText('Cancel'))
    expect(screen.getByText('+ Add option')).toBeInTheDocument()
  })

  it('copies an option', async () => {
    getBundle.mockResolvedValue(bundleWithOption())
    await openBuilder()

    fireEvent.click(screen.getByRole('button', { name: 'Copy option' }))
    await waitFor(() => expect(templatesGet).toHaveBeenCalledWith('tmpl-1'))
    await waitFor(() =>
      expect(createOption).toHaveBeenCalledWith('p1', 'dt-1', 'slot-1', expect.objectContaining({ label: 'Option 1 (copy)' })),
    )
  })

  it('deletes an option after confirming', async () => {
    getBundle.mockResolvedValue(bundleWithOption())
    await openBuilder()

    const row = screen.getByText('Option 1').closest('div') as HTMLElement
    fireEvent.click(within(row.parentElement as HTMLElement).getByRole('button', { name: 'Delete' }))
    fireEvent.click(screen.getByText('Confirm delete'))
    await waitFor(() => expect(deleteOption).toHaveBeenCalledWith('p1', 'dt-1', 'slot-1', 'opt-1'))
  })
})
