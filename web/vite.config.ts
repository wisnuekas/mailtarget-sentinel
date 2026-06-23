import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const apiProxy = process.env.VITE_API_PROXY || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: apiProxy, changeOrigin: true },
      '/health': { target: apiProxy, changeOrigin: true },
    },
  },
})
