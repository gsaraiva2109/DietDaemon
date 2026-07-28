import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

const getDocumentMock = vi.fn()
const GlobalWorkerOptions = { workerSrc: '' }

vi.mock('pdfjs-dist', () => ({
  GlobalWorkerOptions,
  getDocument: getDocumentMock,
}))

import { pdfToImages } from './pdfToImages'

function fakePdf(numPages: number) {
  return {
    numPages,
    getPage: vi.fn(() =>
      Promise.resolve({
        getViewport: () => ({ width: 10, height: 10 }),
        render: () => ({ promise: Promise.resolve() }),
      }),
    ),
  }
}

describe('pdfToImages', () => {
  let toBlobSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    getDocumentMock.mockReset()
    GlobalWorkerOptions.workerSrc = ''
    toBlobSpy = vi
      .spyOn(HTMLCanvasElement.prototype, 'toBlob')
      .mockImplementation(function (this: HTMLCanvasElement, cb: BlobCallback) {
        cb(new Blob(['fake-png-bytes'], { type: 'image/png' }))
      })
  })

  afterEach(() => {
    toBlobSpy.mockRestore()
  })

  it('renders every page of the PDF to a PNG blob, in order', async () => {
    getDocumentMock.mockReturnValue({ promise: Promise.resolve(fakePdf(2)) })

    const file = new File(['fake-pdf-bytes'], 'plan.pdf', { type: 'application/pdf' })
    const blobs = await pdfToImages(file)

    expect(blobs).toHaveLength(2)
    for (const blob of blobs) {
      expect(blob.type).toBe('image/png')
    }
    expect(GlobalWorkerOptions.workerSrc).toContain('pdf.worker.min.mjs')
  })

  it('returns no blobs for an empty PDF', async () => {
    getDocumentMock.mockReturnValue({ promise: Promise.resolve(fakePdf(0)) })

    const file = new File(['fake-pdf-bytes'], 'empty.pdf', { type: 'application/pdf' })
    const blobs = await pdfToImages(file)

    expect(blobs).toHaveLength(0)
  })

  it('throws if the canvas cannot produce a blob', async () => {
    toBlobSpy.mockImplementation(function (this: HTMLCanvasElement, cb: BlobCallback) {
      cb(null)
    })
    getDocumentMock.mockReturnValue({ promise: Promise.resolve(fakePdf(1)) })

    const file = new File(['fake-pdf-bytes'], 'plan.pdf', { type: 'application/pdf' })
    await expect(pdfToImages(file)).rejects.toThrow('Failed to render PDF page to an image')
  })
})
