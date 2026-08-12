import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Aliases } from './Aliases'
import { DemoProvider } from '@/lib/demo'
import type { FoodDetail } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      foods: {
        ...actual.api.foods,
        list: vi.fn(),
        search: vi.fn(),
        addAlias: vi.fn(),
        deleteAlias: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const list = vi.mocked(api.foods.list)
const search = vi.mocked(api.foods.search)
const addAlias = vi.mocked(api.foods.addAlias)
const deleteAlias = vi.mocked(api.foods.deleteAlias)

function food(overrides: Partial<FoodDetail> = {}): FoodDetail {
  return {
    food_id: 'f1',
    name: 'Chicken breast',
    source: 'food_library',
    per_100g: { Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Fiber: 0 },
    category: '',
    brand: '',
    barcode: '',
    image_url: '',
    serving_size: 100,
    serving_unit: 'g',
    query_count: 3,
    last_used: '',
    aliases: [],
    in_library: true,
    volume_units_eligible: false,
    ...overrides,
  }
}

function renderAliases({ demo = false }: { demo?: boolean } = {}) {
  localStorage.setItem('dd.demo', demo ? '1' : '0')
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DemoProvider>
          <Aliases />
        </DemoProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  list.mockReset()
  search.mockReset()
  addAlias.mockReset()
  deleteAlias.mockReset()
  localStorage.clear()
})

describe('Aliases loading/empty branches', () => {
  it('shows a spinner while browsing foods is in flight', async () => {
    list.mockReturnValue(new Promise(() => {}))
    renderAliases()

    expect(await screen.findByRole('status')).toHaveTextContent('Loading foods')
  })

  it('shows the empty state when there are no foods', async () => {
    list.mockResolvedValue([])
    renderAliases()

    expect(await screen.findByText('No foods found')).toBeInTheDocument()
    expect(screen.getByText('Try a different search.')).toBeInTheDocument()
  })
})

describe('Aliases list', () => {
  it('renders a card per food with "no aliases" when there are none', async () => {
    list.mockResolvedValue([food({ food_id: 'f1', name: 'Chicken breast', aliases: [] })])
    renderAliases()

    expect(await screen.findByText('Chicken breast')).toBeInTheDocument()
    expect(screen.getByText('No aliases yet.')).toBeInTheDocument()
  })

  it('renders existing alias chips', async () => {
    list.mockResolvedValue([
      food({
        food_id: 'f1',
        name: 'Chicken breast',
        aliases: [{ food_id: 'f1', alias: 'chx', normalized: 'chx' }],
      }),
    ])
    renderAliases()

    expect(await screen.findByText('chx')).toBeInTheDocument()
  })

  it('adds an alias on submit and clears the input', async () => {
    list.mockResolvedValue([food({ food_id: 'f1', name: 'Chicken breast' })])
    addAlias.mockResolvedValue({ status: 'ok' })
    renderAliases()

    const input = await screen.findByLabelText('Add alias for Chicken breast')
    fireEvent.change(input, { target: { value: 'chix' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(addAlias).toHaveBeenCalledWith('f1', 'chix'))
    expect(input).toHaveValue('')
  })

  it('removes an alias on click', async () => {
    list.mockResolvedValue([
      food({
        food_id: 'f1',
        name: 'Chicken breast',
        aliases: [{ food_id: 'f1', alias: 'chx', normalized: 'chx' }],
      }),
    ])
    deleteAlias.mockResolvedValue(undefined)
    renderAliases()

    fireEvent.click(await screen.findByRole('button', { name: 'Remove alias chx' }))

    await waitFor(() => expect(deleteAlias).toHaveBeenCalledWith('f1', 'chx'))
  })

  it('debounces search input and queries the search endpoint', async () => {
    list.mockResolvedValue([])
    search.mockResolvedValue([food({ food_id: 'f2', name: 'Chicken thigh' })])
    renderAliases()
    await screen.findByText('No foods found')

    fireEvent.change(screen.getByLabelText('Search foods'), { target: { value: 'chicken' } })

    await waitFor(() => expect(search).toHaveBeenCalledWith('chicken'))
    expect(await screen.findByText('Chicken thigh')).toBeInTheDocument()
  })
})

describe('Aliases demo mode', () => {
  it('shows the read-only notice and hides add/remove controls', async () => {
    list.mockResolvedValue([
      food({
        food_id: 'f1',
        name: 'Chicken breast',
        aliases: [{ food_id: 'f1', alias: 'chx', normalized: 'chx' }],
      }),
    ])
    renderAliases({ demo: true })

    expect(await screen.findByText('Aliases are read only here.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Remove alias chx' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Add alias for Chicken breast')).not.toBeInTheDocument()
  })
})
