import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { NudgeSettings } from './NudgeSettings'
import { DemoProvider } from '@/lib/demo'
import type { NudgeRuleView } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      nudges: {
        ...actual.api.nudges,
        get: vi.fn(),
        set: vi.fn(),
        reset: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const nudgesGet = vi.mocked(api.nudges.get)
const nudgesSet = vi.mocked(api.nudges.set)
const nudgesReset = vi.mocked(api.nudges.reset)

function macroRule(overrides: Partial<NudgeRuleView> = {}): NudgeRuleView {
  return {
    rule_id: 'macro-protein',
    kind: 'macro',
    enabled: true,
    rule: { AfterHour: 18, MinFraction: 0.5 },
    ...overrides,
  }
}

function digestRule(): NudgeRuleView {
  return { rule_id: 'weekly-digest', kind: 'digest', enabled: false, rule: { CheckHour: 9 } }
}

function renderPage() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <NudgeSettings />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  localStorage.clear()
  nudgesGet.mockReset()
  nudgesSet.mockReset().mockResolvedValue({ status: 'ok' })
  nudgesReset.mockReset().mockResolvedValue({ status: 'ok' })
})

describe('NudgeSettings loading and empty states', () => {
  it('shows a spinner while the rules are loading', async () => {
    nudgesGet.mockReturnValue(new Promise(() => {}))
    renderPage()
    expect(await screen.findByText('Loading nudge rules', { exact: false })).toBeInTheDocument()
  })

  it('shows an empty state when there are no rules', async () => {
    nudgesGet.mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('No nudge rules found')).toBeInTheDocument()
  })
})

describe('NudgeSettings with rules', () => {
  it('renders rule groups, titles, and a back link to settings', async () => {
    nudgesGet.mockResolvedValue([macroRule(), digestRule()])
    renderPage()

    expect(await screen.findByText('Macro nudges')).toBeInTheDocument()
    expect(screen.getByText('Weekly digest')).toBeInTheDocument()
    expect(screen.getByText('Macro Protein')).toBeInTheDocument()
    expect(screen.getByText('Weekly Digest')).toBeInTheDocument()
    // Groups with no matching rules (health, weekly budget, smart meal) don't render.
    expect(screen.queryByText('Health nudges')).not.toBeInTheDocument()

    expect(screen.getByRole('link', { name: /Settings/ })).toHaveAttribute('href', '/settings')
  })

  it('toggles a rule on/off', async () => {
    nudgesGet.mockResolvedValue([macroRule({ enabled: true })])
    renderPage()

    const toggle = await screen.findByLabelText('Enable Macro Protein')
    fireEvent.click(toggle)

    await waitFor(() =>
      expect(nudgesSet).toHaveBeenCalledWith({ rule_id: 'macro-protein', enabled: false }),
    )
  })

  it('edits a numeric field, then saves the tuned params', async () => {
    nudgesGet.mockResolvedValue([macroRule()])
    renderPage()

    const afterHourInput = await screen.findByLabelText('After hour')
    fireEvent.change(afterHourInput, { target: { value: '20' } })

    const saveButton = screen.getByRole('button', { name: 'Save' })
    expect(saveButton).toBeEnabled()
    fireEvent.click(saveButton)

    await waitFor(() =>
      expect(nudgesSet).toHaveBeenCalledWith({
        rule_id: 'macro-protein',
        enabled: true,
        params: { AfterHour: 20, MinFraction: 0.5 },
      }),
    )
  })

  it('save is disabled until a field is dirty', async () => {
    nudgesGet.mockResolvedValue([macroRule()])
    renderPage()

    await screen.findByLabelText('After hour')
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('resets a rule to its default', async () => {
    nudgesGet.mockResolvedValue([macroRule()])
    renderPage()

    await screen.findByText('Macro Protein')
    fireEvent.click(screen.getByRole('button', { name: 'Reset to default' }))

    await waitFor(() => expect(nudgesReset).toHaveBeenCalledWith('macro-protein'))
  })

  it('a rule with no editable fields (e.g. fasting) shows no numeric inputs but still offers reset', async () => {
    nudgesGet.mockResolvedValue([{ rule_id: 'fasting-window', kind: 'health', enabled: true, rule: { Domain: 'fasting' } }])
    renderPage()

    await screen.findByText('Fasting Window')
    expect(screen.queryByRole('spinbutton')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset to default' })).toBeInTheDocument()
  })

  it('renders the rule message when present', async () => {
    nudgesGet.mockResolvedValue([macroRule({ rule: { AfterHour: 18, MinFraction: 0.5, Message: 'Custom nudge copy' } })])
    renderPage()
    expect(await screen.findByText('Custom nudge copy')).toBeInTheDocument()
  })
})

describe('NudgeSettings in demo mode', () => {
  beforeEach(() => {
    localStorage.setItem('dd.demo', '1')
  })

  it('shows a read-only banner, disables toggles/inputs, and hides save/reset controls', async () => {
    nudgesGet.mockResolvedValue([macroRule()])
    renderPage()

    expect(await screen.findByText('Nudge rules are read only here.')).toBeInTheDocument()
    expect(await screen.findByLabelText('Enable Macro Protein')).toBeDisabled()
    expect(screen.getByLabelText('After hour')).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Reset to default' })).not.toBeInTheDocument()
  })
})
