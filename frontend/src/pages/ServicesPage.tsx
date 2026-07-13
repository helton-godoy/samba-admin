import { useMutation, useQuery } from '@tanstack/react-query';
import { Activity, Pause, Play, RefreshCw, RotateCw } from 'lucide-react';
import { api } from '../api/client';
import { useCapabilityGate } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

export function ServicesPage() {
  const query = useQuery({ queryKey: ['services'], queryFn: api.services });
  const serviceCapability = useCapabilityGate('samba.service.manage');
  const action = useMutation({ mutationFn: ({ id, actionName }: { id: string; actionName: string }) => api.serviceAction(id, actionName) });
  if (query.isLoading) return <Loading />;
  if (query.error) return <ErrorState error={query.error} message="Falha ao listar servicos." />;

  const execute = (id: string, actionName: string) => {
    if (!serviceCapability.available) return;
    action.mutate({ id, actionName });
  };

  return <>
    <PageHeader title="Serviços" description="Administração condicionada ao perfil operacional instalado." />
    <Panel title="Estado dos serviços" subtitle="Paradas e reinicializações exibem impacto antes da confirmação">
      {!serviceCapability.available && <div className="alert alert-warning"><strong>Operacoes de servico bloqueadas.</strong> {serviceCapability.reason}</div>}
      <div className="service-grid">{query.data!.map((service) => <article className="service-card" key={service.id}>
        <div className="service-card__title"><div><Activity size={18}/><strong>{service.name}</strong></div><StatusBadge state={service.state}/></div>
        <p>{service.description}</p>
        <dl><div><dt>PID</dt><dd>{service.pid || '—'}</dd></div><div><dt>Inicialização</dt><dd>{service.enabledAtBoot ? 'Automática' : 'Manual'}</dd></div></dl>
        <small>{service.lastMessage}</small>
        <div className="icon-actions service-actions">
          <button title={`Iniciar. ${serviceCapability.reason}`} disabled={!serviceCapability.available || action.isPending} onClick={() => execute(service.id, 'start')}><Play size={15}/></button>
          <button title={`Parar. ${serviceCapability.reason}`} disabled={!serviceCapability.available || action.isPending} onClick={() => execute(service.id, 'stop')}><Pause size={15}/></button>
          <button title={`Recarregar. ${serviceCapability.reason}`} disabled={!serviceCapability.available || action.isPending} onClick={() => execute(service.id, 'reload')}><RefreshCw size={15}/></button>
          <button title={`Reiniciar. ${serviceCapability.reason}`} disabled={!serviceCapability.available || action.isPending} onClick={() => execute(service.id, 'restart')}><RotateCw size={15}/></button>
        </div>
      </article>)}</div>
      {action.isPending && <Loading label="Criando operacao controlada..." />}
      {action.error && <ErrorState error={action.error} />}
      {action.data && <div className="alert alert-warning">Ação <strong>{action.data.action}</strong> aceita. {action.data.warning || 'Uma verificação de saúde será executada.'}</div>}
    </Panel>
  </>;
}
