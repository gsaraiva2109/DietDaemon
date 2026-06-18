/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Set VITE_ENABLE_DEMO=1 at build time to expose the demo-mode toggle in a
  // production build. Demo controls are always available in dev.
  readonly VITE_ENABLE_DEMO?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
