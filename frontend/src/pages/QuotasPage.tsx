import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, Plus } from 'lucide-react';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON, useCapabilityGate } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

export function QuotasPage(){const q=useQuery({queryKey:['quotas'],queryFn:api.quotas});const writeCapability=useCapabilityGate('quota.ufs.write');if(q.isLoading)return <Loading/>;if(q.error)return <ErrorState error={q.error} message="Falha ao consultar o inventário de cotas."/>;if(!q.data)return <ErrorState message="A API não retornou o inventário de cotas."/>;const writeReason=writeCapability.reason || READ_ONLY_RELEASE_REASON;return <>
  <PageHeader title="Cotas UFS" description="Limites por usuário ou grupo vinculados a mapeamentos de identidade estáveis." actions={<button className="button" disabled title={writeReason}><Plus size={16}/> Nova cota</button>}/>
  <div className="alert alert-warning"><strong>Escrita de cotas indisponível.</strong> {writeReason}</div>
  <Panel title="Consumo e limites" subtitle="Cotas de domínio ficam bloqueadas quando SID ↔ UID/GID não é determinístico">
    <div className="table-scroll"><table><thead><tr><th>Principal</th><th>Tipo</th><th>Uso</th><th>Flexível</th><th>Rígido</th><th>Arquivos</th><th>Tolerância</th><th>Mapeamento</th></tr></thead><tbody>{q.data!.map((quota)=>{const pct=Math.round((quota.usedGiB/quota.hardGiB)*100);return <tr key={quota.id}><td>{quota.principal}</td><td>{quota.kind==='group'?'Grupo':'Usuário'}</td><td><div className="progress"><span style={{width:`${Math.min(pct,100)}%`}}/></div>{quota.usedGiB} GiB ({pct}%)</td><td>{quota.softGiB} GiB</td><td>{quota.hardGiB} GiB</td><td>{quota.filesUsed.toLocaleString('pt-BR')}</td><td>{quota.graceDays} dias</td><td><StatusBadge state={quota.mappingStable?'suportado':'indisponivel'}/></td></tr>})}</tbody></table></div>
  </Panel>
  <Panel title="Definir cota" subtitle={READ_ONLY_RELEASE_REASON}><fieldset disabled><div className="form-grid"><label>Sistema de arquivos<select><option>Sistema detectado</option></select></label><label>Tipo<select><option>Grupo</option><option>Usuário</option></select></label><label>Principal<input placeholder="DOMINIO\\grupo"/></label><label>Modelo<select><option>Modelo de referência</option><option>Personalizado</option></select></label><label>Limite flexível (GiB)<input type="number" defaultValue="400"/></label><label>Limite rígido (GiB)<input type="number" defaultValue="450"/></label></div></fieldset><div className="alert alert-warning"><AlertTriangle size={18}/> Não será aplicada cota a SID não resolvido, UID transitório ou faixa idmap sobreposta.</div><button className="button" disabled title={writeReason}>Validar correspondência e simular</button></Panel>
</>}
