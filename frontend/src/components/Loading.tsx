import type { ReactNode } from 'react';
import { isApiError } from '../api/client';

export function Loading({ label = 'Carregando dados...' }: { label?: string }) {
  return <div className="loading" role="status"><span className="spinner" />{label}</div>;
}

export function ErrorState({
  message,
  error,
  action
}: {
  message?: string;
  error?: unknown;
  action?: ReactNode;
}) {
  const apiError = isApiError(error) ? error : undefined;
  const detail = apiError?.message || message || 'Nao foi possivel concluir a operacao.';
  const errorCode = apiError?.code?.toLowerCase() || '';
  const csrfDenied = errorCode.startsWith('csrf') || errorCode.includes('csrf_') || errorCode.includes('csrf-');
  const capabilityUnavailable = errorCode.includes('capability_unavailable') || errorCode.includes('capability-unavailable');
  const agentUnavailable = errorCode.includes('agent_unavailable') || errorCode.includes('agent-unavailable');
  const heading = apiError?.status === 401
    ? 'Sessao expirada ou nao autenticada.'
    : apiError?.status === 403
      ? csrfDenied
        ? 'Protecao CSRF recusada.'
        : 'Permissao insuficiente.'
      : apiError?.status === 409
        ? 'Conflito operacional detectado.'
        : apiError?.status === 423
          ? 'Recurso bloqueado por outra operacao.'
        : apiError?.status === 422
          ? 'Os dados enviados precisam de correcao.'
          : apiError?.status === 503
            ? capabilityUnavailable
              ? 'Capability indisponivel neste host.'
              : agentUnavailable
                ? 'Agente privilegiado indisponivel.'
                : 'Servico em manutencao ou indisponivel.'
            : 'Nao foi possivel concluir a operacao.';

  return <div className="alert alert-danger" role="alert">
    <strong>{heading}</strong><br />
    {detail}
    {apiError?.fieldErrors && apiError.fieldErrors.length > 0 && (
      <ul className="error-list">
        {apiError.fieldErrors.map((fieldError) => <li key={`${fieldError.field}-${fieldError.code}`}>{fieldError.field}: {fieldError.message}</li>)}
      </ul>
    )}
    {apiError?.retryAfterSeconds !== undefined && <div className="muted">Tente novamente em aproximadamente {apiError.retryAfterSeconds}s.</div>}
    {apiError?.correlationId && <div className="technical-detail">Correlacao: <code>{apiError.correlationId}</code></div>}
    {action && <div className="error-action">{action}</div>}
  </div>;
}
