import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DeletedChatSessions } from './DeletedChatSessions'
import type { ChatSession } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      chat: {
        ...actual.api.chat,
        listDeletedSessions: vi.fn(),
        restoreSession: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const listDeletedSessions = vi.mocked(api.chat.listDeletedSessions)
const restoreSession = vi.mocked(api.chat.restoreSession)

function session(overrides: Partial<ChatSession> = {}): ChatSession {
  return {
    id: 's1',
    title: 'Breakfast planning',
    created_at: '2024-06-01T00:00:00Z',
    updated_at: '2024-06-01T00:00:00Z',
    ...overrides,
  }
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeletedChatSessions />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  listDeletedSessions.mockReset()
  restoreSession.mockReset()
})

describe('DeletedChatSessions loading/error/empty branches', () => {
  it('shows a spinner while the deleted sessions query is in flight', async () => {
    listDeletedSessions.mockReturnValue(new Promise(() => {}))
    renderPage()

    expect(await screen.findByRole('status')).toHaveTextContent('Loading deleted conversations')
  })

  it('shows an error empty-state with the thrown message when the query fails', async () => {
    listDeletedSessions.mockRejectedValue(new Error('network down'))
    renderPage()

    expect(await screen.findByText("Couldn't load deleted conversations")).toBeInTheDocument()
    expect(screen.getByText('network down')).toBeInTheDocument()
  })

  it('shows the empty state when there are no deleted sessions', async () => {
    listDeletedSessions.mockResolvedValue([])
    renderPage()

    expect(await screen.findByText('Nothing deleted')).toBeInTheDocument()
    expect(screen.getByText('Conversations you delete from Chat show up here for 30 days.')).toBeInTheDocument()
  })
})

describe('DeletedChatSessions list', () => {
  it('renders a row per session, falling back to "New conversation" for an untitled one', async () => {
    listDeletedSessions.mockResolvedValue([session({ id: 's1', title: 'Breakfast planning' }), session({ id: 's2', title: '' })])
    renderPage()

    expect(await screen.findByText('Breakfast planning')).toBeInTheDocument()
    expect(screen.getByText('New conversation')).toBeInTheDocument()
  })

  it('restores a session on click', async () => {
    listDeletedSessions.mockResolvedValue([session({ id: 's1', title: 'Breakfast planning' })])
    restoreSession.mockResolvedValue(undefined)
    renderPage()

    fireEvent.click(await screen.findByRole('button', { name: 'Restore' }))

    await waitFor(() => expect(restoreSession).toHaveBeenCalledWith('s1'))
  })

  it('shows a restore-failed alert when the restore mutation rejects', async () => {
    listDeletedSessions.mockResolvedValue([session({ id: 's1', title: 'Breakfast planning' })])
    restoreSession.mockRejectedValue(new Error('boom'))
    renderPage()

    fireEvent.click(await screen.findByRole('button', { name: 'Restore' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('boom')
  })
})
