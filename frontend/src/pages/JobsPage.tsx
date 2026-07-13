import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleOff, Radio, RotateCcw, WifiOff } from 'lucide-react';
import { useEffect } from 'react';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON } from '../capabilities';
import type { Job } from '../api/generated';
import { useJobEvents } from '../hooks/useJobEvents';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

function upsertJob(current: Job[] | undefined, job: Job): Job[] {
  const previous = current || [];
  const index = previous.findIndex((item) => item.id === job.id);
  if (index < 0) return [job, ...previous];
  const next = [...previous];
  next[index] = { ...next[index], ...job };
  return next;
}

export function JobsPage() {
  const queryClient = useQueryClient();
  const stream = useJobEvents({
    onJobEvent: ({ job }) => {
      queryClient.setQueryData<Job[]>(['jobs'], (current) => upsertJob(current, job));
    }
  });
  const query = useQuery({
    queryKey: ['jobs'],
    queryFn: api.jobs,
    refetchInterval: stream.shouldPoll ? 5_000 : false
  });
  const cancel = useMutation({
    mutationFn: (id: string) => api.cancelJob(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['jobs'] })
  });
  const rollback = useMutation({
    mutationFn: (id: string) => api.rollbackJob(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['jobs'] })
  });

  useEffect(() => {
    if (stream.connectionState === 'connected') {
      void queryClient.invalidateQueries({ queryKey: ['jobs'] });
    }
  }, [queryClient, stream.connectionState]);

  if (query.isLoading) return <Loading label="Carregando tarefas persistentes..." />;
  if (query.error) return <ErrorState error={query.error} message="Falha ao listar tarefas." />;

  const streamDetail = stream.connectionState === 'connected'
    ? 'Atualizacao em tempo real conectada.'
    : stream.connectionState === 'reconnecting'
      ? `Reconectando ao fluxo de eventos (tentativa ${stream.reconnectAttempt})...`
      : stream.shouldPoll
        ? 'SSE indisponivel; atualizacao por polling a cada cinco segundos.'
        : 'Conectando ao fluxo de eventos...';

  return <>
    <PageHeader title="Central de tarefas" description="Acompanhamento de validacao, backup, aplicacao, health check, cancelamento e rollback." />
    <Panel title="Historico de operacoes" subtitle={streamDetail}>
      <div className={`stream-status stream-${stream.connectionState}`}>
        {stream.shouldPoll ? <WifiOff size={16} /> : <Radio size={16} />}
        <span>{streamDetail}</span>
      </div>
      <div className="table-scroll"><table>
        <thead><tr><th>ID</th><th>Solicitante</th><th>Operacao</th><th>Progresso</th><th>Resultado</th><th>Resumo</th><th>Acao</th></tr></thead>
        <tbody>{query.data!.map((job) => <tr key={job.id}>
          <td><code>{job.id}</code><div className="muted">{new Date(job.requestedAt).toLocaleString('pt-BR')}</div></td>
          <td>{job.requestedBy}</td>
          <td>{job.operation}</td>
          <td><div className="progress"><span style={{ width: `${job.progress}%` }} /></div>{job.progress}%</td>
          <td><StatusBadge state={job.status} /></td>
          <td>{job.summary}{job.error && <div className="field-error">{job.error}</div>}</td>
          <td><div className="job-actions">
            {job.cancellable && <button className="button button-secondary button-small" disabled title={READ_ONLY_RELEASE_REASON}><CircleOff size={14} /> Cancelar</button>}
            {job.rollbackAvailable && <button className="button button-secondary button-small" disabled title={READ_ONLY_RELEASE_REASON}><RotateCcw size={14} /> Rollback</button>}
          </div></td>
        </tr>)}</tbody>
      </table></div>
      {cancel.isPending && <Loading label="Solicitando cancelamento controlado..." />}
      {rollback.isPending && <Loading label="Restaurando backup e verificando saude..." />}
      {cancel.error && <ErrorState error={cancel.error} />}
      {rollback.error && <ErrorState error={rollback.error} />}
      {cancel.data && <div className="alert alert-warning">Cancelamento solicitado para <code>{cancel.data.jobId}</code>. O estado sera confirmado pela tarefa.</div>}
      {rollback.data && <div className="alert alert-success">{rollback.data.message}</div>}
    </Panel>
  </>;
}
