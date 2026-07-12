// Bridges our own session backend (POST/GET /chat/sessions, soft-delete +
// restore) into assistant-ui's RemoteThreadListAdapter so the sidebar can be
// built from ThreadListPrimitive/ThreadListItemPrimitive instead of hand-
// rolled session-list markup. "Archive"/"Unarchive" map onto our soft-delete
// (30-day retention) + restore endpoints — assistant-ui's own archived/regular
// split does double duty as our deleted/active split, so Settings' "recently
// deleted" list and the sidebar both read from the same classification.
//
// `rename` and `delete` (permanent) have no backend support and no UI trigger
// in this app (only Archive/Unarchive are wired up) — implemented as
// ponytail-minimal stubs so the adapter still type-checks.

import { createAssistantStream } from 'assistant-stream'
import type { RemoteThreadListAdapter } from '@assistant-ui/react'
import { api } from './api'
import type { ChatSession } from './types'

function toMetadata(s: ChatSession, status: 'regular' | 'archived') {
  return {
    status,
    remoteId: s.id,
    title: s.title || undefined,
    lastMessageAt: new Date(s.updated_at),
  }
}

export const chatThreadListAdapter: RemoteThreadListAdapter = {
  async list() {
    const [active, deleted] = await Promise.all([api.chat.listSessions(), api.chat.listDeletedSessions()])
    return {
      threads: [
        ...active.map((s) => toMetadata(s, 'regular')),
        ...deleted.map((s) => toMetadata(s, 'archived')),
      ],
    }
  },

  // ponytail: no backend call — session created lazily on first message
  // in handleChatMessage (AppendChatMessage → ErrNotFound → auto-create).
  async initialize() {
    return { remoteId: crypto.randomUUID(), externalId: undefined }
  },

  async archive(remoteId) {
    await api.chat.deleteSession(remoteId)
  },

  async unarchive(remoteId) {
    await api.chat.restoreSession(remoteId)
  },

  // No hard-delete affordance in the UI (only Archive/Unarchive are wired up)
  // — treat as the same soft-delete so the adapter stays correct if this is
  // ever invoked. Add real permanent-delete if a UI trigger for it shows up.
  async delete(remoteId) {
    await api.chat.deleteSession(remoteId)
  },

  // No rename endpoint exists yet and no UI offers renaming — no-op stub.
  async rename() {},

  async fetch(remoteId) {
    const [active, deleted] = await Promise.all([api.chat.listSessions(), api.chat.listDeletedSessions()])
    const found = active.find((s) => s.id === remoteId)
    if (found) return toMetadata(found, 'regular')
    const deletedFound = deleted.find((s) => s.id === remoteId)
    if (deletedFound) return toMetadata(deletedFound, 'archived')
    throw new Error('Chat session not found')
  },

  // Auto-titling needs a title-generation backend call we don't have; this
  // never emits text, so titles just stay whatever list()/fetch() reported
  // (empty today — same as before this rewrite, no regression).
  generateTitle: async () => createAssistantStream((controller) => controller.close()),
}
