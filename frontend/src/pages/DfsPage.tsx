import { useQuery } from '@tanstack/react-query';
import { GitBranch, Network, Plus } from 'lucide-react';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

export function DfsPage(){const q=useQuery({queryKey:['dfs'],queryFn:api.dfs});if(q.isLoading)return <Loading/>;if(q.error)return <ErrorState error={q.error} message="Falha ao consultar o inventário DFS."/>;if(!q.data)return <ErrorState message="A API não retornou o inventário DFS."/>;return <>
  <PageHeader title="DFS Namespace" description="Namespaces, links, referrals e destinos; não representa replicação física de dados." actions={<button className="button" disabled title={READ_ONLY_RELEASE_REASON}><Plus size={16}/> Nova raiz</button>}/>
  <div className="alert alert-warning"><Network size={18}/><strong>DFS Namespace não é DFS-R.</strong> Referrals distribuem caminhos e destinos, mas não sincronizam arquivos automaticamente.</div>
  <div className="alert alert-warning"><strong>Alterações bloqueadas.</strong> {READ_ONLY_RELEASE_REASON}</div>
  {q.data!.map((root)=><Panel key={root.id} title={`Raiz ${root.name}`} subtitle={root.unc} actions={<StatusBadge state={root.enabled?'saudavel':'critico'}/>}>
    <div className="table-scroll"><table><thead><tr><th>Link</th><th>Caminho UNC</th><th>Destinos</th><th>Referral</th></tr></thead><tbody>{root.links.map((link)=><tr key={link.name}><td><GitBranch size={16}/> {link.name}</td><td><code>{link.path}</code></td><td>{link.targets.map((t)=><div key={t}><code>{t}</code></div>)}</td><td><StatusBadge state={link.health}/></td></tr>)}</tbody></table></div>
  </Panel>)}
  <Panel title="Criar link DFS" subtitle={READ_ONLY_RELEASE_REASON}><fieldset disabled><div className="form-grid"><label>Nome do link<input placeholder="Nome do link"/></label><label>Raiz<select><option>Raiz detectada</option></select></label><label className="span-2">Destino UNC<input placeholder="\\SERVIDOR\COMPARTILHAMENTO"/></label><label>Prioridade<select><option>Normal</option><option>Primeiro entre iguais</option></select></label><label>Estado<select><option>Habilitado</option><option>Desabilitado</option></select></label></div></fieldset><button className="button" disabled title={READ_ONLY_RELEASE_REASON}>Testar referral</button></Panel>
</>}
