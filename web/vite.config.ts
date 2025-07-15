import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// Get API details from Netronome env vars
const apiHost = process.env.NETRONOME__HOST || '127.0.0.1'
const apiPort = process.env.NETRONOME__PORT || '7575'
const baseUrl = process.env.NETRONOME__BASE_URL || ''

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: baseUrl,
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: undefined
      }
    }
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    proxy: {
      '/api': {
        target: `http://${apiHost}:${apiPort}`,
        changeOrigin: true,
        secure: false,
      }
    }
  }
})
