// Thin wrappers around @simplewebauthn/browser that pair the begin/finish API
// endpoints with the browser ceremony. begin → server options → browser prompt
// → finish → server verify. The browser half can't be mocked headlessly; the
// dev mock returns valid-shaped options and accepts any result.

import {
  startRegistration,
  startAuthentication,
  browserSupportsWebAuthn,
} from '@simplewebauthn/browser'
import { api } from './api'
import type { LoginResponse, Passkey, SessionResponse } from './types'

export { browserSupportsWebAuthn }

// Register a new passkey for the signed-in user.
export async function registerPasskey(label: string): Promise<Passkey> {
  const optionsJSON = await api.auth.passkeys.registerBegin()
  const credential = await startRegistration({ optionsJSON })
  return api.auth.passkeys.registerFinish(label, credential)
}

// Sign in with an existing passkey (discoverable credential; email optional).
// May return a session directly, or an MFA challenge when TOTP is enabled.
export async function loginWithPasskey(email?: string): Promise<LoginResponse> {
  const optionsJSON = await api.auth.passkeys.loginBegin(email)
  const credential = await startAuthentication({ optionsJSON })
  return api.auth.passkeys.loginFinish(credential)
}

// Complete MFA step-up with a passkey.
export async function mfaWithPasskey(challengeToken: string): Promise<SessionResponse> {
  const optionsJSON = await api.auth.mfa.passkeyBegin(challengeToken)
  const credential = await startAuthentication({ optionsJSON })
  return api.auth.mfa.passkeyFinish(challengeToken, credential)
}

// User aborted the native prompt (NotAllowedError), treat as a soft cancel.
export function isWebAuthnCancel(err: unknown): boolean {
  return err instanceof Error && (err.name === 'NotAllowedError' || err.name === 'AbortError')
}
