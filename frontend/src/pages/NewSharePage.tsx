import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ArrowLeft, ShieldAlert } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Link } from 'react-router-dom';
import { z } from 'zod';
import { api } from '../api/client';
import { READ_ONLY_RELEASE_REASON, useCapabilityGate } from '../capabilities';
import { CodeBlock } from '../components/CodeBlock';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import type { Share } from '../types';

const schema = z.object({
  name: z.string().min(2, 'Informe pelo menos 2 caracteres').regex(/^[A-Za-z0-9_$-]+$/, 'Use apenas letras, números, _, $ e -'),
  description: z.string().min(4, 'Informe uma descrição'),
  path: z.string().startsWith('/srv/dados/', 'O caminho deve estar em /srv/dados/').refine((v)=>!v.includes('..'), 'Travessia de diretórios não é permitida'),
  readOnly: z.boolean(),
  allowedPrincipal: z.string().min(1, 'Selecione um usuário ou grupo autorizado'),
  encryption: z.enum(['desired','required']),
  signing: z.enum(['default','mandatory']),
  auditProfile: z.enum(['off','minimum','security','changes','complete']),
  recycleBin: z.boolean(),
  dfs: z.boolean(),
  maxConnections: z.number().min(1).max(5000)
});
type FormValues = z.infer<typeof schema>;

export function NewSharePage() {
  const principals = useQuery({ queryKey:['identities'], queryFn:api.identities });
  const testparmCapability = useCapabilityGate('samba.testparm');
  const [preview, setPreview] = useState<Awaited<ReturnType<typeof api.previewShare>> | null>(null);
  const { register, handleSubmit, formState:{ errors } } = useForm<FormValues>({
    resolver:zodResolver(schema),
    defaultValues:{ readOnly:false, encryption:'desired', signing:'mandatory', auditProfile:'security', recycleBin:true, dfs:false, maxConnections:200 }
  });
  const previewMutation = useMutation({ mutationFn:(data:FormValues)=>api.previewShare(toShare(data)), onSuccess:setPreview });

  function toShare(data:FormValues):Share { return {
    id:'pending', name:data.name, description:data.description, path:data.path, enabled:true, readOnly:data.readOnly,
    guestAccess:false, allowedPrincipals:[data.allowedPrincipal], deniedPrincipals:[], encryption:data.encryption, signing:data.signing,
    auditProfile:data.auditProfile, recycleBin:data.recycleBin, dfs:data.dfs,
    vfsModules:['acl_xattr', ...(data.recycleBin?['recycle']:[]), ...(data.auditProfile!=='off'?['full_audit']:[])],
    maxConnections:data.maxConnections, aclModel:'nfsv4'
  };}

  if (principals.isLoading) return <Loading/>;
  if (principals.error) return <ErrorState message="Falha ao carregar usuários e grupos." error={principals.error}/>;
  return <>
    <PageHeader title="Novo compartilhamento" description="Fluxo seguro com validação tipada, política de caminho, prévia e diff." actions={<Link className="button button-secondary" to="/compartilhamentos"><ArrowLeft size={16}/> Voltar</Link>} />
    <form onSubmit={handleSubmit((data)=>previewMutation.mutate(data))}>
      <Panel title="Identificação e acesso" subtitle="O navegador envia dados estruturados; não constrói comandos de shell">
        <div className="form-grid">
          <label>Nome<input {...register('name')} placeholder="Projetos"/>{errors.name&&<span className="field-error">{errors.name.message}</span>}</label>
          <label>Descrição<input {...register('description')} placeholder="Documentos de projetos institucionais"/>{errors.description&&<span className="field-error">{errors.description.message}</span>}</label>
          <label className="span-2">Caminho autorizado<input {...register('path')} placeholder="/srv/dados/projetos"/>{errors.path&&<span className="field-error">{errors.path.message}</span>}<small>/boot, /dev, /root, /bin, /sbin, /lib e demais caminhos do sistema são bloqueados.</small></label>
          <label>Usuário ou grupo permitido<select {...register('allowedPrincipal')}><option value="">Selecione...</option>{principals.data!.filter((p)=>p.mappingStable).map((p)=><option key={p.id} value={p.name}>{p.displayName} — {p.name}</option>)}</select>{errors.allowedPrincipal&&<span className="field-error">{errors.allowedPrincipal.message}</span>}</label>
          <label>Limite de conexões<input type="number" {...register('maxConnections', { valueAsNumber: true })}/>{errors.maxConnections&&<span className="field-error">{errors.maxConnections.message}</span>}</label>
        </div>
      </Panel>
      <Panel title="Segurança SMB e recursos" subtitle="Valores padrão compatíveis com Windows 11 e princípio do menor privilégio">
        <div className="form-grid">
          <label>Criptografia<select {...register('encryption')}><option value="desired">Desejada</option><option value="required">Obrigatória</option></select></label>
          <label>Assinatura<select {...register('signing')}><option value="mandatory">Obrigatória</option><option value="default">Padrão do servidor</option></select></label>
          <label>Auditoria<select {...register('auditProfile')}><option value="off">Desativada</option><option value="minimum">Mínima</option><option value="security">Segurança</option><option value="changes">Alterações</option><option value="complete">Completa</option></select></label>
          <label className="check-label"><input type="checkbox" {...register('readOnly')}/> Somente leitura</label>
          <label className="check-label"><input type="checkbox" {...register('recycleBin')}/> Lixeira</label>
          <label className="check-label"><input type="checkbox" {...register('dfs')}/> Publicar como destino DFS</label>
        </div>
        <div className="alert alert-info"><ShieldAlert size={18}/> Acesso guest é fixado como <strong>desabilitado</strong>; SMB1 não é oferecido pela interface.</div>
      </Panel>
      {!testparmCapability.available && <div className="alert alert-warning"><strong>Validacao indisponivel.</strong> {testparmCapability.reason}</div>}
      <button className="button" type="submit" disabled={previewMutation.isPending || !testparmCapability.available} title={testparmCapability.reason}>Validar e gerar prévia</button>
    </form>
    {previewMutation.isPending&&<Loading label="Gerando configuração e executando validações conceituais..."/>}
    {previewMutation.error&&<ErrorState error={previewMutation.error}/>} 
    {preview&&<Panel title="Prévia antes da aplicação" subtitle={preview.testparm} tone="info">
      <div className="impact-grid"><div><span>Sessões ativas</span><strong>{preview.impact.activeSessions}</strong></div><div><span>Arquivos abertos</span><strong>{preview.impact.openFiles}</strong></div><div><span>Ação do serviço</span><strong>{preview.restartRequired?'Reiniciar':'Recarregar'}</strong></div></div>
      <div className="two-column"><CodeBlock label="Trecho proposto de smb4.conf">{preview.config}</CodeBlock><CodeBlock label="Diff">{preview.diff}</CodeBlock></div>
      <div className="alert alert-warning"><strong>Aplicação bloqueada.</strong> {READ_ONLY_RELEASE_REASON} A prévia não realizou gravação.</div>
      <button className="button button-danger" disabled title={READ_ONLY_RELEASE_REASON}>Confirmar aplicação</button>
    </Panel>}
  </>;
}
