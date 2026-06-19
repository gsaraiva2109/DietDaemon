// Auth state for the dashboard. Sessions are server-side (HttpOnly cookie); the
// client never sees a token. We learn who we are by probing GET /auth/session:
// 200 → authed (with the user), 401 → anonymous. A single 401 interceptor
// (registered into api.ts) flips us back to anon and routes to /login whenever
// any request finds the session gone (expired/revoked). Demo mode short-circuits
// the whole thing to "authed" with no backend.

import {
  createContext,
  use,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { api, setUnauthorizedHandler } from './api'
import { useDemo } from './demo'
import { isMfaChallenge, type User } from './types'

type AuthStatus = 'checking' | 'authed' | 'anon'

// login() either signs in (1FA done) or hands back an MFA challenge token the
// caller must satisfy via verifyTotp() before a session is issued.
export type LoginResult =
  | { status: 'ok' }
  | { status: 'mfa_required'; challengeToken: string }

interface AuthValue {
  status: AuthStatus
  user: User | null
  login: (email: string, password: string, remember: boolean) => Promise<LoginResult>
  verifyTotp: (challengeToken: string, code: string, recovery?: boolean) => Promise<void>
  register: (email: string, password: string, displayName: string) => Promise<void>
  logout: () => Promise<void>
  /** Re-probe the session (e.g. after enabling/disabling a factor). */
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const { demo } = useDemo()
  const navigate = useNavigate()
  const [status, setStatus] = useState<AuthStatus>('checking')
  const [user, setUser] = useState<User | null>(null)
  // Guard against a stray 401 redirect firing repeatedly.
  const expiringRef = useRef(false)

  // Probe the session on boot (skipped entirely in demo).
  useEffect(() => {
    if (demo) {
      setStatus('authed')
      return
    }
    let alive = true
    api.auth
      .session()
      .then((res) => {
        if (!alive) return
        setUser(res.user)
        setStatus('authed')
      })
      .catch(() => {
        if (!alive) return
        setUser(null)
        setStatus('anon')
      })
    return () => {
      alive = false
    }
  }, [demo])

  // Register the single 401 interceptor. Demo bypasses it (no real backend).
  useEffect(() => {
    if (demo) {
      setUnauthorizedHandler(null)
      return
    }
    setUnauthorizedHandler(() => {
      if (expiringRef.current) return
      expiringRef.current = true
      setUser(null)
      setStatus('anon')
      toast.error('Your session expired. Please sign in again.')
      const next = window.location.pathname + window.location.search
      navigate(`/login?next=${encodeURIComponent(next)}`, { replace: true })
    })
    return () => setUnauthorizedHandler(null)
  }, [demo, navigate])

  const login = useCallback(
    async (email: string, password: string, remember: boolean): Promise<LoginResult> => {
      const res = await api.auth.login(email.trim().toLowerCase(), password, remember)
      if (isMfaChallenge(res)) {
        return { status: 'mfa_required', challengeToken: res.challenge_token }
      }
      expiringRef.current = false
      setUser(res.user)
      setStatus('authed')
      return { status: 'ok' }
    },
    [],
  )

  const verifyTotp = useCallback(
    async (challengeToken: string, code: string, recovery = false) => {
      const res = await api.auth.totp.challenge(challengeToken, code, recovery)
      expiringRef.current = false
      setUser(res.user)
      setStatus('authed')
    },
    [],
  )

  const refresh = useCallback(async () => {
    try {
      const res = await api.auth.session()
      setUser(res.user)
      setStatus('authed')
    } catch {
      setUser(null)
      setStatus('anon')
    }
  }, [])

  const register = useCallback(
    async (email: string, password: string, displayName: string) => {
      const res = await api.auth.register(email.trim().toLowerCase(), password, displayName.trim())
      expiringRef.current = false
      setUser(res.user)
      setStatus('authed')
    },
    [],
  )

  const logout = useCallback(async () => {
    try {
      await api.auth.logout()
    } catch {
      // Even if the call fails, drop local state — the cookie is gone or stale.
    }
    setUser(null)
    setStatus('anon')
  }, [])

  // Reflect demo toggles immediately.
  useEffect(() => {
    if (demo) setStatus('authed')
  }, [demo])

  const value = useMemo<AuthValue>(
    () => ({ status, user, login, verifyTotp, register, logout, refresh }),
    [status, user, login, verifyTotp, register, logout, refresh],
  )
  return <AuthContext value={value}>{children}</AuthContext>
}

export function useAuth(): AuthValue {
  const ctx = use(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
