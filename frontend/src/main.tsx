import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { AuthProvider } from './auth/AuthProvider';
import './styles.css';

async function enableMocking() {
  // O worker é uma ferramenta exclusiva do servidor de desenvolvimento.
  // VITE_USE_MSW nunca deve habilitá-lo em um bundle de produção.
  if (!import.meta.env.DEV || import.meta.env.MODE === 'test') return;
  const explicitlyEnabled = import.meta.env.VITE_USE_MSW === 'true';
  const enabledByDefaultInDev = import.meta.env.VITE_USE_MSW !== 'false';
  if (!explicitlyEnabled && !enabledByDefaultInDev) return;
  const { worker } = await import('./mocks/browser');
  return worker.start({ onUnhandledRequest: 'bypass' });
}

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 15_000 }, mutations: { retry: 0 } }
});

enableMocking().then(() => {
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter><AuthProvider><App /></AuthProvider></BrowserRouter>
      </QueryClientProvider>
    </React.StrictMode>
  );
});
