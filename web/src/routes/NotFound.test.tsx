import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import { MemoryRouter } from 'react-router-dom'
import { expect, test, vi } from 'vitest'
import '@/lib/i18n'
import { ThemeProvider } from '@/lib/theme'
import { NotFound } from './NotFound'

vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))

test('renders a recovery link', () => {
  render(
    <ThemeProvider>
      <MemoryRouter>
        <NotFound />
      </MemoryRouter>
    </ThemeProvider>,
  )

  expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Go to dashboard' })).toHaveAttribute('href', '/')
})
