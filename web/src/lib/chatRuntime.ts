// Custom assistant-ui ChatModelAdapter: fetches our own SSE endpoint and
// parses the event stream by hand (assistant-ui's useLocalRuntime is
// transport-agnostic, so this owns the entire wire format — no Vercel AI SDK
// protocol involved). Event catalog matches
// .context/prompts/backend-chat-assistant.md exactly:
//   event: delta        data: {"text": "..."}
//   event: tool-call     data: {"id": "...", "name": "...", "args": "..."}
//   event: tool-result   data: {"id": "...", "text": "..."}
//   event: suggestions  data: {"options": ["...", ...]}   (see 08b brief; backend-optional)
//   event: done          data: {}
//   event: error         data: {"message": "..."}
//
// The backend owns conversation history server-side (session-scoped), so a
// run only ever sends the latest user message, not the full transcript.

import type {
  ChatModelAdapter,
  ChatModelRunResult,
  SuggestionAdapter,
  ThreadAssistantMessagePart,
  ThreadMessage,
} from '@assistant-ui/react'
import { api } from './api'

function extractText(message: ThreadMessage): string {
  return message.content
    .map((part) => (part.type === 'text' ? part.text : ''))
    .join('')
    .trim()
}

interface TextPart {
  type: 'text'
  text: string
}

interface ToolPart {
  type: 'tool-call'
  toolCallId: string
  toolName: string
  args: Record<string, never>
  argsText: string
  result?: string
}

function parseSSEBlock(block: string): { event: string; data: string } | null {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) event = line.slice(6).trim()
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
  }
  if (dataLines.length === 0) return null
  return { event, data: dataLines.join('\n') }
}

// Strips a trailing ```suggestions ... ``` fence the model may have echoed
// inline into its reply text (the structured `suggestions` SSE event is
// authoritative; this just hides the raw fence so it isn't shown twice).
// Written with plain string scanning rather than a regex: the equivalent
// /\n```suggestions\s*\n[\s\S]*?```\s*$/ has ambiguous \s*\n / \s*$ segments
// that are super-linear to backtrack on non-matching input (S8786).
function stripSuggestionsFence(text: string): string {
  const openIdx = text.lastIndexOf('\n```suggestions')
  if (openIdx === -1) return text
  const bodyStart = text.indexOf('\n', openIdx + 1)
  if (bodyStart === -1) return text
  const closeIdx = text.indexOf('```', bodyStart)
  if (closeIdx === -1) return text
  const afterFence = text.slice(closeIdx + 3)
  if (afterFence.trim() !== '') return text // fence must be the trailing content
  return text.slice(0, openIdx)
}

interface StreamState {
  content: (TextPart | ToolPart)[]
  toolParts: Map<string, ToolPart>
  currentText: TextPart | null
}

function appendDelta(state: StreamState, data: string): void {
  const payload = JSON.parse(data) as { text: string }
  if (!state.currentText) {
    state.currentText = { type: 'text', text: '' }
    state.content.push(state.currentText)
  }
  state.currentText.text += payload.text
}

function appendToolCall(state: StreamState, data: string): void {
  const payload = JSON.parse(data) as { id: string; name: string; args: string }
  state.currentText = null // next delta (if any) starts a fresh block after this tool run
  const part: ToolPart = { type: 'tool-call', toolCallId: payload.id, toolName: payload.name, args: {}, argsText: payload.args }
  state.toolParts.set(payload.id, part)
  state.content.push(part)
}

function applyToolResult(state: StreamState, data: string): void {
  const payload = JSON.parse(data) as { id: string; text: string }
  const tc = state.toolParts.get(payload.id)
  if (tc) tc.result = payload.text
}

// Returns the new suggestion list so the caller can stash it for
// suggestionAdapter.generate(); everything else mutates `state` in place.
function applySuggestions(state: StreamState, data: string): string[] {
  const payload = JSON.parse(data) as { options: string[] }
  if (state.currentText) state.currentText.text = stripSuggestionsFence(state.currentText.text)
  return payload.options ?? []
}

function raiseStreamError(data: string): never {
  const payload = JSON.parse(data) as { message: string }
  throw new Error(payload.message)
}

// Applies one parsed SSE block to the running state, returning a fresh
// suggestion list when the event carries one (undefined otherwise).
function applyStreamEvent(state: StreamState, event: string, data: string): string[] | undefined {
  switch (event) {
    case 'delta':
      appendDelta(state, data)
      return undefined
    case 'tool-call':
      appendToolCall(state, data)
      return undefined
    case 'tool-result':
      applyToolResult(state, data)
      return undefined
    case 'suggestions':
      return applySuggestions(state, data)
    case 'error':
      raiseStreamError(data)
  }
  return undefined
}

function snapshotOf(state: StreamState): ChatModelRunResult {
  return { content: state.content.map((p) => ({ ...p })) as ThreadAssistantMessagePart[] }
}

export interface ChatAdapters {
  modelAdapter: ChatModelAdapter
  suggestionAdapter: SuggestionAdapter
}

// Bundles the chat model adapter with a SuggestionAdapter fed by the same SSE
// run: the backend (see 08b-chat-suggestions.md) may end a turn with a
// `suggestions` event carrying quick-reply options. assistant-ui calls
// suggestionAdapter.generate() itself once a run finishes, so `run` just has
// to stash the latest options where `generate` can hand them back — no event
// means no suggestions, exactly as the backend brief allows.
export function createChatAdapters(getSessionID: () => string | null): ChatAdapters {
  let latestSuggestions: string[] = []

  const modelAdapter: ChatModelAdapter = {
    async *run({ messages, abortSignal }) {
      const sessionID = getSessionID()
      if (!sessionID) throw new Error('No active chat session yet.')

      latestSuggestions = []
      const text = extractText(messages.at(-1)!)
      const res = await api.chat.sendMessage(sessionID, text, abortSignal)
      if (!res.ok || !res.body) {
        throw new Error(`Chat request failed (${res.status})`)
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      // Parts in arrival order — a tool-call run splits surrounding text into
      // separate blocks instead of merging everything into one leading blob,
      // so text that streams in after tools finish shows up below them.
      const state: StreamState = { content: [], toolParts: new Map(), currentText: null }

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const blocks = buffer.split('\n\n')
        buffer = blocks.pop() ?? ''

        for (const block of blocks) {
          const parsed = parseSSEBlock(block)
          if (!parsed) continue
          const suggestions = applyStreamEvent(state, parsed.event, parsed.data)
          if (suggestions) latestSuggestions = suggestions
          yield snapshotOf(state)
        }
      }
    },
  }

  const suggestionAdapter: SuggestionAdapter = {
    generate: async () => latestSuggestions.map((prompt) => ({ prompt })),
  }

  return { modelAdapter, suggestionAdapter }
}
