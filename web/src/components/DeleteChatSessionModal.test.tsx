import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DeleteChatSessionModal } from './DeleteChatSessionModal'

function renderModal() {
  const onCancel = vi.fn()
  const onConfirm = vi.fn()
  render(<DeleteChatSessionModal onCancel={onCancel} onConfirm={onConfirm} />)
  return { onCancel, onConfirm }
}

function getBackdropButton() {
  return screen.getByRole('button', { name: 'Dismiss' })
}

describe('DeleteChatSessionModal', () => {
  it('renders the dialog with title and body', () => {
    renderModal()
    expect(screen.getByRole('dialog', { name: 'Delete conversation' })).toBeInTheDocument()
    expect(screen.getByText('Delete this conversation?')).toBeInTheDocument()
  })

  it('calls onConfirm when Delete is clicked', () => {
    const { onConfirm } = renderModal()
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
  })

  it('calls onCancel when Cancel is clicked', () => {
    const { onCancel } = renderModal()
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('calls onCancel when the close icon button is clicked', () => {
    const { onCancel } = renderModal()
    const closeButtons = screen.getAllByRole('button', { name: 'Close' })
    const iconClose = closeButtons.find((b) => !b.className.includes('inset-0'))!
    fireEvent.click(iconClose)
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('backdrop is a native, focusable button that closes on click', () => {
    const { onCancel } = renderModal()
    const overlay = getBackdropButton()
    expect(overlay.tagName).toBe('BUTTON')
    fireEvent.click(overlay)
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('backdrop is keyboard-reachable and activatable (regression test for the a11y fix)', () => {
    // Previously a bare <div onClick>: not focusable, no keyboard access at all.
    // As a native <button> it is tab-reachable and Enter/Space-activates by
    // default browser behavior (no onKeyDown wiring needed), unlike the div.
    renderModal()
    const overlay = getBackdropButton()
    expect(overlay.tagName).toBe('BUTTON')
    expect(overlay).toHaveAttribute('type', 'button')
    overlay.focus()
    expect(overlay).toHaveFocus()
  })
})
