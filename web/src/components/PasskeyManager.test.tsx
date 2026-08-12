import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { PasskeyManager } from './PasskeyManager'
import { DemoProvider } from '@/lib/demo'
import type { Passkey } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      auth: {
        ...actual.api.auth,
        passkeys: {
          ...actual.api.auth.passkeys,
          list: vi.fn(),
          rename: vi.fn(),
          remove: vi.fn(),
        },
      },
    },
  }
})

vi.mock('@/lib/webauthn', () => ({
  registerPasskey: vi.fn(),
  browserSupportsWebAuthn: vi.fn(() => true),
  isWebAuthnCancel: vi.fn(() => false),
}))

import { api } from '@/lib/api'
import { registerPasskey, browserSupportsWebAuthn, isWebAuthnCancel } from '@/lib/webauthn'

const list = vi.mocked(api.auth.passkeys.list)
const rename = vi.mocked(api.auth.passkeys.rename)
const remove = vi.mocked(api.auth.passkeys.remove)
const registerPasskeyMock = vi.mocked(registerPasskey)
const browserSupportsWebAuthnMock = vi.mocked(browserSupportsWebAuthn)
const isWebAuthnCancelMock = vi.mocked(isWebAuthnCancel)

function passkey(overrides: Partial<Passkey> = {}): Passkey {
  return {
    id: 'p1',
    label: 'MacBook Touch ID',
    created_at: '2024-06-01T00:00:00Z',
    last_used_at: null,
    ...overrides,
  }
}

function renderManager() {
  localStorage.setItem('dd.demo', '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <PasskeyManager />
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  list.mockReset()
  rename.mockReset()
  remove.mockReset()
  registerPasskeyMock.mockReset()
  browserSupportsWebAuthnMock.mockReset().mockReturnValue(true)
  isWebAuthnCancelMock.mockReset().mockReturnValue(false)
  localStorage.clear()
})

describe('PasskeyManager unsupported browser', () => {
  it('shows an unsupported message and no form when WebAuthn is unavailable', async () => {
    browserSupportsWebAuthnMock.mockReturnValue(false)
    renderManager()

    expect(await screen.findByText('This browser does not support passkeys.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add passkey' })).not.toBeInTheDocument()
  })
})

describe('PasskeyManager list branches', () => {
  it('shows a spinner while the passkeys query is in flight', async () => {
    list.mockReturnValue(new Promise(() => {}))
    renderManager()

    expect(await screen.findByRole('status')).toBeInTheDocument()
  })

  it('shows the empty state when there are no passkeys', async () => {
    list.mockResolvedValue([])
    renderManager()

    expect(await screen.findByText('No passkeys yet.')).toBeInTheDocument()
  })

  it('renders a row per passkey, including last-used when present', async () => {
    list.mockResolvedValue([
      passkey({ id: 'p1', label: 'MacBook Touch ID', last_used_at: '2024-07-01T00:00:00Z' }),
      passkey({ id: 'p2', label: 'YubiKey', last_used_at: null }),
    ])
    renderManager()

    expect(await screen.findByText('MacBook Touch ID')).toBeInTheDocument()
    expect(screen.getByText('YubiKey')).toBeInTheDocument()
    expect(screen.getByText(/Last used/)).toBeInTheDocument()
  })
})

describe('PasskeyManager add passkey', () => {
  it('registers a new passkey and clears the name field on success', async () => {
    list.mockResolvedValue([])
    registerPasskeyMock.mockResolvedValue(passkey())
    renderManager()
    await screen.findByText('No passkeys yet.')

    const input = screen.getByLabelText('New passkey name')
    fireEvent.change(input, { target: { value: 'My new key' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add passkey' }))

    await waitFor(() => expect(registerPasskeyMock).toHaveBeenCalledWith('My new key'))
    await waitFor(() => expect(input).toHaveValue(''))
  })

  it('does not submit when the name field is blank', async () => {
    list.mockResolvedValue([])
    renderManager()
    await screen.findByText('No passkeys yet.')

    expect(screen.getByRole('button', { name: 'Add passkey' })).toBeDisabled()
    expect(registerPasskeyMock).not.toHaveBeenCalled()
  })

  it('silently ignores a cancelled WebAuthn ceremony', async () => {
    list.mockResolvedValue([])
    isWebAuthnCancelMock.mockReturnValue(true)
    registerPasskeyMock.mockRejectedValue(new Error('cancelled'))
    renderManager()
    await screen.findByText('No passkeys yet.')

    fireEvent.change(screen.getByLabelText('New passkey name'), { target: { value: 'My new key' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add passkey' }))

    await waitFor(() => expect(registerPasskeyMock).toHaveBeenCalled())
    expect(await screen.findByRole('button', { name: 'Add passkey' })).toBeInTheDocument()
  })

  it('shows the waiting-for-device label while registration is pending', async () => {
    list.mockResolvedValue([])
    let resolveRegister!: (p: Passkey) => void
    registerPasskeyMock.mockReturnValue(new Promise((resolve) => { resolveRegister = resolve }))
    renderManager()
    await screen.findByText('No passkeys yet.')

    fireEvent.change(screen.getByLabelText('New passkey name'), { target: { value: 'My new key' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add passkey' }))

    expect(await screen.findByRole('button', { name: 'Waiting for device…' })).toBeInTheDocument()
    resolveRegister(passkey())
    expect(await screen.findByRole('button', { name: 'Add passkey' })).toBeInTheDocument()
  })
})

describe('PasskeyManager rename', () => {
  it('renames a passkey and returns to the display state', async () => {
    list.mockResolvedValue([passkey({ id: 'p1', label: 'MacBook Touch ID' })])
    rename.mockResolvedValue(passkey({ id: 'p1', label: 'Work laptop' }))
    renderManager()

    fireEvent.click(await screen.findByText('MacBook Touch ID'))

    const nameInput = screen.getByLabelText('Passkey name')
    fireEvent.change(nameInput, { target: { value: 'Work laptop' } })
    fireEvent.click(screen.getByLabelText('Save name'))

    await waitFor(() => expect(rename).toHaveBeenCalledWith('p1', 'Work laptop'))
    expect(screen.queryByLabelText('Passkey name')).not.toBeInTheDocument()
  })

  it('does not call rename when the label is unchanged', async () => {
    list.mockResolvedValue([passkey({ id: 'p1', label: 'MacBook Touch ID' })])
    renderManager()

    fireEvent.click(await screen.findByText('MacBook Touch ID'))
    fireEvent.click(screen.getByLabelText('Save name'))

    await waitFor(() => expect(screen.queryByLabelText('Passkey name')).not.toBeInTheDocument())
    expect(rename).not.toHaveBeenCalled()
  })
})

describe('PasskeyManager delete', () => {
  it('deletes a passkey on click', async () => {
    list.mockResolvedValue([passkey({ id: 'p1', label: 'MacBook Touch ID' })])
    remove.mockResolvedValue(undefined)
    renderManager()
    await screen.findByText('MacBook Touch ID')

    fireEvent.click(screen.getByLabelText('Delete MacBook Touch ID'))

    await waitFor(() => expect(remove).toHaveBeenCalledWith('p1'))
  })
})
