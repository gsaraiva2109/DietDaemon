import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  AssistantRuntimeProvider,
  useAui,
  useLocalRuntime,
  type ChatModelAdapter,
  type ChatModelRunResult,
  type SuggestionAdapter,
  type ToolCallMessagePartProps,
} from '@assistant-ui/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DemoProvider } from '@/lib/demo'
import { Chat, parseLogMealResult, LogMealToolCard, Suggestions } from './Chat'
import type { ChatSession, ChatMessageRecord } from '@/lib/types'

// assistant-ui's composer auto-resize watches its textarea via ResizeObserver,
// which jsdom doesn't implement.
class FakeResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
vi.stubGlobal('ResizeObserver', FakeResizeObserver)

// AnimatePresence keeps exiting elements mounted until their exit animation
// finishes, which jsdom never drives to completion — same workaround as
// CommandPalette.test.tsx. Chat.tsx (and the tool-call cards it renders via
// ToolCallChip/ToolCallGroup) only ever use motion.div/motion.p.
vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: { children: React.ReactNode }) => children,
  motion: {
    div: ({ variants, initial, animate, exit, transition, ...rest }: Record<string, unknown>) => {
      void variants
      void initial
      void animate
      void exit
      void transition
      return <div {...rest} />
    },
    p: ({ variants, initial, animate, exit, transition, ...rest }: Record<string, unknown>) => {
      void variants
      void initial
      void animate
      void exit
      void transition
      return <p {...rest} />
    },
  },
}))

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      chat: {
        ...actual.api.chat,
        listSessions: vi.fn(),
        listDeletedSessions: vi.fn(),
        getMessages: vi.fn(),
        sendMessage: vi.fn(),
        deleteSession: vi.fn(),
        restoreSession: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const listSessions = vi.mocked(api.chat.listSessions)
const listDeletedSessions = vi.mocked(api.chat.listDeletedSessions)
const getMessages = vi.mocked(api.chat.getMessages)

const SESSION: ChatSession = { id: 's1', title: 'My Chat', created_at: '', updated_at: new Date().toISOString() }

function renderChat() {
  const queryClient = new QueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <Chat />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  localStorage.removeItem('dd.demo')
  listSessions.mockReset().mockResolvedValue([SESSION])
  listDeletedSessions.mockReset().mockResolvedValue([])
  getMessages.mockReset().mockResolvedValue([])
  vi.mocked(api.chat.deleteSession).mockReset()
  vi.mocked(api.chat.restoreSession).mockReset()
  vi.mocked(api.chat.sendMessage).mockReset()
})

// --- Issue 2: S6848/S1082 — mobile rail backdrop must be a native, keyboard-
// operable control, not a bare <div onClick>. Drives the real <Chat/> route:
// this path never touches the composer/runtime send flow, so it's stable.
describe('Chat mobile rail backdrop (S6848/S1082 fix)', () => {
  it('is a native, focusable button that closes the rail on click', async () => {
    renderChat()
    fireEvent.click(await screen.findByRole('button', { name: 'History' }))

    const backdrop = await screen.findByRole('button', { name: 'Close' })
    expect(backdrop.tagName).toBe('BUTTON')
    expect(backdrop).toHaveAttribute('type', 'button')

    // Keyboard-reachable: a bare <div onClick> (the pre-fix shape) can never
    // receive focus at all. jsdom doesn't simulate the browser's native
    // Enter/Space-activates-button behavior, so focusability is the
    // observable proxy for "keyboard operable" here (same pattern used by
    // DeleteChatSessionModal.test.tsx's backdrop regression test).
    backdrop.focus()
    expect(backdrop).toHaveFocus()

    fireEvent.click(backdrop)
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Close' })).not.toBeInTheDocument())
  })
})

