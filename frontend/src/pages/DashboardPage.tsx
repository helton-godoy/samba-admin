import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, Cpu, Files, HardDrive, MemoryStick, Network, Printer, ShieldCheck } from 'lucide-react';
import { api } from '../api/client';
import { Loading, ErrorState } from '../components/Loading';
import { MetricCard } from '../components/MetricCard';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

export function DashboardPage() {
  const system = useQuery({ queryKey: ['system'], queryFn: api.system });
  const samba = useQuery({ queryKey: ['samba'], queryFn: api.samba });
  const cups = useQuery({ queryKey: ['cups'], queryFn: api.cups });
  const filesystems = useQuery({ queryKey: ['filesystems'], queryFn: api.filesystems });
  const capabilities = useQuery({ queryKey: ['capabilities'], queryFn: api.capabilities });
  const audit = useQuery({ queryKey: ['audit'], queryFn: api.audit });

  if (system.isLoading || samba.isLoading || cups.isLoading || filesystems.isLoading || capabilities.isLoading || audit.isLoading) return <Loading label="Consolidando inventários somente leitura..." />;
  const error = system.error || samba.error || cups.error || filesystems.error || capabilities.error || audit.error;
  if (error) return <ErrorState error={error} message="Falha ao consultar o estado do RC1." />;
  if (!system.data || !samba.data || !cups.data || !filesystems.data || !capabilities.data || !audit.data) return <ErrorState message="A API retornou inventários incompletos." />;
  const s = system.data;
  const detectedProfile = samba.data.profile || s.profile;
  const profileLabel: Record<string, string> = {
    standalone: 'Servidor independente',
    'domain-member': 'Membro de domínio',
    'ad-dc': 'Controlador de domínio',
    'additional-dc': 'Controlador adicional'
  };

  return (
    <>
      <PageHeader
        title="Painel"
        description="Visão consolidada dos inventários reais e das capabilities do host, sem habilitar mutações."
        actions={<button className="button button-secondary" onClick={() => Promise.all([system.refetch(), samba.refetch(), cups.refetch(), filesystems.refetch(), capabilities.refetch(), audit.refetch()])}>Atualizar estado</button>}
      />

      <div className="metrics-grid">
        <MetricCard title="Perfil operacional" value={profileLabel[detectedProfile] || detectedProfile || 'Não detectado'} detail={s.domainName || undefined} icon={<Network />} />
        <MetricCard title="Sessões SMB" value={samba.data.sessions} detail={`${samba.data.openFiles} arquivos abertos`} icon={<Files />} />
        <MetricCard title="CPU" value={`${s.cpuPercent}%`} detail="Carga instantânea informada pelo provider" icon={<Cpu />} />
        <MetricCard title="Memória" value={`${s.memoryPercent}%`} detail="Uso do sistema" icon={<MemoryStick />} />
        <MetricCard title="Armazenamento" value={`${s.diskPercent}%`} detail="Uso agregado" icon={<HardDrive />} />
        <MetricCard title="Filas de impressão" value={cups.data.printers.length} detail={cups.data.version || 'CUPS não identificado'} icon={<Printer />} />
      </div>

      <div className="two-column">
        <Panel title="Identidade e serviços essenciais" subtitle="Dados detectados pelos providers somente leitura">
          <dl className="definition-grid">
            <div><dt>Hostname</dt><dd>{s.hostname}</dd></div>
            <div><dt>FQDN</dt><dd>{s.fqdn}</dd></div>
            <div><dt>FreeBSD</dt><dd>{s.freebsdVersion}</dd></div>
            <div><dt>Samba</dt><dd>{samba.data.version || s.sambaVersion}</dd></div>
            <div><dt>Pacote</dt><dd>{samba.data.package || s.sambaPackage}</dd></div>
            <div><dt>Origem</dt><dd>{samba.data.origin || s.sambaOrigin}</dd></div>
            <div><dt>DC preferencial</dt><dd>{s.preferredDc || 'Não detectado'}</dd></div>
            <div><dt>Saúde geral</dt><dd><StatusBadge state={s.health} /></dd></div>
            <div><dt>Sincronização de horário</dt><dd><StatusBadge state={s.timeSync} /></dd></div>
            <div><dt>DNS</dt><dd><StatusBadge state={s.dnsHealth} /></dd></div>
            <div><dt>Encaminhamento de logs</dt><dd><StatusBadge state={s.logForwarding} /></dd></div>
          </dl>
        </Panel>

        <Panel title="Alertas operacionais" subtitle="Itens que exigem validação antes de uso em produção" tone="warning">
          <div className="alert-list">
            <div><AlertTriangle size={18} /><span>O suporte a AD DC ainda não foi validado contra as opções reais de compilação do pacote.</span></div>
            <div><AlertTriangle size={18} /><span>O transporte TLS de syslog depende de pacote ou agente adicional.</span></div>
            <div><AlertTriangle size={18} /><span>Há uma identidade sem mapeamento SID ↔ UID/GID estável.</span></div>
            <div><ShieldCheck size={18} /><span>SMB1 e acesso guest estão bloqueados pela política padrão.</span></div>
          </div>
        </Panel>
      </div>

      <Panel title="Sistemas de arquivos montados" subtitle="O formato de ACL é propriedade do ponto de montagem, não da pasta">
        <div className="table-scroll">
          <table>
            <thead><tr><th>Dispositivo</th><th>Montagem</th><th>Tipo</th><th>Uso</th><th>ACL ativa</th><th>Quotas</th><th>Compartilhamentos</th></tr></thead>
            <tbody>
              {filesystems.data.map((fs) => (
                <tr key={fs.id}>
                  <td><code>{fs.device}</code></td>
                  <td><code>{fs.mountPoint}</code></td>
                  <td>{fs.type.toUpperCase()}</td>
                  <td>{fs.sizeGiB > 0 ? `${Math.round((fs.usedGiB / fs.sizeGiB) * 100)}%` : '—'}</td>
                  <td><span className="mono-pill">{fs.aclModel === 'nfsv4' ? 'NFSv4' : fs.aclModel === 'posix' ? 'POSIX.1e' : 'Sem ACL estendida'}</span></td>
                  <td>{fs.userQuota || fs.groupQuota ? 'Ativas' : 'Desativadas'}</td>
                  <td>{fs.associatedShares.join(', ') || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Matriz de viabilidade detectada" subtitle="Estados conservadores; recursos não verificados permanecem bloqueados">
        <div className="table-scroll">
          <table>
            <thead><tr><th>Funcionalidade</th><th>Estado</th><th>Escopo</th><th>Evidência</th></tr></thead>
            <tbody>
              {capabilities.data.map((cap) => (
                <tr key={cap.id}><td>{cap.feature}</td><td><StatusBadge state={cap.state} /></td><td>{cap.scope}</td><td>{cap.evidence}</td></tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Eventos de auditoria recentes" subtitle="Conteúdo resumido; segredos e conteúdo de arquivos nunca são registrados">
        <div className="table-scroll">
          <table>
            <thead><tr><th>Data</th><th>Usuário</th><th>Operação</th><th>Resultado</th><th>Caminho</th><th>Correlação</th></tr></thead>
            <tbody>
              {audit.data.map((event) => (
                <tr key={event.id}>
                  <td>{new Date(event.timestamp).toLocaleString('pt-BR')}</td>
                  <td>{event.domain}\\{event.user}</td>
                  <td><code>{event.operation}</code></td>
                  <td><StatusBadge state={event.result === 'success' ? 'success' : 'failed'} /></td>
                  <td><code>{event.path}</code></td>
                  <td><code>{event.correlationId}</code></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>
    </>
  );
}
