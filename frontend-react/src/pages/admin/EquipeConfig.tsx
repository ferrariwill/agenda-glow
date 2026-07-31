import { useMemo, useState } from 'react'
import {
  BarChart3,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Clock,
  Edit,
  Filter,
  MoreVertical,
  Search,
  UserPlus,
  Users,
} from 'lucide-react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { ActionsDropdown } from '../../components/ui/ActionsDropdown'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { useAuth } from '../../contexts/AuthContext'
import { PlanLimitExceededError, type Profissional } from '../../types'
import {
  getDb,
  getEquipeStats,
  getProfissionalEmail,
  profissionalStatusLabel,
  updateProfissional,
} from '../../utils/mockDb'
import { formatProfissionalDisponibilidade } from '../../utils/format'

const PAGE_SIZE = 8
const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]'

const DEFAULT_AVATAR =
  'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80'

type StatusFilter = '' | 'ativo' | 'inativo' | 'pendente'

function statusPill(status: ReturnType<typeof profissionalStatusLabel>) {
  switch (status) {
    case 'ATIVO':
      return 'bg-[#f0f4f1] text-[#2d4a3e]'
    case 'PENDENTE':
      return 'bg-[#f1dfd4] text-[#50443c]'
    default:
      return 'bg-[#f2f2f2] text-[#4a4a4a]'
  }
}

