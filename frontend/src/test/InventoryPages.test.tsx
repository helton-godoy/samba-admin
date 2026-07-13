import { HttpResponse, http } from 'msw';
import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { server } from '../mocks/server';
import { PrintingPage } from '../pages/PrintingPage';
import { SambaPage } from '../pages/SambaPage';
import { renderWithProviders } from './render';

describe('Inventários Samba e CUPS', () => {
  it('exibe o inventário Samba estruturado sem oferecer mutações', async () => {
    renderWithProviders(<SambaPage />);

    expect(await screen.findByRole('heading', { name: 'Inventário Samba' })).toBeInTheDocument();
    expect(screen.getByText('samba423-4.23.2 (simulado)')).toBeInTheDocument();
    expect(screen.getByText('RC1 somente leitura.')).toBeInTheDocument();
    expect(screen.getByText('Assistencial, Administrativo, print$')).toBeInTheDocument();
  });

  it('integra o contrato CUPS e preserva as filas detalhadas legadas', async () => {
    renderWithProviders(<PrintingPage />);

    expect(await screen.findByRole('heading', { name: 'Servidor de impressão' })).toBeInTheDocument();
    expect(screen.getByText('cups-2.4.11 (simulado)')).toBeInTheDocument();
    expect(screen.getByText('HUUFCAT-ADM-COR-01')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Nova impressora' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Validar e gerar configuração' })).toBeDisabled();
  });

  it('apresenta o 503 CUPS com identificador de correlação', async () => {
    server.use(http.get('/api/v1/cups', () => HttpResponse.json({
      type: 'capability_unavailable',
      title: 'Inventário CUPS indisponível',
      status: 503,
      detail: 'A capability CUPS não está disponível no agente.',
      correlationId: 'corr-teste-cups-503'
    }, { status: 503, headers: { 'Content-Type': 'application/problem+json' } })));

    renderWithProviders(<PrintingPage />);

    expect(await screen.findByText('Capability indisponivel neste host.')).toBeInTheDocument();
    expect(screen.getByText('A capability CUPS não está disponível no agente.')).toBeInTheDocument();
    expect(screen.getByText('corr-teste-cups-503')).toBeInTheDocument();
  });
});
