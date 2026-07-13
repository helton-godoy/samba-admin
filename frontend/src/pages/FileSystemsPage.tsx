import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { AlertTriangle, CheckCircle2, HardDrive, ShieldAlert } from 'lucide-react';
import { api } from '../api/client';
import { useCapabilityGate } from '../capabilities';
import { CodeBlock } from '../components/CodeBlock';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';

export function FileSystemsPage() {
  const query = useQuery({ queryKey: ['filesystems'], queryFn: api.filesystems });
  const [selected, setSelected] = useState('fs-print');
  const [target, setTarget] = useState('nfsv4');
  const aclCapability = useCapabilityGate('acl.nfsv4.read');
  const simulation = useMutation({ mutationFn: ({ filesystemId, targetModel }: { filesystemId: string; targetModel: string }) => api.simulateAclConversion(filesystemId, targetModel) });

  if (query.isLoading) return <Loading />;
  if (query.error) return <ErrorState error={query.error} message="Falha ao listar sistemas de arquivos." />;
  if (!query.data) return <ErrorState message="A API não retornou o inventário de sistemas de arquivos." />;
  const ufsFilesystems = query.data.filter((item) => item.type.toLowerCase().startsWith('ufs'));
  const fs = query.data.find((item) => item.id === selected) || ufsFilesystems[0] || query.data[0];

  return (
    <>
      <PageHeader title="Sistemas de arquivos" description="Inventário UFS2, opções de montagem, ACLs e quotas por ponto de montagem." />
      <Panel title="Inventário de montagens" subtitle="ACL POSIX e ACL NFSv4 não podem coexistir no mesmo UFS2 montado">
        <div className="table-scroll">
          <table>
            <thead><tr><th>Dispositivo</th><th>Ponto</th><th>Tamanho</th><th>Livre</th><th>Opções</th><th>ACL</th><th>xattr</th><th>Quotas</th></tr></thead>
            <tbody>{query.data.map((item) => <tr key={item.id}>
              <td><code>{item.device}</code></td><td><code>{item.mountPoint}</code></td><td>{item.sizeGiB} GiB</td><td>{item.sizeGiB-item.usedGiB} GiB</td>
              <td>{item.mountOptions.map((o) => <span className="mono-pill" key={o}>{o}</span>)}</td>
              <td>{item.aclModel === 'nfsv4' ? 'NFSv4' : item.aclModel === 'posix' ? 'POSIX.1e' : 'Ausente'}</td>
              <td>{item.extendedAttributes ? 'Sim' : 'Não'}</td><td>{item.userQuota || item.groupQuota ? 'Ativas' : 'Inativas'}</td>
            </tr>)}</tbody>
          </table>
        </div>
      </Panel>

      <Panel title="Assistente de alteração do modelo de ACL" subtitle="Somente simulação: a operação real exigirá backup, janela de manutenção e backend privilegiado" tone="warning">
        <div className="form-grid">
          <label>Sistema de arquivos
            <select value={fs?.id || ''} disabled={!fs} onChange={(e) => { setSelected(e.target.value); simulation.reset(); }}>
              {ufsFilesystems.map((item) => <option key={item.id} value={item.id}>{item.mountPoint} — {item.aclModel}</option>)}
            </select>
          </label>
          <label>Modelo de destino
            <select value={target} onChange={(e) => { setTarget(e.target.value); simulation.reset(); }}>
              <option value="nfsv4">ACL NFSv4</option>
              <option value="posix">ACL POSIX.1e</option>
            </select>
          </label>
        </div>
        <div className="callout">
          <HardDrive size={20} />
          <div><strong>Impacto conhecido</strong><p>{fs ? <>Compartilhamentos afetados: {fs.associatedShares.join(', ') || 'nenhum'}. A montagem atual usa <code>{fs.mountOptions.join(',')}</code>.</> : 'Nenhum sistema de arquivos UFS foi detectado.'}</p></div>
        </div>
        {!aclCapability.available && <div className="alert alert-warning"><strong>Simulacao indisponivel.</strong> {aclCapability.reason}</div>}
        {fs?.aclModel === target ? <div className="alert alert-warning"><AlertTriangle size={18} /> O sistema de arquivos já utiliza o modelo selecionado.</div> : (
          <button className="button" onClick={() => { if (fs) simulation.mutate({ filesystemId: fs.id, targetModel: target }); }} disabled={!fs || simulation.isPending || !aclCapability.available} title={aclCapability.reason}>Simular conversão e gerar relatório</button>
        )}
        {simulation.isPending && <Loading label="Inventariando permissões e simulando perdas semânticas..." />}
        {simulation.error && <ErrorState error={simulation.error} />}
        {simulation.data && <div className="simulation-result">
          <div className="metrics-grid compact">
            <div className="mini-stat"><span>Objetos analisados</span><strong>{simulation.data.scannedObjects.toLocaleString('pt-BR')}</strong></div>
            <div className="mini-stat success"><span>Preservados</span><strong>{simulation.data.preserved.toLocaleString('pt-BR')}</strong></div>
            <div className="mini-stat warning"><span>Aproximados</span><strong>{simulation.data.approximated.toLocaleString('pt-BR')}</strong></div>
            <div className="mini-stat danger"><span>Não representáveis</span><strong>{simulation.data.notRepresentable.toLocaleString('pt-BR')}</strong></div>
          </div>
          <div className="two-column">
            <div>
              <h3><ShieldAlert size={18} /> Riscos identificados</h3>
              <ul>{simulation.data.risks.map((risk) => <li key={risk}>{risk}</li>)}</ul>
            </div>
            <div>
              <h3><CheckCircle2 size={18} /> Plano controlado</h3>
              <ol>{simulation.data.plan.map((step) => <li key={step}>{step}</li>)}</ol>
            </div>
          </div>
          <CodeBlock label="Artefato de backup planejado">{simulation.data.backupArtifact}</CodeBlock>
          <div className="alert alert-danger"><strong>Operação bloqueada no protótipo.</strong> A reversão restaura dados exportados, mas não é apresentada como semanticamente perfeita.</div>
        </div>}
      </Panel>
    </>
  );
}
