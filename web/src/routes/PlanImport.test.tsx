import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Plan } from './Plan'
import { DemoProvider } from '@/lib/demo'
import type { DietPlan, FoodDetail, PlanBundle, PlanDraft } from '@/lib/types'

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

// Real PDF-canvas/text-layer work is unreliable under jsdom, so the
// PDF-selection UI path is tested against canned pdfToText/pdfToImages
// results instead of exercising pdfjs-dist for real (see those modules' own
// comments; that path needs manual verification).
vi.mock('@/lib/pdfToImages', () => ({
  pdfToImages: vi.fn(),
}))
vi.mock('@/lib/pdfToText', () => ({
  pdfToText: vi.fn(),
}))

import { api, ApiError } from '@/lib/api'
import { pdfToImages } from '@/lib/pdfToImages'
import { pdfToText } from '@/lib/pdfToText'
import type { PdfTextResult } from '@/lib/pdfToText'

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
const pdfToImagesMock = vi.mocked(pdfToImages)
const pdfToTextMock = vi.mocked(pdfToText)

const TARGETS = { Calories: 1800, Protein: 140, Carbs: 150, Fat: 55, Fiber: 20 }

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

// One day-type, one slot, one option, one item — the smallest draft that
// still exercises every level of the review tree.
function draft(overrides: Partial<PlanDraft> = {}): PlanDraft {
  return {
    plan_name: 'Imported plan',
    unreadable: false,
    notes: null,
    day_types: [
      {
        name: 'High carb',
        targets: TARGETS,
        water_goal_ml: 2000,
        low_confidence_fields: [],
        slots: [
          {
            label: 'Breakfast',
            time_of_day: '07:00',
            options: [
              {
                label: 'Option 1',
                low_confidence_fields: [],
                items: [{ raw_name: 'Peito de frango grelhado', quantity: 150, unit: null, ad_libitum: false }],
              },
            ],
          },
        ],
      },
    ],
    ...overrides,
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

beforeEach(() => {
  list.mockReset().mockResolvedValue([])
  active.mockReset().mockRejectedValue(new ApiError(404, 'not found'))
  getBundle.mockReset().mockResolvedValue(bundle(plan()))
  createPlan.mockReset().mockResolvedValue(plan())
  extractFromText.mockReset()
  extractFromImage.mockReset()
  pdfToImagesMock.mockReset()
  pdfToTextMock.mockReset()
  createDayType.mockReset()
  createSlot.mockReset()
  createOption.mockReset()
  searchCatalog.mockReset().mockResolvedValue([])
})

async function openPasteForm() {
  renderPlan()
  fireEvent.click(await screen.findByRole('button', { name: 'Import from text' }))
}

async function extractAndReachReview(d: PlanDraft) {
  extractFromText.mockResolvedValue(d)
  await openPasteForm()
  fireEvent.change(screen.getByLabelText('Paste the plan text here…'), { target: { value: 'some pasted plan text' } })
  fireEvent.click(screen.getByText('Extract'))
  await screen.findByText('Review the extracted plan')
}

describe('Plan import from text', () => {
  it('renders the review screen with unresolved items and a disabled confirm button', async () => {
    await extractAndReachReview(draft())

    expect(screen.getByText('High carb')).toBeInTheDocument()
    expect(screen.getByText('Breakfast')).toBeInTheDocument()
    expect(screen.getByText('Peito de frango grelhado')).toBeInTheDocument()
    expect(screen.getByText('Confirm import')).toBeDisabled()
  })

  it('enables the confirm button once every item is resolved', async () => {
    searchCatalog.mockResolvedValue([foodDetail()])
    await extractAndReachReview(draft())

    fireEvent.change(screen.getByLabelText('Search the food catalog for Peito de frango grelhado'), {
      target: { value: 'frango' },
    })
    fireEvent.click(await screen.findByText('Chicken breast'))

    await waitFor(() => expect(screen.getByText('Confirm import')).not.toBeDisabled())
  })

  it('confirms by firing the create sequence in order and lands on the builder', async () => {
    searchCatalog.mockResolvedValue([foodDetail()])
    createPlan.mockResolvedValue(plan({ id: 'new-plan-1' }))
    createDayType.mockResolvedValue({ id: 'dt-1', plan_id: 'new-plan-1', name: 'High carb', position: 0, targets: TARGETS, water_goal_ml: 2000 })
    createSlot.mockResolvedValue({ id: 'slot-1', day_type_id: 'dt-1', position: 0, time_of_day: '07:00', label: 'Breakfast' })
    createOption.mockResolvedValue({ id: 'opt-1', slot_id: 'slot-1', position: 0, label: 'Option 1', template_id: 'tmpl-1' })
    getBundle.mockResolvedValue(bundle(plan({ id: 'new-plan-1', name: 'Imported plan' })))
    await extractAndReachReview(draft())

    fireEvent.change(screen.getByLabelText('Search the food catalog for Peito de frango grelhado'), {
      target: { value: 'frango' },
    })
    fireEvent.click(await screen.findByText('Chicken breast'))
    await waitFor(() => expect(screen.getByText('Confirm import')).not.toBeDisabled())

    fireEvent.click(screen.getByText('Confirm import'))

    await waitFor(() =>
      expect(createPlan).toHaveBeenCalledWith(
        expect.objectContaining({ name: 'Imported plan', cycle_anchor_date: expect.any(String) }),
      ),
    )
    await waitFor(() =>
      expect(createDayType).toHaveBeenCalledWith(
        'new-plan-1',
        expect.objectContaining({ name: 'High carb', position: 0, water_goal_ml: 2000 }),
      ),
    )
    await waitFor(() =>
      expect(createSlot).toHaveBeenCalledWith('new-plan-1', 'dt-1', expect.objectContaining({ label: 'Breakfast', position: 0 })),
    )
    await waitFor(() =>
      expect(createOption).toHaveBeenCalledWith(
        'new-plan-1',
        'dt-1',
        'slot-1',
        expect.objectContaining({
          label: 'Option 1',
          items: [expect.objectContaining({ Match: expect.objectContaining({ FoodID: 'food-1' }) })],
        }),
      ),
    )

    // Ordering: plan create resolved strictly before the calls that depend on it.
    const planOrder = createPlan.mock.invocationCallOrder[0]
    const dayTypeOrder = createDayType.mock.invocationCallOrder[0]
    const slotOrder = createSlot.mock.invocationCallOrder[0]
    const optionOrder = createOption.mock.invocationCallOrder[0]
    expect(planOrder).toBeLessThan(dayTypeOrder)
    expect(dayTypeOrder).toBeLessThan(slotOrder)
    expect(slotOrder).toBeLessThan(optionOrder)

    // onCreated hands off to the normal PlanBuilder view of the new plan.
    expect(await screen.findByRole('heading', { name: 'Imported plan' })).toBeInTheDocument()
  })

  it('shows an error with a working manual-builder fallback when extraction fails', async () => {
    extractFromText.mockRejectedValue(new Error('boom'))
    await openPasteForm()
    fireEvent.change(screen.getByLabelText('Paste the plan text here…'), { target: { value: 'garbage' } })
    fireEvent.click(screen.getByText('Extract'))

    expect(await screen.findByText('boom')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Build it by hand instead'))

    // Back to the collapsed entry point, with the manual builder's own entry
    // point (NewPlanCard) visible again.
    expect(await screen.findByText('New plan')).toBeInTheDocument()
  })

  it('shows the unreadable error state instead of a garbage draft', async () => {
    await extractAndUnreadable()

    expect(screen.getByText(/couldn't be read/)).toBeInTheDocument()
    fireEvent.click(screen.getByText('Build it by hand instead'))
    expect(await screen.findByText('New plan')).toBeInTheDocument()
  })
})

async function extractAndUnreadable() {
  extractFromText.mockResolvedValue(draft({ unreadable: true }))
  await openPasteForm()
  fireEvent.change(screen.getByLabelText('Paste the plan text here…'), { target: { value: 'not a plan at all' } })
  fireEvent.click(screen.getByText('Extract'))
  await screen.findByRole('alert')
}

async function openPhotoForm() {
  renderPlan()
  fireEvent.click(await screen.findByRole('button', { name: 'Import from photo/PDF' }))
}

function pngFile(name = 'plan.png'): File {
  return new File(['fake-image-bytes'], name, { type: 'image/png' })
}

function pdfFile(name = 'plan.pdf'): File {
  return new File(['fake-pdf-bytes'], name, { type: 'application/pdf' })
}

function selectPhotoFile(file: File) {
  fireEvent.change(screen.getByLabelText('Choose a photo or PDF'), { target: { files: [file] } })
}

function oversizedFile(name = 'plan.pdf'): File {
  const file = pdfFile(name)
  Object.defineProperty(file, 'size', { value: 26 * 1024 * 1024 })
  return file
}

function textResult(overrides: Partial<PdfTextResult> = {}): PdfTextResult {
  return { text: '--- Page 1 ---\nBreakfast: 100g oats', pageCount: 1, status: 'ok', ...overrides }
}

// Reuses the same DraftReview screen the text-import path already covers in
// depth above; these mainly prove the new photo/PDF entry point reaches it,
// and that it picks the right path (native text vs. scanned images) per
// #220/#222's client-side logic.
describe('Plan import from photo/PDF', () => {
  it('uploads a photo directly, without touching pdfToText or pdfToImages', async () => {
    extractFromImage.mockResolvedValue(draft())
    await openPhotoForm()
    selectPhotoFile(pngFile())

    await screen.findByText('Review the extracted plan')
    expect(extractFromImage).toHaveBeenCalledWith([expect.any(File)])
    expect(pdfToTextMock).not.toHaveBeenCalled()
    expect(pdfToImagesMock).not.toHaveBeenCalled()
  })

  it('extracts a multi-page PDF via native text when the text layer looks usable', async () => {
    pdfToTextMock.mockResolvedValue(textResult({ pageCount: 3, text: '--- Page 1 ---\n...' }))
    extractFromText.mockResolvedValue(draft())
    await openPhotoForm()
    selectPhotoFile(pdfFile())

    await screen.findByText('Review the extracted plan')
    expect(pdfToTextMock).toHaveBeenCalledWith(expect.any(File))
    expect(extractFromText).toHaveBeenCalledWith('--- Page 1 ---\n...')
    expect(pdfToImagesMock).not.toHaveBeenCalled()
    expect(extractFromImage).not.toHaveBeenCalled()
  })

  it('auto-falls-back to the scanned (image) path when the PDF has no usable text', async () => {
    pdfToTextMock.mockResolvedValue(textResult({ status: 'empty', pageCount: 2, text: '' }))
    pdfToImagesMock.mockResolvedValue([
      new Blob(['page-1'], { type: 'image/png' }),
      new Blob(['page-2'], { type: 'image/png' }),
    ])
    extractFromImage.mockResolvedValue(draft())
    await openPhotoForm()
    selectPhotoFile(pdfFile())

    await screen.findByText('Review the extracted plan')
    expect(pdfToImagesMock).toHaveBeenCalledWith(expect.any(File))
    expect(extractFromImage).toHaveBeenCalledWith([expect.any(File), expect.any(File)])
    expect(extractFromText).not.toHaveBeenCalled()
  })

  it('shows a text preview and an explicit retry-as-scan action for malformed text, without auto-extracting', async () => {
    pdfToTextMock.mockResolvedValue(textResult({ status: 'malformed', text: 'g@rbl3d t3xt' }))
    await openPhotoForm()
    selectPhotoFile(pdfFile())

    expect(await screen.findByText(/looks garbled/)).toBeInTheDocument()
    expect(screen.getByText('g@rbl3d t3xt')).toBeInTheDocument()
    expect(extractFromText).not.toHaveBeenCalled()
    expect(extractFromImage).not.toHaveBeenCalled()

    pdfToImagesMock.mockResolvedValue([new Blob(['page-1'], { type: 'image/png' })])
    extractFromImage.mockResolvedValue(draft())
    fireEvent.click(screen.getByText('Retry as scan'))

    await screen.findByText('Review the extracted plan')
    expect(pdfToImagesMock).toHaveBeenCalledWith(expect.any(File))
    expect(extractFromImage).toHaveBeenCalledWith([expect.any(File)])
  })

  it('rejects a PDF over the page-count limit before calling extract', async () => {
    pdfToTextMock.mockResolvedValue(textResult({ pageCount: 11 }))
    await openPhotoForm()
    selectPhotoFile(pdfFile())

    expect(await screen.findByText(/This PDF has 11 pages/)).toBeInTheDocument()
    expect(extractFromText).not.toHaveBeenCalled()
    expect(extractFromImage).not.toHaveBeenCalled()
  })

  it('rejects an over-limit file by size before any network or PDF work', async () => {
    await openPhotoForm()
    selectPhotoFile(oversizedFile())

    expect(await screen.findByText(/too large/)).toBeInTheDocument()
    expect(pdfToTextMock).not.toHaveBeenCalled()
    expect(pdfToImagesMock).not.toHaveBeenCalled()
    expect(extractFromText).not.toHaveBeenCalled()
    expect(extractFromImage).not.toHaveBeenCalled()
  })

  it('shows an error with a working manual-builder fallback when image extraction fails', async () => {
    extractFromImage.mockRejectedValue(new Error('boom'))
    await openPhotoForm()
    selectPhotoFile(pngFile())

    expect(await screen.findByText('boom')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Build it by hand instead'))
    expect(await screen.findByText('New plan')).toBeInTheDocument()
  })

  it('shows an error when the PDF fails to render, without calling extract', async () => {
    pdfToTextMock.mockRejectedValue(new Error('corrupt pdf'))
    await openPhotoForm()
    selectPhotoFile(pdfFile())

    expect(await screen.findByText('corrupt pdf')).toBeInTheDocument()
    expect(extractFromText).not.toHaveBeenCalled()
    expect(extractFromImage).not.toHaveBeenCalled()
  })

  it('shows an error when the scanned fallback has no pages, without calling extract', async () => {
    pdfToTextMock.mockResolvedValue(textResult({ status: 'empty', text: '' }))
    pdfToImagesMock.mockResolvedValue([])
    await openPhotoForm()
    selectPhotoFile(pdfFile())

    expect(await screen.findByText('Could not extract a plan from that text. Please try again.')).toBeInTheDocument()
    expect(extractFromImage).not.toHaveBeenCalled()
  })
})
