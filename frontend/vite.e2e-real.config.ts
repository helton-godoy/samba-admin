import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const apiTarget = process.env.SAMBA_ADMIN_E2E_API_URL || 'http://127.0.0.1:18080';

export default defineConfig({
  plugins: [react()],
  publicDir: false,
  server: {
    proxy: {
      '/api': apiTarget,
      '/healthz': apiTarget,
      '/readyz': apiTarget
    }
  }
});
