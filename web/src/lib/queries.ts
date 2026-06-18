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
import { useDemo, demoToday, demoRange, DEMO_MEALS } from './demo'
import type { DailyRollup, Meal, ResolvedItem } from './types'

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

export function useCorrectItem(mealID: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ index, corrected }: { index: number; corrected: ResolvedItem }) =>
      api.correctItem(mealID, index, corrected),
    onSuccess: (updated: Meal) => {
      qc.setQueryData(keys.meal(mealID, false), updated)
      qc.invalidateQueries({ queryKey: ['rollup'] })
      qc.invalidateQueries({ queryKey: ['meals'] })
    },
  })
}
