// Fixture-driven regression test for the photo/PDF import path (#224 req. 3):
// runs the REAL pdfToText (unlike PlanImport.test.tsx, which mocks it) over
// the real 4-page dietbox-plan-4page.pdf fixture, feeds the resulting text
// into a (still-mocked, as in PlanImport.test.tsx) api.plans.extract.fromText
// using a canned PlanDraft that mirrors every day type/meal/substitution/note
// the fixture actually contains, and asserts DraftReview renders all of it —
// proving a complete multi-page prescription reaches the review screen, and
// that dropping/reordering a page would be caught (see the fixture's own
// generator script for the manual negative-case verification of that).
//
// pdfjs-dist's browser build can't run in jsdom/Node (no `Worker` global —
// see pdfToText.fixture.test.ts's comment for the full story), so this file
// remaps the `pdfjs-dist` specifier to the real legacy build, same as there.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Plan } from './Plan'
import { DemoProvider } from '@/lib/demo'
import type { DietPlan, FoodDetail, PlanBundle, PlanDraft } from '@/lib/types'

vi.mock('pdfjs-dist', async () => {
  ;(globalThis as unknown as { pdfjsWorker?: unknown }).pdfjsWorker = await import(
    'pdfjs-dist/legacy/build/pdf.worker.min.mjs'
  )
  return import('pdfjs-dist/legacy/build/pdf.mjs')
})

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
        get: vi.fn(),
        create: vi.fn(),
        update: vi.fn(),
        extract: { fromText: vi.fn(), fromImage: vi.fn() },
        dayTypes: { ...actual.api.plans.dayTypes, create: vi.fn() },
        slots: { ...actual.api.plans.slots, create: vi.fn() },
        options: { ...actual.api.plans.options, create: vi.fn() },
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
const getBundle = vi.mocked(api.plans.get)
const createPlan = vi.mocked(api.plans.create)
const extractFromText = vi.mocked(api.plans.extract.fromText)
const extractFromImage = vi.mocked(api.plans.extract.fromImage)
const createDayType = vi.mocked(api.plans.dayTypes.create)
const createSlot = vi.mocked(api.plans.slots.create)
const createOption = vi.mocked(api.plans.options.create)
const searchCatalog = vi.mocked(api.foods.searchCatalog)

const TARGETS = { Calories: 1800, Protein: 140, Carbs: 150, Fat: 55, Fiber: 20 }
const PAGE_MARKERS = ['TRK-P1-AVEIA', 'TRK-P2-FRANGO', 'RST-P3-IOGURTE', 'RST-P4-SALMAO']

