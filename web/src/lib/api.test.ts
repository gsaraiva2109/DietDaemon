import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError, RateLimitError, readCookie, setUnauthorizedHandler, UnauthorizedError } from './api'

afterEach(() => {
  vi.unstubAllGlobals()
  setUnauthorizedHandler(null)
  document.cookie.split(';').forEach((c) => {
    const name = c.split('=')[0]?.trim()
    if (name) document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/`
  })
})

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

describe('request()', () => {
  it('wraps a fetch failure in a network ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('failed to fetch')))

    const err = await api.rollupToday().catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 0, code: 'service_unavailable' })
  })

  it('falls back to a generic message when the error body is not JSON', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('not json', { status: 500 })))

    await expect(api.rollupToday()).rejects.toMatchObject({
      status: 500,
      code: 'internal_error',
      message: 'Request failed (500)',
    })
  })

  it('fires the global 401 handler unless suppressed', async () => {
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 401 })))

    await expect(api.rollupToday()).rejects.toBeInstanceOf(UnauthorizedError)
    // Fired out-of-band via queueMicrotask; flush the microtask queue.
    await Promise.resolve()
    await Promise.resolve()
    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it('suppresses the 401 handler for the anon session probe', async () => {
    const onUnauthorized = vi.fn()
    setUnauthorizedHandler(onUnauthorized)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 401 })))

    await expect(api.auth.session()).rejects.toBeInstanceOf(UnauthorizedError)
    await Promise.resolve()
    await Promise.resolve()
    expect(onUnauthorized).not.toHaveBeenCalled()
  })

  it('parses Retry-After into RateLimitError.retryAfter', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(null, { status: 429, headers: { 'Retry-After': '30' } })),
    )

    const err = await api.rollupToday().catch((e: unknown) => e)
    expect(err).toBeInstanceOf(RateLimitError)
    expect((err as RateLimitError).retryAfter).toBe(30)
  })

  it('leaves retryAfter null when Retry-After is absent', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 429 })))

    const err = await api.rollupToday().catch((e: unknown) => e)
    expect((err as RateLimitError).retryAfter).toBeNull()
  })

  it('returns undefined for a 204 No Content response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))

    await expect(api.auth.logout()).resolves.toBeUndefined()
  })

  it('attaches the CSRF header only for mutating methods', async () => {
    document.cookie = 'dd_csrf=csrf-token-abc'
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({}), { status: 200 })))
    vi.stubGlobal('fetch', fetchMock)

    await api.rollupToday()
    const getHeaders = fetchMock.mock.calls[0][1].headers as Headers
    expect(getHeaders.get('X-CSRF-Token')).toBeNull()

    await api.templates.delete('t1')
    const deleteHeaders = fetchMock.mock.calls[1][1].headers as Headers
    expect(deleteHeaders.get('X-CSRF-Token')).toBe('csrf-token-abc')
  })
})

describe('readCookie', () => {
  it('returns null when the cookie is absent', () => {
    expect(readCookie('missing_cookie')).toBeNull()
  })

  it('reads and decodes a matching cookie, ignoring similarly-prefixed names', () => {
    document.cookie = 'dd_csrfx=wrong'
    document.cookie = 'dd_csrf=hello%20world'
    expect(readCookie('dd_csrf')).toBe('hello world')
  })

  it('escapes regex-special characters in the cookie name', () => {
    // A name containing regex metacharacters must be treated literally, not
    // as part of the constructed pattern.
    document.cookie = 'a.b=literal'
    document.cookie = 'axb=wrong-if-dot-were-a-wildcard'
    expect(readCookie('a.b')).toBe('literal')
  })
})

describe('foods URL construction (flattened nested template literals)', () => {
  it('omits the source param when source is empty', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify([]), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.foods.list()
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/foods?limit=30&offset=0')
  })

  it('appends an encoded source param when provided', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify([]), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.foods.list('open food facts', 10, 5)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/foods?limit=10&offset=5&source=open%20food%20facts')
  })

  it('searchCatalog appends an encoded source param when provided', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify([]), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await api.foods.searchCatalog('egg', 'usda', 5, 0)
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/catalog/search?q=egg&limit=5&offset=0&source=usda')
  })
})
