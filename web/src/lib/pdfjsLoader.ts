// Shared pdfjs-dist loader for pdfToImages.ts and pdfToText.ts. Imported
// dynamically so it never lands in the main bundle, and never loads in
// environments (like jsdom under Vitest) that never call it.
let workerConfigured = false

export async function loadPdfjs() {
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
