import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'

afterEach(cleanup)

// assistant-ui scrolls thread viewports asynchronously; jsdom has no
// Element.scrollTo implementation.
Object.defineProperty((globalThis as unknown as { HTMLElement: { prototype: object } }).HTMLElement.prototype, 'scrollTo', {
  value: vi.fn(),
  writable: true,
})
