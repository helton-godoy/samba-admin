import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DomainPage } from '../pages/DomainPage';
import { renderWithProviders } from './render';

describe('Active Directory', () => {
  it('mantém o fluxo como membro separado do AD DC', async () => {
    renderWithProviders(<DomainPage />);
    expect(await screen.findByText('Perfil atual: servidor membro')).toBeInTheDocument();
    expect(screen.getByText(/Trocar membro de domínio por AD DC exige reprovisionamento separado/)).toBeInTheDocument();
    const password = screen.getByPlaceholderText('Coleta bloqueada nesta release');
    expect(password).toHaveAttribute('type', 'password');
    expect(password).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Preparar ingresso como membro' })).toBeDisabled();
  });
});
