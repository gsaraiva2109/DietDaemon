// Typed fetch wrapper for the DietDaemon REST API. Base path is /api/v1, which
// is same-origin in production (Go serves the SPA) and proxied in dev (see
// vite.config.ts). Auth is cookie-based: an HttpOnly `dd_session` cookie the
// browser sends automatically (we just set `credentials: 'include'`), paired
// with a readable `dd_csrf` cookie echoed in `X-CSRF-Token` on mutations
// (double-submit CSRF). No token lives in JS/localStorage anymore.

import type {
  AuthenticationResponseJSON,
  PublicKeyCredentialCreationOptionsJSON,
  PublicKeyCredentialRequestOptionsJSON,
  RegistrationResponseJSON,
} from '@simplewebauthn/browser'
import type {
  AIKeyStatus,
  ApiKey,
  AssistantSettings,
  BackupConfig,
  BodyCompositionSummary,
  ChatMessageRecord,
  ChatSession,
  DailyRollup,
  Fast,
  FoodDetail,
  GoalSuggestion,
  HevyImportResult,
  HevyKeyStatus,
  LinkedIdentity,
  LoginResponse,
  Macros,
  Meal,
  MealTemplate,
  MeasurementEntry,
  NewApiKey,
  NudgeRuleUpdate,
  NudgeRuleView,
  Passkey,
  PendingAlias,
  ProgressPhoto,
  ProvidersResponse,
  RecoveryCodesResponse,
  ResolvedItem,
  SessionResponse,
  SleepLog,
  SleepQuality,
  StreakResponse,
  TDEEResult,
  TotpEnrollResponse,
  UserProfile,
  WaterLog,
  WaterToday,
  WeeklyBudgetResponse,
  WeightEntry,
  WeightTrend,
  Workout,
  WorkoutIntensity,
} from './types'

export const BASE = '/api/v1'

// Generic, field-agnostic auth copy, never reveal which field was wrong.
export const AUTH_ERROR = 'Invalid email or password.'

// Read a non-HttpOnly cookie value (used for the CSRF double-submit token).
export function readCookie(name: string): string | null {
  const match = document.cookie.match(
    new RegExp('(?:^|; )' + name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1') + '=([^;]*)'),
  )
  return match ? decodeURIComponent(match[1]) : null
}

// A single 401 interceptor. AuthProvider registers a callback here so any
// request that 401s (session expired, revoked) flips the app back to anon and
// routes to /login, without each call site handling it.
type UnauthorizedHandler = () => void
let onUnauthorized: UnauthorizedHandler | null = null
export function setUnauthorizedHandler(fn: UnauthorizedHandler | null) {
  onUnauthorized = fn
}

const MUTATING = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
    this.name = 'ApiError'
  }
}

// Thrown on 401 so the app can bounce to the login screen.
export class UnauthorizedError extends ApiError {
  constructor(message = 'unauthorized') {
    super(401, message)
    this.name = 'UnauthorizedError'
  }
}

// Thrown on 429 (auth lockout). Carries the Retry-After seconds when present.
export class RateLimitError extends ApiError {
  retryAfter: number | null
  constructor(retryAfter: number | null, message = 'too many attempts') {
    super(429, message)
    this.name = 'RateLimitError'
    this.retryAfter = retryAfter
  }
}

function handleUnauthorized(suppress = false): UnauthorizedError {
  // Fire the interceptor out-of-band so the throw still propagates to callers.
  // `suppress` is set for the anonymous boot/route-guard probe, where a 401 is
  // expected and means "not signed in", not an expired session.
  if (!suppress && onUnauthorized) queueMicrotask(onUnauthorized)
  return new UnauthorizedError()
}

interface RequestOpts {
  // Skip the global 401 handler (the "session expired" toast + redirect).
  suppressUnauthorized?: boolean
}

