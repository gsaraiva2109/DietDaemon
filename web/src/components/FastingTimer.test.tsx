import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { FastingTimer } from './FastingTimer'

function renderTimer() {
  render(<FastingTimer />)
}

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('FastingTimer idle state', () => {
  it('shows the intro copy and a start button when no fast is active', () => {
    renderTimer()
    expect(screen.getByText(/Start a fast to track your eating window/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Start fast/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Stop & reset' })).not.toBeInTheDocument()
  })

  it('defaults to a 16h goal, marked pressed among the goal options', () => {
    renderTimer()
    expect(screen.getByRole('button', { name: '16h' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: '12h' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('selecting a different goal updates the pressed state and persists to localStorage', () => {
    renderTimer()
    fireEvent.click(screen.getByRole('button', { name: '20h' }))

    expect(screen.getByRole('button', { name: '20h' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: '16h' })).toHaveAttribute('aria-pressed', 'false')
    expect(localStorage.getItem('dd.fast.goal')).toBe('20')
  })

  it('starting a fast switches to the running view and writes the start time', () => {
    renderTimer()
    fireEvent.click(screen.getByRole('button', { name: /Start fast/ }))

    expect(localStorage.getItem('dd.fast.start')).not.toBeNull()
    expect(screen.getByRole('button', { name: 'Stop & reset' })).toBeInTheDocument()
    expect(screen.getByText('00:00:00')).toBeInTheDocument()
  })
})

describe('FastingTimer running state (pre-seeded localStorage)', () => {
  it('shows the elapsed time and hours-toward-goal pill for an in-progress fast', () => {
    vi.useFakeTimers()
    const start = new Date('2026-08-10T10:00:00.000Z')
    vi.setSystemTime(new Date('2026-08-10T12:30:00.000Z')) // 2.5h later
    localStorage.setItem('dd.fast.start', start.toISOString())
    localStorage.setItem('dd.fast.goal', '16')

    renderTimer()

    expect(screen.getByText('02:30:00')).toBeInTheDocument()
    expect(screen.getByText('2h / 16h')).toBeInTheDocument()
  })

  it('shows the goal-reached pill once elapsed time passes the goal', () => {
    vi.useFakeTimers()
    const start = new Date('2026-08-10T00:00:00.000Z')
    vi.setSystemTime(new Date('2026-08-10T17:00:00.000Z')) // 17h later, goal 16h
    localStorage.setItem('dd.fast.start', start.toISOString())
    localStorage.setItem('dd.fast.goal', '16')

    renderTimer()

    expect(screen.getByText('Goal reached')).toBeInTheDocument()
    expect(screen.queryByText(/h \/ 16h/)).not.toBeInTheDocument()
  })

  it('ticks the elapsed clock forward once per second while running', () => {
    vi.useFakeTimers()
    const start = new Date('2026-08-10T10:00:00.000Z')
    vi.setSystemTime(start)
    localStorage.setItem('dd.fast.start', start.toISOString())

    renderTimer()
    expect(screen.getByText('00:00:00')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(3000)
    })

    expect(screen.getByText('00:00:03')).toBeInTheDocument()
  })

  it('stopping the fast clears localStorage and returns to the idle view', () => {
    localStorage.setItem('dd.fast.start', new Date().toISOString())
    renderTimer()

    fireEvent.click(screen.getByRole('button', { name: 'Stop & reset' }))

    expect(localStorage.getItem('dd.fast.start')).toBeNull()
    expect(screen.getByRole('button', { name: /Start fast/ })).toBeInTheDocument()
  })
})
