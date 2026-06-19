// TanStack Query hooks. No WebSocket exists on the backend, so "live" data is
// polled. Logging is async (202) — after it we invalidate today's rollup and
// the meal list so the dashboard reflects the new meal once the pipeline runs.
//
// Demo mode (useDemo) short-circuits every read to sample data so the UI can be
// explored with no backend. The demo flag is part of each query key, so
// toggling it refetches.

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from '@tanstack/react-query'
import { api, ApiError } from './api'
import {
  useDemo,
  demoToday,
  demoRange,
  DEMO_MEALS,
  DEMO_FOODS,
  demoFoodSearch,
  DEMO_TEMPLATES,
  DEMO_WEIGHT,
  demoWeightTrend,
  demoBodySummary,
  DEMO_MEASUREMENTS,
  DEMO_PROFILE,
} from './demo'
import type {
  DailyRollup,
  FoodDetail,
  Macros,
  Meal,
  MeasurementEntry,
  ResolvedItem,
  UserProfile,
} from './types'

const POLL_MS = 30_000

export const keys = {
  today: (demo: boolean) => ['rollup', 'today', demo] as const,
  range: (start: string, end: string, demo: boolean) => ['rollup', 'range', start, end, demo] as const,
  meals: (limit: number, demo: boolean) => ['meals', limit, demo] as const,
  meal: (id: string, demo: boolean) => ['meal', id, demo] as const,
}

// today's rollup 404s on an empty day; treat that as "no data yet", not error.
export function useToday(): UseQueryResult<DailyRollup | null> {
  const { demo } = useDemo()
  return useQuery({
    queryKey: keys.today(demo),
    queryFn: async () => {
      if (demo) return demoToday()
      try {
        return await api.rollupToday()
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null
        throw err
      }
    },
    refetchInterval: demo ? false : POLL_MS,
  })
}

export function useRange(start: string, end: string) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: keys.range(start, end, demo),
    queryFn: () => (demo ? demoRange(start, end) : api.rollupRange(start, end)),
    enabled: Boolean(start && end),
  })
}

export function useMeals(limit = 20) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: keys.meals(limit, demo),
    queryFn: () => (demo ? DEMO_MEALS.slice(0, limit) : api.meals(limit)),
    refetchInterval: demo ? false : POLL_MS,
  })
}

export function useMeal(id: string | undefined) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: keys.meal(id ?? '', demo),
    queryFn: () => (demo ? (DEMO_MEALS.find((x) => x.ID === id) ?? DEMO_MEALS[0]) : api.meal(id as string)),
    enabled: Boolean(id),
  })
}

export function useLogMeal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (text: string) => api.logMeal(text),
    onSuccess: () => {
      // Pipeline is async; give it a beat, then refresh.
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ['rollup'] })
        qc.invalidateQueries({ queryKey: ['meals'] })
      }, 1200)
    },
  })
}

function refreshMeal(qc: ReturnType<typeof useQueryClient>, mealID: string, updated: Meal) {
  qc.setQueryData(keys.meal(mealID, false), updated)
  qc.invalidateQueries({ queryKey: ['rollup'] })
  qc.invalidateQueries({ queryKey: ['meals'] })
}

export function useCorrectItem(mealID: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ index, corrected }: { index: number; corrected: ResolvedItem }) =>
      api.correctItem(mealID, index, corrected),
    onSuccess: (updated: Meal) => refreshMeal(qc, mealID, updated),
  })
}

export function useAddItem(mealID: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (item: ResolvedItem) => api.addItem(mealID, item),
    onSuccess: (updated: Meal) => refreshMeal(qc, mealID, updated),
  })
}

export function useDeleteItem(mealID: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (index: number) => api.deleteItem(mealID, index),
    onSuccess: (updated: Meal) => refreshMeal(qc, mealID, updated),
  })
}

export function useTargets() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['targets', demo],
    queryFn: async () => {
      try {
        return (await api.getTargets()).Targets
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null
        throw err
      }
    },
    enabled: !demo,
  })
}

export function useSetTargets() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (targets: Macros) => api.setTargets(targets),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['targets'] })
      qc.invalidateQueries({ queryKey: ['rollup'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Phase 2 — Food Discovery
// ---------------------------------------------------------------------------

export function useFoods(source = '') {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['foods', 'list', source, demo],
    queryFn: () =>
      demo
        ? DEMO_FOODS.filter((f: FoodDetail) => !source || f.source === source)
        : api.foods.list(source, 60),
  })
}

export function useSearchFoods(q: string) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['foods', 'search', q, demo],
    queryFn: () => (demo ? demoFoodSearch(q) : api.foods.search(q)),
    enabled: q.trim().length > 0,
  })
}

export function useFrequentFoods(limit = 12) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['foods', 'frequent', limit, demo],
    queryFn: () => (demo ? DEMO_FOODS.slice(0, limit) : api.foods.frequent(limit)),
  })
}

export function useFood(id: string | undefined) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['foods', 'detail', id ?? '', demo],
    queryFn: () =>
      demo
        ? (DEMO_FOODS.find((f: FoodDetail) => f.food_id === id) ?? DEMO_FOODS[0])
        : api.foods.get(id as string),
    enabled: Boolean(id),
  })
}

export function useAddAlias(foodID: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (alias: string) => api.foods.addAlias(foodID, alias),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['foods'] }),
  })
}

