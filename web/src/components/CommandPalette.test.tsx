import {describe, expect, it, vi} from 'vitest'
import {fireEvent, render, screen} from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import {MemoryRouter} from 'react-router-dom'
import '@/lib/i18n'
import {ThemeProvider} from '@/lib/theme'
import {DemoProvider} from '@/lib/demo'
import {CommandPalette} from './CommandPalette'
import * as React from "react";

vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))

// framer-motion's AnimatePresence keeps exiting elements mounted until their
// exit animation finishes, which jsdom never drives to completion. Replace
// it with plain passthrough elements so open/close is a normal synchronous
// mount/unmount, matching how the component actually behaves for users
// once the (short) exit animation completes.
vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: { children: React.ReactNode }) => children,
  motion: {
    div: ({ initial, animate, exit, ...rest }: Record<string, unknown>) => {
      void initial
      void animate
      void exit
      return <div {...rest} />
    },
  },
}))

function renderPalette() {
  return render(
    <ThemeProvider>
      <DemoProvider>
        <MemoryRouter>
          <CommandPalette />
        </MemoryRouter>
      </DemoProvider>
    </ThemeProvider>,
  )
}

function openPalette() {
  fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
}

describe('CommandPalette', () => {
  it('is not rendered until opened', () => {
    renderPalette()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('opens on Ctrl+K and lists commands, closes on Escape', () => {
    renderPalette()
    openPalette()

    expect(screen.getByRole('dialog', { name: 'Command palette' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Go to Today' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Open Chat' })).toBeInTheDocument()

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('filters commands by the typed query', () => {
    renderPalette()
    openPalette()

    const input = screen.getByPlaceholderText('Type a command…')
    fireEvent.change(input, { target: { value: 'trends' } })

    expect(screen.getByRole('button', { name: 'Go to Trends' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Go to Today' })).not.toBeInTheDocument()
  })

  it('shows a no-results message when nothing matches', () => {
    renderPalette()
    openPalette()

    const input = screen.getByPlaceholderText('Type a command…')
    fireEvent.change(input, { target: { value: 'zzz-nope' } })

    expect(screen.getByText('No commands')).toBeInTheDocument()
  })

  it('running a command closes the palette', () => {
    renderPalette()
    openPalette()

    fireEvent.click(screen.getByRole('button', { name: 'Go to Today' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('running the theme toggle command closes the palette', () => {
    renderPalette()
    openPalette()

    fireEvent.click(screen.getByRole('button', { name: /Switch to (dark|light) mode/ }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('running the demo toggle command closes the palette', () => {
    renderPalette()
    openPalette()

    fireEvent.click(screen.getByRole('button', { name: /Turn sample data (on|off)/ }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('navigates the list with arrow keys and runs the active command on Enter', () => {
    renderPalette()
    openPalette()

    const dialog = screen.getByRole('dialog')
    fireEvent.keyDown(dialog, { key: 'ArrowDown' })
    fireEvent.keyDown(dialog, { key: 'ArrowDown' })
    fireEvent.keyDown(dialog, { key: 'ArrowUp' })
    fireEvent.mouseMove(screen.getByRole('button', { name: 'Log a meal' }))
    fireEvent.keyDown(dialog, { key: 'Enter' })

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  // Regression test for the S6848/S1082 fix: the backdrop overlay used to be
  // a bare <div onClick=...> with no keyboard support. It's now a native
  // <button> with an explicit onKeyDown, so Enter/Space must close the
  // palette exactly like a click does.
  describe('backdrop overlay (S6848/S1082 fix)', () => {
    it('closes on click', () => {
      renderPalette()
      openPalette()

      fireEvent.click(screen.getByRole('button', { name: 'Close command palette' }))
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('closes on Enter key', () => {
      renderPalette()
      openPalette()

      const backdrop = screen.getByRole('button', { name: 'Close command palette' })
      fireEvent.keyDown(backdrop, { key: 'Enter' })
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('closes on Space key', () => {
      renderPalette()
      openPalette()

      const backdrop = screen.getByRole('button', { name: 'Close command palette' })
      fireEvent.keyDown(backdrop, { key: ' ' })
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('ignores other keys', () => {
      renderPalette()
      openPalette()

      const backdrop = screen.getByRole('button', { name: 'Close command palette' })
      fireEvent.keyDown(backdrop, { key: 'a' })
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
  })
})
