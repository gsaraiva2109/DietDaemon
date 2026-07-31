import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@testing-library/jest-dom/vitest'
import '@/lib/i18n'
import { MenuDiningCard } from './DiningMode'
import type { MenuDraft } from '@/lib/types'

vi.mock('@/lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      menu: {
        ...actual.api.menu,
        extractFromImage: vi.fn(),
        logDish: vi.fn(),
      },
    },
  }
})

import { api } from '@/lib/api'

const extractFromImage = vi.mocked(api.menu.extractFromImage)
const logDish = vi.mocked(api.menu.logDish)

function draft(overrides: Partial<MenuDraft> = {}): MenuDraft {
  return {
    unreadable: false,
    dishes: [
      { name: 'Grilled salmon', description: 'With rice and vegetables' },
      { name: 'Caesar salad', description: 'Chicken, romaine, parmesan' },
    ],
    ...overrides,
  }
}

function renderCard() {
  const queryClient = new QueryClient()
  render(
    <QueryClientProvider client={queryClient}>
      <MenuDiningCard />
    </QueryClientProvider>,
  )
}

function menuFile(name = 'menu.jpg'): File {
  return new File(['fake-image-bytes'], name, { type: 'image/jpeg' })
}

function selectPhotoFile(file: File) {
  fireEvent.change(screen.getByLabelText('Choose a photo'), { target: { files: [file] } })
}

beforeEach(() => {
  extractFromImage.mockReset()
  logDish.mockReset()
})

describe('MenuDiningCard', () => {
  it('walks upload -> pick dish -> edit -> low-confidence badge -> log with edited values', async () => {
    extractFromImage.mockResolvedValue(draft())
    logDish.mockResolvedValue({} as never)
    renderCard()

    selectPhotoFile(menuFile())
    await waitFor(() => expect(extractFromImage).toHaveBeenCalledWith(expect.any(File)))

    expect(await screen.findByText('Grilled salmon')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Caesar salad'))

    expect(await screen.findByText('Estimate — low confidence')).toBeInTheDocument()
    expect(screen.getByLabelText('Dish name')).toHaveValue('Caesar salad')

    fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'Chicken, romaine, extra parmesan' } })
    fireEvent.click(screen.getByRole('button', { name: 'Log this' }))

    await waitFor(() =>
      expect(logDish).toHaveBeenCalledWith({ name: 'Caesar salad', description: 'Chicken, romaine, extra parmesan' }),
    )
    expect(await screen.findByText(/Logged\./)).toBeInTheDocument()
  })

  it('shows the unreadable message and lets the user try another photo', async () => {
    extractFromImage.mockResolvedValue(draft({ unreadable: true }))
    renderCard()

    selectPhotoFile(menuFile())

    expect(await screen.findByRole('alert')).toHaveTextContent(/couldn't read a menu/i)
    fireEvent.click(screen.getByText('Try again'))
    expect(await screen.findByLabelText('Choose a photo')).toBeInTheDocument()
  })

  it('shows an error with a retry when extraction fails', async () => {
    extractFromImage.mockRejectedValue(new Error('boom'))
    renderCard()

    selectPhotoFile(menuFile())

    expect(await screen.findByText('boom')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Try again'))
    expect(await screen.findByLabelText('Choose a photo')).toBeInTheDocument()
  })
})
