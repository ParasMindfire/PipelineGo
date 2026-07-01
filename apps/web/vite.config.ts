import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Reads .env from the project root; frontend calls backend directly via VITE_API_URL
export default defineConfig({
  plugins: [react()],
  // Look for .env files two levels up (project root), not inside apps/web
  envDir: path.resolve(__dirname, '../../'),
  server: {
    port: 5173,
  },
})
