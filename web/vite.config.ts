import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Get API details from Netronome env vars
const apiHost = process.env.NETRONOME__HOST || 'localhost'
const apiPort = process.env.NETRONOME__PORT || '7575'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
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
