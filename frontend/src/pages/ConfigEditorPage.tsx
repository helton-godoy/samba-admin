import { useMutation, useQuery } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, History, Lock, Save } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON } from '../capabilities';
import { CodeBlock } from '../components/CodeBlock';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';

const allowedFiles=['/usr/local/etc/smb4.conf','/etc/rc.conf','/etc/fstab','/etc/krb5.conf','/etc/nsswitch.conf','/etc/syslog.conf','/usr/local/etc/cups/cupsd.conf'];
export function ConfigEditorPage(){
  const[path,setPath]=useState(allowedFiles[0]); const[content,setContent]=useState(''); const[dirty,setDirty]=useState(false);
  const file=useQuery({queryKey:['config-file',path],queryFn:()=>api.configFile(path)});
  const validate=useMutation({mutationFn:()=>api.validateConfig(path,content)});
  useEffect(()=>{if(file.data){setContent(file.data.content);setDirty(false);validate.reset()}},[file.data]);
  return <>
    <PageHeader title="Editor de configuração" description="Edição permitida somente para arquivos pré-autorizados, com locking, validação, histórico e rollback." />
    <div className="alert alert-warning"><Lock size={18}/> Senhas, keytabs, hashes, tickets Kerberos, tokens, chaves privadas e outros segredos não são expostos neste editor.</div>
    <Panel title="Arquivo autorizado" subtitle="O backend real usará bloqueio otimista por versão e gravação atômica">
      <label>Arquivo<select value={path} onChange={(e)=>setPath(e.target.value)}>{allowedFiles.map((p)=><option key={p}>{p}</option>)}</select></label>
      {file.isLoading&&<Loading/>}{file.error&&<ErrorState error={file.error}/>}
      {file.data&&<>
        <div className="editor-meta"><span>Codificação: {file.data.encoding}</span><span>Final de linha: {file.data.eol}</span><span>Versão: <code>{file.data.version}</code></span><span>{dirty?'Alterações não salvas':'Sem alterações'}</span></div>
        <div className="editor-wrap"><div className="line-numbers" aria-hidden="true">{content.split('\n').map((_,i)=><span key={i}>{i+1}</span>)}</div><textarea aria-label="Conteúdo do arquivo" spellCheck={false} value={content} onChange={(e)=>{setContent(e.target.value);setDirty(e.target.value!==file.data.content)}}/></div>
        <div className="button-row"><button className="button button-secondary" onClick={()=>setContent(file.data.previousContent)}><History size={16}/> Comparar versão anterior</button><button className="button" disabled={!dirty||validate.isPending} onClick={()=>validate.mutate()}><Save size={16}/> Validar alterações</button></div>
      </>}
    </Panel>
    {validate.isPending&&<Loading label="Executando validação sintática, política e teste técnico..."/>}
    {validate.error&&<ErrorState error={validate.error}/>}
    {validate.data&&<Panel title="Resultado da validação" subtitle="Nenhuma gravação foi realizada"><div className="alert alert-success"><CheckCircle2 size={18}/> Validação aprovada por: {validate.data.validators.join(', ')}.</div><CodeBlock label="Diff resumido">{validate.data.diff}</CodeBlock><div className="alert alert-warning"><AlertTriangle size={18}/> A aplicação exigiria backup, gravação atômica, verificação de versão, {validate.data.reloadRequired?'recarga do serviço':'teste de leitura'} e rollback.</div><button className="button button-danger" disabled title={READ_ONLY_RELEASE_REASON}>Criar tarefa de aplicação</button></Panel>}
  </>;
}
