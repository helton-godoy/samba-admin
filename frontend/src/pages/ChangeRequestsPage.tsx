import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Play, ShieldAlert, X } from 'lucide-react';
import { useState, type FormEvent } from 'react';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON } from '../capabilities';
import type { ChangeRequest, ChangeRequestInput } from '../api/generated';
import { useAuth } from '../auth/AuthProvider';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

const requiresSegregation = (change: ChangeRequest) => ['high', 'destructive', 'emergency'].includes(change.risk);

function toISO(value: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

export function ChangeRequestsPage() {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [decisionReason, setDecisionReason] = useState('Revisão técnica concluída; impacto e rollback conferidos.');
  const [form, setForm] = useState({
    operation: 'samba.service.restart',
    resource: 'service:smbd',
    justification: 'Aplicar a mudança durante a janela aprovada pela infraestrutura.',
    impact: 'Pode interromper temporariamente sessões SMB e arquivos abertos.',
    rollbackPlan: 'Restaurar a versão anterior e validar a saúde dos serviços.',
    maintenanceStart: '',
    maintenanceEnd: ''
  });

  const changes = useQuery({ queryKey: ['change-requests'], queryFn: () => api.changeRequests() });
  const create = useMutation({
    mutationFn: (input: ChangeRequestInput) => api.createChangeRequest(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['change-requests'] })
  });
  const decide = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'approve' | 'reject' }) => decision === 'approve'
      ? api.approveChangeRequest(id, decisionReason)
      : api.rejectChangeRequest(id, decisionReason),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['change-requests'] })
  });
  const execute = useMutation({
    mutationFn: (id: string) => api.executeChangeRequest(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['change-requests'] })
  });

  const submit = (event: FormEvent) => {
    event.preventDefault();
    create.mutate({
      operation: form.operation,
      resource: form.resource,
      justification: form.justification,
      impact: form.impact,
      rollbackPlan: form.rollbackPlan,
      maintenanceStart: toISO(form.maintenanceStart),
      maintenanceEnd: toISO(form.maintenanceEnd),
      payload: { resource: form.resource }
    });
  };

  if (changes.isLoading) return <Loading label="Carregando solicitações de mudança..." />;
  if (changes.error) return <ErrorState message="Falha ao listar solicitações de mudança." error={changes.error} />;

  return <>
    <PageHeader title="Aprovação de mudanças" description="Solicitação, decisão segregada, janela de manutenção e execução rastreável como tarefa." />
    <Panel title="Nova solicitação" subtitle="A API recalcula o risco e aplica a política; o valor enviado pelo navegador não é confiado.">
      <form onSubmit={submit}>
        <div className="form-grid">
          <label>Operação<select value={form.operation} onChange={(event) => setForm({ ...form, operation: event.target.value })}>
            <option value="samba.service.restart">Reiniciar Samba</option>
            <option value="share.delete" disabled>Excluir compartilhamento (mutação bloqueada)</option>
            <option value="domain.member.join" disabled>Ingressar como membro AD (homologação pendente)</option>
          </select></label>
          <label>Recurso<input required minLength={3} value={form.resource} onChange={(event) => setForm({ ...form, resource: event.target.value })} /></label>
          <label className="span-2">Justificativa<textarea required minLength={10} value={form.justification} onChange={(event) => setForm({ ...form, justification: event.target.value })} /></label>
          <label className="span-2">Impacto<textarea required minLength={10} value={form.impact} onChange={(event) => setForm({ ...form, impact: event.target.value })} /></label>
          <label className="span-2">Plano de rollback<textarea required minLength={10} value={form.rollbackPlan} onChange={(event) => setForm({ ...form, rollbackPlan: event.target.value })} /></label>
          <label>Início da janela<input type="datetime-local" value={form.maintenanceStart} onChange={(event) => setForm({ ...form, maintenanceStart: event.target.value })} /></label>
          <label>Fim da janela<input type="datetime-local" value={form.maintenanceEnd} onChange={(event) => setForm({ ...form, maintenanceEnd: event.target.value })} /></label>
        </div>
        <button className="button" type="submit" disabled={create.isPending}>Solicitar aprovação</button>
      </form>
      {create.isPending && <Loading label="Registrando solicitação auditável..." />}
      {create.data && <div className="alert alert-success">Solicitação <code>{create.data.id}</code> criada com risco <strong>{create.data.risk}</strong>.</div>}
      {create.error && <ErrorState error={create.error} />}
    </Panel>
    <Panel title="Fila de decisão" subtitle="Solicitante e aprovador devem ser pessoas distintas quando a política exigir segregação.">
      <label>Fundamentação da decisão<input value={decisionReason} minLength={5} onChange={(event) => setDecisionReason(event.target.value)} /></label>
      <div className="table-scroll"><table>
        <thead><tr><th>ID</th><th>Operação</th><th>Risco</th><th>Solicitante</th><th>Janela</th><th>Estado</th><th>Ações</th></tr></thead>
        <tbody>{changes.data!.map((change) => {
          const selfApprovalBlocked = requiresSegregation(change)
            && (change.requestedBy === user?.id || change.requestedBy === user?.username);
          return <tr key={change.id}>
            <td><code>{change.id}</code><div className="muted">{change.resource}</div></td>
            <td>{change.operation}<div className="muted">{change.justification}</div></td>
            <td>{change.risk}</td>
            <td>{change.requestedBy}</td>
            <td>{change.maintenanceStart ? new Date(change.maintenanceStart).toLocaleString('pt-BR') : 'Não definida'}</td>
            <td><StatusBadge state={change.status} /></td>
            <td><div className="job-actions">
              {change.status === 'pending_approval' && <>
                <button className="button button-secondary button-small" type="button" aria-label={`Aprovar ${change.id}`} title={selfApprovalBlocked ? 'Autoaprovação bloqueada pela segregação de funções.' : 'Aprovar solicitação'} disabled={selfApprovalBlocked || decisionReason.trim().length < 5 || decide.isPending} onClick={() => decide.mutate({ id: change.id, decision: 'approve' })}><Check size={14} /> Aprovar</button>
                <button className="button button-secondary button-small" type="button" aria-label={`Rejeitar ${change.id}`} disabled={decisionReason.trim().length < 5 || decide.isPending} onClick={() => decide.mutate({ id: change.id, decision: 'reject' })}><X size={14} /> Rejeitar</button>
              </>}
              {change.status === 'approved' && <button className="button button-small" type="button" aria-label={`Executar ${change.id}`} disabled title={READ_ONLY_RELEASE_REASON}><Play size={14} /> Executar</button>}
              {selfApprovalBlocked && <span className="muted"><ShieldAlert size={13} /> Aprovador distinto obrigatório</span>}
            </div></td>
          </tr>;
        })}</tbody>
      </table></div>
      {decide.error && <ErrorState error={decide.error} />}
      {execute.error && <ErrorState error={execute.error} />}
      {execute.data && <div className="alert alert-success">Execução enviada como tarefa <code>{execute.data.id}</code>.</div>}
    </Panel>
  </>;
}
