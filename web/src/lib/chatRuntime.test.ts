import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ChatModelRunOptions, ChatModelRunResult, ThreadMessage } from '@assistant-ui/react'
import { createChatAdapters } from './chatRuntime'

vi.mock('./api', () => ({
  api: { chat: { sendMessage: vi.fn() } },
}))

import { api } from './api'

afterEach(() => vi.resetAllMocks())

// --- fixtures --------------------------------------------------------

function userMessage(text: string): ThreadMessage {
  return {
    id: 'm1',
    role: 'user',
    content: [{ type: 'text', text }],
    createdAt: new Date(),
  } as unknown as ThreadMessage
}

function runOptions(text: string): ChatModelRunOptions {
  return {
    messages: [userMessage(text)],
    runConfig: {},
    abortSignal: new AbortController().signal,
    context: {},
    unstable_getMessage: () => userMessage(text),
  } as unknown as ChatModelRunOptions
}

// Builds an SSE body exactly matching the backend's wire format (see the
// event catalog documented at the top of chatRuntime.ts).
function sseBody(events: { event: string; data: unknown }[]): string {
  return events.map(({ event, data }) => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`).join('')
}

function streamOf(text: string): ReadableStream<Uint8Array> {
  const bytes = new TextEncoder().encode(text)
  let sent = false
  return new ReadableStream({
    pull(controller) {
      if (sent) {
        controller.close()
        return
      }
      sent = true
      controller.enqueue(bytes)
    },
  })
}

function fakeResponse(body: string, status = 200): Response {
  return { ok: status < 400, status, body: streamOf(body) } as unknown as Response
}

async function collect(gen: AsyncGenerator<ChatModelRunResult>): Promise<ChatModelRunResult[]> {
  const results: ChatModelRunResult[] = []
  for await (const r of gen) results.push(r)
  return results
}

// --- tests -----------------------------------------------------------

describe('createChatAdapters().modelAdapter.run', () => {
  it('throws when there is no active session', async () => {
    const { modelAdapter } = createChatAdapters(() => null)
    await expect(collect(modelAdapter.run(runOptions('hi')) as AsyncGenerator<ChatModelRunResult>)).rejects.toThrow(
      'No active chat session yet.',
    )
  })

  it('throws when the SSE request itself fails', async () => {
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse('', 500))
    const { modelAdapter } = createChatAdapters(() => 'session-1')
    await expect(collect(modelAdapter.run(runOptions('hi')) as AsyncGenerator<ChatModelRunResult>)).rejects.toThrow(
      'Chat request failed (500)',
    )
  })

  it('streams delta text into a single growing text part', async () => {
    const body = sseBody([
      { event: 'delta', data: { text: 'Hel' } },
      { event: 'delta', data: { text: 'lo!' } },
    ])
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse(body))
    const { modelAdapter } = createChatAdapters(() => 'session-1')

    const results = await collect(modelAdapter.run(runOptions('hi')) as AsyncGenerator<ChatModelRunResult>)

    expect(results).toHaveLength(2)
    expect(results.at(-1)).toEqual({ content: [{ type: 'text', text: 'Hello!' }] })
  })

  it('splits text around a tool call and records the tool result', async () => {
    const body = sseBody([
      { event: 'delta', data: { text: 'Let me check.' } },
      { event: 'tool-call', data: { id: 'tc1', name: 'lookupFood', args: '{"q":"egg"}' } },
      { event: 'tool-result', data: { id: 'tc1', text: 'Egg: 143 kcal/100g' } },
      { event: 'delta', data: { text: 'Eggs are 143 kcal per 100g.' } },
    ])
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse(body))
    const { modelAdapter } = createChatAdapters(() => 'session-1')

    const results = await collect(modelAdapter.run(runOptions('how many calories in eggs?')) as AsyncGenerator<ChatModelRunResult>)
    const final = results.at(-1)

    expect(final?.content).toEqual([
      { type: 'text', text: 'Let me check.' },
      {
        type: 'tool-call',
        toolCallId: 'tc1',
        toolName: 'lookupFood',
        args: {},
        argsText: '{"q":"egg"}',
        result: 'Egg: 143 kcal/100g',
      },
      { type: 'text', text: 'Eggs are 143 kcal per 100g.' },
    ])
  })

  it('throws on an error event, surfacing the backend message', async () => {
    const body = sseBody([{ event: 'error', data: { message: 'AI key not configured.' } }])
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse(body))
    const { modelAdapter } = createChatAdapters(() => 'session-1')

    await expect(collect(modelAdapter.run(runOptions('hi')) as AsyncGenerator<ChatModelRunResult>)).rejects.toThrow(
      'AI key not configured.',
    )
  })

  it('stashes suggestions for the suggestionAdapter and strips an inline fence from the text', async () => {
    const body = sseBody([
      { event: 'delta', data: { text: 'Try one of these:\n```suggestions\nMore protein\nLess sugar\n```' } },
      { event: 'suggestions', data: { options: ['More protein', 'Less sugar'] } },
    ])
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse(body))
    const { modelAdapter, suggestionAdapter } = createChatAdapters(() => 'session-1')

    const results = await collect(modelAdapter.run(runOptions('what should I eat?')) as AsyncGenerator<ChatModelRunResult>)
    const final = results.at(-1)

    expect(final?.content).toEqual([{ type: 'text', text: 'Try one of these:' }])
    await expect(suggestionAdapter.generate({ messages: [] })).resolves.toEqual([
      { prompt: 'More protein' },
      { prompt: 'Less sugar' },
    ])
  })

  it('leaves text untouched when no suggestions fence is present', async () => {
    const body = sseBody([
      { event: 'delta', data: { text: 'Just a plain reply.' } },
      { event: 'suggestions', data: { options: [] } },
    ])
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse(body))
    const { modelAdapter, suggestionAdapter } = createChatAdapters(() => 'session-1')

    const results = await collect(modelAdapter.run(runOptions('hi')) as AsyncGenerator<ChatModelRunResult>)

    expect(results.at(-1)?.content).toEqual([{ type: 'text', text: 'Just a plain reply.' }])
    await expect(suggestionAdapter.generate({ messages: [] })).resolves.toEqual([])
  })


  // Regression test for the S8786 ReDoS fix: the old
  // /\n```suggestions\s*\n[\s\S]*?```\s*$/ regex had ambiguous \s*\n and
  // \s*$ segments that a static analyzer flags as super-linear. A large,
  // never-matching "suggestions fence" body must resolve well within a test
  // timeout, not hang.
  it('handles a pathological suggestions-fence-shaped input without hanging', async () => {
    const poison = '\n```suggestions' + ' \n'.repeat(50_000)
    const body = sseBody([
      { event: 'delta', data: { text: `Normal prefix.${poison}` } },
      { event: 'suggestions', data: { options: ['ok'] } },
    ])
    vi.mocked(api.chat.sendMessage).mockResolvedValue(fakeResponse(body))
    const { modelAdapter } = createChatAdapters(() => 'session-1')

    const start = performance.now()
    const results = await collect(modelAdapter.run(runOptions('hi')) as AsyncGenerator<ChatModelRunResult>)
    const elapsedMs = performance.now() - start

    expect(elapsedMs).toBeLessThan(1000)
    // No closing fence and no trailing-only whitespace after it, so the
    // (unterminated) fence-shaped text is left untouched.
    const firstPart = results.at(-1)?.content?.[0] as { text: string } | undefined
    expect(firstPart?.text).toBe(`Normal prefix.${poison}`)
  })
})
