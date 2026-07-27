// Renders each page of a PDF to a PNG image, client-side, so a multi-page
// prescription can be fed through the same single-image extraction endpoint
// as a photo (#194). pdfjs-dist is imported dynamically so it never lands in
// the main bundle, and never loads in environments (like jsdom under
// Vitest) that never call this function.
import type { PDFDocumentProxy } from 'pdfjs-dist'

let workerConfigured = false

async function loadPdfjs() {
  const pdfjs = await import('pdfjs-dist')
  if (!workerConfigured) {
    // Vite statically resolves this new URL(...) call and copies the worker
    // into the build output with a hashed name; the same expression also
    // resolves correctly in dev and under `vite build`.
    pdfjs.GlobalWorkerOptions.workerSrc = new URL(
      'pdfjs-dist/build/pdf.worker.min.mjs',
      import.meta.url,
    ).toString()
    workerConfigured = true
  }
  return pdfjs
}

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

// Renders every page of `file` to a PNG Blob. Multi-page merging into a
// single extraction call is out of scope (#194 PR2) — the caller decides
// what to do when more than one page comes back.
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
