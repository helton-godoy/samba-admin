import { AlertOctagon, DatabaseBackup, KeyRound } from 'lucide-react';
import { useCapabilityGate } from '../capabilities';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

export function AdDcPage(){
  const ad = useCapabilityGate('samba.ad_dc');
  const state = ad.state === 'loading' || ad.state === 'error' ? 'nao_verificado' : ad.state;
  return <>
    <PageHeader title="Samba AD DC" description="Módulo operacional separado para nova floresta ou controlador adicional." />
    <div className="profile-banner danger"><KeyRound/><div><strong>Perfil não habilitado</strong><p>Este host está configurado como membro de domínio. Uma instância não pode ser apresentada simultaneamente como membro e AD DC.</p></div><StatusBadge state={state}/></div>
    <Panel title="Matriz de pré-requisitos" subtitle="O provisionamento fica bloqueado até que o backend prove cada requisito">
      <div className="table-scroll"><table><thead><tr><th>Verificação</th><th>Estado</th><th>Motivo</th></tr></thead><tbody>
        <tr><td>Versão do Samba</td><td><StatusBadge state="nao_verificado"/></td><td>Detectar versão e comparar com a política institucional.</td></tr>
        <tr><td>Opções de compilação do pacote</td><td><StatusBadge state="nao_verificado"/></td><td>Confirmar AD DC, DNS, Python, LDB e Kerberos.</td></tr>
        <tr><td>Backend Kerberos</td><td><StatusBadge state="nao_verificado"/></td><td>Validar implementação e compatibilidade da compilação.</td></tr>
        <tr><td>ACL exigida para SYSVOL</td><td><StatusBadge state="experimental"/></td><td>Exige prova funcional no UFS2 e testes de permissões equivalentes.</td></tr>
        <tr><td>Backup e restore do domínio</td><td><StatusBadge state="nao_verificado"/></td><td>Testar ferramenta da versão instalada e restauração isolada.</td></tr>
        <tr><td>Replicação como DC adicional</td><td><StatusBadge state="nao_verificado"/></td><td>Validar DNS, DRS, SYSVOL e interoperabilidade do domínio alvo.</td></tr>
      </tbody></table></div>
    </Panel>
    <div className="two-column">
      <Panel title="Nova floresta" subtitle="Provisionamento somente em host dedicado e vazio">
        <ol><li>Validar build e dependências.</li><li>Reservar DNS, IP, hostname e NTP.</li><li>Gerar backup da configuração.</li><li>Provisionar em ambiente isolado.</li><li>Testar Kerberos, LDAP, DNS, SYSVOL e GPO.</li></ol>
        <button className="button" disabled title={ad.reason}>Provisionar nova floresta</button>
      </Panel>
      <Panel title="Controlador adicional" subtitle="Não equivale a ingressar como membro">
        <ol><li>Verificar nível funcional.</li><li>Testar conectividade DRS.</li><li>Realizar backup dos DCs existentes.</li><li>Ingressar como DC adicional.</li><li>Verificar replicação e FSMO.</li></ol>
        <button className="button" disabled title={ad.reason}>Ingressar como DC adicional</button>
      </Panel>
    </div>
    <Panel title="Bloqueio de segurança" subtitle="Proteção contra promoção acidental de servidor de arquivos" tone="danger">
      <div className="callout"><AlertOctagon/><div><strong>Backup obrigatório</strong><p>Sem backup validado, restauração testada e aprovação administrativa, qualquer mudança de perfil permanece indisponível.</p></div><DatabaseBackup/></div>
    </Panel>
  </>;
}
