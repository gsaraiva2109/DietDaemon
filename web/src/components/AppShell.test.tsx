import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { AppShell } from './AppShell'
import { DemoProvider } from '@/lib/demo'
import { ThemeProvider } from '@/lib/theme'
import { AuthProvider } from '@/lib/auth'

vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))

// Demo mode makes AuthProvider report authed with no session probe and no
// backend, so the whole shell (nav + UtilityBar + VerifyEmailBanner) renders
// with no API mocking, matching the pattern used by App.test.tsx.
beforeEach(() => {
  localStorage.setItem('dd.demo', '1')
})

function renderShell(initialPath = '/') {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter initialEntries={[initialPath]}>
          <ThemeProvider>
            <AuthProvider>
              <AppShell>
                <p>page content</p>
              </AppShell>
            </AuthProvider>
          </ThemeProvider>
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

describe('AppShell', () => {
  it('renders the brand, grouped nav sections, and the routed children', () => {
    renderShell()

    expect(screen.getByText('DietDaemon')).toBeInTheDocument()
    expect(screen.getByText('page content')).toBeInTheDocument()

    // Group headings only render for groups that declare a headingKey.
    expect(screen.getByText('Discover')).toBeInTheDocument()
    expect(screen.getByText('Track')).toBeInTheDocument()

    // Nav items appear in both the desktop sidebar and the mobile bottom bar.
    for (const label of ['Today', 'Foods', 'Settings']) {
      expect(screen.getAllByText(label).length).toBeGreaterThanOrEqual(1)
    }
    // Desktop-only items (not in the curated mobile bar) appear once.
    expect(screen.getAllByText('Chat')).toHaveLength(1)
    expect(screen.getAllByText('Trends')).toHaveLength(1)
  })

  it('marks the active route with aria-current on its nav link(s)', () => {
    renderShell('/foods')

    const foodsLinks = screen.getAllByRole('link', { name: /Foods/ })
    expect(foodsLinks.length).toBeGreaterThanOrEqual(1)
    for (const link of foodsLinks) {
      expect(link).toHaveAttribute('aria-current', 'page')
    }

    const todayLinks = screen.getAllByRole('link', { name: /Today/ })
    for (const link of todayLinks) {
      expect(link).not.toHaveAttribute('aria-current')
    }
  })

  it('shows the demo banner when demo mode is on', () => {
    renderShell()
    expect(screen.getByText('Sample data, no backend needed.')).toBeInTheDocument()
  })

  it('hides the demo banner and shows no verify-email banner when demo mode is off', () => {
    localStorage.setItem('dd.demo', '0')
    renderShell()

    expect(screen.queryByText('Sample data, no backend needed.')).not.toBeInTheDocument()
    // No authenticated user in this test (anon/demo-off), so the
    // verify-email banner (which requires a real unverified user) never renders.
    expect(screen.queryByText(/verify/i)).not.toBeInTheDocument()
  })

  it('preloads a route module on hover, for both the desktop sidebar and the mobile bottom bar', () => {
    renderShell()

    // Every nav link (desktop sidebar or mobile bottom bar) wires an
    // onMouseEnter that fires the item's lazy route import ahead of the click.
    // Hovering must not throw, on either surface.
    const [desktopLink, mobileLink] = screen.getAllByRole('link', { name: /Foods/ })
    expect(() => fireEvent.mouseEnter(desktopLink)).not.toThrow()
    expect(() => fireEvent.mouseEnter(mobileLink)).not.toThrow()
  })

  it('links point at their expected routes', () => {
    renderShell()

    expect(screen.getAllByRole('link', { name: /Foods/ })[0]).toHaveAttribute('href', '/foods')
    expect(screen.getAllByRole('link', { name: /Settings/ })[0]).toHaveAttribute('href', '/settings')
  })
})
