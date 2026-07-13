import { useMutation, useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { ArrowDown, ArrowUp, Calculator, Plus, ShieldAlert, Trash2 } from 'lucide-react';
import { api } from '../api/client';
import { useCapabilityGate } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';
import type { Ace } from '../types';

const permissionGroups: Record<string, string[]> = {
  'Controle total': ['read_data','write_data','append_data','execute','delete','read_acl','write_acl','write_owner'],
  Modificar: ['read_data','write_data','append_data','execute','delete','read_acl'],
  'Ler e executar': ['read_data','execute','read_acl'],
  Leitura: ['read_data','read_acl'],
  Gravação: ['write_data','append_data']
};

export function AclPage() {
  const aclQuery = useQuery({queryKey:['acls'],queryFn:api.acls});
  const principalQuery = useQuery({queryKey:['identities'],queryFn:api.identities});
  const writeCapability = useCapabilityGate('acl.nfsv4.write');
  const [entries,setEntries] = useState<Ace[]>([]);
  const [mode,setMode] = useState<'windows'|'technical'|'posix'>('windows');
  const [selectedPrincipal,setSelectedPrincipal] = useState('EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL');
  const effective = useMutation({mutationFn:()=>api.effectiveAcl(selectedPrincipal)});
  const current = entries.length ? entries : (aclQuery.data || []);
  const compatibility = useMemo(()=>({ preserved:current.filter((a)=>a.type==='ALLOW').length, approximated:current.filter((a)=>a.type==='DENY').length, notRepresentable:principalQuery.data?.filter((p)=>!p.mappingStable).length||0 }),[current,principalQuery.data]);
  if(aclQuery.isLoading||principalQuery.isLoading)return <Loading/>;
  if(aclQuery.error||principalQuery.error)return <ErrorState error={aclQuery.error || principalQuery.error} message="Falha ao carregar ACLs."/>;
  const update = (next:Ace[])=>setEntries(next.map((item,index)=>({...item,order:index+1})));
  const move=(index:number,dir:-1|1)=>{const next=[...current];const target=index+dir;if(target<0||target>=next.length)return;[next[index],next[target]]=[next[target],next[index]];update(next)};
  const addAce=()=>update([...current,{id:`ace-${Date.now()}`,principal:'everyone@',source:'special',type:'ALLOW',permissions:['read_data','read_acl'],flags:['file_inherit','directory_inherit'],inherited:false,order:current.length+1}]);
  return <>
    <PageHeader title="Permissões e ACLs" description="Visualizações POSIX, NFSv4 técnica e compatibilidade semelhante ao Windows." actions={<button className="button" disabled title={writeCapability.reason} onClick={addAce}><Plus size={16}/> Adicionar ACE</button>}/>
    <div className="alert alert-warning"><strong>Escrita de ACL bloqueada.</strong> {writeCapability.reason}</div>
    <div className="tabs" role="tablist">
      <button className={mode==='windows'?'active':''} onClick={()=>setMode('windows')}>Compatibilidade Windows</button>
      <button className={mode==='technical'?'active':''} onClick={()=>setMode('technical')}>NFSv4 técnica</button>
      <button className={mode==='posix'?'active':''} onClick={()=>setMode('posix')}>POSIX</button>
    </div>
    {mode==='windows'&&<Panel title="Editor de segurança" subtitle="Caminho: /srv/dados/assistencial — sistema de arquivos com ACL NFSv4">
      <div className="table-scroll"><table><thead><tr><th>Ordem</th><th>Principal</th><th>Tipo</th><th>Permissões agrupadas</th><th>Herança</th><th>Origem</th><th>Ações</th></tr></thead>
      <tbody>{current.map((ace,index)=><tr key={ace.id}>
        <td>{ace.order}</td><td><strong>{ace.principal}</strong><div className="muted">{ace.source}</div></td><td><StatusBadge state={ace.type}/></td>
        <td><select disabled title={writeCapability.reason} value={Object.entries(permissionGroups).find(([,p])=>p.every((x)=>ace.permissions.includes(x)))?.[0]||'Permissões especiais'} onChange={(e)=>{const perms=permissionGroups[e.target.value];if(perms)update(current.map((item)=>item.id===ace.id?{...item,permissions:perms}:item))}}><option>Permissões especiais</option>{Object.keys(permissionGroups).map((p)=><option key={p}>{p}</option>)}</select></td>
        <td>{ace.flags.includes('file_inherit')?'Arquivos e pastas':'Somente este objeto'}{ace.inherited&&<div className="muted">Herdada</div>}</td><td>{ace.inherited?'Pai':'Explícita'}</td>
        <td><div className="icon-actions"><button disabled title={writeCapability.reason} aria-label="Mover acima" onClick={()=>move(index,-1)}><ArrowUp size={15}/></button><button disabled title={writeCapability.reason} aria-label="Mover abaixo" onClick={()=>move(index,1)}><ArrowDown size={15}/></button><button disabled title={writeCapability.reason} aria-label="Excluir" onClick={()=>update(current.filter((a)=>a.id!==ace.id))}><Trash2 size={15}/></button></div></td>
      </tr>)}</tbody></table></div>
    </Panel>}
    {mode==='technical'&&<Panel title="Entradas NFSv4" subtitle="Ordem das ACEs, flags e permissões individuais são semanticamente relevantes">
      {current.map((ace)=><article className="ace-card" key={ace.id}><div><strong>#{ace.order} {ace.principal}</strong><StatusBadge state={ace.type}/></div><code>{ace.type.toLowerCase()}:{ace.principal}:{ace.permissions.join('/')}:flags={ace.flags.join('/')}</code></article>)}
    </Panel>}
    {mode==='posix'&&<Panel title="Visão POSIX aproximada" subtitle="Não representa perfeitamente a ACL NFSv4 atualmente armazenada" tone="warning">
      <div className="posix-grid"><div><strong>Proprietário</strong><span>root</span><code>rwx</code></div><div><strong>Grupo</strong><span>EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL</span><code>rwx</code></div><div><strong>Outros</strong><span>everyone@ aproximado</span><code>---</code></div><div><strong>Máscara</strong><span>Não aplicável diretamente</span><code>rwx</code></div></div>
    </Panel>}
    <div className="two-column">
      <Panel title="Compatibilidade semântica" subtitle="Estimativa para apresentação a clientes Windows">
        <div className="impact-grid"><div><span>Preservadas</span><strong>{compatibility.preserved}</strong></div><div><span>Aproximadas</span><strong>{compatibility.approximated}</strong></div><div><span>Não representáveis</span><strong>{compatibility.notRepresentable}</strong></div></div>
        <div className="alert alert-warning"><ShieldAlert size={18}/> Uma ACE DENY pode alterar precedência quando reordenada. A interface nunca normaliza a ordem silenciosamente.</div>
      </Panel>
      <Panel title="Permissões efetivas" subtitle="Cálculo simulado com advertência sobre grupos aninhados">
        <label>Usuário ou grupo<select value={selectedPrincipal} onChange={(e)=>setSelectedPrincipal(e.target.value)}>{principalQuery.data!.map((p)=><option key={p.id} value={p.name}>{p.displayName}</option>)}</select></label>
        <button className="button" onClick={()=>effective.mutate()}><Calculator size={16}/> Calcular</button>
        {effective.isPending&&<Loading label="Resolvendo token de acesso..."/>}
        {effective.data&&<div className="effective-result"><h3>Permitido</h3><p>{effective.data.effective.join(', ')}</p><h3>Negado</h3><p>{effective.data.denied.join(', ')}</p><small>{effective.data.warnings[0]}</small></div>}
      </Panel>
    </div>
  </>;
}
