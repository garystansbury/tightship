import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The build lands inside the Go module so `go build` embeds it (internal/webui). In development
// the API is proxied to the Go binary, so the app is always talking to the real route table.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: { '/api': 'http://127.0.0.1:8090', '/healthz': 'http://127.0.0.1:8090' },
  },
})
