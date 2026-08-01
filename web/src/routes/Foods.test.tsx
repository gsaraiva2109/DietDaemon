import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { Foods } from './Foods'
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
        searchCatalog: vi.fn(),
        frequent: vi.fn(),
      },
    },
  }
})

vi.mock('@/components/FoodDetailModal', () => ({
  FoodDetailModal: ({ foodID, onClose }: { foodID: string; onClose: () => void }) => (
    <div data-testid="food-detail-modal">
      detail:{foodID}
      <button onClick={onClose}>close-detail</button>
    </div>
  ),
}))

vi.mock('@/components/CustomFoodModal', () => ({
  CustomFoodModal: ({ food, onClose }: { food?: FoodDetail; onClose: () => void }) => (
    <div data-testid="custom-food-modal">
      custom:{food ? food.food_id : 'new'}
      <button onClick={onClose}>close-custom</button>
    </div>
  ),
}))

import { api } from '@/lib/api'

const list = vi.mocked(api.foods.list)
const search = vi.mocked(api.foods.search)
const searchCatalog = vi.mocked(api.foods.searchCatalog)
const frequent = vi.mocked(api.foods.frequent)

function food(overrides: Partial<FoodDetail>): FoodDetail {
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
    in_library: true,
    volume_units_eligible: false,
    ...overrides,
  }
}

function renderFoods() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <DemoProvider>
        <MemoryRouter>
          <Foods />
        </MemoryRouter>
      </DemoProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  list.mockReset().mockResolvedValue([])
  search.mockReset().mockResolvedValue([])
  searchCatalog.mockReset().mockResolvedValue([])
  frequent.mockReset().mockResolvedValue([])
})

describe('Foods library tab', () => {
  it('shows the library empty state with an add-custom button (top and inline) when there are no library foods', async () => {
    renderFoods()

    expect(await screen.findByText('No foods yet')).toBeInTheDocument()
    expect(
      screen.getByText(
        "Foods appear here as you log meals. Looking for something you haven't logged yet? Browse the full imported catalog in the Catalog tab.",
      ),
    ).toBeInTheDocument()
    // One button at the top of the page, one inline under the empty state.
    expect(screen.getAllByRole('button', { name: 'Add custom food' })).toHaveLength(2)
  })

  it('shows the no-matches empty state when a library search has no results, without hiding the add-custom button', async () => {
    search.mockResolvedValue([])
    renderFoods()
    await screen.findByText('No foods yet')

    fireEvent.change(screen.getByLabelText('Search foods'), { target: { value: 'nonexistent' } })

    await waitFor(() => expect(search).toHaveBeenCalledWith('nonexistent'))
    expect(await screen.findByText('No matches')).toBeInTheDocument()
    expect(
      screen.getByText(
        "This only searches foods you've already logged — try a different name/alias, or log the meal directly.",
      ),
    ).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Add custom food' })).toHaveLength(2)
  })

  it('renders a food card per library result and opens the detail modal on click', async () => {
    list.mockResolvedValue([food({ food_id: 'f1', name: 'Chicken breast' })])
    renderFoods()

    const card = await screen.findByText('Chicken breast')
    fireEvent.click(card)

    expect(await screen.findByTestId('food-detail-modal')).toHaveTextContent('detail:f1')
  })

  it('shows the loading spinner while the library list is loading', async () => {
    list.mockImplementation(() => new Promise(() => {}))
    renderFoods()

    const status = await screen.findByRole('status')
    expect(status).toHaveTextContent('Loading foods')
  })

  it('hides the source filter row only on the library tab while actively searching', async () => {
    list.mockResolvedValue([])
    search.mockResolvedValue([])
    renderFoods()
    await screen.findByText('No foods yet')

    // Library, not searching: filters visible.
    expect(screen.getByRole('button', { name: 'OpenFoodFacts' })).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Search foods'), { target: { value: 'chicken' } })
    await waitFor(() => expect(search).toHaveBeenCalledWith('chicken'))

    // Library, searching: filters hidden.
    expect(screen.queryByRole('button', { name: 'OpenFoodFacts' })).not.toBeInTheDocument()
  })

  it('opens the custom-food modal for a new food from the top button', async () => {
    renderFoods()
    await screen.findByText('No foods yet')

    fireEvent.click(screen.getAllByRole('button', { name: 'Add custom food' })[0])

    expect(await screen.findByTestId('custom-food-modal')).toHaveTextContent('custom:new')
  })
})

describe('Foods catalog tab', () => {
  async function openCatalogTab() {
    renderFoods()
    await screen.findByText('No foods yet')
    fireEvent.click(screen.getByRole('button', { name: 'Catalog' }))
  }

  it('shows the catalog empty state without the inline add-custom button', async () => {
    searchCatalog.mockResolvedValue([])
    await openCatalogTab()

    expect(await screen.findByText('No catalog matches')).toBeInTheDocument()
    expect(screen.getByText('Try a different search or source filter.')).toBeInTheDocument()
    // Only the top "Add custom food" button remains, the inline one is library-only.
    expect(screen.getAllByRole('button', { name: 'Add custom food' })).toHaveLength(1)
  })

  it('keeps the source filter row visible on the catalog tab even while searching', async () => {
    searchCatalog.mockResolvedValue([])
    await openCatalogTab()
    await screen.findByText('No catalog matches')

    fireEvent.change(screen.getByLabelText('Search foods'), { target: { value: 'rice' } })
    await waitFor(() => expect(searchCatalog).toHaveBeenCalledWith('rice', '', 30))

    expect(screen.getByRole('button', { name: 'TACO' })).toBeInTheDocument()
  })

  it('shows a load-more button at a full page and requests the next page on click', async () => {
    const page = Array.from({ length: 30 }, (_, i) => food({ food_id: `c${i}`, name: `Catalog food ${i}` }))
    searchCatalog.mockResolvedValue(page)
    await openCatalogTab()

    const loadMore = await screen.findByRole('button', { name: 'Load more' })
    fireEvent.click(loadMore)

    await waitFor(() => expect(searchCatalog).toHaveBeenCalledWith('', '', 60))
  })
})
