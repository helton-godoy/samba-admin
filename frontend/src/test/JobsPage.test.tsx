import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { JobsPage } from '../pages/JobsPage';
import { renderWithProviders } from './render';

describe('Central de tarefas', () => {
  it('usa polling de contingência quando EventSource está indisponível', async () => {
    vi.stubGlobal('EventSource', undefined);
    renderWithProviders(<JobsPage />);

    expect(await screen.findAllByText(/SSE indisponivel; atualizacao por polling/i)).not.toHaveLength(0);
    expect(screen.getByText('JOB-2026-0711-0042')).toBeInTheDocument();
    vi.unstubAllGlobals();
  });
});
