import { useQuery } from '@tanstack/react-query';
import { api } from './api/client';
import type { Capability, CapabilityState } from './api/generated';

export const READ_ONLY_RELEASE_REASON = 'Operações mutáveis permanecem bloqueadas no RC1 somente leitura.';

const legacyIds: Record<string, string[]> = {
  'acl.nfsv4.read': ['cap-nfsv4', 'cap-nfsv4-ufs'],
  'samba.testparm': ['cap-smb3'],
  'domain.member.diagnose': ['cap-member', 'cap-ad-member'],
  'samba.ad_dc': ['cap-ad-dc'],
  'cups.inspect': ['cap-print'],
  'syslog.tls': ['cap-syslog-tls'],
  'samba.service.manage': ['cap-smb3']
};

export interface CapabilityGate {
  id: string;
  capability?: Capability;
  state: CapabilityState | 'loading' | 'error';
  available: boolean;
  reason: string;
}

function operationAvailable(state: CapabilityState): boolean {
  return state === 'suportado'
    || state === 'restrito'
    || state === 'suportado_com_restricoes'
    || state === 'supported'
    || state === 'supported_with_restrictions';
}

export function resolveCapability(capabilities: Capability[] | undefined, id: string): CapabilityGate {
  const exact = capabilities?.find((capability) => capability.id === id);
  const legacy = capabilities?.find((capability) => legacyIds[id]?.includes(capability.id));
  const capability = exact || legacy;

  if (!capability) {
    return {
      id,
      state: 'indisponivel',
      available: false,
      reason: `A capability ${id} ainda nao foi comprovada por este host.`
    };
  }

  const legacyNotice = !exact ? ' Evidencia recebida por capability legada; confirme a deteccao especifica antes de liberar producao.' : '';
  return {
    id,
    capability,
    state: capability.state,
    available: operationAvailable(capability.state),
    reason: `${capability.evidence}${legacyNotice}`
  };
}

export function useCapabilityGate(id: string): CapabilityGate {
  const query = useQuery({ queryKey: ['capabilities'], queryFn: api.capabilities, staleTime: 15_000 });
  if (query.isLoading) {
    return { id, state: 'loading', available: false, reason: 'Verificando capabilities do host.' };
  }
  if (query.error) {
    return { id, state: 'error', available: false, reason: 'Nao foi possivel verificar a capability exigida.' };
  }
  return resolveCapability(query.data, id);
}
