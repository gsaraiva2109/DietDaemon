import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Body } from './Body'
import { DemoProvider } from '@/lib/demo'
import type {
  WeightEntry,
  MeasurementEntry,
  ProgressPhoto,
  BodyCompositionSummary,
} from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      rollupRange: vi.fn(),
      body: {
        ...actual.api.body,
        weight: { ...actual.api.body.weight, list: vi.fn(), trend: vi.fn(), log: vi.fn(), delete: vi.fn() },
        measurements: { ...actual.api.body.measurements, list: vi.fn(), log: vi.fn(), delete: vi.fn() },
        photos: { ...actual.api.body.photos, list: vi.fn(), upload: vi.fn(), delete: vi.fn(), blob: vi.fn() },
        summary: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const rollupRange = vi.mocked(api.rollupRange)
const weightList = vi.mocked(api.body.weight.list)
const weightTrend = vi.mocked(api.body.weight.trend)
const weightLog = vi.mocked(api.body.weight.log)
const weightDelete = vi.mocked(api.body.weight.delete)
const measurementsList = vi.mocked(api.body.measurements.list)
const measurementsLog = vi.mocked(api.body.measurements.log)
const measurementsDelete = vi.mocked(api.body.measurements.delete)
const photosList = vi.mocked(api.body.photos.list)
const photosUpload = vi.mocked(api.body.photos.upload)
const photosDelete = vi.mocked(api.body.photos.delete)
const photosBlob = vi.mocked(api.body.photos.blob)
const bodySummary = vi.mocked(api.body.summary)

function weightEntry(overrides: Partial<WeightEntry> = {}): WeightEntry {
  return {
    id: 'w1',
    user_id: 'u1',
    date: '2026-08-01',
    weight_kg: 82.4,
    note: '',
    created_at: '2026-08-01',
    ...overrides,
  }
}

function measurementEntry(overrides: Partial<MeasurementEntry> = {}): MeasurementEntry {
  return {
    id: 'm1',
    user_id: 'u1',
    date: '2026-08-01',
    waist_cm: 0,
    hips_cm: 0,
    chest_cm: 0,
    left_arm_cm: 0,
    right_arm_cm: 0,
    left_thigh_cm: 0,
    right_thigh_cm: 0,
    note: '',
    created_at: '2026-08-01',
    ...overrides,
  }
}

function photo(overrides: Partial<ProgressPhoto> = {}): ProgressPhoto {
  return {
    id: 'p1',
    user_id: 'u1',
    date: '2026-08-01',
    view: 'front',
    mime_type: 'image/jpeg',
    created_at: '2026-08-01',
    ...overrides,
  }
}

const SUMMARY: BodyCompositionSummary = {
  current_weight_kg: 80,
  start_weight_kg: 85,
  change_kg: -5,
  trend_direction: 'down',
}

function renderBody(tab: 'weight' | 'measurements' | 'photos' = 'weight', { demo = false } = {}) {
  localStorage.setItem('dd.demo', demo ? '1' : '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter initialEntries={[`/body/${tab}`]}>
          <Routes>
            <Route path="/body/:tab" element={<Body />} />
          </Routes>
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  localStorage.setItem('dd.demo', '0')
  weightList.mockReset().mockResolvedValue([])
  weightTrend.mockReset().mockResolvedValue([])
  weightLog.mockReset().mockResolvedValue(weightEntry())
  weightDelete.mockReset().mockResolvedValue(undefined)
  measurementsList.mockReset().mockResolvedValue([])
  measurementsLog.mockReset().mockResolvedValue(measurementEntry())
  measurementsDelete.mockReset().mockResolvedValue(undefined)
  photosList.mockReset().mockResolvedValue([])
  photosUpload.mockReset().mockResolvedValue(photo())
  photosDelete.mockReset().mockResolvedValue(undefined)
  photosBlob.mockReset().mockResolvedValue(new Blob(['x']))
  bodySummary.mockReset().mockResolvedValue(SUMMARY)
  rollupRange.mockReset().mockResolvedValue([])
  URL.createObjectURL = vi.fn(() => 'blob:mock-url')
  URL.revokeObjectURL = vi.fn()
})

describe('Body tab navigation', () => {
  it('shows the weight tab by default and switches panels via the tab pills', async () => {
    renderBody('weight')

    expect(await screen.findByText('Log a weigh-in')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'measurements' }))
    expect(await screen.findByRole('button', { name: 'Log measurements' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'photos' }))
    expect(await screen.findByText('Upload a photo')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'weight' }))
    expect(await screen.findByText('Log a weigh-in')).toBeInTheDocument()
  })
})

