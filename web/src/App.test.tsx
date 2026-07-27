import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import { beforeEach, expect, test, vi } from 'vitest'
import '@/lib/i18n'
import App from './App'

vi.stubGlobal(
  'matchMedia',
  vi.fn().mockReturnValue({
    matches: false,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  }),
)

// App wires its own BrowserRouter/providers, so it isn't injectable the way
// individual routes are. Demo mode makes every query hook short-circuit to
// sample data (queries.ts) and skips AuthProvider's session probe entirely,
// so the whole tree renders with no backend and no network mocking needed.
beforeEach(() => {
  localStorage.setItem('dd.demo', '1')
})

test('renders the authenticated app shell and default (dashboard) route in demo mode', async () => {
  render(<App />)

  // AppShell nav (desktop sidebar) renders with the brand and nav items --
  // proves AppShell, the routing tree, and ProtectedRoute's authed branch
  // (demo reports as authed) all wired up correctly.
  expect(await screen.findByText('DietDaemon')).toBeInTheDocument()
  expect(screen.getAllByText('Today').length).toBeGreaterThan(0)
  expect(screen.getAllByText('Settings').length).toBeGreaterThan(0)

  // "/" resolves to the lazy-loaded Dashboard behind Suspense.
  expect(await screen.findByText("Today's meals")).toBeInTheDocument()
})
