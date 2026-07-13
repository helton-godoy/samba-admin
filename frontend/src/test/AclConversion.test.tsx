import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { FileSystemsPage } from '../pages/FileSystemsPage';
import { renderWithProviders } from './render';

describe('Conversão de ACL', () => {
  it('produz relatório de perdas e mantém aplicação bloqueada', async () => {
    const user = userEvent.setup();
    renderWithProviders(<FileSystemsPage />);
    await screen.findByText('Assistente de alteração do modelo de ACL');
    await user.click(screen.getByRole('button', { name: 'Simular conversão e gerar relatório' }));
    expect(await screen.findByText('Não representáveis')).toBeInTheDocument();
    expect(screen.getByText(/Operação bloqueada no protótipo/)).toBeInTheDocument();
    expect(screen.getByText(/não garante equivalência semântica perfeita/)).toBeInTheDocument();
  });
});
