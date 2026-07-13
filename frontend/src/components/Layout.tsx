import {
  Activity,
  BookOpenCheck,
  Boxes,
  ChevronRight,
  ClipboardList,
  ClipboardCheck,
  Database,
  FileCode2,
  Files,
  FolderCog,
  Gauge,
  HardDrive,
  KeyRound,
  LogOut,
  Network,
  Printer,
  ScrollText,
  Server,
  ServerCog,
  ShieldCheck,
  SlidersHorizontal,
  UsersRound
} from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { NavLink, Outlet } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth } from '../auth/AuthProvider';

const nav = [
  ['/', 'Painel', Gauge],
  ['/samba', 'Inventário Samba', Server],
  ['/sistemas-arquivos', 'Sistemas de arquivos', HardDrive],
  ['/compartilhamentos', 'Compartilhamentos', FolderCog],
  ['/acl', 'Permissões e ACLs', ShieldCheck],
  ['/arquivos', 'Gerenciador de arquivos', Files],
  ['/configuracoes', 'Editor de configuração', FileCode2],
  ['/dominio', 'Active Directory', UsersRound],
  ['/ad-dc', 'Samba AD DC', KeyRound],
  ['/impressao', 'Impressão', Printer],
  ['/dfs', 'DFS Namespace', Network],
  ['/cotas', 'Cotas', Database],
  ['/logs', 'Logs e SIEM', ScrollText],
  ['/auditoria', 'Auditoria', BookOpenCheck],
  ['/mudancas', 'Aprovações', ClipboardCheck],
  ['/servicos', 'Serviços', ServerCog],
  ['/tarefas', 'Central de tarefas', ClipboardList],
  ['/arquitetura', 'Arquitetura e API', Boxes]
] as const;

export function Layout() {
  const { user, logout } = useAuth();
  const system = useQuery({ queryKey: ['system'], queryFn: api.system });
  const profile = system.data?.profile;
  const profileLabel: Record<string, string> = {
    standalone: 'independente',
    'domain-member': 'membro de domínio',
    'ad-dc': 'controlador de domínio',
    'additional-dc': 'controlador adicional'
  };
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark"><SlidersHorizontal size={22} /></div>
          <div>
            <strong>Samba Console</strong>
            <span>{system.data?.freebsdVersion || 'FreeBSD'}</span>
          </div>
        </div>
        <nav aria-label="Navegação principal">
          {nav.map(([to, label, Icon]) => (
            <NavLink key={to} to={to} end={to === '/'} className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
              <Icon size={18} aria-hidden="true" />
              <span>{label}</span>
              <ChevronRight className="nav-chevron" size={15} aria-hidden="true" />
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="sidebar-health"><Activity size={16} /> API protegida</div>
          <span>Nenhum comando de shell é executado pelo navegador.</span>
        </div>
      </aside>
      <div className="main-column">
        <header className="topbar">
          <div>
            <strong>{system.data?.hostname || (system.error ? 'Servidor indisponível' : 'Servidor administrado')}</strong>
            <span>{system.data?.fqdn || (system.error ? 'Inventário não disponível' : 'Carregando inventário...')}</span>
          </div>
          <div className="topbar-meta">
            <span className="environment-pill">RC1 SOMENTE LEITURA</span>
            <span>Perfil: {profile ? (profileLabel[profile] || profile) : 'não detectado'}</span>
            <span className="authenticated-user" title={user?.roles.join(', ')}>{user?.displayName || user?.username}</span>
            <button className="icon-button" type="button" title="Encerrar sessao" onClick={() => void logout()}><LogOut size={16} /></button>
          </div>
        </header>
        <main className="content"><Outlet /></main>
      </div>
    </div>
  );
}
