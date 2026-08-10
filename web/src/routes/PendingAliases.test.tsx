import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { PendingAliases } from './PendingAliases'
import { DemoProvider } from '@/lib/demo'
import type { PendingAlias } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      aliases: {
        ...actual.api.aliases,
        pending: {
          ...actual.api.aliases.pending,
          list: vi.fn(),
          confirm: vi.fn(),
          reject: vi.fn(),
        },
      },
    },
  }
})

import { api } from '@/lib/api'

const list = vi.mocked(api.aliases.pending.list)
const confirm = vi.mocked(api.aliases.pending.confirm)
const reject = vi.mocked(api.aliases.pending.reject)

function pendingAlias(overrides: Partial<PendingAlias> = {}): PendingAlias {
  return {
    id: 'pa1',
    user_id: 'u1',
    phrase: 'chx breast',
    food_id: 'f1',
    food_name: 'Chicken breast',
    match_score: 0.87,
    created_at: '',
    ...overrides,
  }
}

function renderPage({ demo = false }: { demo?: boolean } = {}) {
  localStorage.setItem('dd.demo', demo ? '1' : '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DemoProvider>
          <PendingAliases />
        </DemoProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  list.mockReset()
  confirm.mockReset()
  reject.mockReset()
  localStorage.clear()
})

describe('PendingAliases loading/empty branches', () => {
  it('shows a spinner while the list is in flight', async () => {
    list.mockReturnValue(new Promise(() => {}))
    renderPage()

    expect(await screen.findByRole('status')).toHaveTextContent('Loading pending aliases')
  })

  it('shows the empty state when there is nothing pending', async () => {
    list.mockResolvedValue([])
    renderPage()

    expect(await screen.findByText('Nothing waiting for review')).toBeInTheDocument()
    expect(screen.getByText('New near-miss matches will show up here for confirmation.')).toBeInTheDocument()
  })
})

describe('PendingAliases list', () => {
  it('renders each pending alias with its phrase, matched food, and match percent', async () => {
    list.mockResolvedValue([
      pendingAlias({ id: 'pa1', phrase: 'chx breast', food_name: 'Chicken breast', match_score: 0.87 }),
    ])
    renderPage()

    expect(await screen.findByText('"chx breast"')).toBeInTheDocument()
    expect(screen.getByText('matches Chicken breast')).toBeInTheDocument()
    expect(screen.getByText('87% match')).toBeInTheDocument()
  })

  it('confirms a pending alias on click', async () => {
    list.mockResolvedValue([pendingAlias({ id: 'pa1', phrase: 'chx breast' })])
    confirm.mockResolvedValue({ status: 'ok' })
    renderPage()

    fireEvent.click(await screen.findByLabelText('Confirm alias for chx breast'))

    await waitFor(() => expect(confirm).toHaveBeenCalledWith('pa1'))
  })

  it('rejects a pending alias on click', async () => {
    list.mockResolvedValue([pendingAlias({ id: 'pa1', phrase: 'chx breast' })])
    reject.mockResolvedValue(undefined)
    renderPage()

    fireEvent.click(await screen.findByLabelText('Reject alias for chx breast'))

    await waitFor(() => expect(reject).toHaveBeenCalledWith('pa1'))
  })

  it('links back to settings', async () => {
    list.mockResolvedValue([])
    renderPage()

    await screen.findByText('Nothing waiting for review')
    const backLink = screen.getByRole('link', { name: /Settings/ })
    expect(backLink).toHaveAttribute('href', '/settings')
  })
})

describe('PendingAliases demo mode', () => {
  it('shows the read-only notice and hides confirm/reject controls', async () => {
    list.mockResolvedValue([pendingAlias({ id: 'pa1', phrase: 'chx breast' })])
    renderPage({ demo: true })

    expect(await screen.findByText('Pending aliases are read only here.')).toBeInTheDocument()
    expect(screen.queryByLabelText('Confirm alias for chx breast')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Reject alias for chx breast')).not.toBeInTheDocument()
  })
})
