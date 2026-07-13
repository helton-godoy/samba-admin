import { Copy, Download, Edit3, FileText, Folder, FolderPlus, Move, Search, Trash2, Upload } from 'lucide-react';
import { useState } from 'react';
import { READ_ONLY_RELEASE_REASON } from '../capabilities';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';

const entries=[
  {name:'Administrativo',type:'folder',owner:'root',group:'EBSERHNET\\GDL-HUUFCAT-ADMINISTRATIVO',size:'84 GiB',modified:'11/07/2026 10:20'},
  {name:'Assistencial',type:'folder',owner:'root',group:'EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL',size:'312 GiB',modified:'11/07/2026 11:58'},
  {name:'Projetos',type:'folder',owner:'root',group:'EBSERHNET\\GDL-HUUFCAT-PROJETOS',size:'118 GiB',modified:'10/07/2026 17:42'},
  {name:'LEIA-ME.txt',type:'file',owner:'root',group:'wheel',size:'4 KiB',modified:'08/07/2026 09:00'}
];
export function FileManagerPage(){const[filter,setFilter]=useState('');const[selected,setSelected]=useState('');return <>
  <PageHeader title="Gerenciador de arquivos" description="Navegação restrita a volumes autorizados; caminhos do sistema ficam fora do escopo." actions={<div className="button-row"><button className="button button-secondary" disabled title={READ_ONLY_RELEASE_REASON}><Upload size={16}/> Enviar</button><button className="button" disabled title={READ_ONLY_RELEASE_REASON}><FolderPlus size={16}/> Nova pasta</button></div>}/>
  <div className="alert alert-warning"><strong>Operações de arquivo bloqueadas.</strong> {READ_ONLY_RELEASE_REASON}</div>
  <div className="alert alert-info"><strong>Raiz autorizada:</strong> <code>/srv/dados</code>. Caminhos como <code>/boot</code>, <code>/dev</code>, <code>/root</code>, <code>/bin</code>, <code>/sbin</code>, <code>/lib</code> e <code>/usr/sbin</code> são bloqueados.</div>
  <Panel title="/srv/dados" subtitle="Operações destrutivas exigirão impacto, arquivos abertos e confirmação explícita" actions={<label className="search-box"><Search size={16}/><input value={filter} onChange={(e)=>setFilter(e.target.value)} placeholder="Localizar"/></label>}>
    <div className="file-toolbar"><button disabled title={READ_ONLY_RELEASE_REASON}><Copy size={15}/> Copiar</button><button disabled title={READ_ONLY_RELEASE_REASON}><Move size={15}/> Mover</button><button disabled title={READ_ONLY_RELEASE_REASON}><Edit3 size={15}/> Renomear</button><button disabled title="Download ainda não integrado ao provider somente leitura."><Download size={15}/> Baixar</button><button disabled title={READ_ONLY_RELEASE_REASON} className="danger"><Trash2 size={15}/> Excluir</button></div>
    <div className="table-scroll"><table><thead><tr><th></th><th>Nome</th><th>Tamanho</th><th>Dono</th><th>Grupo</th><th>Modificado</th></tr></thead><tbody>{entries.filter((e)=>e.name.toLowerCase().includes(filter.toLowerCase())).map((e)=><tr key={e.name} className={selected===e.name?'selected-row':''} onClick={()=>setSelected(e.name)}><td>{e.type==='folder'?<Folder size={18}/>:<FileText size={18}/>}</td><td><strong>{e.name}</strong></td><td>{e.size}</td><td>{e.owner}</td><td>{e.group}</td><td>{e.modified}</td></tr>)}</tbody></table></div>
  </Panel>
  {selected&&<Panel title="Propriedades e impacto" subtitle={`/srv/dados/${selected}`} tone={selected==='Assistencial'?'warning':'info'}><div className="impact-grid"><div><span>Objetos estimados</span><strong>{selected==='Assistencial'?'142.300':'1'}</strong></div><div><span>Tamanho</span><strong>{entries.find((e)=>e.name===selected)?.size}</strong></div><div><span>Compartilhamento</span><strong>{selected}</strong></div><div><span>Usuários conectados</span><strong>{selected==='Assistencial'?'28':'0'}</strong></div></div><p>A exclusão real ficaria bloqueada enquanto houver arquivos abertos ou sessões ativas, salvo operação administrativa explicitamente autorizada e auditada.</p></Panel>}
</>}
