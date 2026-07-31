import { afterEach, beforeEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'

afterEach(cleanup)

let errorSpy: ReturnType<typeof vi.spyOn>
let warnSpy: ReturnType<typeof vi.spyOn>

beforeEach(() => {
  errorSpy = vi.spyOn(console, 'error').mockImplementation((...args) => {
    throw new Error(`Unexpected console.error: ${args.map(String).join(' ')}`)
  })
  warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    throw new Error(`Unexpected console.warn: ${args.map(String).join(' ')}`)
  })
})

afterEach(() => {
  errorSpy.mockRestore()
  warnSpy.mockRestore()
})

// assistant-ui scrolls thread viewports asynchronously; jsdom has no
// Element.scrollTo implementation.
Object.defineProperty((globalThis as unknown as { HTMLElement: { prototype: object } }).HTMLElement.prototype, 'scrollTo', {
  value: vi.fn(),
  writable: true,
})
