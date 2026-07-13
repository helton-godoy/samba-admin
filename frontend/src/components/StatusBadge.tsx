import type { CapabilityState, HealthState, JobStatus } from '../types';

type ChangeStatus = 'pending_approval' | 'approved' | 'rejected' | 'executing' | 'completed' | 'rolled_back' | 'expired';
type State = CapabilityState | HealthState | JobStatus | ChangeStatus | 'running' | 'stopped' | 'degraded' | 'ALLOW' | 'DENY';

const labels: Record<string, string> = {
  suportado: 'Suportado',
  supported: 'Suportado',
  restrito: 'Suportado com restrições',
  suportado_com_restricoes: 'Suportado com restrições',
  supported_with_restrictions: 'Suportado com restrições',
  experimental: 'Experimental',
  indisponivel: 'Indisponível',
  unavailable: 'Indisponível',
  nao_verificado: 'Ainda não verificado',
  not_verified: 'Ainda não verificado',
  saudavel: 'Saudável',
  atencao: 'Atenção',
  critico: 'Crítico',
  desconhecido: 'Desconhecido',
  queued: 'Na fila',
  validating: 'Validando',
  running: 'Em execução',
  testing: 'Em teste',
  success: 'Concluída',
  partial: 'Falha parcial',
  failed: 'Falhou',
  'rolled-back': 'Revertida',
  rolled_back: 'Revertida',
  'cancellation-requested': 'Cancelamento solicitado',
  cancelled: 'Cancelada',
  pending_approval: 'Aguardando aprovação',
  approved: 'Aprovada',
  rejected: 'Rejeitada',
  executing: 'Em execução',
  completed: 'Concluída',
  expired: 'Expirada',
  stopped: 'Parado',
  degraded: 'Degradado',
  ALLOW: 'Permitir',
  DENY: 'Negar'
};

export function StatusBadge({ state }: { state: State }) {
  return <span className={`status-badge status-${state}`}>{labels[state] || state}</span>;
}
