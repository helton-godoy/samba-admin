import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig(({ command }) => ({
  plugins: [react()],
  // O único arquivo público atual é o worker do MSW. Ele é servido por Vite
  // no desenvolvimento, mas nunca copiado para o frontend-dist de produção.
  publicDir: command === 'serve' ? 'public' : false,
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
      '/readyz': 'http://localhost:8080'
    }
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    exclude: ['**/node_modules/**', '**/dist/**', '**/e2e/**', '**/e2e-real/**'],
    css: true
  }
}));
