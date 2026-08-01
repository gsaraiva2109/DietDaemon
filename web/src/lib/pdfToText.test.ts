import { describe, it, expect, vi, beforeEach } from 'vitest'

const getDocumentMock = vi.fn()
const GlobalWorkerOptions = { workerSrc: '' }

vi.mock('pdfjs-dist', () => ({
  GlobalWorkerOptions,
  getDocument: getDocumentMock,
}))

import { pdfToText } from './pdfToText'

function fakePdf(pageTexts: string[]) {
  return {
    numPages: pageTexts.length,
    getPage: vi.fn((pageNumber: number) =>
      Promise.resolve({
        getTextContent: () =>
          Promise.resolve({ items: [{ str: pageTexts[pageNumber - 1] }] }),
      }),
    ),
  }
}

beforeEach(() => {
  getDocumentMock.mockReset()
  GlobalWorkerOptions.workerSrc = ''
})

describe('pdfToText', () => {
  it('joins clean multi-page text with page-boundary markers, in order', async () => {
    getDocumentMock.mockReturnValue({
      promise: Promise.resolve(fakePdf(['Breakfast: 100g oats', 'Lunch: 150g chicken breast'])),
    })

    const file = new File(['fake-pdf-bytes'], 'plan.pdf', { type: 'application/pdf' })
    const result = await pdfToText(file)

    expect(result.status).toBe('ok')
    expect(result.pageCount).toBe(2)
    expect(result.text).toBe('--- Page 1 ---\nBreakfast: 100g oats\n\n--- Page 2 ---\nLunch: 150g chicken breast')
  })

  it('handles a single page', async () => {
    getDocumentMock.mockReturnValue({
      promise: Promise.resolve(fakePdf(['A complete diet plan written on a single page of text'])),
    })

    const file = new File(['fake-pdf-bytes'], 'plan.pdf', { type: 'application/pdf' })
    const result = await pdfToText(file)

    expect(result.status).toBe('ok')
    expect(result.pageCount).toBe(1)
    expect(result.text).toBe('--- Page 1 ---\nA complete diet plan written on a single page of text')
  })

  it('reports "empty" for a scanned PDF with no real text layer', async () => {
    getDocumentMock.mockReturnValue({ promise: Promise.resolve(fakePdf(['', ' '])) })

    const file = new File(['fake-pdf-bytes'], 'scanned.pdf', { type: 'application/pdf' })
    const result = await pdfToText(file)

    expect(result.status).toBe('empty')
    expect(result.pageCount).toBe(2)
  })

  it('reports "malformed" when the text is mostly single-character/replacement tokens', async () => {
    const garbled = Array.from({ length: 30 }, () => '�').join(' ')
    getDocumentMock.mockReturnValue({ promise: Promise.resolve(fakePdf([garbled])) })

    const file = new File(['fake-pdf-bytes'], 'garbled.pdf', { type: 'application/pdf' })
    const result = await pdfToText(file)

    expect(result.status).toBe('malformed')
  })
})
