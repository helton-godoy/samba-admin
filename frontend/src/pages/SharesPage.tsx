import { useQuery } from '@tanstack/react-query';
import { Copy, Eye, Plus, Search, ShieldCheck } from 'lucide-react';
import { useState } from 'react';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';

export function SharesPage() {
  const query = useQuery({ queryKey: ['shares'], queryFn: api.shares });
  const [filter, setFilter] = useState('');
  if (query.isLoading) return <Loading />;
  if (query.error) return <ErrorState error={query.error} message="Falha ao listar compartilhamentos." />;
  if (!query.data) return <ErrorState message="A API não retornou o contrato legado de compartilhamentos." />;
  const rows = query.data.filter((share) => `${share.name} ${share.path} ${share.description}`.toLowerCase().includes(filter.toLowerCase()));
  return <>
    <PageHeader title="Compartilhamentos Samba" description="Contrato legado preservado; ainda não convertido para inventário real pelo parser Samba." actions={<button className="button" disabled title={READ_ONLY_RELEASE_REASON}><Plus size={17} /> Novo compartilhamento</button>} />
    <div className="alert alert-warning"><strong>Alterações bloqueadas.</strong> {READ_ONLY_RELEASE_REASON}</div>
    <Panel title="Compartilhamentos configurados" subtitle="SMB1 e guest permanecem desabilitados por padrão" actions={<label className="search-box"><Search size={16}/><input value={filter} onChange={(e)=>setFilter(e.target.value)} placeholder="Buscar compartilhamento" /></label>}>
      <div className="table-scroll"><table>
        <thead><tr><th>Nome</th><th>Caminho</th><th>ACL</th><th>Acesso</th><th>Criptografia</th><th>VFS</th><th>Estado</th><th>Ações</th></tr></thead>
        <tbody>{rows.map((share)=><tr key={share.id}>
          <td><strong>{share.name}</strong><div className="muted">{share.description}</div></td>
          <td><code>{share.path}</code></td><td>{share.aclModel.toUpperCase()}</td>
          <td>{share.readOnly?'Somente leitura':'Leitura e gravação'}<div className="muted">Guest: {share.guestAccess?'sim':'não'}</div></td>
          <td>{share.encryption === 'required' ? 'Obrigatória' : 'Desejada'}</td>
          <td>{share.vfsModules.map((module)=><span className="mono-pill" key={module}>{module}</span>)}</td>
          <td><span className={`status-badge ${share.enabled?'status-success':'status-stopped'}`}>{share.enabled?'Ativo':'Desabilitado'}</span></td>
          <td><div className="icon-actions"><button title="Visualizar configuração"><Eye size={16}/></button><button disabled title={`Duplicar. ${READ_ONLY_RELEASE_REASON}`}><Copy size={16}/></button><button title="Validar sem aplicar"><ShieldCheck size={16}/></button></div></td>
        </tr>)}</tbody>
      </table></div>
    </Panel>
  </>;
}
