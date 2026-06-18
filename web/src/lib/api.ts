// Typed fetch wrapper for the DietDaemon REST API. Base path is /api/v1, which
// is same-origin in production (Go serves the SPA) and proxied in dev (see
// vite.config.ts). The Bearer token, when set, lives in localStorage.

import type { DailyRollup, Meal, ResolvedItem } from './types'

const BASE = '/api/v1'
const TOKEN_KEY = 'dd.token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}
export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
    this.name = 'ApiError'
  }
}

// Thrown on 401 so the app can bounce to the token gate.
export class UnauthorizedError extends ApiError {
  constructor(message = 'unauthorized') {
    super(401, message)
    this.name = 'UnauthorizedError'
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers = new Headers(init?.headers)
  headers.set('Accept', 'application/json')
  if (init?.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let res: Response
  try {
    res = await fetch(`${BASE}${path}`, { ...init, headers })
  } catch {
    throw new ApiError(0, 'Network error — is the DietDaemon server running?')
  }

  if (res.status === 401) throw new UnauthorizedError()

  if (!res.ok) {
    let msg = `Request failed (${res.status})`
    try {
      const body = (await res.json()) as { error?: string }
      if (body?.error) msg = body.error
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, msg)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  rollupToday: () => request<DailyRollup>('/rollups/today'),

  rollupRange: (start: string, end: string) =>
    request<DailyRollup[]>(`/rollups/range?start=${start}&end=${end}`),

  meals: (limit = 20) => request<Meal[]>(`/meals?limit=${limit}`),

  meal: (mealID: string) => request<Meal>(`/meals/${encodeURIComponent(mealID)}`),

  // Returns the updated meal. itemIndex is the zero-based position in Items.
  correctItem: (mealID: string, itemIndex: number, corrected: ResolvedItem) =>
    request<Meal>(
      `/meals/${encodeURIComponent(mealID)}/items/${itemIndex}/correct`,
      { method: 'POST', body: JSON.stringify(corrected) },
    ),

  // 202 Accepted; processing is asynchronous (poll rollup/meals afterward).
  logMeal: (text: string) =>
    request<{ status: string }>('/meals/log', {
      method: 'POST',
      body: JSON.stringify({ text }),
    }),

  // Lightweight auth probe used by the token gate.
  ping: () => request<DailyRollup>('/rollups/today'),
}
