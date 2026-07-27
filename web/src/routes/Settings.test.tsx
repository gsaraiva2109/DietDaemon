import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import i18n from '@/lib/i18n'
import { Settings } from './Settings'
import { DemoProvider } from '@/lib/demo'
import { AuthProvider } from '@/lib/auth'

vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))

// Demo mode short-circuits useToday to sample data and skips AuthProvider's
// session probe entirely, so Settings renders fully without any api mocking.
// i18n is a module-level singleton shared across test files, so the language
// switcher test below must restore 'en' or every later test in this run sees
// Portuguese copy.
beforeEach(() => {
  localStorage.setItem('dd.demo', '1')
})

afterEach(() => {
  void i18n.changeLanguage('en')
})

function renderSettings() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DemoProvider>
          <AuthProvider>
            <Settings />
          </AuthProvider>
        </DemoProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

test('renders daily targets (read-only in demo mode) and the settings menu', async () => {
  renderSettings()

  expect(await screen.findByText('Daily targets')).toBeInTheDocument()
  // Demo mode: a "read only" pill shows and inputs are disabled. Wait for the
  // targets query to resolve (past the loading spinner) before inspecting inputs.
  expect(screen.getByText('read only')).toBeInTheDocument()
  const inputs = await screen.findAllByRole('spinbutton')
  expect(inputs).toHaveLength(5) // Calories, Protein, Carbs, Fat, Fiber
  for (const input of inputs) {
    expect(input).toBeDisabled()
  }
  expect(screen.getByRole('button', { name: 'Save targets' })).toBeDisabled()

  // Settings menu links to the other settings surfaces.
  expect(screen.getByRole('link', { name: /Security/ })).toHaveAttribute('href', '/settings/security')
  expect(screen.getByRole('link', { name: /Diet plan/ })).toHaveAttribute('href', '/plan')

  expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument()

  // "Edit body profile" re-opens the onboarding wizard via a window event;
  // OnboardingWizard itself lives outside this tree, so just prove the event
  // fires with no listener attached (nothing to assert on) rather than throws.
  const dispatchSpy = vi.spyOn(window, 'dispatchEvent')
  fireEvent.click(screen.getByRole('button', { name: /Edit body profile/ }))
  expect(dispatchSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'dd:onboarding' }))
})

test('language switcher toggles the active language pill', async () => {
  renderSettings()

  await screen.findByText('Daily targets')
  const enButton = screen.getByRole('button', { name: 'English' })
  const ptButton = screen.getByRole('button', { name: 'Português (Brasil)' })
  expect(enButton).toHaveAttribute('aria-pressed', 'true')

  ptButton.click()
  expect(await screen.findByText('Metas diárias')).toBeInTheDocument()
})

test('export data button opens the export modal', async () => {
  renderSettings()

  await screen.findByText('Daily targets')
  screen.getByRole('button', { name: /Export data/ }).click()
  expect(await screen.findByText('Download your data')).toBeInTheDocument()

  // Closing hands control back to Settings (onClose flips exporting off).
  screen.getByRole('button', { name: 'Close' }).click()
  await waitFor(() => expect(screen.queryByText('Download your data')).not.toBeInTheDocument())
})
