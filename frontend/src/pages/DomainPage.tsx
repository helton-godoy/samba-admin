import { useMutation, useQuery } from '@tanstack/react-query';
import { CheckCircle2, Network, ShieldAlert } from 'lucide-react';
import { api } from '../api/client';
import { useCapabilityGate } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

export function DomainPage(){
  const query=useQuery({queryKey:['domain'],queryFn:api.domain});
  const test=useMutation({mutationFn:api.testDomain});
  const diagnosisCapability = useCapabilityGate('domain.member.diagnose');
  const joinCapability = useCapabilityGate('domain.member.join');
  if(query.isLoading)return <Loading/>; if(query.error)return <ErrorState error={query.error} message="Falha ao consultar o domínio."/>;
  if(!query.data)return <ErrorState message="A API não retornou o inventário de domínio."/>;
  const d=query.data;
  return <>
    <PageHeader title="Active Directory" description="Assistente exclusivo para o perfil membro de domínio; não se mistura ao fluxo de AD DC." />
    <div className="profile-banner"><Network/><div><strong>Perfil atual: servidor membro</strong><p>Executa smbd/winbindd e utiliza um domínio Microsoft Active Directory existente.</p></div><StatusBadge state={d.joined?'saudavel':'critico'}/></div>
    <Panel title="Configuração do ingresso" subtitle="A senha de ingresso é efêmera e nunca deve ser persistida">
      <div className="form-grid">
        <label>Domínio DNS<input defaultValue={d.dnsDomain}/></label><label>Realm Kerberos<input defaultValue={d.realm}/></label>
        <label>Nome NetBIOS<input defaultValue={d.netbios}/></label><label>Site do AD<input defaultValue={d.site}/></label>
        <label className="span-2">OU do computador<input defaultValue={d.computerOu}/></label>
        <label>DC preferencial 1<input defaultValue={d.preferredDcs[0]}/></label><label>DC preferencial 2<input defaultValue={d.preferredDcs[1]}/></label>
        <label>Estratégia de ID mapping<select defaultValue={d.idmapStrategy}><option value="rid">idmap_rid — determinístico pelo RID</option><option value="ad">idmap_ad — atributos RFC2307 obrigatórios</option></select></label>
        <label>Faixa UID/GID<input defaultValue={d.uidRange}/></label>
        <label>Conta de ingresso<input disabled placeholder="Disponível somente após homologação" autoComplete="off" title={joinCapability.reason}/></label><label>Senha temporária<input disabled type="password" placeholder="Coleta bloqueada nesta release" autoComplete="new-password" title={joinCapability.reason}/></label>
      </div>
      <div className="alert alert-info"><ShieldAlert size={18}/> Faixas de idmap internas e do domínio devem ser não sobrepostas. Em <code>idmap_ad</code>, contas sem uidNumber/gidNumber coerentes não devem receber acesso ou quota.</div>
      <div className="alert alert-warning"><strong>Ingresso real bloqueado.</strong> {joinCapability.reason} Nenhuma credencial é solicitada nesta release.</div>
      {!diagnosisCapability.available&&<div className="alert alert-warning"><strong>Diagnostico indisponivel.</strong> {diagnosisCapability.reason}</div>}
      <button className="button" onClick={()=>test.mutate()} disabled={!diagnosisCapability.available||test.isPending} title={diagnosisCapability.reason}>Executar bateria de testes</button>
      <button className="button button-secondary" type="button" disabled title={joinCapability.reason}>Preparar ingresso como membro</button>
      {test.isPending&&<Loading label="Testando DNS, horário, Kerberos, LDAP, SMB e resolução de identidades..."/>}
      {test.error&&<ErrorState error={test.error}/>} 
    </Panel>
    <Panel title="Diagnóstico do domínio" subtitle="Comandos são apenas modelados; o navegador não executa utilitários do sistema">
      <div className="check-grid">{(test.data?.checks||d.tests).map((check)=><div className="check-card" key={check.name}><StatusBadge state={check.state}/><strong>{check.name}</strong><p>{check.details}</p></div>)}</div>
      {test.data&&<div className="alert alert-success"><CheckCircle2 size={18}/> Testes aprovados. Utilitários modelados: {test.data.commandsModeled.join(', ')}.</div>}
    </Panel>
    <Panel title="Consequências da mudança de perfil" subtitle="Trocar membro de domínio por AD DC exige reprovisionamento separado" tone="danger">
      <p>O protótipo bloqueia a promoção direta deste servidor de arquivos. Antes de qualquer troca, são exigidos backup, inventário de compartilhamentos, saída controlada do domínio, validação do pacote e plano de restauração.</p>
    </Panel>
  </>;
}