describe('Chat top-level states (demo mode / backend unavailable)', () => {
  it('renders the demo-mode empty state instead of a live thread list', async () => {
    localStorage.setItem('dd.demo', '1')
    renderChat()
    expect(await screen.findByText('Chat needs a real account')).toBeInTheDocument()
    // The health check query itself still fires (it's unconditional), but its
    // result is ignored in favor of the demo empty state.
    expect(screen.queryByRole('button', { name: 'History' })).not.toBeInTheDocument()
  })

  it('renders the unavailable state when the health check fails', async () => {
    listSessions.mockReset().mockRejectedValue(new Error('network down'))
    renderChat()
    // The query's own `retry: 1` delays settling into isError by one retry
    // backoff (~1s) before the unavailable screen shows.
    expect(await screen.findByText('network down', {}, { timeout: 5000 })).toBeInTheDocument()
  }, 10000)
})

describe('Chat session delete flow (SessionRow)', () => {
  it('confirms and archives the session via the delete modal', async () => {
    const deleteSession = vi.mocked(api.chat.deleteSession).mockResolvedValue(undefined)
    renderChat()

    fireEvent.click(await screen.findByRole('button', { name: 'Delete conversation' }))
    expect(await screen.findByText('Delete this conversation?')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(deleteSession).toHaveBeenCalledWith('s1'))
    expect(screen.queryByText('Delete this conversation?')).not.toBeInTheDocument()
  })

  it('cancels without archiving', async () => {
    const deleteSession = vi.mocked(api.chat.deleteSession)
    renderChat()

    fireEvent.click(await screen.findByRole('button', { name: 'Delete conversation' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Cancel' }))

    expect(screen.queryByText('Delete this conversation?')).not.toBeInTheDocument()
    expect(deleteSession).not.toHaveBeenCalled()
  })
})

// --- Issue 3: S1874 — useThreadListItemRuntime() migrated to
// useAui().threadListItem(). Exercised through the real per-thread runtime
// hook (useChatThreadRuntime) by selecting the one seeded session and
// checking its messages load correctly (proves getSessionID/remoteId
// resolution still works post-migration).
describe('Chat useAui() thread-list-item migration (S1874 fix)', () => {
  it('resolves the correct session and fetches its messages after selecting it', async () => {
    getMessages.mockResolvedValue([
      { id: 'm1', role: 'user', content: 'hi', created_at: '' },
      { id: 'm2', role: 'assistant', content: 'hello!', created_at: '' },
    ] satisfies ChatMessageRecord[])
    renderChat()

    fireEvent.click(await screen.findByRole('button', { name: 'My Chat' }))

    expect(await screen.findByText('hello!')).toBeInTheDocument()
    expect(getMessages).toHaveBeenCalledWith('s1')
  })
})

// --- Issue 1: S8786/S5843 — the LOGMEAL_HEADER_RE/LOGMEAL_NUTRIENT_RE split
// replacing the old catastrophic-backtracking-prone regex. Pure-function
// tests against parseLogMealResult (no rendering) plus a couple of rendered
// LogMealToolCard cases for the conditional macro spans.
describe('parseLogMealResult (S8786/S5843 regex fix)', () => {
  it('parses all four fields when every macro is present', () => {
    expect(parseLogMealResult('Logged: 200g grilled chicken\n300 kcal · 40g protein · 10g carbs · 5g fat')).toEqual({
      rawText: '200g grilled chicken',
      kcal: '300',
      protein: '40',
      carbs: '10',
      fat: '5',
    })
  })

  it('parses kcal-only text, leaving absent macros undefined', () => {
    expect(parseLogMealResult('Logged: banana\n90 kcal')).toEqual({
      rawText: 'banana',
      kcal: '90',
      protein: undefined,
      carbs: undefined,
      fat: undefined,
    })
  })

  it('parses a partial macro set (protein and fat, no carbs) regardless of gaps', () => {
    expect(parseLogMealResult('Logged: eggs\n150 kcal · 12g protein · 5g fat')).toEqual({
      rawText: 'eggs',
      kcal: '150',
      protein: '12',
      carbs: undefined,
      fat: '5',
    })
  })

  it('tolerates decimal values and a non-standard separator', () => {
    expect(parseLogMealResult('Logged: mystery\n120.5 kcal, 5.5g protein, 15g carbs, 2g fat')).toEqual({
      rawText: 'mystery',
      kcal: '120.5',
      protein: '5.5',
      carbs: '15',
      fat: '2',
    })
  })

  it('returns null (no match, no throw) for text that does not fit the expected shape', () => {
    expect(parseLogMealResult('unexpected backend response shape')).toBeNull()
    expect(parseLogMealResult('garbage')).toBeNull()
  })

  // Regression guard for the reported super-linear-backtracking shape: a long
  // run of non-digit characters that never resolves into a valid "<n>g X"
  // token used to make the old single regex re-try many splits of `\D+`
  // against every optional block. The new header+token-scan split has no
  // equivalent overlapping-quantifier pair, so this must resolve near-instantly.
  it('resolves quickly for a long non-matching tail (no catastrophic backtracking)', () => {
    const pathological = `Logged: bad\n100 kcal ${'x'.repeat(5000)}`
    const start = performance.now()
    const result = parseLogMealResult(pathological)
    const elapsed = performance.now() - start
    expect(result).toEqual({ rawText: 'bad', kcal: '100', protein: undefined, carbs: undefined, fat: undefined })
    expect(elapsed).toBeLessThan(200)
  })
})

// A minimal, non-nested runtime (no useRemoteThreadListRuntime/useChatThreadRuntime
// involved) so LogMealToolCard/Suggestions can be exercised directly against
// their real assistant-ui hooks (useAui, useAuiState) without the SSE/session
// plumbing the full route pulls in.
function runtimeHarness(run: ChatModelAdapter['run'], suggestionAdapter?: SuggestionAdapter) {
  function Harness({ children }: { children: React.ReactNode }) {
    const runtime = useLocalRuntime({ run }, suggestionAdapter ? { adapters: { suggestion: suggestionAdapter } } : undefined)
    return <AssistantRuntimeProvider runtime={runtime}>{children}</AssistantRuntimeProvider>
  }
  return Harness

}

// Builds a full ToolCallMessagePartProps: the data fields Chat.tsx's
// LogMealToolCard actually reads, plus no-op stand-ins for the
// action-callback fields the type requires but this fix never touches.
function toolCallProps(overrides: Partial<ToolCallMessagePartProps>): ToolCallMessagePartProps {
  return {
    type: 'tool-call',
    toolCallId: 't1',
    toolName: 'logmeal',
    args: {},
    argsText: '{}',
    status: { type: 'complete' },
    addResult: vi.fn(),
    resume: vi.fn(),
    respondToApproval: vi.fn(),
    ...overrides,
  }
}

describe('LogMealToolCard rendering (S8786/S5843 regex fix)', () => {
  async function* noopRun(): AsyncGenerator<ChatModelRunResult> {
    yield { content: [] }
  }

  function renderCard(result: string | undefined) {
    const Harness = runtimeHarness(noopRun)
    return render(
      <Harness>
        <LogMealToolCard {...toolCallProps({ result })} />
      </Harness>,
    )
  }

  it('renders the full macro breakdown when all fields are present, and "log a different amount" is clickable', () => {
    renderCard('Logged: 200g grilled chicken\n300 kcal · 40g protein · 10g carbs · 5g fat')
    expect(screen.getByText('200g grilled chicken')).toBeInTheDocument()
    expect(screen.getByText('300 kcal')).toBeInTheDocument()
    expect(screen.getByText('40g protein')).toBeInTheDocument()
    expect(screen.getByText('10g carbs')).toBeInTheDocument()
    expect(screen.getByText('5g fat')).toBeInTheDocument()

    // Just proves the handler runs without throwing; asserting on the
    // resulting composer text needs a reactive (useAuiState) reader, which
    // is a separate concern from what this fix touches.
    expect(() => fireEvent.click(screen.getByRole('button', { name: 'Log a different amount' }))).not.toThrow()
  })

  it('falls back to the generic tool chip for a non-string result (e.g. still streaming)', () => {
    const Harness = runtimeHarness(noopRun)
    render(
      <Harness>
        <LogMealToolCard {...toolCallProps({ toolCallId: 't0', result: undefined, status: { type: 'running' } })} />
      </Harness>,
    )
    expect(screen.queryByText('kcal', { exact: false })).not.toBeInTheDocument()
    expect(screen.getByText(/logmeal/i)).toBeInTheDocument()
  })

  it('renders kcal-only and omits macro spans that are absent', () => {
    renderCard('Logged: banana\n90 kcal')
    expect(screen.getByText('banana')).toBeInTheDocument()
    expect(screen.getByText('90 kcal')).toBeInTheDocument()
    expect(screen.queryByText(/protein/)).not.toBeInTheDocument()
    expect(screen.queryByText(/carbs/)).not.toBeInTheDocument()
    expect(screen.queryByText(/fat/)).not.toBeInTheDocument()
  })

  it('falls back to the generic tool chip (no macro card, no throw) for a non-matching result', () => {
    renderCard('unexpected backend response shape')
    expect(screen.queryByText('kcal', { exact: false })).not.toBeInTheDocument()
    // ToolCallChip's generic rendering shows the raw tool name somewhere.
    expect(screen.getByText(/logmeal/i)).toBeInTheDocument()
  })
})

// --- Issue 4: S6479 — Suggestions used to key its chips by array index;
// now keyed by prompt text. Driven through a real assistant-ui run + a
// SuggestionAdapter (the actual mechanism thread.suggestions is populated
// through in production), on the same minimal non-nested runtime harness.
function SendTrigger() {
  const aui = useAui()
  return (
    <button type="button" onClick={() => aui.thread().append('go')}>
      trigger send
    </button>
  )
}

describe('Suggestions list (S6479 array-index-key fix)', () => {
  it('keys chips by prompt text: renders all options, survives a reordered list, and no React key warning fires', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    let options = ['Log breakfast', 'Check macros', 'Suggest dinner']
    async function* run(): AsyncGenerator<ChatModelRunResult> {
      yield { content: [{ type: 'text', text: 'ok' }] }
    }
    const suggestionAdapter: SuggestionAdapter = {
      generate: async () => options.map((prompt) => ({ prompt })),
    }
    const Harness = runtimeHarness(run, suggestionAdapter)

    // Suggestions is populated after a run completes, driven via a real
    // thread.append() run (the actual mechanism thread.suggestions is
    // populated through in production, via SuggestionAdapter.generate()).
    render(
      <Harness>
        <SendTrigger />
        <Suggestions />
      </Harness>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'trigger send' }))

    const first = await screen.findByRole('button', { name: 'Log breakfast' })
    expect(first).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Check macros' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Suggest dinner' })).toBeInTheDocument()

    // Clicking a suggestion chip appends its prompt as the next user message.
    fireEvent.click(screen.getByRole('button', { name: 'Check macros' }))

    // Second turn: same three prompts, reversed order. With an index key this
    // would reuse stale DOM nodes positionally; with a prompt key (the fix)
    // React reorders/remounts by identity and nothing warns.
    options = ['Suggest dinner', 'Check macros', 'Log breakfast']
    fireEvent.click(screen.getByRole('button', { name: 'trigger send' }))

    await waitFor(() => {
      const buttons = screen.getAllByRole('button', { name: /Log breakfast|Check macros|Suggest dinner/ })
      expect(buttons.map((b) => b.textContent)).toEqual(['Suggest dinner', 'Check macros', 'Log breakfast'])
    })

    const keyWarning = consoleError.mock.calls.some((args) =>
      args.some((a) => typeof a === 'string' && a.includes('unique "key" prop')),
    )
    expect(keyWarning).toBe(false)

    consoleError.mockRestore()
  })
})
