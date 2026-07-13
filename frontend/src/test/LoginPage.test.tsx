import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { screen, render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AuthProvider } from '../auth/AuthProvider';
import { LoginPage } from '../pages/LoginPage';

function renderLogin() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}>
    <MemoryRouter initialEntries={['/login']}>
      <AuthProvider><Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<div>Sessao autenticada</div>} />
      </Routes></AuthProvider>
    </MemoryRouter>
  </QueryClientProvider>);
}

describe('LoginPage', () => {
  it('armazena a sessão autenticada após login bem-sucedido', async () => {
    const user = userEvent.setup();
    renderLogin();

    await user.type(screen.getByLabelText('Usuario'), 'admin.homologacao');
    await user.type(screen.getByLabelText('Senha'), 'senha-de-teste');
    await user.click(screen.getByRole('button', { name: 'Entrar' }));

    expect(await screen.findByText('Sessao autenticada')).toBeInTheDocument();
  });

  it('solicita o código TOTP somente quando o servidor exige MFA', async () => {
    const user = userEvent.setup();
    renderLogin();

    await user.type(screen.getByLabelText('Usuario'), 'admin.mfa');
    await user.type(screen.getByLabelText('Senha'), 'mfa-required');
    await user.click(screen.getByRole('button', { name: 'Entrar' }));

    expect(await screen.findByLabelText('Codigo de autenticacao')).toBeInTheDocument();
    await user.type(screen.getByLabelText('Codigo de autenticacao'), '000000');
    await user.click(screen.getByRole('button', { name: 'Confirmar MFA' }));
    expect(await screen.findByText('Sessao autenticada')).toBeInTheDocument();
  });
});
