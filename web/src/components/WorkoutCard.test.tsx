import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { WorkoutCard } from './WorkoutCard'
import { DemoProvider } from '@/lib/demo'
import type { Workout } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      body: {
        ...actual.api.body,
        workouts: { ...actual.api.body.workouts, list: vi.fn(), log: vi.fn() },
      },
    },
  }
})

import { api, ApiError } from '@/lib/api'

const workoutsList = vi.mocked(api.body.workouts.list)
const workoutsLog = vi.mocked(api.body.workouts.log)

function workout(overrides: Partial<Workout> = {}): Workout {
  return {
    id: 'w1',
    user_id: 'u1',
    name: 'Upper body',
    duration_min: 45,
    intensity: 'moderate',
    logged_at: '2026-08-09T07:00:00Z',
    ...overrides,
  }
}

function renderCard(queryClient = new QueryClient()) {
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <WorkoutCard />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  workoutsList.mockReset()
  workoutsLog.mockReset()
})

describe('WorkoutCard loading state', () => {
  it('shows a spinner while the query is pending', () => {
    workoutsList.mockImplementation(() => new Promise(() => {}))
    renderCard()
    expect(screen.getByRole('status')).toBeInTheDocument()
  })
})

describe('WorkoutCard error state', () => {
  it('shows a retry button and refetches on click', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    workoutsList.mockRejectedValue(new ApiError(500, 'boom'))
    renderCard(queryClient)

    const retryBtn = await screen.findByText("Couldn't load, retry")
    workoutsList.mockClear()
    workoutsList.mockRejectedValue(new ApiError(500, 'boom again'))
    fireEvent.click(retryBtn)

    await waitFor(() => expect(workoutsList).toHaveBeenCalled())
  })
})

describe('WorkoutCard empty state', () => {
  it('shows the empty message when there are no workouts (or the backend 404s)', async () => {
    workoutsList.mockRejectedValue(new ApiError(404, 'not found'))
    renderCard()

    expect(await screen.findByText('No workouts logged this week. Log one above.')).toBeInTheDocument()
  })
})

describe('WorkoutCard with data', () => {
  it('renders each workout with its duration and a raw intensity pill', async () => {
    workoutsList.mockResolvedValue([
      workout({ id: 'w1', name: 'Upper body', duration_min: 45, intensity: 'moderate' }),
      workout({ id: 'w2', name: 'Sprint intervals', duration_min: 20, intensity: 'heavy' }),
    ])
    renderCard()

    expect(await screen.findByText('Upper body')).toBeInTheDocument()
    expect(screen.getByText('45 min')).toBeInTheDocument()
    expect(screen.getByText('Sprint intervals')).toBeInTheDocument()
    expect(screen.getByText('20 min')).toBeInTheDocument()
    expect(screen.getByText('moderate')).toBeInTheDocument()
    expect(screen.getByText('heavy')).toBeInTheDocument()
  })
})

describe('WorkoutCard log form', () => {
  it('toggles the inline log form open and closed via the header button', async () => {
    workoutsList.mockResolvedValue([])
    renderCard()
    await screen.findByText('No workouts logged this week. Log one above.')

    fireEvent.click(screen.getByRole('button', { name: 'Log' }))
    expect(screen.getByPlaceholderText('e.g. Upper body')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByPlaceholderText('e.g. Upper body')).not.toBeInTheDocument()
  })

  it('does not submit when the name is blank or minutes is zero/negative', async () => {
    workoutsList.mockResolvedValue([])
    renderCard()
    await screen.findByText('No workouts logged this week. Log one above.')
    fireEvent.click(screen.getByRole('button', { name: 'Log' }))

    // Zero minutes with a valid name.
    fireEvent.change(screen.getByPlaceholderText('e.g. Upper body'), { target: { value: 'Leg day' } })
    fireEvent.change(screen.getByPlaceholderText('min'), { target: { value: '0' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(workoutsLog).not.toHaveBeenCalled()

    // Whitespace-only name with valid minutes.
    fireEvent.change(screen.getByPlaceholderText('e.g. Upper body'), { target: { value: '   ' } })
    fireEvent.change(screen.getByPlaceholderText('min'), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(workoutsLog).not.toHaveBeenCalled()
  })

  it('logs a workout with the chosen name/minutes/intensity and resets the form on success', async () => {
    workoutsList.mockResolvedValue([])
    workoutsLog.mockResolvedValue(workout())
    renderCard()
    await screen.findByText('No workouts logged this week. Log one above.')

    fireEvent.click(screen.getByRole('button', { name: 'Log' }))
    fireEvent.change(screen.getByPlaceholderText('e.g. Upper body'), { target: { value: 'Leg day' } })
    fireEvent.change(screen.getByPlaceholderText('min'), { target: { value: '50' } })
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'heavy' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() =>
      expect(workoutsLog).toHaveBeenCalledWith({ name: 'Leg day', duration_min: 50, intensity: 'heavy' }),
    )
    // Form closes and resets after a successful save.
    await waitFor(() => expect(screen.queryByPlaceholderText('e.g. Upper body')).not.toBeInTheDocument())
  })

  it('disables the save button while the mutation is pending', async () => {
    workoutsList.mockResolvedValue([])
    workoutsLog.mockImplementation(() => new Promise(() => {}))
    renderCard()
    await screen.findByText('No workouts logged this week. Log one above.')

    fireEvent.click(screen.getByRole('button', { name: 'Log' }))
    fireEvent.change(screen.getByPlaceholderText('e.g. Upper body'), { target: { value: 'Leg day' } })
    fireEvent.change(screen.getByPlaceholderText('min'), { target: { value: '50' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled())
  })
})
