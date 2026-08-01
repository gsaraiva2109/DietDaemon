// Renders each page of a PDF to a PNG image, client-side, so a scanned
// prescription (or one where native text extraction failed/looked garbled,
// see pdfToText.ts) can be fed through the same multi-image extraction
// endpoint as a set of photos (#194, #220).
import type { PDFDocumentProxy } from 'pdfjs-dist'
import { loadPdfjs } from './pdfjsLoader'

async function renderPage(pdf: PDFDocumentProxy, pageNumber: number): Promise<Blob> {
  const page = await pdf.getPage(pageNumber)
  const viewport = page.getViewport({ scale: 2 })
  const canvas = document.createElement('canvas')
  canvas.width = viewport.width
  canvas.height = viewport.height
  await page.render({ canvas, viewport }).promise
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'))
  if (!blob) throw new Error('Failed to render PDF page to an image')
  return blob
}

// Renders every page of `file` to a PNG Blob, in order.
export async function pdfToImages(file: File): Promise<Blob[]> {
  const pdfjs = await loadPdfjs()
  const data = await file.arrayBuffer()
  const pdf = await pdfjs.getDocument({ data }).promise
  const blobs: Blob[] = []
  for (let pageNumber = 1; pageNumber <= pdf.numPages; pageNumber++) {
    blobs.push(await renderPage(pdf, pageNumber))
  }
  return blobs
}
