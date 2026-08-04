import { useMemo, useState } from 'react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
  MetricCard,
  TablePagination,
} from '../../components/superadmin/SuperAdminLayout'
import { Badge } from '../../components/ui/Badge'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { getPlatformClients, getSuperAdminStats, type PlatformClient } from '../../utils/mockDb'
import { formatBRL, formatRelativeDate, initials } from '../../utils/format'

const PAGE_SIZE = 5

export function ClientesSuperAdmin() {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const stats = getSuperAdminStats()

  const clients = useMemo(() => {
    let list = getPlatformClients()
    if (search.trim()) {
      const q = search.toLowerCase()
      list = list.filter(
        (c) =>
          c.nome.toLowerCase().includes(q) ||
          c.telefone.includes(q) ||
          c.tenant_nome.toLowerCase().includes(q),
      )
    }
    return list
  }, [search])

  const totalPages = Math.max(1, Math.ceil(clients.length / PAGE_SIZE))
  const slice = clients.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  const columns: EntityColumn<PlatformClient>[] = [
    {
      header: 'Nome do Cliente',
      cell: (c) => (
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary">
            {initials(c.nome)}
          </div>
          <div className="min-w-0">
            <p className="font-medium">{c.nome}</p>
            {c.visitas > 2 && <Badge variant="success">VIP</Badge>}
            {c.visitas === 1 && <Badge variant="muted">NOVO</Badge>}
          </div>
        </div>
      ),
    },
    {
      header: 'Salão',
      cell: (c) => <span className="text-aura-muted">{c.tenant_nome}</span>,
    },
    {
      header: 'Telefone',
      cell: (c) => (
        <span className="whitespace-nowrap text-aura-muted">{c.telefone}</span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Última Visita',
      cell: (c) => (
        <span className="whitespace-nowrap text-aura-muted">
          {formatRelativeDate(c.ultima_visita)}
        </span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Gasto Total',
      cell: (c) => (
        <span className="whitespace-nowrap font-medium">{formatBRL(c.gasto_total)}</span>
      ),
      align: 'right',
    },
  ]

  const renderCard = (c: PlatformClient) => (
    <div className="rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary">
          {initials(c.nome)}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="font-semibold text-aura-anthracite">{c.nome}</p>
            {c.visitas > 2 && <Badge variant="success">VIP</Badge>}
            {c.visitas === 1 && <Badge variant="muted">NOVO</Badge>}
          </div>
          <p className="truncate text-xs text-aura-muted">{c.tenant_nome}</p>
        </div>
      </div>
      <div className="mt-3 flex items-center justify-between gap-3">
        <span className="whitespace-nowrap text-xs text-aura-muted">
          {formatRelativeDate(c.ultima_visita)}
        </span>
        <span className="shrink-0 whitespace-nowrap text-sm font-semibold text-[#7d5141]">
          {formatBRL(c.gasto_total)}
        </span>
      </div>
    </div>
  )

  return (
    <SuperAdminLayout
      searchPlaceholder="Buscar clientes…"
      searchValue={search}
      onSearchChange={(v) => {
        setSearch(v)
        setPage(1)
      }}
    >
      <PageHeader
        title="Gestão de Clientes"
        subtitle="Visualize e gerencie a base de clientes da plataforma Aura Beauty."
      />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Total de Clientes" value={stats.totalClientes.toLocaleString('pt-BR')} />
        <MetricCard label="Salões Ativos" value={String(stats.ativos)} highlight="success" />
        <MetricCard label="Taxa de Retorno" value={`${stats.taxaRetorno}%`} />
        <MetricCard label="Ticket Médio" value={formatBRL(stats.ticketMedio)} />
      </div>

      <div className="overflow-hidden rounded-lg border border-aura-border bg-white shadow-sm">
        <ResponsiveEntityList
          items={slice}
          getKey={(c) => c.id}
          renderCard={renderCard}
          columns={columns}
          emptyTitle="Nenhum cliente encontrado."
        />
        <TablePagination
          page={page}
          totalPages={totalPages}
          total={clients.length}
          pageSize={PAGE_SIZE}
          onPageChange={setPage}
          label="clientes"
        />
      </div>

      <SuperAdminFooter />
    </SuperAdminLayout>
  )
}
