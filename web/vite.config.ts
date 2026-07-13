import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

// Same-origin in production (the Go binary serves the built assets and the API
// from one server). In dev, proxy /api to the running Go server on :8080 so
// there is no CORS to configure on either side.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    rollupOptions: {
      output: {
        // Split rarely-changing vendor code out of the main entry chunk so
        // it caches independently and stays under the 500kB warning.
        manualChunks(id) {
          if (id.includes('node_modules/framer-motion')) return 'motion'
          if (id.includes('node_modules/i18next') || id.includes('node_modules/react-i18next')) return 'i18n'
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
