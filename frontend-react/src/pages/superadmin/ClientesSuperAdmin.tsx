import { useMemo, useState } from 'react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
  MetricCard,
  TablePagination,
} from '../../components/superadmin/SuperAdminLayout'
import { Badge } from '../../components/ui/Badge'
import { getPlatformClients, getSuperAdminStats } from '../../utils/mockDb'
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
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-aura-border bg-aura-surface/50 text-left text-aura-muted">
                <th className="px-4 py-3 font-medium">Nome do Cliente</th>
                <th className="px-4 py-3 font-medium">Salão</th>
                <th className="px-4 py-3 font-medium">Telefone</th>
                <th className="px-4 py-3 font-medium">Última Visita</th>
                <th className="px-4 py-3 font-medium">Gasto Total</th>
              </tr>
            </thead>
            <tbody>
              {slice.map((c) => (
                <tr key={c.id} className="border-b border-aura-border/60 hover:bg-aura-surface/30">
                  <td className="px-4 py-3.5">
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary">
                        {initials(c.nome)}
                      </div>
                      <div>
                        <p className="font-medium">{c.nome}</p>
                        {c.visitas > 2 && <Badge variant="success">VIP</Badge>}
                        {c.visitas === 1 && <Badge variant="muted">NOVO</Badge>}
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3.5 text-aura-muted">{c.tenant_nome}</td>
                  <td className="px-4 py-3.5 text-aura-muted">{c.telefone}</td>
                  <td className="px-4 py-3.5 text-aura-muted">{formatRelativeDate(c.ultima_visita)}</td>
                  <td className="px-4 py-3.5 font-medium">{formatBRL(c.gasto_total)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
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