async function request<T>(path: string, init?: RequestInit, opts?: RequestOpts): Promise<T> {
  const method = (init?.method ?? 'GET').toUpperCase()
  const headers = new Headers(init?.headers)
  headers.set('Accept', 'application/json')
  if (init?.body) headers.set('Content-Type', 'application/json')
  if (MUTATING.has(method)) {
    const csrf = readCookie('dd_csrf')
    if (csrf) headers.set('X-CSRF-Token', csrf)
  }

  let res: Response
  try {
    res = await fetch(`${BASE}${path}`, { ...init, headers, credentials: 'include' })
  } catch {
    throw new ApiError(0, 'Network error, is the DietDaemon server running?')
  }

  if (res.status === 401) throw handleUnauthorized(opts?.suppressUnauthorized)
  if (res.status === 429) {
    const ra = res.headers.get('Retry-After')
    throw new RateLimitError(ra ? Number(ra) : null)
  }

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

  // Append an item to a meal; returns the updated meal.
  addItem: (mealID: string, item: ResolvedItem) =>
    request<Meal>(`/meals/${encodeURIComponent(mealID)}/items`, {
      method: 'POST',
      body: JSON.stringify(item),
    }),

  // Remove the item at the zero-based index; returns the updated meal.
  deleteItem: (mealID: string, itemIndex: number) =>
    request<Meal>(`/meals/${encodeURIComponent(mealID)}/items/${itemIndex}`, {
      method: 'DELETE',
    }),

  getTargets: () => request<{ UserID: string; Targets: Macros }>('/targets'),

  // Body is a bare Macros object; returns the saved targets.
  setTargets: (targets: Macros) =>
    request<{ UserID: string; Targets: Macros }>('/targets', {
      method: 'PUT',
      body: JSON.stringify(targets),
    }),

  // 202 Accepted; processing is asynchronous (poll rollup/meals afterward).
  logMeal: (text: string) =>
    request<{ status: string }>('/meals/log', {
      method: 'POST',
      body: JSON.stringify({ text }),
    }),

  // --- Auth: sessions, registration, API keys ------------------
  auth: {
    // Boot probe + route guard. 401 → anonymous (UnauthorizedError). Suppresses
    // the global 401 handler: an anonymous load is normal, not an expiry.
    session: () => request<SessionResponse>('/auth/session', undefined, { suppressUnauthorized: true }),
    // May resolve to a session OR an MFA challenge.
    login: (email: string, password: string, remember: boolean) =>
      request<LoginResponse>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password, remember }),
      }),
    register: (email: string, password: string, displayName: string) =>
      request<SessionResponse>('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ email, password, display_name: displayName }),
      }),
    logout: () => request<void>('/auth/logout', { method: 'POST' }),
    // Drives login/register gating.
    providers: () => request<ProvidersResponse>('/auth/providers'),
    // --- OIDC -------------------------------------------------
    // Full-page redirect target that begins a provider sign-in (or link).
    oidcStartUrl: (id: string, link = false) =>
      `${BASE}/auth/oidc/${encodeURIComponent(id)}/start${link ? '?link=1' : ''}`,
    identities: {
      list: () => request<LinkedIdentity[]>('/auth/identities'),
      unlink: (id: string) =>
        request<void>(`/auth/identities/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
    // --- Email verification & change --------------------------
    email: {
      verify: (verifyToken: string) =>
        request<void>('/auth/email/verify', {
          method: 'POST',
          body: JSON.stringify({ token: verifyToken }),
        }),
      resendVerify: () => request<void>('/auth/email/verify/resend', { method: 'POST' }),
      change: (email: string) =>
        request<void>('/auth/email/change', {
          method: 'POST',
          body: JSON.stringify({ email }),
        }),
    },
    // --- Passwordless magic code / link -----------------------
    magic: {
      // Request a sign-in code + link by email. Always responds generically.
      request: (email: string) =>
        request<void>('/auth/magic/request', {
          method: 'POST',
          body: JSON.stringify({ email }),
        }),
      // Complete sign-in with the emailed code (scoped by email). May return
      // a session directly, or an MFA challenge when TOTP is enabled.
      verifyCode: (email: string, code: string) =>
        request<LoginResponse>('/auth/magic/verify', {
          method: 'POST',
          body: JSON.stringify({ email, code }),
        }),
      // … or the one-click link token. Same MFA-aware return type.
      verifyToken: (magicToken: string) =>
        request<LoginResponse>('/auth/magic/verify', {
          method: 'POST',
          body: JSON.stringify({ token: magicToken }),
        }),
    },
    // --- Passkeys / WebAuthn ----------------------------------
    passkeys: {
      list: () => request<Passkey[]>('/auth/passkeys'),
      registerBegin: () =>
        request<PublicKeyCredentialCreationOptionsJSON>('/auth/passkeys/register/begin', {
          method: 'POST',
        }),
      registerFinish: (label: string, credential: RegistrationResponseJSON) =>
        request<Passkey>('/auth/passkeys/register/finish', {
          method: 'POST',
          body: JSON.stringify({ label, credential }),
        }),
      loginBegin: (email?: string) =>
        request<PublicKeyCredentialRequestOptionsJSON>('/auth/passkeys/login/begin', {
          method: 'POST',
          body: JSON.stringify(email ? { email } : {}),
        }),
      loginFinish: (credential: AuthenticationResponseJSON) =>
        request<LoginResponse>('/auth/passkeys/login/finish', {
          method: 'POST',
          body: JSON.stringify({ credential }),
        }),
      rename: (id: string, label: string) =>
        request<Passkey>(`/auth/passkeys/${encodeURIComponent(id)}`, {
          method: 'PATCH',
          body: JSON.stringify({ label }),
        }),
      remove: (id: string) =>
        request<void>(`/auth/passkeys/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
    // --- Password reset. forgot() always responds generically. ---
    password: {
      forgot: (email: string) =>
        request<void>('/auth/password/forgot', {
          method: 'POST',
          body: JSON.stringify({ email }),
        }),
      reset: (resetToken: string, newPassword: string) =>
        request<void>('/auth/password/reset', {
          method: 'POST',
          body: JSON.stringify({ token: resetToken, password: newPassword }),
        }),
    },
    changePassword: (currentPassword: string, newPassword: string) =>
      request<void>('/auth/change-password', {
        method: 'POST',
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
      }),
    apiKeys: {
      list: () => request<ApiKey[]>('/auth/api-keys'),
      // Raw key returned ONCE; surface it immediately, never persist it.
      create: (label: string) =>
        request<NewApiKey>('/auth/api-keys', {
          method: 'POST',
          body: JSON.stringify({ label }),
        }),
      revoke: (id: string) =>
        request<void>(`/auth/api-keys/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
    // --- TOTP / MFA -------------------------------------------
    // --- MFA step-up: passkey + email-OTP fallback ------------
    mfa: {
      passkeyBegin: (challengeToken: string) =>
        request<PublicKeyCredentialRequestOptionsJSON>('/auth/mfa/passkey/begin', {
          method: 'POST',
          body: JSON.stringify({ challenge_token: challengeToken }),
        }),
      passkeyFinish: (challengeToken: string, credential: AuthenticationResponseJSON) =>
        request<SessionResponse>('/auth/mfa/passkey/finish', {
          method: 'POST',
          body: JSON.stringify({ challenge_token: challengeToken, credential }),
        }),
      emailSend: (challengeToken: string) =>
        request<void>('/auth/mfa/email/send', {
          method: 'POST',
          body: JSON.stringify({ challenge_token: challengeToken }),
        }),
      emailVerify: (challengeToken: string, code: string) =>
        request<SessionResponse>('/auth/mfa/email/verify', {
          method: 'POST',
          body: JSON.stringify({ challenge_token: challengeToken, code }),
        }),
    },
    totp: {
      // Begin enrollment: returns otpauth_url (QR) + base32 secret.
      enroll: () => request<TotpEnrollResponse>('/auth/totp/enroll', { method: 'POST' }),
      // Confirm enrollment with a 6-digit code; returns recovery codes once.
      verify: (code: string) =>
        request<RecoveryCodesResponse>('/auth/totp/verify', {
          method: 'POST',
          body: JSON.stringify({ code }),
        }),
      // Second step of a 2FA login. `recovery` switches code → recovery_code.
      challenge: (challengeToken: string, code: string, recovery = false) =>
        request<SessionResponse>('/auth/totp/challenge', {
          method: 'POST',
          body: JSON.stringify(
            recovery
              ? { challenge_token: challengeToken, recovery_code: code }
              : { challenge_token: challengeToken, code },
          ),
        }),
      disable: () => request<void>('/auth/totp', { method: 'DELETE' }),
      regenerateRecovery: () =>
        request<RecoveryCodesResponse>('/auth/totp/recovery-codes/regenerate', {
          method: 'POST',
        }),
    },
  },

  // --- Food Discovery -------------------------------------------
  foods: {
    list: (source = '', limit = 30, offset = 0) =>
      request<FoodDetail[]>(
        `/foods?limit=${limit}&offset=${offset}${source ? `&source=${encodeURIComponent(source)}` : ''}`,
      ),
    search: (q: string) => request<FoodDetail[]>(`/foods/search?q=${encodeURIComponent(q)}`),
    frequent: (limit = 12) => request<FoodDetail[]>(`/foods/frequent?limit=${limit}`),
    get: (foodID: string) => request<FoodDetail>(`/foods/${encodeURIComponent(foodID)}`),
    addAlias: (foodID: string, alias: string) =>
      request<{ status: string }>(`/foods/${encodeURIComponent(foodID)}/aliases`, {
        method: 'POST',
        body: JSON.stringify({ alias }),
      }),
    deleteAlias: (foodID: string, alias: string) =>
      request<void>(
        `/foods/${encodeURIComponent(foodID)}/aliases/${encodeURIComponent(alias)}`,
        { method: 'DELETE' },
      ),
  },

  // --- Pending Aliases --------------------------------------------
  aliases: {
    pending: {
      list: () => request<PendingAlias[]>('/aliases/pending'),
      confirm: (id: string) =>
        request<{ status: string }>(`/aliases/pending/${encodeURIComponent(id)}/confirm`, {
          method: 'POST',
        }),
      reject: (id: string) =>
        request<void>(`/aliases/pending/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
  },

  // --- Nutrition Source Precedence ----------------------------------
  precedence: {
    get: () => request<{ order: string[] }>('/settings/precedence'),
    set: (order: string[]) =>
      request<{ status: string }>('/settings/precedence', {
        method: 'PUT',
        body: JSON.stringify({ order }),
      }),
  },

  // --- Meal Templates -------------------------------------------
  templates: {
    list: () => request<MealTemplate[]>('/templates'),
    get: (id: string) => request<MealTemplate>(`/templates/${encodeURIComponent(id)}`),
    create: (name: string, items: ResolvedItem[]) =>
      request<MealTemplate>('/templates', {
        method: 'POST',
        body: JSON.stringify({ name, items }),
      }),
    compose: (name: string, items: { food_id: string; grams: number }[]) =>
      request<MealTemplate>('/templates/compose', {
        method: 'POST',
        body: JSON.stringify({ name, items }),
      }),
    delete: (id: string) =>
      request<void>(`/templates/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    log: (id: string) =>
      request<{ status: string; meal_id: string }>(
        `/templates/${encodeURIComponent(id)}/log`,
        { method: 'POST' },
      ),
  },

  // POST /meals/{id}/duplicate, clones a past meal as a fresh "today" meal.
  duplicateMeal: (mealID: string) =>
    request<{ status: string; meal_id: string }>(
      `/meals/${encodeURIComponent(mealID)}/duplicate`,
      { method: 'POST' },
    ),

  // --- Body Tracking --------------------------------------------
  body: {
    weight: {
      list: (days = 90) => request<WeightEntry[]>(`/body/weight?days=${days}`),
      trend: (days = 90) => request<WeightTrend[]>(`/body/weight/trend?days=${days}`),
      log: (date: string, weightKg: number, note = '') =>
        request<WeightEntry>('/body/weight', {
          method: 'POST',
          body: JSON.stringify({ date, weight_kg: weightKg, note }),
        }),
      delete: (id: string) =>
        request<void>(`/body/weight/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
    measurements: {
      list: (days = 180) => request<MeasurementEntry[]>(`/body/measurements?days=${days}`),
      log: (entry: Partial<MeasurementEntry>) =>
        request<MeasurementEntry>('/body/measurements', {
          method: 'POST',
          body: JSON.stringify(entry),
        }),
      delete: (id: string) =>
        request<void>(`/body/measurements/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
    photos: {
      list: () => request<ProgressPhoto[]>('/body/photos'),
      // Multipart upload, the request() helper is JSON-only, so go direct.
      upload: (file: File, view: string, date: string) => {
        const fd = new FormData()
        fd.append('file', file)
        fd.append('view', view)
        fd.append('date', date)
        return multipart<ProgressPhoto>('/body/photos', fd)
      },
      // The raw binary endpoint, for <img src>. Token goes in the URL-less
      // header path is impossible for <img>, so callers fetch a blob instead.
      dataURL: (id: string) => `${BASE}/body/photos/${encodeURIComponent(id)}/data`,
      blob: (id: string) => blobRequest(`/body/photos/${encodeURIComponent(id)}/data`),
      delete: (id: string) =>
        request<void>(`/body/photos/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    },
    summary: () => request<BodyCompositionSummary>('/body/summary'),
    // --- Health domains (backend pending; 404 → empty in queries) ---
    water: {
      today: () => request<WaterToday>('/body/water'),
      log: (amountMl: number, note?: string) =>
        request<WaterLog>('/body/water', {
          method: 'POST',
          body: JSON.stringify({ amount_ml: amountMl, note }),
        }),
    },
    workouts: {
      list: (limit = 5) => request<Workout[]>(`/body/workouts?limit=${limit}`),
      log: (w: { name: string; duration_min: number; intensity: WorkoutIntensity; note?: string }) =>
        request<Workout>('/body/workouts', { method: 'POST', body: JSON.stringify(w) }),
    },
    sleep: {
      list: (days = 7) => request<SleepLog[]>(`/body/sleep?days=${days}`),
      log: (s: { sleep_at: string; wake_at: string; quality: SleepQuality; note?: string }) =>
        request<SleepLog>('/body/sleep', { method: 'POST', body: JSON.stringify(s) }),
    },
  },

  // --- Fasting -------------------------
  fasting: {
    // GET active fast; 404 when none in progress (treat as "no active fast").
    active: () => request<Fast>('/fasting/active'),
    history: (limit = 10) => request<Fast[]>(`/fasting/history?limit=${limit}`),
    start: (targetHours?: number) =>
      request<Fast>('/fasting/start', {
        method: 'POST',
        body: JSON.stringify({ target_hours: targetHours ?? 16 }),
      }),
    end: () => request<Fast>('/fasting/end', { method: 'POST' }),
  },

  // --- Bot account linking --------------------------------------
  bot: {
    // Generate a one-time code to link a chat platform to this account.
    createLinkCode: (platform: string) =>
      request<{ code: string }>('/bot/link-code', {
        method: 'POST',
        body: JSON.stringify({ platform }),
      }),
    // Dashboard-side completion (the bot's /link command is the usual path).
    completeLink: (code: string) =>
      request<{ status: string }>('/bot/link', {
        method: 'POST',
        body: JSON.stringify({ code }),
      }),
    // SSE stream that pushes a "linked" event when the bot consumes the code.
    streamLinkCode: (code: string) =>
      new EventSource(`${BASE}/bot/link-code/${encodeURIComponent(code)}/stream`, {
        withCredentials: true,
      }),
  },

  // --- Goals & Planning -----------------------------------------
  profile: {
    get: () => request<UserProfile>('/profile'),
    put: (profile: UserProfile) =>
      request<UserProfile>('/profile', { method: 'PUT', body: JSON.stringify(profile) }),
  },
  tdee: (p: { weight_kg: number; height_cm: number; age: number; gender: string; activity: string }) =>
    request<TDEEResult>(
      `/tdee?weight_kg=${p.weight_kg}&height_cm=${p.height_cm}&age=${p.age}` +
        `&gender=${encodeURIComponent(p.gender)}&activity=${encodeURIComponent(p.activity)}`,
    ),
  goalSuggestions: () => request<GoalSuggestion>('/goals/suggestions'),

  // --- Nudge settings ---------------------------------------------
  nudges: {
    get: () => request<NudgeRuleView[]>('/settings/nudges'),
    // Save an enabled flag and/or param overrides for one rule.
    set: (update: NudgeRuleUpdate) =>
      request<{ status: string }>('/settings/nudges', {
        method: 'PUT',
        body: JSON.stringify(update),
      }),
    // Remove the override so the rule falls back to its hardcoded default.
    reset: (ruleID: string) =>
      request<{ status: string }>('/settings/nudges', {
        method: 'PUT',
        body: JSON.stringify({ rule_id: ruleID, enabled: true, reset: true }),
      }),
  },

  // --- Streak -----------------------------------------------------
  streak: () => request<StreakResponse>('/streak'),

  // --- Weekly rolling budget --------------------------------------
  budget: {
    weekly: () => request<WeeklyBudgetResponse>('/budget/weekly'),
  },

  // --- Export ---------------------------------------------------
  // Returns a Blob; callers trigger a download. format is "csv" | "json".
  export: {
    meals: (format: string, start: string, end: string) =>
      blobRequest(`/export/meals?format=${format}&start=${start}&end=${end}`),
    rollups: (format: string, start: string, end: string) =>
      blobRequest(`/export/rollups?format=${format}&start=${start}&end=${end}`),
  },

  // --- Scheduled backup ------------------------------------------
  backup: {
    get: () => request<BackupConfig>('/settings/backup'),
    set: (cfg: BackupConfig) =>
      request<BackupConfig>('/settings/backup', { method: 'PUT', body: JSON.stringify(cfg) }),
    runNow: () => request<{ status: string }>('/settings/backup/run', { method: 'POST' }),
  },

  // --- BYOK AI key -----------------------------------------------
  aiKey: {
    status: () => request<AIKeyStatus>('/settings/ai-key'),
    set: (body: { provider: string; key: string }) =>
      request<{ status: string }>('/settings/ai-key', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    delete: () => request<void>('/settings/ai-key', { method: 'DELETE' }),
  },

  // --- Hevy integration ------------------------------------------
  hevyKey: {
    status: () => request<HevyKeyStatus>('/settings/hevy-key'),
    set: (body: { key: string }) =>
      request<{ status: string }>('/settings/hevy-key', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
    delete: () => request<void>('/settings/hevy-key', { method: 'DELETE' }),
  },

  importHevy: () =>
    request<HevyImportResult>('/import/hevy', { method: 'POST' }),

  // --- AI chat assistant -------------------------------------------
  chat: {
    createSession: () => request<ChatSession>('/chat/sessions', { method: 'POST' }),
    listSessions: () => request<ChatSession[]>('/chat/sessions'),
    getMessages: (sessionID: string) =>
      request<ChatMessageRecord[]>(`/chat/sessions/${encodeURIComponent(sessionID)}/messages`),
    // The streaming send is a raw fetch (not the JSON `request()` wrapper):
    // the caller reads the response body as an SSE stream itself. Carries the
    // same credentials + CSRF header as every other mutation.
    sendMessage: (sessionID: string, text: string, signal?: AbortSignal) => {
      const headers = new Headers({ 'Content-Type': 'application/json', Accept: 'text/event-stream' })
      const csrf = readCookie('dd_csrf')
      if (csrf) headers.set('X-CSRF-Token', csrf)
      return fetch(`${BASE}/chat/sessions/${encodeURIComponent(sessionID)}/messages`, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: JSON.stringify({ text }),
        signal,
      })
    },
    settings: {
      get: () => request<AssistantSettings>('/chat/settings'),
      set: (body: AssistantSettings) =>
        request<AssistantSettings>('/chat/settings', { method: 'PUT', body: JSON.stringify(body) }),
    },
  },
}

// multipart sends FormData without forcing a JSON Content-Type (the browser
// sets the multipart boundary itself). POST → carry the CSRF header too.
async function multipart<T>(path: string, body: FormData): Promise<T> {
  const headers = new Headers()
  headers.set('Accept', 'application/json')
  const csrf = readCookie('dd_csrf')
  if (csrf) headers.set('X-CSRF-Token', csrf)
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    body,
    headers,
    credentials: 'include',
  })
  if (res.status === 401) throw handleUnauthorized()
  if (!res.ok) throw new ApiError(res.status, `Upload failed (${res.status})`)
  return (await res.json()) as T
}

// blobRequest fetches binary/file responses (photos, CSV/JSON exports).
async function blobRequest(path: string): Promise<Blob> {
  const res = await fetch(`${BASE}${path}`, { credentials: 'include' })
  if (res.status === 401) throw handleUnauthorized()
  if (!res.ok) throw new ApiError(res.status, `Request failed (${res.status})`)
  return await res.blob()
}

// triggerDownload saves a Blob to disk with the given filename.
export function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
