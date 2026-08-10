import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import { DemoProvider, useDemo, demoAvailable } from './demo'

function Probe() {
  const { demo, setDemo } = useDemo()
  return (
    <div>
      <span data-testid="demo-value">{String(demo)}</span>
      <button onClick={() => setDemo(!demo)}>toggle</button>
    </div>
  )
}

describe('demoAvailable', () => {
  it('is true under vitest (DEV mode)', () => {
    expect(demoAvailable()).toBe(true)
  })
})

describe('DemoProvider / useDemo', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('defaults to false when localStorage has no saved value', () => {
    render(
      <DemoProvider>
        <Probe />
      </DemoProvider>,
    )
    expect(screen.getByTestId('demo-value')).toHaveTextContent('false')
  })

  it('reads the initial value from localStorage', () => {
    localStorage.setItem('dd.demo', '1')
    render(
      <DemoProvider>
        <Probe />
      </DemoProvider>,
    )
    expect(screen.getByTestId('demo-value')).toHaveTextContent('true')
  })

  it('setDemo updates state and persists to localStorage', () => {
    render(
      <DemoProvider>
        <Probe />
      </DemoProvider>,
    )
    fireEvent.click(screen.getByText('toggle'))
    expect(screen.getByTestId('demo-value')).toHaveTextContent('true')
    expect(localStorage.getItem('dd.demo')).toBe('1')

    fireEvent.click(screen.getByText('toggle'))
    expect(screen.getByTestId('demo-value')).toHaveTextContent('false')
    expect(localStorage.getItem('dd.demo')).toBe('0')
  })

  it('useDemo throws when used outside a DemoProvider', () => {
    function renderWithoutProvider() {
      render(<Probe />)
    }
    expect(renderWithoutProvider).toThrow('useDemo must be used within DemoProvider')
  })
})