export function EquipeConfig() {
  const navigate = useNavigate()
  const location = useLocation()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [espFilter, setEspFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')
  const [page, setPage] = useState(1)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(
    (location.state as { success?: string } | null)?.success ?? '',
  )
  const [toast, setToast] = useState('')

  const refresh = () => setDb(getDb())
  const stats = useMemo(() => getEquipeStats(tenantId), [db, tenantId])
  const especialidades = db.especialidades.filter((e) => e.tenant_id === tenantId)

  const profissionais = useMemo(() => {
    const q = search.trim().toLowerCase()
    let list = db.profissionais.filter((p) => p.tenant_id === tenantId)
    if (espFilter) list = list.filter((p) => p.especialidade_id === espFilter)
    if (statusFilter) {
      list = list.filter((p) => {
        const st = profissionalStatusLabel(p)
        if (statusFilter === 'ativo') return st === 'ATIVO'
        if (statusFilter === 'inativo') return st === 'INATIVO'
        if (statusFilter === 'pendente') return st === 'PENDENTE'
        return true
      })
    }
    if (q) {
      list = list.filter((p) => {
        const esp = especialidades.find((e) => e.id === p.especialidade_id)?.nome ?? ''
        return (
          p.nome.toLowerCase().includes(q) ||
          getProfissionalEmail(p).toLowerCase().includes(q) ||
          esp.toLowerCase().includes(q)
        )
      })
    }
    return list.sort((a, b) => a.nome.localeCompare(b.nome))
  }, [db.profissionais, tenantId, espFilter, statusFilter, search, especialidades])

  const totalPages = Math.max(1, Math.ceil(profissionais.length / PAGE_SIZE))
  const pageSafe = Math.min(page, totalPages)
  const pageItems = profissionais.slice((pageSafe - 1) * PAGE_SIZE, pageSafe * PAGE_SIZE)
  const rangeStart = profissionais.length === 0 ? 0 : (pageSafe - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(pageSafe * PAGE_SIZE, profissionais.length)

  const clearFilters = () => {
    setSearch('')
    setEspFilter('')
    setStatusFilter('')
    setPage(1)
  }

  const toggleProfissional = async (prof: Profissional) => {
    setError('')
    try {
      if (prof.pendente_aprovacao) {
        await updateProfissional(prof.id, { ativo: true, pendente_aprovacao: false })
        setSuccess('Profissional aprovado e ativado.')
      } else {
        await updateProfissional(prof.id, { ativo: !prof.ativo })
        setSuccess(prof.ativo ? 'Profissional desativado.' : 'Profissional reativado.')
      }
      refresh()
    } catch (err) {
      if (err instanceof PlanLimitExceededError) {
        setError('Não é possível ativar: limite do plano atingido.')
      }
    }
  }

  const getEspNome = (id: string) =>
    especialidades.find((e) => e.id === id)?.nome ?? '—'

  const renderProfActions = (p: Profissional) => (
    <div className="relative flex justify-end gap-1">
      <button
        type="button"
        title="Ver Desempenho"
        aria-label="Ver desempenho"
        onClick={() => setToast('Relatório de desempenho em breve.')}
        className="inline-flex h-11 w-11 items-center justify-center rounded text-[#514440] transition-all hover:bg-[#996958]/10 hover:text-[#7d5141]"
      >
        <BarChart3 className="h-5 w-5" />
      </button>
      <button
        type="button"
        title="Editar"
        aria-label="Editar"
        onClick={() => navigate(`/admin/equipe/${p.id}/edit`)}
        className="inline-flex h-11 w-11 items-center justify-center rounded text-[#514440] transition-all hover:bg-[#996958]/10 hover:text-[#7d5141]"
      >
        <Edit className="h-5 w-5" />
      </button>
      <ActionsDropdown
        icon={MoreVertical}
        iconClassName="h-5 w-5"
        triggerClassName="inline-flex h-11 w-11 items-center justify-center rounded text-[#514440] transition-all hover:bg-[#996958]/10 hover:text-[#7d5141]"
        menuClassName="border-[#e5d3c8]/50"
        items={
          p.pendente_aprovacao
            ? [
                {
                  label: 'Aprovar profissional',
                  onClick: () => void toggleProfissional(p),
                },
              ]
            : [
                {
                  label: 'Editar perfil',
                  onClick: () => navigate(`/admin/equipe/${p.id}/edit`),
                },
                {
                  label: p.ativo ? 'Desativar' : 'Reativar',
                  onClick: () => void toggleProfissional(p),
                },
              ]
        }
      />
    </div>
  )

  const profissionalColumns: EntityColumn<Profissional>[] = [
    {
      header: 'Profissional',
      cell: (p) => {
        const status = profissionalStatusLabel(p)
        const inactive = status === 'INATIVO'
        return (
          <div className="flex items-center gap-4">
            <img
              src={p.foto_url ?? DEFAULT_AVATAR}
              alt=""
              className={[
                'h-12 w-12 rounded-full border-2 object-cover',
                inactive ? 'border-[#d6c2bd]/30 grayscale' : 'border-[#ffdbcf]',
              ].join(' ')}
            />
            <div>
              <p className="text-lg font-semibold text-[#7d5141]">{p.nome}</p>
              <p className="text-sm text-[#514440]/70">{getProfissionalEmail(p)}</p>
            </div>
          </div>
        )
      },
    },
    {
      header: 'Especialidade',
      cell: (p) => (
        <span className="rounded bg-[#f1dfd4] px-2 py-1 text-xs font-semibold text-[#50443c]">
          {getEspNome(p.especialidade_id)}
        </span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Disponibilidade',
      cell: (p) => (
        <span className="whitespace-nowrap text-sm text-[#514440]">
          {formatProfissionalDisponibilidade(p.expedientes)}
        </span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Status',
      cell: (p) => {
        const status = profissionalStatusLabel(p)
        return (
          <span
            className={[
              'inline-block rounded-full px-4 py-1 text-xs font-bold',
              statusPill(status),
            ].join(' ')}
          >
            {status}
          </span>
        )
      },
    },
    {
      header: 'Ações',
      cell: renderProfActions,
      align: 'right',
    },
  ]

  const renderProfCard = (p: Profissional) => {
    const status = profissionalStatusLabel(p)
    const inactive = status === 'INATIVO'
    return (
      <div className="rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4">
        <div className="flex items-start gap-3">
          <img
            src={p.foto_url ?? DEFAULT_AVATAR}
            alt=""
            className={[
              'h-12 w-12 shrink-0 rounded-full border-2 object-cover',
              inactive ? 'border-[#d6c2bd]/30 grayscale' : 'border-[#ffdbcf]',
            ].join(' ')}
          />
          <div className="min-w-0 flex-1">
            <p className="font-semibold text-[#7d5141]">{p.nome}</p>
            <p className="text-xs text-[#514440]/70">{getEspNome(p.especialidade_id)}</p>
          </div>
          {renderProfActions(p)}
        </div>
        {/* ≤2 decisões: status + disponibilidade */}
        <div className="mt-3 flex items-center justify-between gap-3">
          <span
            className={[
              'inline-block rounded-full px-3 py-1 text-xs font-bold',
              statusPill(status),
            ].join(' ')}
          >
            {status}
          </span>
          <p className="shrink-0 whitespace-nowrap text-xs text-[#514440]">
            {formatProfissionalDisponibilidade(p.expedientes)}
          </p>
        </div>
      </div>
    )
  }

  const novoBtn = (
    <Link
      to="/admin/equipe/novo"
      className="flex items-center gap-2 rounded-xl bg-[#7d5141] px-6 py-3 text-sm font-semibold text-white shadow-lg transition-all hover:opacity-90 active:scale-95"
    >
      <UserPlus className="h-5 w-5" />
      Novo Profissional
    </Link>
  )

  const feedbackMessage = success || toast || null

  return (
    <DonaLayout headerAction={novoBtn}>
      {/* Header */}
      <header className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <nav className="mb-1 flex items-center gap-1 text-sm text-[#514440]">
            <span>Equipe</span>
            <ChevronRight className="h-3.5 w-3.5" />
            <span className="font-semibold text-[#7d5141]">Profissionais</span>
          </nav>
          <h1 className="font-display text-3xl font-bold text-[#7d5141]">
            Equipe do Salão
          </h1>
        </div>
        <div className="sm:hidden">{novoBtn}</div>
      </header>

      {error && (
        <Alert variant="error" className="mb-6" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}
      <ToastFeedback
        message={feedbackMessage}
        onDismiss={() => {
          setSuccess('')
          setToast('')
        }}
      />

      {/* KPIs */}
      <div className="mb-8 grid gap-6 md:grid-cols-3">
        <div className={`flex items-center gap-6 p-6 ${GLASS}`}>
          <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-[#ffdbcf] text-[#7d5141]">
            <Users className="h-8 w-8" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]/70">
              Total de membros
            </p>
            <p className="font-display text-3xl font-bold text-[#7d5141]">
              {String(stats.total).padStart(2, '0')}
            </p>
          </div>
        </div>
        <div className={`flex items-center gap-6 p-6 ${GLASS}`}>
          <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-[#f0f4f1] text-[#2d4a3e]">
            <CheckCircle2 className="h-8 w-8" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]/70">
              Profissionais ativos
            </p>
            <p className="font-display text-3xl font-bold text-[#695c53]">
              {String(stats.ativos).padStart(2, '0')}
            </p>
          </div>
        </div>
        <div className={`flex items-center gap-6 p-6 ${GLASS}`}>
          <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-[#f1dfd4] text-[#50443c]">
            <Clock className="h-8 w-8" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]/70">
              Aguardando aprovação
            </p>
            <p className="font-display text-3xl font-bold text-[#6d6057]">
              {String(stats.pendentes).padStart(2, '0')}
            </p>
          </div>
        </div>
      </div>

      {/* Filtros */}
      <div
        className={`mb-6 flex flex-col items-center justify-between gap-4 p-4 md:flex-row ${GLASS}`}
      >
        <div className="relative w-full md:w-1/3">
          <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[#83746f]" />
          <input
            type="text"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
            placeholder="Buscar por nome ou especialidade..."
            className="w-full rounded-lg border-none bg-[#f4f3f2] py-3 pl-11 pr-4 text-sm transition-all focus:bg-white focus:ring-1 focus:ring-[#7d5141]"
          />
        </div>
        <div className="flex w-full gap-4 md:w-auto">
          <select
            value={espFilter}
            onChange={(e) => {
              setEspFilter(e.target.value)
              setPage(1)
            }}
            className="min-w-[140px] cursor-pointer rounded-lg border-none bg-[#f4f3f2] px-4 py-3 text-sm focus:ring-1 focus:ring-[#7d5141]"
          >
            <option value="">Especialidade</option>
            {especialidades.map((e) => (
              <option key={e.id} value={e.id}>
                {e.nome}
              </option>
            ))}
          </select>
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value as StatusFilter)
              setPage(1)
            }}
            className="min-w-[140px] cursor-pointer rounded-lg border-none bg-[#f4f3f2] px-4 py-3 text-sm focus:ring-1 focus:ring-[#7d5141]"
          >
            <option value="">Status</option>
            <option value="ativo">Ativo</option>
            <option value="inativo">Inativo</option>
            <option value="pendente">Aguardando</option>
          </select>
          <button
            type="button"
            onClick={() => setToast('Filtros aplicados.')}
            className="inline-flex h-11 w-11 items-center justify-center rounded-lg bg-[#e9e8e7] text-[#514440] transition-colors hover:bg-[#d6c2bd]/30"
            aria-label="Aplicar filtros"
          >
            <Filter className="h-5 w-5" />
          </button>
        </div>
      </div>

      {/* Listagem */}
      <div className={`overflow-hidden ${GLASS}`}>
        <ResponsiveEntityList
          items={pageItems}
          getKey={(p) => p.id}
          renderCard={renderProfCard}
          columns={profissionalColumns}
          emptyTitle="Nenhum profissional encontrado"
          emptyDescription="Não conseguimos encontrar membros da equipe que correspondam aos seus filtros atuais. Tente ajustar a busca ou limpe os filtros."
          emptyAction={
            <button
              type="button"
              onClick={clearFilters}
              className="rounded-xl border border-[#7d5141] px-6 py-3 text-sm font-semibold text-[#7d5141] transition-colors hover:bg-[#996958]/10"
            >
              Limpar Filtros
            </button>
          }
          tableFrom="md"
          cardListClassName="space-y-3 p-4"
        />

        {profissionais.length > 0 && (
          <div className="flex items-center justify-between border-t border-[#d6c2bd]/30 bg-[#f4f3f2] px-6 py-4">
            <p className="text-sm text-[#514440]">
              Mostrando {rangeStart}-{rangeEnd} de {profissionais.length} profissionais
            </p>
            <div className="flex gap-2">
              <button
                type="button"
                disabled={pageSafe <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="inline-flex h-11 w-11 items-center justify-center rounded border border-[#d6c2bd]/50 text-[#514440] hover:bg-white disabled:opacity-50"
                aria-label="Página anterior"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
              {Array.from({ length: totalPages }, (_, i) => i + 1)
                .slice(0, 5)
                .map((n) => (
                  <button
                    key={n}
                    type="button"
                    onClick={() => setPage(n)}
                    className={[
                      'inline-flex h-11 min-w-11 items-center justify-center rounded border px-3 text-sm font-bold',
                      n === pageSafe
                        ? 'border-[#d6c2bd]/50 bg-white text-[#7d5141] shadow-sm'
                        : 'border-[#d6c2bd]/50 text-[#514440] hover:bg-white',
                    ].join(' ')}
                  >
                    {n}
                  </button>
                ))}
              <button
                type="button"
                disabled={pageSafe >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                className="inline-flex h-11 w-11 items-center justify-center rounded border border-[#d6c2bd]/50 text-[#514440] hover:bg-white disabled:opacity-50"
                aria-label="Próxima página"
              >
                <ChevronRight className="h-5 w-5" />
              </button>
            </div>
          </div>
        )}
      </div>

      <DonaFooter />
    </DonaLayout>
  )
}
