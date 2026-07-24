import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { DemoProvider } from '@/lib/demo'
import { Templates } from './Templates'
import { DEMO_TEMPLATES } from '@/lib/demoData'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      templates: { ...actual.api.templates, list: vi.fn(), log: vi.fn(), delete: vi.fn() },
    },
  }
})

import { api } from '@/lib/api'

const list = vi.mocked(api.templates.list)
const logTemplate = vi.mocked(api.templates.log)
const deleteTemplate = vi.mocked(api.templates.delete)

const TEMPLATE = DEMO_TEMPLATES[0] // "Grilled chicken + rice", 2 items, real macro shape

function renderTemplates({ demo = false }: { demo?: boolean } = {}) {
  localStorage.setItem('dd.demo', demo ? '1' : '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DemoProvider>
          <Templates />
        </DemoProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  list.mockReset()
  logTemplate.mockReset()
  deleteTemplate.mockReset()
})

describe('Templates list ternary', () => {
  it('loading branch: shows a spinner', async () => {
    list.mockReturnValue(new Promise(() => {}))
    renderTemplates()

    expect(await screen.findByText('Loading templates', { exact: false })).toBeInTheDocument()
  })

  it('empty branch: shows the empty state', async () => {
    list.mockResolvedValue([])
    renderTemplates()

    expect(await screen.findByText('No templates yet')).toBeInTheDocument()
  })

  it('list branch: renders each template with computed kcal and item count', async () => {
    list.mockResolvedValue(DEMO_TEMPLATES)
    renderTemplates()

    expect(await screen.findByText('Grilled chicken + rice')).toBeInTheDocument()
    expect(screen.getByText('Protein shake + oats')).toBeInTheDocument()
    // 330 + 195 = 525 kcal for template 1 (templateKcal sums item.Macros.Calories).
    expect(screen.getByText(/525 kcal/)).toBeInTheDocument()
    expect(screen.getAllByText(/2 items/)).toHaveLength(2) // both demo templates have 2 items
  })
})

describe('Templates compose-from-scratch', () => {
  it('opens ComposeTemplateModal on "New from scratch" and hides it in demo mode', async () => {
    list.mockResolvedValue([])
    renderTemplates()

    expect(await screen.findByRole('button', { name: 'New from scratch' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'New from scratch' }))
    expect(await screen.findByText('Compose a template')).toBeInTheDocument()
  })

  it('hides the "New from scratch" button in demo mode', async () => {
    renderTemplates({ demo: true })

    await screen.findByText('Grilled chicken + rice')
    expect(screen.queryByRole('button', { name: 'New from scratch' })).not.toBeInTheDocument()
  })
})

describe('Templates demo mode', () => {
  it('hides log/delete actions', async () => {
    renderTemplates({ demo: true })

    expect(await screen.findByText('Grilled chicken + rice')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Log' })).not.toBeInTheDocument()
  })
})

describe('TemplateRow action ternary chain', () => {
  it('default branch: shows Log and Delete controls', async () => {
    list.mockResolvedValue([TEMPLATE])
    renderTemplates()

    expect(await screen.findByRole('button', { name: 'Log' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: `Delete ${TEMPLATE.name}` })).toBeInTheDocument()
  })

  it('confirming=log branch: shows Confirm/Cancel, then Confirm logs and shows the Logged pill', async () => {
    list.mockResolvedValue([TEMPLATE])
    logTemplate.mockResolvedValue({ status: 'logged', meal_id: 'm1' })
    renderTemplates()

    fireEvent.click(await screen.findByRole('button', { name: 'Log' }))
    expect(await screen.findByRole('button', { name: 'Confirm' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    await waitFor(() => expect(logTemplate).toHaveBeenCalledWith(TEMPLATE.id))
    expect(await screen.findByText('Logged')).toBeInTheDocument()
  })

  it('confirming=log branch: Cancel returns to the default branch', async () => {
    list.mockResolvedValue([TEMPLATE])
    renderTemplates()

    fireEvent.click(await screen.findByRole('button', { name: 'Log' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Cancel' }))

    expect(await screen.findByRole('button', { name: 'Log' })).toBeInTheDocument()
    expect(logTemplate).not.toHaveBeenCalled()
  })

  it('confirming=delete branch: shows Delete/Cancel, then Delete calls the API', async () => {
    list.mockResolvedValue([TEMPLATE])
    deleteTemplate.mockResolvedValue(undefined)
    renderTemplates()

    fireEvent.click(await screen.findByRole('button', { name: `Delete ${TEMPLATE.name}` }))
    const confirmDelete = await screen.findByRole('button', { name: 'Delete' })
    expect(confirmDelete).toBeInTheDocument()

    fireEvent.click(confirmDelete)
    await waitFor(() => expect(deleteTemplate).toHaveBeenCalledWith(TEMPLATE.id))
  })
})

describe('TemplateRow error-message ternary', () => {
  it('shows the log mutation error message when logging fails', async () => {
    list.mockResolvedValue([TEMPLATE])
    logTemplate.mockRejectedValue(new Error('Server is unavailable'))
    renderTemplates()

    fireEvent.click(await screen.findByRole('button', { name: 'Log' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Server is unavailable')
  })

  it('shows the delete mutation error message when deleting fails', async () => {
    list.mockResolvedValue([TEMPLATE])
    deleteTemplate.mockRejectedValue(new Error('Template is in use'))
    renderTemplates()

    fireEvent.click(await screen.findByRole('button', { name: `Delete ${TEMPLATE.name}` }))
    fireEvent.click(await screen.findByRole('button', { name: 'Delete' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Template is in use')
  })

  it('falls back to the generic error message for a non-Error rejection', async () => {
    list.mockResolvedValue([TEMPLATE])
    logTemplate.mockRejectedValue('nope')
    renderTemplates()

    fireEvent.click(await screen.findByRole('button', { name: 'Log' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Something went wrong')
  })
})
