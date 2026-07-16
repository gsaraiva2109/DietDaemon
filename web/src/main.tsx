import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'

// Self-host Plus Jakarta Sans, no Google Fonts CDN request.
// Latin-subset weights avoid the full charset (70 files, ~800KB).
// The first-paint 400/700 weights are declared in index.css so index.html can preload them.
import '@fontsource/plus-jakarta-sans/latin-500.css'
import '@fontsource/plus-jakarta-sans/latin-600.css'
import '@fontsource/plus-jakarta-sans/latin-800.css'

import App from './App.tsx'
import './lib/i18n'

// A deployment can replace a hashed lazy chunk while an open tab still has the
// previous entry bundle. Reload once to fetch the current index and its assets.
function reloadForStaleChunk() {
  const reloadKey = 'dietdaemon:chunk-reload'
  if (sessionStorage.getItem(reloadKey)) return
  sessionStorage.setItem(reloadKey, '1')
  window.location.reload()
}

window.addEventListener('vite:preloadError', (event) => {
  event.preventDefault()
  reloadForStaleChunk()
})

window.addEventListener('unhandledrejection', (event) => {
  if (!/Failed to fetch dynamically imported module/.test(String(event.reason))) return
  event.preventDefault()
  reloadForStaleChunk()
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
