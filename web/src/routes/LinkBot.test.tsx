import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { LinkBot } from './LinkBot'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      bot: {
        ...actual.api.bot,
        createLinkCode: vi.fn(),
        streamLinkCode: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const createLinkCode = vi.mocked(api.bot.createLinkCode)
const streamLinkCode = vi.mocked(api.bot.streamLinkCode)

type FakeEventSource = {
  addEventListener: ReturnType<typeof vi.fn>
  close: ReturnType<typeof vi.fn>
  onerror: (() => void) | null
  emit: (type: string) => void
}

function fakeEventSource(): FakeEventSource {
  const listeners: Record<string, Array<() => void>> = {}
  return {
    addEventListener: vi.fn((type: string, cb: () => void) => {
      ;(listeners[type] ??= []).push(cb)
    }),
    close: vi.fn(),
    onerror: null,
    emit(type: string) {
      listeners[type]?.forEach((cb) => cb())
    },
  }
}

function renderLinkBot() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <LinkBot />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  createLinkCode.mockReset()
  streamLinkCode.mockReset()
  vi.useRealTimers()
})

describe('LinkBot platform selection', () => {
  it('starts on Telegram and shows the generate-code button', () => {
    renderLinkBot()

    expect(screen.getByRole('button', { name: 'Telegram' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Generate code' })).toBeInTheDocument()
  })

  it('switches the active platform and updates the intro copy', () => {
    renderLinkBot()

    fireEvent.click(screen.getByRole('button', { name: 'Discord' }))

    expect(screen.getByRole('button', { name: 'Discord' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Telegram' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByText(/Log meals and check your day from Discord/)).toBeInTheDocument()
  })

  it('resets an in-progress code when the platform changes', async () => {
    createLinkCode.mockResolvedValue({ code: '123456' })
    streamLinkCode.mockReturnValue(fakeEventSource() as unknown as EventSource)
    renderLinkBot()

    fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))
    expect(await screen.findByText('123456')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Discord' }))

    expect(screen.queryByText('123456')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Generate code' })).toBeInTheDocument()
  })
})

describe('LinkBot code generation', () => {
  it('generates a code, opens an SSE stream, and shows the countdown panel', async () => {
    createLinkCode.mockResolvedValue({ code: '123456' })
    const es = fakeEventSource()
    streamLinkCode.mockReturnValue(es as unknown as EventSource)
    renderLinkBot()

    fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))

    expect(await screen.findByText('123456')).toBeInTheDocument()
    expect(streamLinkCode).toHaveBeenCalledWith('123456')
    expect(screen.getByText((_, el) => el?.textContent === 'expires in 10:00')).toBeInTheDocument()
  })

  it('shows an alert with the error message when code generation fails', async () => {
    createLinkCode.mockRejectedValue(new Error('server exploded'))
    // generate() doesn't try/catch its mutateAsync call, and the <Button
    // onClick={generate}> wiring never awaits/catches the returned promise
    // either -- so the rejection is genuinely unhandled at the process level
    // (same as it would be in a real browser tab). Swap out Vitest's
    // unhandledRejection listener for a no-op for the span of this test so
    // that real-but-expected rejection doesn't fail the run; the mutation's
    // error state is what actually drives the UI we're asserting on.
    const priorListeners = process.listeners('unhandledRejection')
    priorListeners.forEach((l) => process.removeListener('unhandledRejection', l as NodeJS.UnhandledRejectionListener))
    process.on('unhandledRejection', () => {})
    try {
      renderLinkBot()

      fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))

      expect(await screen.findByRole('alert')).toHaveTextContent('server exploded')
      // Give the fire-and-forget rejection a turn to be flagged while our
      // no-op listener is still the only one attached.
      await new Promise((resolve) => setTimeout(resolve, 0))
    } finally {
      process.removeAllListeners('unhandledRejection')
      priorListeners.forEach((l) => process.on('unhandledRejection', l as NodeJS.UnhandledRejectionListener))
    }
  })

  it('copies the code to the clipboard on click', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    createLinkCode.mockResolvedValue({ code: '654321' })
    streamLinkCode.mockReturnValue(fakeEventSource() as unknown as EventSource)
    renderLinkBot()

    fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))
    const codeButton = await screen.findByTitle('Click to copy')
    fireEvent.click(codeButton)

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('654321'))
  })
})

describe('LinkBot linked state', () => {
  it('shows the success panel when the SSE stream reports linked, and closes the stream', async () => {
    createLinkCode.mockResolvedValue({ code: '123456' })
    const es = fakeEventSource()
    streamLinkCode.mockReturnValue(es as unknown as EventSource)
    renderLinkBot()

    fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))
    await screen.findByText('123456')

    act(() => es.emit('linked'))

    expect(await screen.findByText('Connected!')).toBeInTheDocument()
    expect(es.close).toHaveBeenCalled()
  })

  it('lets you link another platform after a successful link', async () => {
    createLinkCode.mockResolvedValue({ code: '123456' })
    const es = fakeEventSource()
    streamLinkCode.mockReturnValue(es as unknown as EventSource)
    renderLinkBot()

    fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))
    await screen.findByText('123456')
    act(() => es.emit('linked'))
    await screen.findByText('Connected!')

    fireEvent.click(screen.getByRole('button', { name: 'Link another platform' }))

    expect(screen.getByRole('button', { name: 'Generate code' })).toBeInTheDocument()
    expect(screen.queryByText('Connected!')).not.toBeInTheDocument()
  })
})

describe('LinkBot countdown and expiry', () => {
  it('counts the code down and switches to the expired panel once it hits zero', async () => {
    createLinkCode.mockResolvedValue({ code: '123456' })
    streamLinkCode.mockReturnValue(fakeEventSource() as unknown as EventSource)

    vi.useFakeTimers()
    try {
      renderLinkBot()

      fireEvent.click(screen.getByRole('button', { name: 'Generate code' }))
      // Flush the mutateAsync microtask chain that sets `code` and mounts the
      // countdown effect -- all under the fake clock, so the interval it
      // schedules is one the fake clock actually controls.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0)
      })
      expect(screen.getByText('123456')).toBeInTheDocument()

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000)
      })
      expect(screen.getByText((_, el) => el?.textContent === 'expires in 9:59')).toBeInTheDocument()

      await act(async () => {
        await vi.advanceTimersByTimeAsync(599_000)
      })
      expect(screen.getByText('Code expired, generate a new one.')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Generate new code' })).toBeInTheDocument()
    } finally {
      vi.useRealTimers()
    }
  })
})
