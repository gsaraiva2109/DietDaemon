/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ENABLE_DEMO?: string
}

// pdfjs-dist ships no types for this worker entry point; only its module
// shape (imported for its side effect of registering the fake worker) is
// needed by the *.fixture.test.ts(x) files that import it directly.
declare module 'pdfjs-dist/legacy/build/pdf.worker.min.mjs'
