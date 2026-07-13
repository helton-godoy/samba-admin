import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, Plus, Printer, ShieldCheck } from 'lucide-react';
import { api } from '../api/client';
import { useCapabilityGate } from '../capabilities';
import { ErrorState, Loading } from '../components/Loading';
import { PageHeader } from '../components/PageHeader';
import { Panel } from '../components/Panel';
import { StatusBadge } from '../components/StatusBadge';

const READ_ONLY_REASON = 'Operações mutáveis de CUPS e drivers permanecem bloqueadas no RC1 somente leitura.';

export function PrintingPage() {
  const cups = useQuery({ queryKey: ['cups'], queryFn: api.cups });
  const inspectCapability = useCapabilityGate('cups.inspect');
  // /printers permanece consumido para preservar o contrato detalhado legado.
  const printers = useQuery({ queryKey: ['printers'], queryFn: api.printers });
  const drivers = useQuery({ queryKey: ['drivers'], queryFn: api.printDrivers });

  if (cups.isLoading || printers.isLoading || drivers.isLoading) return <Loading label="Consultando CUPS pelo agente..." />;
  const error = cups.error || printers.error || drivers.error;
  if (error) return <ErrorState error={error} message="Falha ao carregar o inventário de impressão." />;
  if (!cups.data || !printers.data || !drivers.data) return <ErrorState message="O inventário de impressão retornou dados incompletos." />;

  return <>
    <PageHeader
      title="Servidor de impressão"
      description="Inventário CUPS real, filas detalhadas e compatibilidade de drivers em modo somente leitura."
      actions={<button className="button" disabled title={READ_ONLY_REASON}><Plus size={16} /> Nova impressora</button>}
    />

    <div className="alert alert-warning"><ShieldCheck size={18} /><strong>Alterações bloqueadas.</strong> {READ_ONLY_REASON}</div>
    {!inspectCapability.available && <div className="alert alert-warning"><strong>Capability de inventário CUPS indisponível.</strong> {inspectCapability.reason}</div>}

    <Panel title="Estado do CUPS" subtitle="Contrato estruturado obtido por API → UDS → agente">
      <dl className="definition-grid">
        <div><dt>Versão/pacote</dt><dd>{cups.data.version || 'Não detectado'}</dd></div>
        <div><dt>Validação do serviço</dt><dd><StatusBadge state={cups.data.service === 'valid' ? 'suportado' : 'indisponivel'} /></dd></div>
        <div><dt>Filas detectadas</dt><dd>{cups.data.printers.length}</dd></div>
        <div><dt>Backends</dt><dd>{cups.data.backends.join(', ') || 'Não detectados'}</dd></div>
        <div><dt>PPDs</dt><dd>{cups.data.ppds.length}</dd></div>
      </dl>
      {cups.data.errors.length > 0 && <div className="alert alert-warning"><AlertTriangle size={18} /><div><strong>Diagnóstico parcial.</strong><ul>{cups.data.errors.map((errorMessage) => <li key={errorMessage}>{errorMessage}</li>)}</ul></div></div>}
    </Panel>

    <Panel title="Filas de impressão" subtitle="O endpoint legado /api/v1/printers continua preservado">
      {printers.data.length === 0 ? <p className="muted">Nenhuma fila foi detectada.</p> : <div className="table-scroll"><table><thead><tr><th>Fila</th><th>URI</th><th>Modelo</th><th>Driver</th><th>Trabalhos</th><th>Windows 11</th><th>Estado</th></tr></thead><tbody>{printers.data.map((printer) => <tr key={printer.id}><td><strong>{printer.name}</strong></td><td><code>{printer.uri || '—'}</code></td><td>{printer.model || 'Não detectado'}</td><td>{printer.driver || 'Não detectado'}</td><td>{printer.jobs}</td><td><StatusBadge state={printer.windows11Validated ? 'suportado' : 'nao_verificado'} /></td><td>{printer.paused ? 'Pausada' : 'Ativa'}</td></tr>)}</tbody></table></div>}
    </Panel>

    <Panel title="Drivers Windows" subtitle="Assinatura e package-aware não garantem instalação sem elevação; GPOs também precisam ser validadas">
      <div className="table-scroll"><table><thead><tr><th>Driver</th><th>Versão</th><th>Arquitetura</th><th>Assinado</th><th>Package-aware</th><th>Windows 11</th></tr></thead><tbody>{drivers.data.map((driver) => <tr key={driver.id}><td>{driver.name}</td><td>{driver.version}</td><td>{driver.architecture}</td><td>{driver.signed ? 'Sim' : 'Não'}</td><td>{driver.packageAware ? 'Sim' : 'Não'}</td><td><StatusBadge state={driver.windows11Compatible ? 'suportado' : 'nao_verificado'} /></td></tr>)}</tbody></table></div>
      <div className="check-grid">
        <div className="check-card"><ShieldCheck /><strong>Servidor aprovado</strong><p>Restringir Point and Print aos servidores explicitamente confiáveis.</p></div>
        <div className="check-card"><ShieldCheck /><strong>Driver assinado</strong><p>Validar cadeia, integridade, arquitetura e origem do pacote.</p></div>
        <div className="check-card"><AlertTriangle /><strong>Elevação</strong><p>Não prometer instalação silenciosa para usuário comum sem avaliar a política vigente.</p></div>
      </div>
    </Panel>

    <Panel title="Nova fila" subtitle={READ_ONLY_REASON} tone="warning">
      <fieldset disabled>
        <div className="form-grid"><label>Nome da fila<input placeholder="FILA-INSTITUCIONAL" /></label><label>URI/backend<input placeholder="ipp://ENDERECO/ipp/print" /></label><label>Fabricante/modelo<input placeholder="Modelo detectado" /></label><label>Driver<select><option>Microsoft IPP Class Driver</option>{drivers.data.map((driver) => <option key={driver.id}>{driver.name}</option>)}</select></label><label>Tamanho do papel<select><option>A4</option><option>A3</option></select></label><label>Duplex<select><option>Automático</option><option>Desativado</option></select></label></div>
      </fieldset>
      <button className="button" disabled title={READ_ONLY_REASON}><Printer size={16} /> Validar e gerar configuração</button>
    </Panel>
  </>;
}
