// Auth state for the single-user dashboard. The backend may run with no auth
// (localhost, API_AUTH_TOKEN empty), a static token, or multi-user tokens.
// We can't know which from the client, so the gate works by probing: a 401
// means "token required / wrong token"; anything else (200, even 404 for an
// empty day) means the token — or no token — was accepted.

import {
  createContext,
  use,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, getToken, setToken, UnauthorizedError } from './api'

type AuthStatus = 'checking' | 'authed' | 'needs-token'

interface AuthValue {
  status: AuthStatus
  token: string | null
  /** Probe the API with a candidate token; persists it on success. */
  signIn: (token: string) => Promise<void>
  signOut: () => void
}

const AuthContext = createContext<AuthValue | null>(null)

async function probe(): Promise<boolean> {
  try {
    await api.ping()
    return true
  } catch (err) {
    if (err instanceof UnauthorizedError) return false
    // Network/404/500 — the request was authorized, surface those elsewhere.
    return true
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('checking')
  const [token, setTok] = useState<string | null>(getToken())

  useEffect(() => {
    let alive = true
    probe().then((ok) => {
      if (alive) setStatus(ok ? 'authed' : 'needs-token')
    })
    return () => {
      alive = false
    }
  }, [])

  const signIn = useCallback(async (candidate: string) => {
    setToken(candidate.trim())
    const ok = await probe()
    if (!ok) {
      setToken(null)
      throw new UnauthorizedError('That token was rejected.')
    }
    setTok(candidate.trim())
    setStatus('authed')
  }, [])

  const signOut = useCallback(() => {
    setToken(null)
    setTok(null)
    setStatus('needs-token')
  }, [])

  const value = useMemo<AuthValue>(
    () => ({ status, token, signIn, signOut }),
    [status, token, signIn, signOut],
  )
  return <AuthContext value={value}>{children}</AuthContext>
}

export function useAuth(): AuthValue {
  const ctx = use(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
