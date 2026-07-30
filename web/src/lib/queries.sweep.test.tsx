import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { DemoProvider } from '@/lib/demo'
import * as queries from './queries'

// `queries.ts` is ~90 thin useQuery/useMutation hooks wrapping api.ts calls
// (already covered by its own generic sweep in api.test.ts). Most of these
// hooks have no test anywhere: nothing renders the body-tracking, auth
// settings, backup, AI/Hevy key, or chat-settings routes yet. Hand-writing a
// renderHook + assertion per hook would mostly re-prove "useQuery calls
// queryFn" over and over, so instead: walk every exported `use*` hook, render
// it with demo mode off (so the real api.* branch executes against a mocked
// fetch), fire .mutate() for mutations, and prove it settles out of
// 'pending' — the same "reaches the network without throwing" contract as
// the api.ts sweep, one level up.
//
// Argument shapes don't matter for coverage purposes: every hook only ever
// forwards its arguments into a template literal, JSON.stringify, or a
// destructuring pattern (which tolerates extra/missing keys), so one dummy
// positional tuple and one kitchen-sink mutate() object cover every hook.

type AnyFn = (...args: unknown[]) => unknown

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return (
    <QueryClientProvider client={qc}>
      <DemoProvider>{children}</DemoProvider>
    </QueryClientProvider>
  )
}

// Generic positional args for hook construction (usePlanBundle(planID),
// useAddServingUnit(foodID), useCatalogSearch(query, source, limit), ...).
const HOOK_ARGS: unknown[] = ['dummy-id', 20, false, 'dummy2', 'dummy3']

// Kitchen-sink argument for every mutate() call, covering every property
// name any mutationFn destructures or reads across all of queries.ts.
const MUTATE_ARG = {
  index: 0,
  corrected: { Parsed: {}, Match: {}, Macros: {} },
  date: '2026-01-01',
  weightKg: 70,
  note: 'note',
  file: new File(['x'], 'x.png', { type: 'image/png' }),
  view: 'front',
  amountMl: 250,
  name: 'Run',
  duration_min: 30,
  intensity: 'moderate',
  sleep_at: '22:00',
  wake_at: '06:00',
  quality: 'good',
  current: 'old-pass',
  next: 'new-pass',
  token: 'tok',
  password: 'new-pass',
  email: 'e@x.com',
  code: '123456',
  id: 'id-1',
  label: 'label-1',
  items: [],
  planSlotID: '',
  planOptionID: '',
  grams: 100,
} as never

afterEach(() => {
  vi.unstubAllGlobals()
})

// useTDEE is disabled by the generic dummy args (it needs real positive
// numbers to satisfy its `enabled` gate) and stays 'pending' forever when
// disabled -- it gets its own dedicated test below instead.
const hookNames = Object.keys(queries)
  .filter((k) => k.startsWith('use') && k !== 'useTDEE')
  .sort()

describe('queries.ts hook sweep (every use* hook reaches the network)', () => {
  it('found every known hook (sanity check the walk itself)', () => {
    expect(hookNames.length).toBeGreaterThan(60)
  })

  it.each(hookNames)('%s settles without throwing', async (name) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 200 })))

    const hookFn = (queries as unknown as Record<string, AnyFn>)[name]
    const { result } = renderHook(() => hookFn(...HOOK_ARGS), { wrapper })

    const initial = result.current
    if (initial && typeof initial === 'object' && typeof (initial as Record<string, unknown>).mutate === 'function') {
      ;(initial as Record<string, AnyFn>).mutate(MUTATE_ARG)
    }

    await waitFor(() => {
      const current = result.current
      if (!current || typeof current !== 'object') return
      const c = current as Record<string, unknown>
      if ('status' in c) {
        expect(c.status).not.toBe('pending')
        return
      }
      // Composite results (useSharedDashboard returns {today, meals, ...}):
      // every sub-query must itself have settled.
      for (const v of Object.values(c)) {
        if (v && typeof v === 'object' && 'status' in (v as Record<string, unknown>)) {
          expect((v as { status: unknown }).status).not.toBe('pending')
        }
      }
    })
  })

  // useTDEE gates its queryFn on `enabled`, which requires real positive
  // numbers -- the generic dummy tuple above (a string) never satisfies it,
  // so it's covered explicitly here instead.
  it('useTDEE fires once given a complete, valid params object', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 200 })))

    const { result } = renderHook(
      () => queries.useTDEE({ weight_kg: 80, height_cm: 180, age: 30, gender: 'male', activity: 'moderate' }),
      { wrapper },
    )

    await waitFor(() => expect(result.current.status).not.toBe('pending'))
  })

  it('stops polling after a terminal query error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 500 })))
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={qc}>
        <DemoProvider>{children}</DemoProvider>
      </QueryClientProvider>
    )

    const { result } = renderHook(() => queries.useMeals(), { wrapper })

    await waitFor(() => expect(result.current.status).toBe('error'))

    const query = qc.getQueryCache().find({ queryKey: queries.keys.meals(20, false) })
    expect(query).toBeDefined()
    const { refetchInterval } = query!.observers[0].options
    if (typeof refetchInterval !== 'function') throw new Error('Expected a polling callback')
    expect(refetchInterval(query!)).toBe(false)
  })
})
