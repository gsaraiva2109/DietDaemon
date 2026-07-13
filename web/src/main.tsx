import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'

// Self-host Plus Jakarta Sans, no Google Fonts CDN request.
// Latin-subset imports ship only the woff2 files we need (5 files,
// ~200KB total) vs the full charset (70 files, ~800KB).
// Vite extracts woff2 files as separate assets, served same-origin.
import '@fontsource/plus-jakarta-sans/latin-400.css'
import '@fontsource/plus-jakarta-sans/latin-500.css'
import '@fontsource/plus-jakarta-sans/latin-600.css'
import '@fontsource/plus-jakarta-sans/latin-700.css'
import '@fontsource/plus-jakarta-sans/latin-800.css'

import App from './App.tsx'
import './lib/i18n'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