export function useDeleteAlias(foodID: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (alias: string) => api.foods.deleteAlias(foodID, alias),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['foods'] }),
  })
}

// ---------------------------------------------------------------------------
// Phase 3 — Meal Templates
// ---------------------------------------------------------------------------

export function useTemplates() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['templates', demo],
    queryFn: () => (demo ? DEMO_TEMPLATES : api.templates.list()),
  })
}

export function useCreateTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, items }: { name: string; items: ResolvedItem[] }) =>
      api.templates.create(name, items),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['templates'] }),
  })
}

export function useDeleteTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.templates.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['templates'] }),
  })
}

export function useLogTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.templates.log(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['templates'] })
      qc.invalidateQueries({ queryKey: ['rollup'] })
      qc.invalidateQueries({ queryKey: ['meals'] })
    },
  })
}

export function useDuplicateMeal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (mealID: string) => api.duplicateMeal(mealID),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['rollup'] })
      qc.invalidateQueries({ queryKey: ['meals'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Phase 4 — Body Tracking
// ---------------------------------------------------------------------------

export function useWeightLog(days = 90) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['body', 'weight', days, demo],
    queryFn: () => (demo ? DEMO_WEIGHT.slice(-days) : api.body.weight.list(days)),
  })
}

export function useWeightTrend(days = 90) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['body', 'weight', 'trend', days, demo],
    queryFn: () => (demo ? demoWeightTrend(days) : api.body.weight.trend(days)),
  })
}

export function useLogWeight() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ date, weightKg, note }: { date: string; weightKg: number; note?: string }) =>
      api.body.weight.log(date, weightKg, note ?? ''),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['body'] }),
  })
}

export function useDeleteWeight() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.body.weight.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['body'] }),
  })
}

export function useMeasurements(days = 180) {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['body', 'measurements', days, demo],
    queryFn: () => (demo ? DEMO_MEASUREMENTS : api.body.measurements.list(days)),
  })
}

export function useLogMeasurements() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (entry: Partial<MeasurementEntry>) => api.body.measurements.log(entry),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['body'] }),
  })
}

export function useDeleteMeasurement() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.body.measurements.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['body'] }),
  })
}

export function usePhotos() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['body', 'photos', demo],
    queryFn: () => (demo ? [] : api.body.photos.list()),
  })
}

export function useUploadPhoto() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ file, view, date }: { file: File; view: string; date: string }) =>
      api.body.photos.upload(file, view, date),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['body', 'photos'] }),
  })
}

export function useDeletePhoto() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.body.photos.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['body', 'photos'] }),
  })
}

export function useBodySummary() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['body', 'summary', demo],
    queryFn: () => (demo ? demoBodySummary() : api.body.summary()),
  })
}

// ---------------------------------------------------------------------------
// Phase 5 — Goals & Planning
// ---------------------------------------------------------------------------

export function useProfile() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['profile', demo],
    queryFn: () => (demo ? DEMO_PROFILE : api.profile.get()),
  })
}

export function useUpsertProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (profile: UserProfile) => api.profile.put(profile),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['profile'] })
      qc.invalidateQueries({ queryKey: ['goals'] })
    },
  })
}

// TDEE is a pure calculation endpoint; gate it on having all inputs.
export function useTDEE(params: {
  weight_kg: number
  height_cm: number
  age: number
  gender: string
  activity: string
} | null) {
  return useQuery({
    queryKey: ['tdee', params],
    queryFn: () => api.tdee(params as NonNullable<typeof params>),
    enabled: Boolean(
      params && params.weight_kg > 0 && params.height_cm > 0 && params.age > 0 && params.gender && params.activity,
    ),
  })
}

// ---------------------------------------------------------------------------
// Auth (Phase 1) — providers gating, API keys, change password.
// Session/login/register/logout state lives in the AuthProvider (auth.tsx),
// the single source of truth; these hooks cover the data-ish surfaces.
// Demo short-circuits reads so the screens render with no backend.
// ---------------------------------------------------------------------------

export function useProviders() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['auth', 'providers', demo],
    queryFn: () =>
      demo
        ? { registration_mode: 'open' as const, providers: [] }
        : api.auth.providers(),
    staleTime: 5 * 60_000,
  })
}

export function useApiKeys() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['auth', 'api-keys', demo],
    queryFn: () => (demo ? [] : api.auth.apiKeys.list()),
  })
}

export function useCreateApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (label: string) => api.auth.apiKeys.create(label),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['auth', 'api-keys'] }),
  })
}

export function useRevokeApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.auth.apiKeys.revoke(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['auth', 'api-keys'] }),
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: ({ current, next }: { current: string; next: string }) =>
      api.auth.changePassword(current, next),
  })
}

export function useGoalSuggestions() {
  const { demo } = useDemo()
  return useQuery({
    queryKey: ['goals', 'suggestions', demo],
    queryFn: () =>
      demo
        ? {
            current_intake_kcal: 2050,
            recommended_kcal: 1900,
            current_loss_kg: 0.3,
            target_loss_kg: 0.5,
            message: "You're losing 0.3 kg/week at ~2050 kcal. To hit 0.5 kg/week, try ~1900 kcal.",
          }
        : api.goalSuggestions(),
  })
}
