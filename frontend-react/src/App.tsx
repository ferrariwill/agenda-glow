import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { DataProvider } from './data/DataProvider'
import { PublicCatalogLoader } from './data/PublicCatalogLoader'
import { MockModeBanner } from './components/layout/MockModeBanner'
import { ClienteProtectedRoute } from './components/layout/ClienteProtectedRoute'
import { ProtectedRoute } from './components/layout/ProtectedRoute'
import { LoginForm } from './pages/auth/LoginForm'
import { SubscriptionBlocked } from './pages/admin/SubscriptionBlocked'
import { DashboardDona } from './pages/admin/DashboardDona'
import { ClientesDona } from './pages/admin/ClientesDona'
import { ClienteDetalheDona } from './pages/admin/ClienteDetalheDona'
import { AgendarClienteDona } from './pages/admin/AgendarClienteDona'
import { CalendarioGeral } from './pages/admin/CalendarioGeral'
import { EquipeConfig } from './pages/admin/EquipeConfig'
import { ProfissionalFormDona } from './pages/admin/ProfissionalFormDona'
import { ServicosDona } from './pages/admin/ServicosDona'
import { ServicoFormDona } from './pages/admin/ServicoFormDona'
import { InsumosDona } from './pages/admin/InsumosDona'
import { InsumoFormDona } from './pages/admin/InsumoFormDona'
import { EspecialidadesDona } from './pages/admin/EspecialidadesDona'
import { FinanceiroDona } from './pages/admin/FinanceiroDona'
import { TransacaoFormDona } from './pages/admin/TransacaoFormDona'
import { ConfiguracoesSalao } from './pages/admin/ConfiguracoesSalao'
import { WhatsAppIntegracao } from './pages/admin/WhatsAppIntegracao'
import { DashboardSuperAdmin } from './pages/superadmin/DashboardSuperAdmin'
import { GerenciamentoSaloes } from './pages/superadmin/GerenciamentoSaloes'
import { PlanosSaaS } from './pages/superadmin/PlanosSaaS'
import { FinanceiroSuperAdmin } from './pages/superadmin/FinanceiroSuperAdmin'
import { AgendaProfissional } from './pages/profissional/AgendaProfissional'
import { DashboardProfissional } from './pages/profissional/DashboardProfissional'
import { SecretariaClientes } from './pages/secretaria/SecretariaClientes'
import { PaginaCatalogo } from './pages/public/PaginaCatalogo'
import { ClienteLogin } from './pages/public/ClienteLogin'
import { ClienteCadastro } from './pages/public/ClienteCadastro'
import { ClienteAgendar } from './pages/public/ClienteAgendar'
import { ClienteMinhaConta } from './pages/public/ClienteMinhaConta'
import { AprovacaoPublica } from './pages/public/AprovacaoPublica'
import { CancelamentoRecuperacao } from './pages/public/CancelamentoRecuperacao'
import { AntecipacaoOferta } from './pages/public/AntecipacaoOferta'
import { GestaoAgendamentoPublica } from './pages/public/GestaoAgendamentoPublica'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <DataProvider>
          <MockModeBanner />
          <Routes>
          <Route path="/login" element={<LoginForm />} />
          <Route path="/login/dona" element={<LoginForm />} />
          <Route path="/login/profissional" element={<LoginForm />} />
          <Route path="/login/secretaria" element={<LoginForm />} />
          <Route path="/login/superadmin" element={<LoginForm />} />

          <Route element={<ProtectedRoute allowedRoles={['SUPER_ADMIN']} />}>
            <Route path="/superadmin/dashboard" element={<DashboardSuperAdmin />} />
            <Route path="/superadmin/saloes" element={<GerenciamentoSaloes />} />
            <Route path="/superadmin/planos" element={<PlanosSaaS />} />
            <Route path="/superadmin/financeiro" element={<FinanceiroSuperAdmin />} />
          </Route>

          <Route element={<ProtectedRoute allowedRoles={['DONA']} checkSubscription />}>
            <Route path="/admin/dashboard" element={<DashboardDona />} />
            <Route path="/admin/clientes" element={<ClientesDona />} />
            <Route path="/admin/clientes/:id" element={<ClienteDetalheDona />} />
            <Route path="/admin/clientes/:id/agendar" element={<AgendarClienteDona />} />
            <Route path="/admin/calendario" element={<CalendarioGeral />} />
            <Route path="/admin/minha-agenda" element={<AgendaProfissional />} />
            <Route path="/admin/insumos" element={<InsumosDona />} />
            <Route path="/admin/insumos/novo" element={<InsumoFormDona />} />
            <Route path="/admin/insumos/:id/edit" element={<InsumoFormDona />} />
            <Route path="/admin/servicos" element={<ServicosDona />} />
            <Route path="/admin/servicos/novo" element={<ServicoFormDona />} />
            <Route path="/admin/servicos/:id/edit" element={<ServicoFormDona />} />
            <Route path="/admin/especialidades" element={<EspecialidadesDona />} />
            <Route path="/admin/equipe" element={<EquipeConfig />} />
            <Route path="/admin/equipe/novo" element={<ProfissionalFormDona />} />
            <Route path="/admin/equipe/:id/edit" element={<ProfissionalFormDona />} />
            <Route path="/admin/financeiro" element={<FinanceiroDona />} />
            <Route path="/admin/financeiro/novo" element={<TransacaoFormDona />} />
            <Route path="/admin/financeiro/:id/edit" element={<TransacaoFormDona />} />
            <Route path="/admin/configuracoes" element={<ConfiguracoesSalao />} />
            <Route path="/admin/whatsapp" element={<WhatsAppIntegracao />} />
            <Route path="/admin/bloqueado" element={<SubscriptionBlocked />} />
            <Route path="/admin/comissoes" element={<Navigate to="/admin/financeiro" replace />} />
          </Route>

          <Route element={<ProtectedRoute allowedRoles={['PROFISSIONAL']} checkSubscription />}>
            <Route path="/profissional/dashboard" element={<DashboardProfissional />} />
            <Route path="/profissional/agenda" element={<AgendaProfissional />} />
            <Route
              path="/dashboard/profissional"
              element={<Navigate to="/profissional/dashboard" replace />}
            />
          </Route>

          <Route element={<ProtectedRoute allowedRoles={['SECRETARIA']} checkSubscription />}>
            <Route path="/secretaria/agenda" element={<CalendarioGeral variant="secretaria" />} />
            <Route path="/secretaria/clientes" element={<SecretariaClientes />} />
            <Route path="/secretaria" element={<Navigate to="/secretaria/agenda" replace />} />
          </Route>

          <Route path="/publico/aprovacao/:id" element={<AprovacaoPublica />} />
          <Route path="/publico/cancelar/:id" element={<CancelamentoRecuperacao />} />
          <Route path="/p/agendamento/:token" element={<GestaoAgendamentoPublica />} />

          {/* Links públicos por token — antes de /:slug para não serem engolidos pelo catálogo. */}
          <Route path="/p/antecipacao/:token" element={<AntecipacaoOferta />} />

          <Route path="/:slug/login" element={<PublicCatalogLoader><ClienteLogin /></PublicCatalogLoader>} />
          <Route path="/:slug/cadastro" element={<PublicCatalogLoader><ClienteCadastro /></PublicCatalogLoader>} />
          <Route element={<ClienteProtectedRoute />}>
            <Route path="/:slug/agendar" element={<PublicCatalogLoader><ClienteAgendar /></PublicCatalogLoader>} />
            <Route path="/:slug/conta" element={<PublicCatalogLoader><ClienteMinhaConta /></PublicCatalogLoader>} />
          </Route>
          <Route path="/:slug" element={<PublicCatalogLoader><PaginaCatalogo /></PublicCatalogLoader>} />

          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
        </DataProvider>
      </AuthProvider>
    </BrowserRouter>
  )
}
