import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { NewSharePage } from '../pages/NewSharePage';
import { renderWithProviders } from './render';

describe('Novo compartilhamento', () => {
  it('bloqueia caminho fora da allowlist', async () => {
    const user = userEvent.setup();
    renderWithProviders(<NewSharePage />);
    await screen.findByText('Identificação e acesso');
    await user.type(screen.getByPlaceholderText('Projetos'), 'Teste');
    await user.type(screen.getByPlaceholderText('Documentos de projetos institucionais'), 'Compartilhamento de teste');
    await user.type(screen.getByPlaceholderText('/srv/dados/projetos'), '/etc');
    await user.selectOptions(screen.getByLabelText('Usuário ou grupo permitido'), 'EBSERHNET\\GLO-SEC-HUUFCAT-FS-ADM');
    await user.click(screen.getByRole('button', { name: 'Validar e gerar prévia' }));
    expect(await screen.findByText('O caminho deve estar em /srv/dados/')).toBeInTheDocument();
  });

  it('gera prévia do smb4.conf para entrada válida', async () => {
    const user = userEvent.setup();
    renderWithProviders(<NewSharePage />);
    await screen.findByText('Identificação e acesso');
    await user.type(screen.getByPlaceholderText('Projetos'), 'Pesquisa');
    await user.type(screen.getByPlaceholderText('Documentos de projetos institucionais'), 'Dados de pesquisa institucional');
    await user.type(screen.getByPlaceholderText('/srv/dados/projetos'), '/srv/dados/pesquisa');
    await user.selectOptions(screen.getByLabelText('Usuário ou grupo permitido'), 'EBSERHNET\\GLO-SEC-HUUFCAT-FS-ADM');
    await user.click(screen.getByRole('button', { name: 'Validar e gerar prévia' }));
    expect(await screen.findByText('Prévia antes da aplicação')).toBeInTheDocument();
    expect(screen.getAllByText(/\[Pesquisa\]/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/Loaded services file OK/)).toBeInTheDocument();
  });
});