function plan(overrides: Partial<DietPlan> = {}): DietPlan {
  return {
    id: 'new-plan-1',
    user_id: 'u1',
    name: 'Imported plan',
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

function bundle(p: DietPlan): PlanBundle {
  return { plan: p, day_types: [] }
}

function foodDetail(): FoodDetail {
  return {
    food_id: 'food-1',
    name: 'Matched catalog food',
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
  }
}

// Mirrors dietbox-plan-4page.pdf's content exactly: 2 day types split across
// the 4 pages, meals/slots spread realistically (not all on one page), the
// standalone substitution note from page 2, and the general plan note from
// page 4 — see generate-dietbox-fixture.mjs for the source text.
function fixtureDraft(): PlanDraft {
  return {
    plan_name: 'DietBox Fixture Plan',
    unreadable: false,
    notes: 'Beber 2 litros de agua por dia',
    weekday_schedule: new Array(7).fill(null),
    substitutions: ['Trocar arroz integral por 150g de batata doce'],
    day_types: [
      {
        name: 'Dia de treino',
        targets: TARGETS,
        water_goal_ml: 2000,
        low_confidence_fields: [],
        slots: [
          {
            label: 'Cafe da manha',
            time_of_day: '07:00',
            options: [
              {
                label: 'Opcao 1',
                low_confidence_fields: [],
                items: [
                  { raw_name: 'Aveia em flocos', quantity: 100, unit: 'g', ad_libitum: false },
                  { raw_name: 'Banana media', quantity: 1, unit: null, ad_libitum: false },
                ],
              },
            ],
          },
          {
            label: 'Almoco',
            time_of_day: '12:30',
            options: [
              {
                label: 'Opcao 1',
                low_confidence_fields: [],
                items: [
                  { raw_name: 'Peito de frango grelhado', quantity: 150, unit: 'g', ad_libitum: false },
                  { raw_name: 'Arroz integral', quantity: 100, unit: 'g', ad_libitum: false },
                ],
              },
            ],
          },
        ],
      },
      {
        name: 'Dia de descanso',
        targets: TARGETS,
        water_goal_ml: 2000,
        low_confidence_fields: [],
        slots: [
          {
            label: 'Cafe da manha',
            time_of_day: '08:00',
            options: [
              {
                label: 'Opcao 1',
                low_confidence_fields: [],
                items: [
                  { raw_name: 'Iogurte natural', quantity: 200, unit: 'ml', ad_libitum: false },
                  { raw_name: 'Granola', quantity: 30, unit: 'g', ad_libitum: false },
                ],
              },
            ],
          },
          {
            label: 'Jantar',
            time_of_day: '19:00',
            options: [
              {
                label: 'Opcao 1',
                low_confidence_fields: [],
                items: [
                  { raw_name: 'Salmao grelhado', quantity: 120, unit: 'g', ad_libitum: false },
                  { raw_name: 'Salada verde', quantity: null, unit: null, ad_libitum: true },
                ],
              },
            ],
          },
        ],
      },
    ],
  }
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

function fixtureFile(): File {
  const bytes = readFileSync(path.join(import.meta.dirname, 'testdata/dietbox-plan-4page.pdf'))
  return new File([bytes], 'dietbox-plan-4page.pdf', { type: 'application/pdf' })
}

// Resolves every unresolved draft item against the same canned catalog match,
// one at a time. Every row's search box starts pre-filled with its raw_name
// and debounces its own search on mount, so several rows' dropdowns can be
// showing "Matched catalog food" at once — resolving the first result
// repeatedly (re-querying each time, since resolving a row swaps it out of
// the search UI entirely) still converges on every item resolved.
async function resolveAllItems() {
  for (let guard = 0; guard < 20; guard++) {
    const inputs = screen.queryAllByLabelText(/^Search the food catalog for/)
    if (inputs.length === 0) break
    // Anchored to the start of the accessible name: a resolved row's remove
    // button is labeled "Remove Matched catalog food", which also contains
    // the food name as a substring, so an unanchored regex would match it
    // too and click "undo" instead of "resolve next" — this only matches
    // the search-result button, whose name starts with the food name itself.
    const matches = await screen.findAllByRole('button', { name: /^Matched catalog food/ })
    fireEvent.click(matches[0])
  }
}

beforeEach(() => {
  list.mockReset().mockResolvedValue([])
  active.mockReset().mockRejectedValue(new ApiError(404, 'not found'))
  getBundle.mockReset().mockResolvedValue(bundle(plan()))
  createPlan.mockReset().mockResolvedValue(plan())
  extractFromText.mockReset()
  extractFromImage.mockReset()
  createDayType.mockReset()
  createSlot.mockReset()
  createOption.mockReset()
  searchCatalog.mockReset().mockResolvedValue([foodDetail()])
})

describe('Plan import from photo/PDF, against a real multi-page PDF fixture', () => {
  // Real pdfjs-dist parsing plus a multi-item resolve loop comfortably clears
  // vitest's 5s default locally, but coverage instrumentation on CI's slower
  // runner pushes it over — give it real headroom instead of a tight budget.
  it('extracts the real PDF text (every page, in order) and renders every day type/meal/substitution/note on review', async () => {
    // See pdfToText.fixture.test.ts's comment: the legacy build can't load
    // glyph-width metrics for a non-embedded standard font without an
    // explicit standardFontDataUrl, and warns once — harmless, since
    // getTextContent()'s extracted strings (asserted below) don't depend on
    // font metrics.
    vi.spyOn(console, 'warn').mockImplementation((...args) => {
      if (typeof args[0] === 'string' && args[0].includes('standardFontDataUrl')) return
      throw new Error(`Unexpected console.warn: ${args.map(String).join(' ')}`)
    })

    extractFromText.mockResolvedValue(fixtureDraft())
    renderPlan()
    fireEvent.click(await screen.findByRole('button', { name: 'Import from photo/PDF' }))
    fireEvent.change(screen.getByLabelText('Choose a photo or PDF'), { target: { files: [fixtureFile()] } })

    await screen.findByText('Review the extracted plan')

    // The text pdfToText actually extracted from the real PDF reached the
    // (mocked) network call, in page order — the core proof of req. 2.
    const sentText = extractFromText.mock.calls[0][0]
    const positions = PAGE_MARKERS.map((m) => sentText.indexOf(m))
    expect(positions.every((p) => p !== -1)).toBe(true)
    expect(positions).toEqual([...positions].sort((a, b) => a - b))

    // Every day type, slot/meal, and item from the fixture draft renders.
    expect(screen.getAllByText('Dia de treino').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Dia de descanso').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Cafe da manha')).toHaveLength(2)
    expect(screen.getByText('Almoco')).toBeInTheDocument()
    expect(screen.getByText('Jantar')).toBeInTheDocument()
    expect(screen.getByText('Aveia em flocos')).toBeInTheDocument()
    expect(screen.getByText('Banana media')).toBeInTheDocument()
    expect(screen.getByText('Peito de frango grelhado')).toBeInTheDocument()
    expect(screen.getByText('Arroz integral')).toBeInTheDocument()
    expect(screen.getByText('Iogurte natural')).toBeInTheDocument()
    expect(screen.getByText('Granola')).toBeInTheDocument()
    expect(screen.getByText('Salmao grelhado')).toBeInTheDocument()
    expect(screen.getByText('Salada verde')).toBeInTheDocument()

    // The standalone substitution note renders in its own section.
    expect(screen.getByText('Substitutions')).toBeInTheDocument()
    expect(screen.getByText('Trocar arroz integral por 150g de batata doce')).toBeInTheDocument()

    // Resolving every item and confirming folds notes + substitutions
    // together (buildNotesWithSubstitutions, already unit-covered in
    // PlanImport.test.tsx — this proves it end-to-end for a real extraction).
    createDayType.mockResolvedValue({
      id: 'dt-1',
      plan_id: 'new-plan-1',
      name: 'Dia de treino',
      position: 0,
      targets: TARGETS,
      water_goal_ml: 2000,
    })
    createSlot.mockResolvedValue({ id: 'slot-1', day_type_id: 'dt-1', position: 0, time_of_day: '07:00', label: 'Cafe da manha' })
    createOption.mockResolvedValue({ id: 'opt-1', slot_id: 'slot-1', position: 0, label: 'Opcao 1', template_id: 'tmpl-1' })

    await resolveAllItems()
    await waitFor(() => expect(screen.getByText('Confirm import')).not.toBeDisabled())
    fireEvent.click(screen.getByText('Confirm import'))

    await waitFor(() =>
      expect(createPlan).toHaveBeenCalledWith(
        expect.objectContaining({
          notes: 'Beber 2 litros de agua por dia\n\nSubstitutions:\n- Trocar arroz integral por 150g de batata doce',
        }),
      ),
    )
  }, 20000)
})
