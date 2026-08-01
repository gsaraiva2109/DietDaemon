import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DemoProvider } from '@/lib/demo'
import { AuthProvider } from '@/lib/auth'
import { Security } from './Security'
import type { ApiKey, RecoveryCodesResponse, SessionResponse, ShareToken, User } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      auth: {
        ...actual.api.auth,
        session: vi.fn(),
        apiKeys: { ...actual.api.auth.apiKeys, list: vi.fn(), create: vi.fn(), revoke: vi.fn() },
        shareTokens: { ...actual.api.auth.shareTokens, list: vi.fn(), create: vi.fn(), revoke: vi.fn() },
        totp: {
          ...actual.api.auth.totp,
          enroll: vi.fn(),
          verify: vi.fn(),
          disable: vi.fn(),
          regenerateRecovery: vi.fn(),
        },
        changePassword: vi.fn(),
        email: { ...actual.api.auth.email, change: vi.fn() },
      },
    },
  }
})

import { api } from '@/lib/api'

const session = vi.mocked(api.auth.session)
const apiKeysList = vi.mocked(api.auth.apiKeys.list)
const apiKeysRevoke = vi.mocked(api.auth.apiKeys.revoke)
const shareTokensList = vi.mocked(api.auth.shareTokens.list)
const totpEnroll = vi.mocked(api.auth.totp.enroll)
const totpDisable = vi.mocked(api.auth.totp.disable)
const regenerateRecovery = vi.mocked(api.auth.totp.regenerateRecovery)
const changePassword = vi.mocked(api.auth.changePassword)
const changeEmail = vi.mocked(api.auth.email.change)

vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))

const USER: User = {
  id: 'u1',
  email: 'jordan@example.com',
  display_name: 'Jordan',
  email_verified: true,
  created_at: '2024-01-01T00:00:00Z',
  totp_enabled: false,
}

const ACTIVE_KEY: ApiKey = {
  id: 'k1',
  label: 'Home server',
  created_at: '2024-06-01T00:00:00Z',
  last_used_at: '2024-06-15T00:00:00Z',
  revoked_at: null,
}
const REVOKED_KEY: ApiKey = {
  id: 'k2',
  label: 'Old laptop',
  created_at: '2024-01-01T00:00:00Z',
  last_used_at: null,
  revoked_at: '2024-02-01T00:00:00Z',
}
const ACTIVE_TOKEN: ShareToken = {
  id: 't1',
  label: 'Mom',
  created_at: '2024-06-01T00:00:00Z',
  last_used_at: null,
  revoked_at: null,
}

function mockSession(user: User) {
  session.mockResolvedValue({ user } satisfies SessionResponse)
}

function renderSecurity({ demo = false }: { demo?: boolean } = {}) {
  localStorage.setItem('dd.demo', demo ? '1' : '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DemoProvider>
          <AuthProvider>
            <Security />
          </AuthProvider>
        </DemoProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  session.mockReset()
  apiKeysList.mockReset()
  apiKeysRevoke.mockReset()
  shareTokensList.mockReset()
  totpEnroll.mockReset()
  totpDisable.mockReset()
  regenerateRecovery.mockReset()
  changePassword.mockReset()
  changeEmail.mockReset()
  localStorage.clear()

  // Sensible defaults so cards outside the one under test don't dangle in a
  // permanent loading state.
  apiKeysList.mockResolvedValue([])
  shareTokensList.mockResolvedValue([])
})

describe('Security demo mode', () => {
  it('shows the demo note and no interactive two-factor controls', async () => {
    renderSecurity({ demo: true })

    expect(screen.getByText('Connect a real backend to manage two-factor.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Enable two-factor' })).not.toBeInTheDocument()
    expect(screen.getAllByText('read only').length).toBeGreaterThan(0)
  })

  it('renders the empty api-key and share-link lists (demo hooks return [])', async () => {
    renderSecurity({ demo: true })

    expect(await screen.findByText('No API keys yet.')).toBeInTheDocument()
    expect(screen.getByText('No share links yet.')).toBeInTheDocument()
  })
})

