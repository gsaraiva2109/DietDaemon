// Custom assistant-ui ChatModelAdapter: fetches our own SSE endpoint and
// parses the event stream by hand (assistant-ui's useLocalRuntime is
// transport-agnostic, so this owns the entire wire format — no Vercel AI SDK
// protocol involved). Event catalog matches
// .context/prompts/backend-chat-assistant.md exactly:
//   event: delta        data: {"text": "..."}
//   event: tool-call     data: {"id": "...", "name": "...", "args": "..."}
//   event: tool-result   data: {"id": "...", "text": "..."}
//   event: done          data: {}
//   event: error         data: {"message": "..."}
//
// The backend owns conversation history server-side (session-scoped), so a
// run only ever sends the latest user message, not the full transcript.

import type {
  ChatModelAdapter,
  ChatModelRunResult,
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

interface ToolCallState {
  toolCallId: string
  toolName: string
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

export function createChatModelAdapter(getSessionID: () => string | null): ChatModelAdapter {
  return {
    async *run({ messages, abortSignal }) {
      const sessionID = getSessionID()
      if (!sessionID) throw new Error('No active chat session yet.')

      const text = extractText(messages[messages.length - 1])
      const res = await api.chat.sendMessage(sessionID, text, abortSignal)
      if (!res.ok || !res.body) {
        throw new Error(`Chat request failed (${res.status})`)
      }

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let assembledText = ''
      const toolCalls = new Map<string, ToolCallState>()
      const toolOrder: string[] = []

      function snapshot(): ChatModelRunResult {
        const content: ThreadAssistantMessagePart[] = []
        if (assembledText) content.push({ type: 'text', text: assembledText })
        for (const id of toolOrder) {
          const tc = toolCalls.get(id)!
          content.push({
            type: 'tool-call',
            toolCallId: tc.toolCallId,
            toolName: tc.toolName,
            args: {},
            argsText: tc.argsText,
            ...(tc.result !== undefined ? { result: tc.result } : {}),
          })
        }
        return { content }
      }

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        const blocks = buffer.split('\n\n')
        buffer = blocks.pop() ?? ''

        for (const block of blocks) {
          const parsed = parseSSEBlock(block)
          if (!parsed) continue
          const { event, data } = parsed

          if (event === 'delta') {
            const payload = JSON.parse(data) as { text: string }
            assembledText += payload.text
            yield snapshot()
          } else if (event === 'tool-call') {
            const payload = JSON.parse(data) as { id: string; name: string; args: string }
            toolCalls.set(payload.id, { toolCallId: payload.id, toolName: payload.name, argsText: payload.args })
            toolOrder.push(payload.id)
            yield snapshot()
          } else if (event === 'tool-result') {
            const payload = JSON.parse(data) as { id: string; text: string }
            const tc = toolCalls.get(payload.id)
            if (tc) tc.result = payload.text
            yield snapshot()
          } else if (event === 'error') {
            const payload = JSON.parse(data) as { message: string }
            throw new Error(payload.message)
          }
          // "done" carries no payload of interest; the stream simply ends.
        }
      }
    },
  }
}
