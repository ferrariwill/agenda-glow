import { Link, useNavigate } from 'react-router-dom'
import { useMemo, useState } from 'react'
import {
  ArrowDownUp,
  Download,
  Filter,
  MessageCircle,
  Pencil,
  Plus,
  RefreshCw,
  UserCheck,
  Users,
  Wallet,
} from 'lucide-react'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { ClienteActionsMenu } from '../../components/dona/ClienteActionsMenu'
import { Alert } from '../../components/ui/Alert'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import type { PlatformClient } from '../../utils/mockDb'
import {
  createCliente,
  getClienteHistorico,
  getDonaStats,
  getTenantClients,
  setClienteAtivo,
  updateCliente,
} from '../../utils/mockDb'
import {
  formatBRL,
  formatDateShortBR,
  formatDateTimeBR,
  formatRelativeDate,
  initials,
} from '../../utils/format'

const PAGE_SIZE = 10
type Filtro = 'TODOS' | 'VIP' | 'NOVOS' | 'INATIVOS'
type ViewMode = 'lista' | 'segmentacao'

type Frequencia = 'SEMANAL' | 'MENSAL' | 'OCASIONAL'
type ClienteStatus = 'VIP GOLD' | 'ATIVO' | 'INATIVO'

const AVATAR_RING = [
  'bg-[#f1dfd4] text-[#7d5141]',
  'bg-[#ffdbcf] text-[#653d2e]',
  'bg-[#ebe0dd] text-[#615b58]',
  'bg-[#efdcd1] text-[#6d6057]',
]

function getFrequencia(c: PlatformClient): Frequencia {
  if (c.visitas === 0) return 'OCASIONAL'
  const daysSince = Math.floor(
    (Date.now() - new Date(`${c.ultima_visita}T12:00:00`).getTime()) / 86400000,
  )
  if (c.visitas >= 6 && daysSince <= 14) return 'SEMANAL'
  if (c.visitas >= 2 && daysSince <= 45) return 'MENSAL'
  return 'OCASIONAL'
}

function getClienteStatus(c: PlatformClient): ClienteStatus {
  if (!c.ativo) return 'INATIVO'
  if (c.visitas > 2 && c.gasto_total >= 300) return 'VIP GOLD'
  return 'ATIVO'
}

function statusClass(status: ClienteStatus) {
  switch (status) {
    case 'VIP GOLD':
      return 'bg-[#d5c3b8]/30 text-[#695c53]'
    case 'ATIVO':
      return 'bg-emerald-100 text-emerald-800'
    case 'INATIVO':
      return 'bg-[#e9e8e7] text-[#514440]'
  }
}

function frequenciaClass(freq: Frequencia) {
  return freq === 'SEMANAL'
    ? 'bg-[#efdcd1] text-[#7d5141]'
    : 'bg-[#e9e8e7] text-[#615b58]'
}

function whatsappUrl(telefone: string) {
  const tel = telefone.replace(/\D/g, '')
  return `https://wa.me/${tel}`
}

function daysSinceVisit(iso: string) {
  return Math.floor((Date.now() - new Date(`${iso}T12:00:00`).getTime()) / 86400000)
}

interface BentoMetricProps {
  icon: React.ReactNode
  badge: string
  label: string
  value: string
}

function BentoMetric({ icon, badge, label, value }: BentoMetricProps) {
  return (
    <div className="rounded-xl border border-[#efdcd1]/30 bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] transition-all hover:-translate-y-0.5 hover:shadow-[0px_10px_30px_rgba(183,132,114,0.12)]">
      <div className="mb-4 flex items-start justify-between">
        <div className="rounded-lg bg-[#efdcd1]/50 p-2 text-[#7d5141]">{icon}</div>
        <span className="rounded-full bg-[#ffdbcf] px-2 py-0.5 text-xs font-bold text-[#7d5141]">
          {badge}
        </span>
      </div>
      <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">{label}</p>
      <h3 className="mt-1 font-display text-2xl font-semibold text-[#7d5141]">{value}</h3>
    </div>
  )
}

