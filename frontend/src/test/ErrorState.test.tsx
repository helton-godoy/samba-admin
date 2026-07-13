import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import { ErrorState } from '../components/Loading';

function error(status: number, code: string) {
  return new ApiError({ message: 'Detalhe seguro para o operador.', status, code });
}

describe('Estado de erro HTTP', () => {
  it.each<[number, string, string]>([
    [403, 'permission_denied', 'Permissao insuficiente.'],
    [403, 'csrf_invalid', 'Protecao CSRF recusada.'],
    [503, 'agent_unavailable', 'Agente privilegiado indisponivel.'],
    [503, 'capability_unavailable', 'Capability indisponivel neste host.'],
    [423, 'resource_locked', 'Recurso bloqueado por outra operacao.']
  ])('distingue HTTP %s com código %s', (status, code, heading) => {
    render(<ErrorState error={error(status, code)} />);
    expect(screen.getByText(heading)).toBeInTheDocument();
  });
});
