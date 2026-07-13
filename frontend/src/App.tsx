import { Route, Routes } from 'react-router-dom';
import { RequireAuth } from './auth/RequireAuth';
import { Layout } from './components/Layout';
import { AclPage } from './pages/AclPage';
import { AdDcPage } from './pages/AdDcPage';
import { ArchitecturePage } from './pages/ArchitecturePage';
import { AuditPage } from './pages/AuditPage';
import { ChangeRequestsPage } from './pages/ChangeRequestsPage';
import { ConfigEditorPage } from './pages/ConfigEditorPage';
import { DashboardPage } from './pages/DashboardPage';
import { DfsPage } from './pages/DfsPage';
import { DomainPage } from './pages/DomainPage';
import { FileManagerPage } from './pages/FileManagerPage';
import { FileSystemsPage } from './pages/FileSystemsPage';
import { JobsPage } from './pages/JobsPage';
import { LoginPage } from './pages/LoginPage';
import { LogsPage } from './pages/LogsPage';
import { NewSharePage } from './pages/NewSharePage';
import { PrintingPage } from './pages/PrintingPage';
import { QuotasPage } from './pages/QuotasPage';
import { ServicesPage } from './pages/ServicesPage';
import { SambaPage } from './pages/SambaPage';
import { SharesPage } from './pages/SharesPage';

export default function App() {
  return <Routes>
    <Route path="login" element={<LoginPage/>}/>
    <Route element={<RequireAuth/>}><Route element={<Layout/>}>
    <Route index element={<DashboardPage/>}/>
    <Route path="samba" element={<SambaPage/>}/>
    <Route path="sistemas-arquivos" element={<FileSystemsPage/>}/>
    <Route path="compartilhamentos" element={<SharesPage/>}/>
    <Route path="compartilhamentos/novo" element={<NewSharePage/>}/>
    <Route path="acl" element={<AclPage/>}/>
    <Route path="arquivos" element={<FileManagerPage/>}/>
    <Route path="configuracoes" element={<ConfigEditorPage/>}/>
    <Route path="dominio" element={<DomainPage/>}/>
    <Route path="ad-dc" element={<AdDcPage/>}/>
    <Route path="impressao" element={<PrintingPage/>}/>
    <Route path="dfs" element={<DfsPage/>}/>
    <Route path="cotas" element={<QuotasPage/>}/>
    <Route path="logs" element={<LogsPage/>}/>
    <Route path="auditoria" element={<AuditPage/>}/>
    <Route path="mudancas" element={<ChangeRequestsPage/>}/>
    <Route path="servicos" element={<ServicesPage/>}/>
    <Route path="tarefas" element={<JobsPage/>}/>
    <Route path="arquitetura" element={<ArchitecturePage/>}/>
    </Route></Route>
  </Routes>;
}