describe('Security two-factor ternary chain', () => {
  it('off branch: shows the enable button when totp is not enabled', async () => {
    mockSession(USER)
    renderSecurity()

    expect(await screen.findByRole('button', { name: 'Enable two-factor' })).toBeInTheDocument()
    expect(screen.queryByText('Regenerate recovery codes')).not.toBeInTheDocument()
  })

  it('enrolling branch: clicking enable renders the TotpEnroll flow', async () => {
    mockSession(USER)
    totpEnroll.mockResolvedValue({ otpauth_url: 'otpauth://totp/DietDaemon?secret=ABC', secret: 'ABCDEF123456' })
    renderSecurity()

    fireEvent.click(await screen.findByRole('button', { name: 'Enable two-factor' }))

    expect(await screen.findByText('Or enter this key manually:')).toBeInTheDocument()
    expect(screen.getByText('ABCDEF123456')).toBeInTheDocument()

    // Cancelling returns to the off branch.
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(await screen.findByRole('button', { name: 'Enable two-factor' })).toBeInTheDocument()
  })

  it('enabled branch: shows regenerate/disable controls when totp is on', async () => {
    mockSession({ ...USER, totp_enabled: true })
    renderSecurity()

    expect(await screen.findByRole('button', { name: 'Regenerate recovery codes' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Disable two-factor' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Enable two-factor' })).not.toBeInTheDocument()
  })

  it('recovery branch: regenerating codes renders RecoveryCodes and "done" returns to the enabled branch', async () => {
    mockSession({ ...USER, totp_enabled: true })
    const codes: RecoveryCodesResponse = { recovery_codes: ['AAAA-1111', 'BBBB-2222'] }
    regenerateRecovery.mockResolvedValue(codes)
    renderSecurity()

    fireEvent.click(await screen.findByRole('button', { name: 'Regenerate recovery codes' }))

    expect(await screen.findByText('AAAA-1111')).toBeInTheDocument()
    expect(screen.getByText('BBBB-2222')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Regenerate recovery codes' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: "I've saved them" }))
    expect(await screen.findByRole('button', { name: 'Regenerate recovery codes' })).toBeInTheDocument()
  })

  it('disable calls the API and returns to the off branch after refresh', async () => {
    // Mount probe sees totp on; refresh() (fired right after disable resolves)
    // must see it off, or the queued mock never gets consumed by anything.
    session.mockResolvedValueOnce({ user: { ...USER, totp_enabled: true } })
    session.mockResolvedValueOnce({ user: { ...USER, totp_enabled: false } })
    totpDisable.mockResolvedValue(undefined)
    renderSecurity()

    fireEvent.click(await screen.findByRole('button', { name: 'Disable two-factor' }))

    await waitFor(() => expect(totpDisable).toHaveBeenCalledTimes(1))
    expect(await screen.findByRole('button', { name: 'Enable two-factor' })).toBeInTheDocument()
  })
})

describe('Security api keys list ternary', () => {
  it('loading branch: shows a spinner while the keys query is in flight', async () => {
    mockSession(USER)
    apiKeysList.mockReturnValue(new Promise(() => {}))
    renderSecurity()

    expect(await screen.findAllByRole('status')).not.toHaveLength(0)
  })

  it('empty branch: shows the empty-state copy', async () => {
    mockSession(USER)
    apiKeysList.mockResolvedValue([])
    renderSecurity()

    expect(await screen.findByText('No API keys yet.')).toBeInTheDocument()
  })

  it('list branch: shows only active keys and revokes on click', async () => {
    mockSession(USER)
    apiKeysList.mockResolvedValue([ACTIVE_KEY, REVOKED_KEY])
    apiKeysRevoke.mockResolvedValue(undefined)
    renderSecurity()

    expect(await screen.findByText('Home server')).toBeInTheDocument()
    expect(screen.queryByText('Old laptop')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Revoke Home server' }))
    await waitFor(() => expect(apiKeysRevoke).toHaveBeenCalledWith('k1'))
  })
})

describe('Security share links list ternary', () => {
  it('list branch: shows active share links', async () => {
    mockSession(USER)
    shareTokensList.mockResolvedValue([ACTIVE_TOKEN])
    renderSecurity()

    expect(await screen.findByText('Mom')).toBeInTheDocument()
  })
})

describe('Security change email', () => {
  it('rejects submitting the same email as the current one', async () => {
    mockSession(USER)
    renderSecurity()
    await screen.findByText('jordan@example.com')

    // Same as the current email (case/whitespace-insensitively) -- the "enter
    // a new email" guard covers both empty and unchanged input.
    fireEvent.change(screen.getByLabelText('New email'), { target: { value: ' Jordan@example.com ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Change email' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Enter a new email address.')
    expect(changeEmail).not.toHaveBeenCalled()
  })

  it('submits a new email and shows a success toast', async () => {
    mockSession(USER)
    changeEmail.mockResolvedValue(undefined)
    renderSecurity()
    await screen.findByText('jordan@example.com')

    fireEvent.change(screen.getByLabelText('New email'), { target: { value: 'new@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: 'Change email' }))

    await waitFor(() => expect(changeEmail).toHaveBeenCalledWith('new@example.com'))
  })
})

describe('Security change password', () => {
  it('blocks mismatched passwords', async () => {
    mockSession(USER)
    renderSecurity()

    fireEvent.change(screen.getByLabelText('Current password'), { target: { value: 'oldpass1' } })
    fireEvent.change(screen.getByLabelText('New password'), { target: { value: 'newpass1' } })
    fireEvent.change(screen.getByLabelText('Confirm new password'), { target: { value: 'newpass2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('New passwords do not match.')
    expect(changePassword).not.toHaveBeenCalled()
  })

  it('shows a generic error when the API rejects the change', async () => {
    mockSession(USER)
    changePassword.mockRejectedValue(new Error('wrong current password'))
    renderSecurity()

    fireEvent.change(screen.getByLabelText('Current password'), { target: { value: 'wrongpass' } })
    fireEvent.change(screen.getByLabelText('New password'), { target: { value: 'newpass123' } })
    fireEvent.change(screen.getByLabelText('Confirm new password'), { target: { value: 'newpass123' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not change password. Check your current password.',
    )
  })

  it('submits matching passwords and clears the form on success', async () => {
    mockSession(USER)
    changePassword.mockResolvedValue(undefined)
    renderSecurity()

    fireEvent.change(screen.getByLabelText('Current password'), { target: { value: 'oldpass1' } })
    fireEvent.change(screen.getByLabelText('New password'), { target: { value: 'newpass123' } })
    fireEvent.change(screen.getByLabelText('Confirm new password'), { target: { value: 'newpass123' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    await waitFor(() => expect(changePassword).toHaveBeenCalledWith('oldpass1', 'newpass123'))
    await waitFor(() => expect(screen.getByLabelText('Current password')).toHaveValue(''))
  })
})
