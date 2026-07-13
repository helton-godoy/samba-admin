import { useQuery } from '@tanstack/react-query';
import { Boxes, FileCheck2, Network, ShieldCheck } from 'lucide-react';
import { api } from '../api/client';
import { useCapabilityGate } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

function availabilityState(value: boolean) {
  return value ? 'suportado' : 'indisponivel';
}

export function SambaPage() {
  const samba = useQuery({ queryKey: ['samba'], queryFn: api.samba });
  const inspectCapability = useCapabilityGate('samba.inspect');

  if (samba.isLoading) return <Loading label="Consultando o inventário Samba pelo agente..." />;
  if (samba.error) return <ErrorState error={samba.error} message="Falha ao consultar o inventário Samba." />;
  if (!samba.data) return <ErrorState message="O agente não retornou o inventário Samba." />;

  const info = samba.data;
  return <>
    <PageHeader title="Inventário Samba" description="Informações somente leitura coletadas exclusivamente pela API por meio do agente privilegiado." />

    <div className="alert alert-info"><ShieldCheck size={18} /><strong>RC1 somente leitura.</strong> Esta tela não altera smb4.conf, serviços, compartilhamentos, domínio ou ACLs.</div>
    {!inspectCapability.available && <div className="alert alert-warning"><strong>Capability de inventário indisponível.</strong> {inspectCapability.reason}</div>}

    <div className="two-column">
      <Panel title="Instalação detectada" subtitle="Pacote e binários identificados no host">
        <dl className="definition-grid">
          <div><dt>Versão</dt><dd>{info.version || 'Não detectada'}</dd></div>
          <div><dt>Pacote</dt><dd>{info.package || 'Não detectado'}</dd></div>
          <div><dt>Origem</dt><dd>{info.origin || 'Não informada'}</dd></div>
          <div><dt>Perfil</dt><dd>{info.profile || 'Não detectado'}</dd></div>
          <div><dt>Configuração</dt><dd><code>{info.configurationPath || 'Não detectada'}</code></dd></div>
          <div><dt>testparm</dt><dd><StatusBadge state={availabilityState(info.testparmAvailable)} /></dd></div>
        </dl>
        <h3><Boxes size={18} /> Binários inventariados</h3>
        {info.binaries.length > 0 ? <ul>{info.binaries.map((binary) => <li key={binary}><code>{binary}</code></li>)}</ul> : <p className="muted">Nenhum binário foi comprovado pelo provider.</p>}
      </Panel>

      <Panel title="Estado operacional" subtitle="Contadores e integrações detectados sem modificar o host">
        <dl className="definition-grid">
          <div><dt>Sessões SMB</dt><dd>{info.sessions}</dd></div>
          <div><dt>Arquivos abertos</dt><dd>{info.openFiles}</dd></div>
          <div><dt>Compartilhamentos detectados</dt><dd>{info.shares.length}</dd></div>
          <div><dt>full_audit</dt><dd><StatusBadge state={availabilityState(info.fullAuditAvailable)} /></dd></div>
          <div><dt>DFS</dt><dd><StatusBadge state={availabilityState(info.dfsAvailable)} /></dd></div>
          <div><dt>Impressão</dt><dd><StatusBadge state={availabilityState(info.printingAvailable)} /></dd></div>
        </dl>
        <h3><Network size={18} /> Nomes detectados pelo parser</h3>
        <p>{info.shares.length > 0 ? info.shares.join(', ') : 'Nenhum nome de compartilhamento detectado.'}</p>
        <p className="muted">A lista acima não substitui o contrato detalhado legado de compartilhamentos.</p>
      </Panel>
    </div>

    <Panel title="Recursos compilados e módulos" subtitle="Evidência de inventário; não representa autorização para operações mutáveis">
      <div className="two-column">
        <div><h3><FileCheck2 size={18} /> Opções do pacote</h3>{info.packageBuildOptions.length > 0 ? <ul>{info.packageBuildOptions.map((option) => <li key={option}><code>{option}</code></li>)}</ul> : <p className="muted">Opções de build não disponíveis.</p>}</div>
        <div><h3><Boxes size={18} /> Módulos VFS</h3>{info.vfsModules.length > 0 ? <ul>{info.vfsModules.map((module) => <li key={module}><code>{module}</code></li>)}</ul> : <p className="muted">Nenhum módulo VFS detectado.</p>}</div>
      </div>
    </Panel>
  </>;
}
