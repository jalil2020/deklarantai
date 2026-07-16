import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Backend (Go) 8080-portda ishlaydi; /api so'rovlarini o'sha yerga yo'naltiramiz.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