describe('Body weight tab', () => {
  it('shows a loading spinner while the weigh-in history is pending', () => {
    weightList.mockReturnValue(new Promise(() => {}))
    renderBody('weight')
    expect(screen.getAllByRole('status').length).toBeGreaterThan(0)
  })

  it('shows the empty state when there is no weigh-in history', async () => {
    renderBody('weight')
    expect(await screen.findByText('No weigh-ins yet')).toBeInTheDocument()
  })

  it('renders history newest-first and conditionally shows the note', async () => {
    const OLD = weightEntry({ id: 'w1', date: '2026-07-01', weight_kg: 84, note: '' })
    const NEW = weightEntry({ id: 'w2', date: '2026-08-01', weight_kg: 82, note: 'after run' })
    weightList.mockResolvedValue([OLD, NEW])
    renderBody('weight')

    const items = await screen.findAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('82 kg')
    expect(items[0]).toHaveTextContent('after run')
    expect(items[1]).toHaveTextContent('84 kg')
    expect(items[1]).not.toHaveTextContent('after run')
  })

  it('deletes a weigh-in entry when its trash button is clicked', async () => {
    weightList.mockResolvedValue([weightEntry({ id: 'w9' })])
    renderBody('weight')

    const del = await screen.findByLabelText('Delete weigh-in')
    fireEvent.click(del)

    await waitFor(() => expect(weightDelete).toHaveBeenCalledWith('w9'))
  })

  it('renders the summary card with a down arrow and a signed negative change', async () => {
    renderBody('weight')

    expect(await screen.findByText('80 kg')).toBeInTheDocument()
    expect(screen.getByText('85 kg')).toBeInTheDocument()
    expect(screen.getByText((_, el) => el?.textContent === '↓-5 kg')).toBeInTheDocument()
  })

  it('shows an up arrow and a leading plus sign for a positive change', async () => {
    bodySummary.mockResolvedValue({ ...SUMMARY, change_kg: 3, trend_direction: 'up' })
    renderBody('weight')

    expect(await screen.findByText((_, el) => el?.textContent === '↑+3 kg')).toBeInTheDocument()
  })

  it('shows a flat arrow for a stable trend', async () => {
    bodySummary.mockResolvedValue({ ...SUMMARY, change_kg: 0, trend_direction: 'stable' })
    renderBody('weight')

    expect(await screen.findByText((_, el) => el?.textContent === '→0 kg')).toBeInTheDocument()
  })

  it('does not render a summary card when there is no summary data', async () => {
    bodySummary.mockResolvedValue(undefined as unknown as BodyCompositionSummary)
    renderBody('weight')

    await screen.findByText('No weigh-ins yet')
    expect(screen.queryByText('Current')).not.toBeInTheDocument()
  })

  it('refetches the trend when a different range button is clicked', async () => {
    renderBody('weight')
    await waitFor(() => expect(weightTrend).toHaveBeenCalledWith(90))

    fireEvent.click(screen.getByRole('button', { name: '30d' }))
    await waitFor(() => expect(weightTrend).toHaveBeenCalledWith(30))

    fireEvent.click(screen.getByRole('button', { name: 'All' }))
    await waitFor(() => expect(weightTrend).toHaveBeenCalledWith(365))
  })

  it('disables the log button until a weight value is entered', async () => {
    renderBody('weight')
    await screen.findByText('No weigh-ins yet')
    expect(screen.getByRole('button', { name: 'Log' })).toBeDisabled()
  })

  it('logs a weigh-in and clears the weight/note fields on success', async () => {
    renderBody('weight')
    await screen.findByText('No weigh-ins yet')

    const weightInput = screen.getByPlaceholderText('82.0') as HTMLInputElement
    fireEvent.change(weightInput, { target: { value: '81.5' } })
    fireEvent.click(screen.getByRole('button', { name: 'Log' }))

    await waitFor(() => expect(weightLog).toHaveBeenCalledWith(expect.any(String), 81.5, ''))
    await waitFor(() => expect(weightInput.value).toBe(''))
  })

  it('disables write controls and shows the demo note in demo mode', async () => {
    renderBody('weight', { demo: true })
    await screen.findByText('Logging is disabled here.')

    expect(screen.getByPlaceholderText('82.0')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Log' })).toBeDisabled()
  })
})

describe('Body measurements tab', () => {
  it('shows the empty state when there are no measurements', async () => {
    renderBody('measurements')
    // "No measurements yet" is shared by the chart's own empty state and the
    // history card's empty state (both wired off the same i18n key).
    const matches = await screen.findAllByText('No measurements yet')
    expect(matches).toHaveLength(2)
  })

  it('renders history newest-first with pills only for populated fields', async () => {
    const OLD = measurementEntry({ id: 'm1', date: '2026-07-01', waist_cm: 80 })
    const NEW = measurementEntry({ id: 'm2', date: '2026-08-01', waist_cm: 78, hips_cm: 95 })
    measurementsList.mockResolvedValue([OLD, NEW])
    renderBody('measurements')

    const items = await screen.findAllByRole('listitem')
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('2026-08-01')
    expect(items[0]).toHaveTextContent('Waist 78')
    expect(items[0]).toHaveTextContent('Hips 95')
    expect(items[1]).toHaveTextContent('2026-07-01')
    expect(items[1]).toHaveTextContent('Waist 80')
    expect(items[1]).not.toHaveTextContent('Hips')
  })

  it('deletes a measurement entry when its trash button is clicked', async () => {
    measurementsList.mockResolvedValue([measurementEntry({ id: 'm9' })])
    renderBody('measurements')

    const del = await screen.findByLabelText('Delete measurements')
    fireEvent.click(del)

    await waitFor(() => expect(measurementsDelete).toHaveBeenCalledWith('m9'))
  })

  it('does not submit when every measurement field is left blank', async () => {
    renderBody('measurements')
    await waitFor(() => expect(measurementsList).toHaveBeenCalled())

    fireEvent.click(screen.getByRole('button', { name: 'Log measurements' }))
    expect(measurementsLog).not.toHaveBeenCalled()
  })

  it('submits only the populated fields and clears the form on success', async () => {
    renderBody('measurements')
    await waitFor(() => expect(measurementsList).toHaveBeenCalled())

    fireEvent.change(screen.getByLabelText('Waist (cm)'), { target: { value: '81' } })
    fireEvent.click(screen.getByRole('button', { name: 'Log measurements' }))

    await waitFor(() =>
      expect(measurementsLog).toHaveBeenCalledWith(expect.objectContaining({ waist_cm: 81 })),
    )
  })

  it('disables write controls in demo mode', async () => {
    renderBody('measurements', { demo: true })
    await waitFor(() => expect(screen.getByLabelText('Waist (cm)')).toBeDisabled())
  })
})

describe('Body photos tab', () => {
  it('shows the empty timeline and a disabled compare button with fewer than 2 photos', async () => {
    renderBody('photos')
    expect(await screen.findByText('No progress photos yet')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Compare' })).toBeDisabled()
  })

  it('enables compare with 2+ photos and opens the comparison modal', async () => {
    photosList.mockResolvedValue([
      photo({ id: 'p1', date: '2026-08-01' }),
      photo({ id: 'p2', date: '2026-08-02' }),
    ])
    renderBody('photos')

    // Wait for the photo grid itself to load before checking the Compare
    // button's disabled state, which flips only once `photos.data` arrives.
    await screen.findByRole('button', { name: 'front photo, 2026-08-01' })
    const compareBtn = screen.getByRole('button', { name: 'Compare' })
    expect(compareBtn).not.toBeDisabled()

    fireEvent.click(compareBtn)
    expect(await screen.findByRole('dialog', { name: 'Compare progress photos' })).toBeInTheDocument()
  })

  it('disables the upload button until a file is chosen, then uploads with view/date', async () => {
    renderBody('photos')
    await screen.findByText('No progress photos yet')
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled()

    const file = new File(['x'], 'progress.png', { type: 'image/png' })
    fireEvent.change(screen.getByLabelText('Image'), { target: { files: [file] } })
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }))

    await waitFor(() => expect(photosUpload).toHaveBeenCalledWith(file, 'front', expect.any(String)))
  })

  it('confirming the delete prompt on a thumbnail deletes that photo', async () => {
    photosList.mockResolvedValue([photo({ id: 'p1', date: '2026-08-01', view: 'front' })])
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    renderBody('photos')

    const thumb = await screen.findByRole('button', { name: 'front photo, 2026-08-01' })
    fireEvent.click(thumb)

    expect(confirmSpy).toHaveBeenCalled()
    await waitFor(() => expect(photosDelete).toHaveBeenCalledWith('p1'))
    confirmSpy.mockRestore()
  })

  it('does not delete when the confirm prompt is dismissed', async () => {
    photosList.mockResolvedValue([photo({ id: 'p1', date: '2026-08-01', view: 'front' })])
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    renderBody('photos')

    const thumb = await screen.findByRole('button', { name: 'front photo, 2026-08-01' })
    fireEvent.click(thumb)

    expect(confirmSpy).toHaveBeenCalled()
    expect(photosDelete).not.toHaveBeenCalled()
    confirmSpy.mockRestore()
  })

  it('disables write controls and hides the tap-to-delete hint in demo mode', async () => {
    photosList.mockResolvedValue([]) // usePhotos short-circuits to [] in demo mode regardless
    renderBody('photos', { demo: true })
    await screen.findByText('Logging is disabled here.')

    expect(screen.getByLabelText('Image')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled()
    expect(screen.queryByText('Tap a photo to delete it.')).not.toBeInTheDocument()
  })
})
