// Typed fetch wrapper for the DietDaemon REST API. Base path is /api/v1, which
// is same-origin in production (Go serves the SPA) and proxied in dev (see
// vite.config.ts). The Bearer token, when set, lives in localStorage.

import type {
  BodyCompositionSummary,
  DailyRollup,
  FoodDetail,
  GoalSuggestion,
  Macros,
  Meal,
  MealTemplate,
  MeasurementEntry,
  ProgressPhoto,
  ResolvedItem,
  TDEEResult,
  UserProfile,
  WeightEntry,
  WeightTrend,
} from './types'

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

  // Lightweight auth probe used by the token gate.
  ping: () => request<DailyRollup>('/rollups/today'),

  // --- Phase 2: Food Discovery -------------------------------------------
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

  // --- Phase 3: Meal Templates -------------------------------------------
  templates: {
    list: () => request<MealTemplate[]>('/templates'),
    get: (id: string) => request<MealTemplate>(`/templates/${encodeURIComponent(id)}`),
    create: (name: string, items: ResolvedItem[]) =>
      request<MealTemplate>('/templates', {
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

  // POST /meals/{id}/duplicate — clones a past meal as a fresh "today" meal.
  duplicateMeal: (mealID: string) =>
    request<{ status: string; meal_id: string }>(
      `/meals/${encodeURIComponent(mealID)}/duplicate`,
      { method: 'POST' },
    ),

  // --- Phase 4: Body Tracking --------------------------------------------
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
      // Multipart upload — the request() helper is JSON-only, so go direct.
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
  },

  // --- Phase 5: Goals & Planning -----------------------------------------
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

  // --- Phase 6: Export ---------------------------------------------------
  // Returns a Blob; callers trigger a download. format is "csv" | "json".
  export: {
    meals: (format: string, start: string, end: string) =>
      blobRequest(`/export/meals?format=${format}&start=${start}&end=${end}`),
    rollups: (format: string, start: string, end: string) =>
      blobRequest(`/export/rollups?format=${format}&start=${start}&end=${end}`),
  },
}

// multipart sends FormData without forcing a JSON Content-Type (the browser
// sets the multipart boundary itself).
async function multipart<T>(path: string, body: FormData): Promise<T> {
  const token = getToken()
  const headers = new Headers()
  headers.set('Accept', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const res = await fetch(`${BASE}${path}`, { method: 'POST', body, headers })
  if (res.status === 401) throw new UnauthorizedError()
  if (!res.ok) throw new ApiError(res.status, `Upload failed (${res.status})`)
  return (await res.json()) as T
}

// blobRequest fetches binary/file responses (photos, CSV/JSON exports).
async function blobRequest(path: string): Promise<Blob> {
  const token = getToken()
  const headers = new Headers()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const res = await fetch(`${BASE}${path}`, { headers })
  if (res.status === 401) throw new UnauthorizedError()
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
