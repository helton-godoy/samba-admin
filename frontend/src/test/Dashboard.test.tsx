import { screen } from '@testing-library/react';
import { HttpResponse, http } from 'msw';
import { describe, expect, it } from 'vitest';
import { sambaInfo, systemInfo } from '../mocks/data';
import { server } from '../mocks/server';
import { DashboardPage } from '../pages/DashboardPage';
import { renderWithProviders } from './render';

describe('Dashboard', () => {
  it('exibe identidade do servidor e matriz de viabilidade', async () => {
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText('CAT-VP-FS01')).toBeInTheDocument();
    expect(screen.getByText('Matriz de viabilidade detectada')).toBeInTheDocument();
    expect(screen.getByText('ACL NFSv4 em UFS2')).toBeInTheDocument();
    expect(screen.getByText('Replicação DFS-R')).toBeInTheDocument();
  });

  it('deriva o perfil operacional do inventário em vez de fixar membro de domínio', async () => {
    server.use(
      http.get('/api/v1/system', () => HttpResponse.json({ ...systemInfo, profile: 'standalone', domainName: null })),
      http.get('/api/v1/samba', () => HttpResponse.json({ ...sambaInfo, profile: 'standalone' }))
    );

    renderWithProviders(<DashboardPage />);
    expect(await screen.findByText('Servidor independente')).toBeInTheDocument();
    expect(screen.queryByText('Membro de domínio')).not.toBeInTheDocument();
  });
});
