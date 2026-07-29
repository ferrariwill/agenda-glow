import { useState } from 'react'
import { Plus } from 'lucide-react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
  MetricCard,
  TablePagination,
} from '../../components/superadmin/SuperAdminLayout'
import { Badge, statusTenantBadge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { getDb, getPlano, getSuperAdminStats } from '../../utils/mockDb'
import { formatBRL, formatDateBR } from '../../utils/format'
import { Link } from 'react-router-dom'

export function DashboardSuperAdmin() {
  const db = getDb()
  const stats = getSuperAdminStats()
  const [page, setPage] = useState(1)
  const pageSize = 5

  const tenants = [...db.tenants].sort(
    (a, b) => b.criado_em.localeCompare(a.criado_em),
  )
  const totalPages = Math.max(1, Math.ceil(tenants.length / pageSize))
  const slice = tenants.slice((page - 1) * pageSize, page * pageSize)

  return (
    <SuperAdminLayout searchPlaceholder="Buscar salões…">
      <PageHeader
        title="Dashboard Global"
        subtitle="Visão executiva da plataforma Aura Beauty SaaS."
        action={
          <Link to="/superadmin/saloes">
            <Button>
              <Plus className="h-4 w-4" />
              Novo Salão
            </Button>
          </Link>
        }
      />

      <div className="mb-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="MRR" value={formatBRL(stats.mrr)} />
        <MetricCard
          label="Salões Ativos"
          value={String(stats.ativos)}
          highlight="success"
        />
        <MetricCard
          label="Assinaturas Vencidas"
          value={String(stats.vencidos)}
          highlight={stats.vencidos > 0 ? 'danger' : 'default'}
        />
        <MetricCard label="Planos Ativos" value={String(db.planos.filter((p) => p.ativo).length)} />
      </div>

      <div className="overflow-hidden rounded-lg border border-aura-border bg-white shadow-sm">
        <div className="border-b border-aura-border px-4 py-4">
          <h2 className="font-display text-lg font-semibold">Salões recentes</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-aura-border bg-aura-surface/50 text-left text-aura-muted">
                <th className="px-4 py-3 font-medium">Salão</th>
                <th className="px-4 py-3 font-medium">Plano</th>
                <th className="px-4 py-3 font-medium">Vencimento</th>
                <th className="px-4 py-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {slice.map((t) => {
                const plano = getPlano(t.plano_id)
                return (
                  <tr key={t.id} className="border-b border-aura-border/60 hover:bg-aura-surface/30">
                    <td className="px-4 py-3.5 font-medium">{t.nome}</td>
                    <td className="px-4 py-3.5 text-aura-muted">{plano?.nome ?? '—'}</td>
                    <td className="px-4 py-3.5 text-aura-muted">{formatDateBR(t.data_vencimento)}</td>
                    <td className="px-4 py-3.5">
                      <Badge variant={statusTenantBadge(t.status)}>{t.status}</Badge>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        <TablePagination
          page={page}
          totalPages={totalPages}
          total={tenants.length}
          pageSize={pageSize}
          onPageChange={setPage}
          label="salões"
        />
      </div>

      <SuperAdminFooter />
    </SuperAdminLayout>
  )
}
