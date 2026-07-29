interface PageHeaderProps {
  title: string
  subtitle: string
  action?: React.ReactNode
}

export function PageHeader({ title, subtitle, action }: PageHeaderProps) {
  return (
    <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <h1 className="font-display text-2xl font-semibold text-aura-anthracite sm:text-3xl">{title}</h1>
        <p className="mt-1 text-sm text-aura-muted">{subtitle}</p>
      </div>
      {action && <div className="w-full shrink-0 sm:w-auto [&>*]:w-full sm:[&>*]:w-auto">{action}</div>}
    </div>
  )
}

interface MetricCardProps {
  label: string
  value: string
  highlight?: 'success' | 'danger' | 'default'
}

export function MetricCard({ label, value, highlight = 'default' }: MetricCardProps) {
  const valueClass =
    highlight === 'success'
      ? 'text-emerald-600'
      : highlight === 'danger'
        ? 'text-red-600'
        : 'text-aura-anthracite'

  return (
    <div className="rounded-lg border border-aura-border bg-white p-4 shadow-sm sm:p-5">
      <p className="text-sm text-aura-muted">{label}</p>
      <p className={`mt-2 font-display text-xl font-semibold sm:text-2xl ${valueClass}`}>{value}</p>
    </div>
  )
}

export function SuperAdminFooter() {
  return (
    <footer className="mt-10 flex flex-col items-center justify-between gap-2 border-t border-aura-border pt-6 text-xs text-aura-muted sm:flex-row">
      <p>© {new Date().getFullYear()} Aura Beauty SaaS. Todos os direitos reservados.</p>
      <div className="flex gap-4">
        <button type="button" className="hover:text-aura-primary">
          Termos de Uso
        </button>
        <button type="button" className="hover:text-aura-primary">
          Privacidade
        </button>
        <button type="button" className="hover:text-aura-primary">
          Suporte
        </button>
      </div>
    </footer>
  )
}

interface PaginationProps {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPageChange: (p: number) => void
  label: string
}

export function TablePagination({
  page,
  totalPages,
  total,
  pageSize,
  onPageChange,
  label,
}: PaginationProps) {
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1
  const end = Math.min(page * pageSize, total)

  return (
    <div className="flex flex-col items-center justify-between gap-3 border-t border-aura-border px-4 py-4 sm:flex-row">
      <p className="text-sm text-aura-muted">
        Mostrando {start}-{end} de {total.toLocaleString('pt-BR')} {label}
      </p>
      <div className="flex items-center gap-1">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          className="rounded-lg px-3 py-1.5 text-sm text-aura-muted disabled:opacity-40"
        >
          ‹
        </button>
        {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
          <button
            key={p}
            type="button"
            onClick={() => onPageChange(p)}
            className={[
              'min-w-[36px] rounded-lg px-3 py-1.5 text-sm font-medium',
              p === page
                ? 'bg-aura-primary text-white'
                : 'text-aura-muted hover:bg-aura-surface',
            ].join(' ')}
          >
            {p}
          </button>
        ))}
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
          className="rounded-lg px-3 py-1.5 text-sm text-aura-muted disabled:opacity-40"
        >
          ›
        </button>
      </div>
    </div>
  )
}