export function ClientesDona() {
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [filtro, setFiltro] = useState<Filtro>('TODOS')
  const [viewMode, setViewMode] = useState<ViewMode>('lista')
  const [sortRentavel, setSortRentavel] = useState(false)
  const [statusFilter, setStatusFilter] = useState<'TODOS' | 'ATIVO' | 'INATIVO'>('TODOS')
  const [tick, setTick] = useState(0)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [historicoOpen, setHistoricoOpen] = useState<PlatformClient | null>(null)
  const [editing, setEditing] = useState<PlatformClient | null>(null)
  const [nome, setNome] = useState('')
  const [telefone, setTelefone] = useState('')
  const [email, setEmail] = useState('')

  const refresh = () => setTick((t) => t + 1)
  const stats = useMemo(() => getDonaStats(tenantId), [tenantId, tick])

  const clients = useMemo(() => {
    let list = getTenantClients(tenantId)
    if (filtro === 'VIP') list = list.filter((c) => getClienteStatus(c) === 'VIP GOLD')
    else if (filtro === 'NOVOS') list = list.filter((c) => c.visitas <= 1 && c.ativo)
    else if (filtro === 'INATIVOS') list = list.filter((c) => !c.ativo)
    if (statusFilter === 'ATIVO') list = list.filter((c) => c.ativo)
    else if (statusFilter === 'INATIVO') list = list.filter((c) => !c.ativo)
    if (search.trim()) {
      const q = search.toLowerCase()
      list = list.filter(
        (c) =>
          c.nome.toLowerCase().includes(q) ||
          c.telefone.includes(q) ||
          c.email?.toLowerCase().includes(q),
      )
    }
    if (sortRentavel) list = [...list].sort((a, b) => b.gasto_total - a.gasto_total)
    return list
  }, [tenantId, search, filtro, statusFilter, sortRentavel, tick])

  const totalPages = Math.max(1, Math.ceil(clients.length / PAGE_SIZE))
  const slice = clients.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  const segmentGroups = useMemo(() => {
    const all = getTenantClients(tenantId)
    return {
      vip: all.filter((c) => getClienteStatus(c) === 'VIP GOLD'),
      novos: all.filter((c) => c.visitas <= 1 && c.ativo),
      outros: all.filter(
        (c) => c.ativo && getClienteStatus(c) !== 'VIP GOLD' && c.visitas > 1,
      ),
    }
  }, [tenantId, tick])

  const openCreate = () => {
    setEditing(null)
    setNome('')
    setTelefone('')
    setEmail('')
    setError('')
    setModalOpen(true)
  }

  const openEdit = (c: PlatformClient) => {
    setEditing(c)
    setNome(c.nome)
    setTelefone(c.telefone)
    setEmail(c.email ?? '')
    setError('')
    setModalOpen(true)
  }

  const saveCliente = async () => {
    setError('')
    if (!nome.trim() || !telefone.trim()) {
      setError('Nome e telefone são obrigatórios')
      return
    }
    try {
      if (editing) {
        updateCliente(editing.cliente_id, { nome, telefone, email })
        setSuccess('Cliente atualizado!')
      } else {
        await createCliente(tenantId, { nome, telefone, email })
        setSuccess('Cliente cadastrado!')
      }
      refresh()
      setModalOpen(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Erro ao salvar')
    }
  }

  const exportCsv = () => {
    const rows = [
      ['Nome', 'Telefone', 'E-mail', 'Visitas', 'Frequência', 'Gasto Total', 'Última Visita', 'Status'],
      ...clients.map((c) => [
        c.nome,
        c.telefone,
        c.email ?? '',
        String(c.visitas),
        getFrequencia(c),
        String(c.gasto_total),
        c.ultima_visita,
        getClienteStatus(c),
      ]),
    ]
    const csv = rows.map((r) => r.join(';')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'clientes.csv'
    a.click()
    URL.revokeObjectURL(url)
  }

  const historico = historicoOpen
    ? getClienteHistorico(tenantId, historicoOpen.telefone)
    : []

  const novoClienteBtn = (
    <Button
      className="bg-[#7d5141] hover:bg-[#996958]"
      onClick={openCreate}
    >
      <Plus className="h-4 w-4" />
      Novo Cliente
    </Button>
  )

  const renderActions = (c: PlatformClient) => (
    <div className="flex items-center justify-end gap-1">
      <a
        href={whatsappUrl(c.telefone)}
        target="_blank"
        rel="noopener noreferrer"
        className="rounded-lg p-2 text-[#25D366] transition-colors hover:bg-[#25D366]/10"
        title="WhatsApp"
        aria-label={`WhatsApp ${c.nome}`}
      >
        <MessageCircle className="h-4 w-4 fill-current" />
      </a>
      <button
        type="button"
        onClick={() => openEdit(c)}
        className="rounded-lg p-2 text-aura-muted transition-colors hover:text-[#7d5141]"
        aria-label={`Editar ${c.nome}`}
      >
        <Pencil className="h-4 w-4" />
      </button>
      <ClienteActionsMenu
        ativo={c.ativo}
        onAgendar={() => navigate(`/admin/clientes/${c.cliente_id}/agendar`)}
        onEdit={() => openEdit(c)}
        onHistorico={() => setHistoricoOpen(c)}
        onToggleAtivo={() => {
          setClienteAtivo(c.cliente_id, !c.ativo)
          refresh()
          setSuccess(c.ativo ? 'Cliente desativado.' : 'Cliente reativado.')
        }}
      />
    </div>
  )

  return (
    <DonaLayout
      searchPlaceholder="Buscar por nome ou telefone…"
      searchValue={search}
      onSearchChange={(v) => {
        setSearch(v)
        setPage(1)
      }}
      headerAction={novoClienteBtn}
    >
      <div className="mb-6 flex flex-col gap-4 sm:mb-8 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="font-display text-3xl font-bold text-[#7d5141] sm:text-4xl">
            Gestão de Clientes
          </h1>
          <p className="mt-2 text-sm text-aura-muted">
            Personalize o atendimento e fidelize seus clientes VIP.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex rounded-full bg-[#e9e8e7] p-1">
            <button
              type="button"
              onClick={() => setViewMode('lista')}
              className={[
                'rounded-full px-4 py-1 text-sm font-semibold transition-colors',
                viewMode === 'lista'
                  ? 'bg-white text-[#7d5141] shadow-sm'
                  : 'text-aura-muted hover:text-[#7d5141]',
              ].join(' ')}
            >
              Lista
            </button>
            <button
              type="button"
              onClick={() => setViewMode('segmentacao')}
              className={[
                'rounded-full px-4 py-1 text-sm font-semibold transition-colors',
                viewMode === 'segmentacao'
                  ? 'bg-white text-[#7d5141] shadow-sm'
                  : 'text-aura-muted hover:text-[#7d5141]',
              ].join(' ')}
            >
              Segmentação
            </button>
          </div>
          <div className="sm:hidden">{novoClienteBtn}</div>
        </div>
      </div>

      {error && !modalOpen && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <BentoMetric
          icon={<Users className="h-5 w-5" />}
          badge={stats.novosMes > 0 ? `+${stats.novosMes}` : '—'}
          label="Total de Clientes"
          value={stats.totalClientes.toLocaleString('pt-BR')}
        />
        <BentoMetric
          icon={<UserCheck className="h-5 w-5" />}
          badge={`${stats.pctAtivos}%`}
          label="Clientes Ativos"
          value={stats.clientesAtivos.toLocaleString('pt-BR')}
        />
        <BentoMetric
          icon={<Wallet className="h-5 w-5" />}
          badge={formatBRL(stats.ticketMedio).replace(/\s/g, '')}
          label="Ticket Médio"
          value={formatBRL(stats.ticketMedio)}
        />
        <BentoMetric
          icon={<RefreshCw className="h-5 w-5" />}
          badge={stats.taxaRetorno >= 60 ? 'Alto' : 'Médio'}
          label="Taxa de Retorno"
          value={`${stats.taxaRetorno}%`}
        />
      </div>

      {viewMode === 'segmentacao' ? (
        <div className="grid gap-6 md:grid-cols-3">
          {(
            [
              { title: 'VIP Gold', items: segmentGroups.vip, color: 'border-[#7d5141]/30' },
              { title: 'Novos', items: segmentGroups.novos, color: 'border-emerald-200' },
              { title: 'Recorrentes', items: segmentGroups.outros, color: 'border-[#d6c2bd]' },
            ] as const
          ).map((group) => (
            <div
              key={group.title}
              className={[
                'rounded-xl border bg-white p-4 shadow-[0px_4px_20px_rgba(183,132,114,0.08)]',
                group.color,
              ].join(' ')}
            >
              <h3 className="mb-3 font-display text-lg font-semibold text-[#7d5141]">
                {group.title}
                <span className="ml-2 text-sm font-normal text-aura-muted">
                  ({group.items.length})
                </span>
              </h3>
              <ul className="space-y-2">
                {group.items.length === 0 ? (
                  <li className="text-sm text-aura-muted">Nenhum cliente neste segmento.</li>
                ) : (
                  group.items.map((c) => (
                    <li
                      key={c.cliente_id}
                      className="flex items-center justify-between rounded-lg bg-[#faf9f8] px-3 py-2"
                    >
                      <div className="min-w-0">
                        <Link
                          to={`/admin/clientes/${c.cliente_id}`}
                          className="truncate text-sm font-medium text-[#7d5141] hover:underline"
                        >
                          {c.nome}
                        </Link>
                        <p className="text-xs text-aura-muted">{formatBRL(c.gasto_total)}</p>
                      </div>
                      {renderActions(c)}
                    </li>
                  ))
                )}
              </ul>
            </div>
          ))}
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-[#efdcd1]/20 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
          <div className="flex flex-col gap-3 border-b border-[#e9e8e7] p-4 md:flex-row md:items-center md:justify-between">
            <div className="flex flex-wrap items-center gap-2 overflow-x-auto pb-1 md:pb-0">
              <button
                type="button"
                onClick={() =>
                  setStatusFilter((s) =>
                    s === 'TODOS' ? 'ATIVO' : s === 'ATIVO' ? 'INATIVO' : 'TODOS',
                  )
                }
                className="flex items-center gap-2 whitespace-nowrap rounded-lg border border-[#d6c2bd]/30 bg-[#f4f3f2] px-3 py-2 text-sm text-aura-muted transition-colors hover:border-[#7d5141]"
              >
                <Filter className="h-4 w-4" />
                {statusFilter === 'TODOS'
                  ? 'Filtrar por status'
                  : statusFilter === 'ATIVO'
                    ? 'Ativos'
                    : 'Inativos'}
              </button>
              <button
                type="button"
                onClick={() => setSortRentavel((v) => !v)}
                className={[
                  'flex items-center gap-2 whitespace-nowrap rounded-lg border px-3 py-2 text-sm transition-colors',
                  sortRentavel
                    ? 'border-[#7d5141] bg-[#ffdbcf]/30 text-[#7d5141]'
                    : 'border-[#d6c2bd]/30 bg-[#f4f3f2] text-aura-muted hover:border-[#7d5141]',
                ].join(' ')}
              >
                <ArrowDownUp className="h-4 w-4" />
                Mais rentáveis
              </button>
              <span className="hidden h-6 w-px bg-[#d6c2bd] md:block" />
              <span className="whitespace-nowrap text-sm font-medium text-[#514440]">
                Segmento:
              </span>
              {(
                [
                  ['TODOS', 'Todos'],
                  ['VIP', 'VIP'],
                  ['NOVOS', 'Novos'],
                ] as const
              ).map(([key, label]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => {
                    setFiltro(key)
                    setPage(1)
                  }}
                  className={[
                    'whitespace-nowrap rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wider transition-colors',
                    filtro === key
                      ? 'bg-[#ffdbcf] text-[#7d5141]'
                      : 'bg-[#f4f3f2] text-aura-muted hover:bg-[#efdcd1]',
                  ].join(' ')}
                >
                  {label}
                </button>
              ))}
            </div>
            <button
              type="button"
              onClick={exportCsv}
              className="self-end rounded-lg border border-[#d6c2bd]/30 p-2 text-aura-muted transition-colors hover:text-[#7d5141] md:self-auto"
              aria-label="Exportar CSV"
            >
              <Download className="h-4 w-4" />
            </button>
          </div>

          {/* Mobile cards */}
          <div className="space-y-3 p-3 md:hidden">
            {slice.length === 0 ? (
              <p className="py-8 text-center text-sm text-aura-muted">Nenhum cliente encontrado.</p>
            ) : (
              slice.map((c, i) => {
                const freq = getFrequencia(c)
                const status = getClienteStatus(c)
                const days = daysSinceVisit(c.ultima_visita)
                return (
                  <div
                    key={c.cliente_id}
                    className="rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex min-w-0 items-center gap-3">
                        <div
                          className={[
                            'flex h-10 w-10 shrink-0 items-center justify-center rounded-full ring-2 ring-[#7d5141]/10',
                            AVATAR_RING[i % AVATAR_RING.length],
                          ].join(' ')}
                        >
                          <span className="text-xs font-bold">{initials(c.nome)}</span>
                        </div>
                        <div className="min-w-0">
                          <Link
                            to={`/admin/clientes/${c.cliente_id}`}
                            className="truncate font-semibold text-[#7d5141] hover:underline"
                          >
                            {c.nome}
                          </Link>
                          <p className="truncate text-xs text-aura-muted">
                            {c.email ?? c.telefone}
                          </p>
                        </div>
                      </div>
                      {renderActions(c)}
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      <span
                        className={[
                          'rounded-full px-2 py-0.5 text-[10px] font-bold uppercase',
                          frequenciaClass(freq),
                        ].join(' ')}
                      >
                        {freq}
                      </span>
                      <span
                        className={[
                          'rounded-full px-2 py-0.5 text-[10px] font-bold uppercase',
                          statusClass(status),
                        ].join(' ')}
                      >
                        {status}
                      </span>
                    </div>
                    <div className="mt-3 flex justify-between text-sm">
                      <div>
                        <p>{formatDateShortBR(c.ultima_visita)}</p>
                        <p
                          className={[
                            'text-[11px] font-medium uppercase',
                            days > 60 ? 'text-red-600' : 'text-aura-muted',
                          ].join(' ')}
                        >
                          {formatRelativeDate(c.ultima_visita)}
                        </p>
                      </div>
                      <p className="font-semibold text-[#7d5141]">{formatBRL(c.gasto_total)}</p>
                    </div>
                  </div>
                )
              })
            )}
          </div>

          {/* Desktop table */}
          <div className="hidden overflow-x-auto md:block">
            <table className="w-full border-collapse text-left">
              <thead>
                <tr className="bg-[#f4f3f2]">
                  {[
                    'Nome do Cliente',
                    'Última Visita',
                    'Frequência',
                    'Total Gasto',
                    'Status',
                    'Ações',
                  ].map((h) => (
                    <th
                      key={h}
                      className={[
                        'border-b border-[#e9e8e7] px-5 py-3 text-[10px] font-bold uppercase tracking-wider text-aura-muted',
                        h === 'Ações' ? 'text-right' : '',
                      ].join(' ')}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e9e8e7]">
                {slice.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-5 py-12 text-center text-aura-muted">
                      Nenhum cliente encontrado.
                    </td>
                  </tr>
                ) : (
                  slice.map((c, i) => {
                    const freq = getFrequencia(c)
                    const status = getClienteStatus(c)
                    const days = daysSinceVisit(c.ultima_visita)
                    return (
                      <tr
                        key={c.cliente_id}
                        className="group transition-colors hover:bg-[#faf9f8]"
                      >
                        <td className="px-5 py-4">
                          <div className="flex items-center gap-4">
                            <div
                              className={[
                                'flex h-10 w-10 shrink-0 items-center justify-center rounded-full ring-2 ring-[#7d5141]/10',
                                AVATAR_RING[i % AVATAR_RING.length],
                              ].join(' ')}
                            >
                              <span className="text-xs font-bold">{initials(c.nome)}</span>
                            </div>
                            <div>
                              <Link
                                to={`/admin/clientes/${c.cliente_id}`}
                                className="text-sm font-semibold text-[#7d5141] group-hover:underline"
                              >
                                {c.nome}
                              </Link>
                              <p className="text-xs text-aura-muted">
                                {c.email ?? c.telefone}
                              </p>
                            </div>
                          </div>
                        </td>
                        <td className="px-5 py-4">
                          <p className="text-sm text-aura-anthracite">
                            {formatDateShortBR(c.ultima_visita)}
                          </p>
                          <p
                            className={[
                              'text-[11px] font-medium uppercase',
                              days > 60 ? 'text-red-600' : 'text-aura-muted',
                            ].join(' ')}
                          >
                            {formatRelativeDate(c.ultima_visita)}
                          </p>
                        </td>
                        <td className="px-5 py-4">
                          <span
                            className={[
                              'inline-flex rounded-full px-3 py-1 text-[11px] font-bold uppercase',
                              frequenciaClass(freq),
                            ].join(' ')}
                          >
                            {freq}
                          </span>
                        </td>
                        <td className="px-5 py-4">
                          <p className="text-sm font-semibold text-[#7d5141]">
                            {formatBRL(c.gasto_total)}
                          </p>
                        </td>
                        <td className="px-5 py-4">
                          <span
                            className={[
                              'inline-flex rounded-full px-3 py-1 text-[11px] font-bold uppercase',
                              statusClass(status),
                            ].join(' ')}
                          >
                            {status}
                          </span>
                        </td>
                        <td className="px-5 py-4">{renderActions(c)}</td>
                      </tr>
                    )
                  })
                )}
              </tbody>
            </table>
          </div>

          <div className="flex flex-col items-center justify-between gap-3 border-t border-[#e9e8e7] bg-[#f4f3f2]/50 px-4 py-4 sm:flex-row sm:px-5">
            <p className="text-sm text-aura-muted">
              Mostrando{' '}
              <span className="font-bold text-[#7d5141]">
                {clients.length === 0
                  ? '0'
                  : `${(page - 1) * PAGE_SIZE + 1}-${Math.min(page * PAGE_SIZE, clients.length)}`}
              </span>{' '}
              de <span className="font-bold text-[#7d5141]">{clients.length}</span> clientes
            </p>
            <div className="flex items-center gap-1">
              <button
                type="button"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="rounded-lg p-2 text-aura-muted hover:bg-[#efdcd1] disabled:opacity-30"
              >
                ‹
              </button>
              {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setPage(p)}
                  className={[
                    'min-w-[2rem] rounded-lg px-2 py-1.5 text-xs font-bold',
                    p === page
                      ? 'bg-[#7d5141] text-white'
                      : 'text-aura-muted hover:bg-[#efdcd1]',
                  ].join(' ')}
                >
                  {p}
                </button>
              ))}
              {totalPages > 5 && <span className="px-2 text-aura-muted">…</span>}
              <button
                type="button"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="rounded-lg p-2 text-aura-muted hover:bg-[#efdcd1] disabled:opacity-30"
              >
                ›
              </button>
            </div>
          </div>
        </div>
      )}

      <DonaFooter />

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editing ? 'Editar cliente' : 'Novo cliente'}
        footer={
          <>
            <Button variant="secondary" onClick={() => setModalOpen(false)}>
              Cancelar
            </Button>
            <Button className="bg-[#7d5141] hover:bg-[#996958]" onClick={saveCliente}>
              Salvar
            </Button>
          </>
        }
      >
        {error && modalOpen && <p className="mb-3 text-sm text-red-600">{error}</p>}
        <div className="space-y-4">
          <Input label="Nome completo" value={nome} onChange={(e) => setNome(e.target.value)} required />
          <Input
            label="WhatsApp (com DDD)"
            value={telefone}
            onChange={(e) => setTelefone(e.target.value)}
            placeholder="5511999999999"
            required
          />
          <Input
            label="E-mail (opcional)"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        </div>
      </Modal>

      <Modal
        open={!!historicoOpen}
        onClose={() => setHistoricoOpen(null)}
        title={`Histórico — ${historicoOpen?.nome ?? ''}`}
        footer={
          <Button variant="secondary" onClick={() => setHistoricoOpen(null)}>
            Fechar
          </Button>
        }
      >
        {historico.length === 0 ? (
          <p className="text-sm text-aura-muted">Nenhum agendamento registrado.</p>
        ) : (
          <ul className="max-h-64 space-y-2 overflow-y-auto">
            {historico.map((h) => (
              <li
                key={h.id}
                className="rounded-lg border border-aura-border bg-aura-surface px-3 py-2 text-sm"
              >
                <p className="font-medium">{formatDateTimeBR(h.data, h.hora_inicio)}</p>
                <p className="text-aura-muted">
                  {h.servico_nome} · {h.profissional_nome}
                </p>
                <Badge variant={h.status === 'CANCELADO' ? 'danger' : 'muted'}>{h.status}</Badge>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </DonaLayout>
  )
}
