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
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { getDb, getPlano, getSuperAdminStats } from '../../utils/mockDb'
import { formatBRL, formatDateBR } from '../../utils/format'
import { Link } from 'react-router-dom'
import type { Tenant } from '../../types'

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

  const columns: EntityColumn<Tenant>[] = [
    {
      header: 'Salão',
      cell: (t) => <span className="font-medium">{t.nome}</span>,
    },
    {
      header: 'Plano',
      cell: (t) => (
        <span className="text-aura-muted">{getPlano(t.plano_id)?.nome ?? '—'}</span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Vencimento',
      cell: (t) => (
        <span className="whitespace-nowrap text-aura-muted">
          {formatDateBR(t.data_vencimento)}
        </span>
      ),
    },
    {
      header: 'Status',
      cell: (t) => <Badge variant={statusTenantBadge(t.status)}>{t.status}</Badge>,
    },
  ]

  const renderCard = (t: Tenant) => (
    <div className="rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-semibold text-aura-anthracite">{t.nome}</p>
          <p className="truncate text-xs text-aura-muted">
            {getPlano(t.plano_id)?.nome ?? '—'}
          </p>
        </div>
        <Badge variant={statusTenantBadge(t.status)}>{t.status}</Badge>
      </div>
      <div className="mt-3">
        <span className="whitespace-nowrap text-sm text-aura-muted">
          Vence {formatDateBR(t.data_vencimento)}
        </span>
      </div>
    </div>
  )

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
        <ResponsiveEntityList
          items={slice}
          getKey={(t) => t.id}
          renderCard={renderCard}
          columns={columns}
          emptyTitle="Nenhum salão cadastrado."
          emptyAction={
            <Link to="/superadmin/saloes">
              <Button>
                <Plus className="h-4 w-4" />
                Novo Salão
              </Button>
            </Link>
          }
        />
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
