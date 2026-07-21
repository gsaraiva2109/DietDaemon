import { afterEach, expect, it, vi } from 'vitest'
import { api } from './api'

afterEach(() => vi.unstubAllGlobals())

it('preserves structured API error details', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: { code: 'validation_error', message: 'Invalid date.' } }), {
        status: 400,
        headers: { 'X-Request-ID': 'request-123' },
      }),
    ),
  )

  await expect(api.rollupToday()).rejects.toMatchObject({
    code: 'validation_error',
    status: 400,
    requestID: 'request-123',
    message: 'Invalid date.',
  })
})
