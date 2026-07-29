import { useMemo, useState } from 'react'
import { Download, Filter, MoreVertical, TrendingDown, TrendingUp } from 'lucide-react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
} from '../../components/superadmin/SuperAdminLayout'
import { Button } from '../../components/ui/Button'
import {
  getChurnStats,
  getDistribuicaoPlanos,
  getFinanceiroInsights,
  getFaturasSaasRecentes,
  getMRR,
  getMRRHistorico6Meses,
  getMRRVariacao,
} from '../../utils/mockDb'
import { formatBRL, formatDateBR, initials } from '../../utils/format'
import type { FaturaSaasStatus } from '../../types'

const AVATAR_BG = [
  'bg-[#ffdbcf] text-[#7d5141]',
  'bg-[#f1dfd4] text-[#695c53]',
  'bg-[#ebe0dd] text-[#615b58]',
  'bg-[#efdcd1] text-[#6d6057]',
]

function statusFaturaLabel(status: FaturaSaasStatus) {
  switch (status) {
    case 'SUCESSO':
      return 'Sucesso'
    case 'PENDENTE':
      return 'Pendente'
    case 'FALHA':
      return 'Falha'
  }
}

function statusFaturaClass(status: FaturaSaasStatus) {
  switch (status) {
    case 'SUCESSO':
      return 'bg-[#e8f5e9] text-[#2e7d32]'
    case 'PENDENTE':
      return 'bg-[#fff8e1] text-[#f57f17]'
    case 'FALHA':
      return 'bg-[#ffebee] text-[#c62828]'
  }
}

function shortFaturaId(id: string) {
  const n = id.replace(/\D/g, '').slice(-5) || id.slice(-5)
  return `#SA-${n.padStart(5, '0').slice(-5)}`
}

function exportFaturasCSV(rows: ReturnType<typeof getFaturasSaasRecentes>) {
  const header = 'Salão;ID;Data;Valor;Status;Referência\n'
  const body = rows
    .map(
      (r) =>
        `${r.tenant_nome};${shortFaturaId(r.id)};${formatDateBR(r.data)};${r.valor.toFixed(2)};${statusFaturaLabel(r.status)};${r.referencia_mes}`,
    )
    .join('\n')
  const blob = new Blob([header + body], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `agendaglow-financeiro-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

const BAR_COLORS = ['bg-[#d5c3b8]', 'bg-[#996958]', 'bg-[#7d5141]']

export function FinanceiroSuperAdmin() {
  const [search, setSearch] = useState('')
  const [showAll, setShowAll] = useState(false)

  const mrr = getMRR()
  const mrrVar = getMRRVariacao()
  const churn = getChurnStats()
  const historico = getMRRHistorico6Meses()
  const maxMrr = Math.max(...historico.map((h) => h.mrr), 1)
  const distribuicao = getDistribuicaoPlanos()
  const insights = getFinanceiroInsights()

  const faturas = useMemo(
    () => getFaturasSaasRecentes(search, showAll ? 50 : 8),
    [search, showAll],
  )

  const exportRows = useMemo(() => getFaturasSaasRecentes(search, 200), [search])

  return (
    <SuperAdminLayout
      searchPlaceholder="Buscar por faturas, IDs ou salões…"
      searchValue={search}
      onSearchChange={setSearch}
    >
      <PageHeader
        title="Financeiro Global"
        subtitle="Visão geral do desempenho econômico da rede AgendaGlow."
        action={
          <Button
            variant="secondary"
            fullWidth
            className="sm:w-auto border-[#695c53] text-[#7d5141] hover:bg-[#f4f3f2]"
            onClick={() => exportFaturasCSV(exportRows)}
          >
            <Download className="h-4 w-4" />
            Exportar relatório
          </Button>
        }
      />

      {/* Métricas + gráfico */}
      <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-4 md:gap-6">
        <div className="group relative overflow-hidden rounded-xl border border-[#695c53]/5 bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] md:col-span-1">
          <div className="absolute right-0 top-0 h-24 w-24 rounded-bl-full bg-[#7d5141]/5 transition-transform duration-500 group-hover:scale-125" />
          <div className="relative flex flex-col gap-2">
            <span className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
              Total MRR
            </span>
            <div className="flex items-baseline gap-1">
              <span className="font-display text-2xl font-semibold text-[#7d5141]">
                {formatBRL(mrr)}
              </span>
              <span className="text-xs text-aura-muted">/mês</span>
            </div>
            <div
              className={[
                'flex items-center gap-1 text-xs font-bold',
                mrrVar.positive ? 'text-[#2e7d32]' : 'text-red-600',
              ].join(' ')}
            >
              {mrrVar.positive ? (
                <TrendingUp className="h-3.5 w-3.5" />
              ) : (
                <TrendingDown className="h-3.5 w-3.5" />
              )}
              {mrrVar.positive ? '+' : '-'}
              {mrrVar.pct}% em relação ao mês anterior
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-[#695c53]/5 bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] md:col-span-1">
          <div className="flex flex-col gap-2">
            <span className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
              Churn rate
            </span>
            <span className="font-display text-2xl font-semibold text-aura-anthracite">
              {churn.rate}%
            </span>
            <div
              className={[
                'flex items-center gap-1 text-xs font-bold',
                churn.delta <= 0 ? 'text-[#2e7d32]' : 'text-red-600',
              ].join(' ')}
            >
              <TrendingDown className="h-3.5 w-3.5" />
              {churn.delta > 0 ? '+' : ''}
              {churn.delta}% vs. último trimestre
            </div>
          </div>
        </div>

        <div className="flex min-h-[160px] flex-col justify-between rounded-xl border border-[#695c53]/5 bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] md:col-span-2">
          <div className="flex items-start justify-between">
            <span className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
              Crescimento de receita (6 meses)
            </span>
            <div className="flex gap-1">
              <div className="h-2 w-2 rounded-full bg-[#7d5141]" />
              <div className="h-2 w-2 rounded-full bg-[#f1dfd4]" />
            </div>
          </div>
          <div className="mt-4 flex h-20 items-end justify-between gap-1 px-1 sm:gap-2 sm:px-2">
            {historico.map((h, i) => {
              const pct = Math.max(12, Math.round((h.mrr / maxMrr) * 100))
              const isLast = i === historico.length - 1
              return (
                <div key={h.label} className="flex flex-1 flex-col items-center gap-1">
                  <div
                    title={formatBRL(h.mrr)}
                    className={[
                      'w-full max-w-[2rem] rounded-t-sm transition-colors',
                      isLast
                        ? 'bg-[#7d5141] hover:bg-[#996958]'
                        : 'bg-[#f1dfd4]/70 hover:bg-[#7d5141]/40',
                    ].join(' ')}
                    style={{ height: `${pct}%` }}
                  />
                </div>
              )
            })}
          </div>
          <div className="mt-2 flex justify-between px-1 text-[10px] font-bold uppercase tracking-widest text-aura-muted sm:px-2">
            {historico.map((h) => (
              <span key={h.label}>{h.label}</span>
            ))}
          </div>
        </div>
      </div>

      {/* Tabela + sidebar */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <div className="flex items-center justify-between">
            <h3 className="font-display text-lg font-semibold text-aura-anthracite">
              Transações recentes
            </h3>
            <button
              type="button"
              className="rounded-lg p-2 text-aura-muted transition-colors hover:bg-[#e9e8e7]"
              aria-label="Filtrar"
            >
              <Filter className="h-4 w-4" />
            </button>
          </div>

          <div className="overflow-hidden rounded-xl border border-[#695c53]/5 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
            <div className="overflow-x-auto">
              <table className="w-full min-w-[640px] border-collapse text-left">
                <thead>
                  <tr className="bg-[#f4f3f2]/50">
                    <th className="border-b border-[#695c53]/10 p-4 text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      Tenant / Salão
                    </th>
                    <th className="border-b border-[#695c53]/10 p-4 text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      Data
                    </th>
                    <th className="border-b border-[#695c53]/10 p-4 text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      Valor
                    </th>
                    <th className="border-b border-[#695c53]/10 p-4 text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                      Status
                    </th>
                    <th className="border-b border-[#695c53]/10 p-4" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#695c53]/5">
                  {faturas.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="p-8 text-center text-sm text-aura-muted">
                        Nenhuma transação encontrada.
                      </td>
                    </tr>
                  ) : (
                    faturas.map((f, i) => (
                      <tr
                        key={f.id}
                        className="transition-colors hover:bg-[#f4f3f2]/30"
                      >
                        <td className="p-4">
                          <div className="flex items-center gap-3">
                            <div
                              className={[
                                'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-xs font-bold',
                                AVATAR_BG[i % AVATAR_BG.length],
                              ].join(' ')}
                            >
                              {initials(f.tenant_nome)}
                            </div>
                            <div className="min-w-0">
                              <p className="truncate text-sm font-bold text-aura-anthracite">
                                {f.tenant_nome}
                              </p>
                              <p className="text-xs text-aura-muted">
                                ID: {shortFaturaId(f.id)}
                              </p>
                            </div>
                          </div>
                        </td>
                        <td className="p-4 text-sm text-aura-muted">
                          {formatDateBR(f.data)}
                        </td>
                        <td className="p-4 text-sm font-bold text-aura-anthracite">
                          {formatBRL(f.valor)}
                        </td>
                        <td className="p-4">
                          <span
                            className={[
                              'inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider',
                              statusFaturaClass(f.status),
                            ].join(' ')}
                          >
                            {statusFaturaLabel(f.status)}
                          </span>
                        </td>
                        <td className="p-4 text-right">
                          <button
                            type="button"
                            className="text-aura-muted transition-colors hover:text-[#7d5141]"
                            aria-label="Mais opções"
                          >
                            <MoreVertical className="h-4 w-4" />
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            <div className="flex justify-center border-t border-[#695c53]/5 bg-white p-4">
              <button
                type="button"
                onClick={() => setShowAll((v) => !v)}
                className="text-xs font-bold uppercase tracking-widest text-[#7d5141] hover:underline"
              >
                {showAll ? 'Ver menos' : 'Ver histórico completo'}
              </button>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-6">
          <div className="relative overflow-hidden rounded-xl border border-white/50 bg-white/70 p-6 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-sm">
            <h3 className="mb-4 font-display text-lg font-semibold text-aura-anthracite">
              Distribuição de planos
            </h3>
            <div className="relative z-10 space-y-4">
              {distribuicao.length === 0 ? (
                <p className="text-sm text-aura-muted">Nenhum salão cadastrado.</p>
              ) : (
                distribuicao.map((d, i) => (
                  <div key={d.id} className="flex flex-col gap-1">
                    <div className="flex justify-between text-sm">
                      <span className="text-aura-muted">{d.nome}</span>
                      <span className="font-bold text-[#7d5141]">{d.pct}%</span>
                    </div>
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-[#efdcd1]/30">
                      <div
                        className={[
                          'h-full rounded-full transition-all',
                          BAR_COLORS[i % BAR_COLORS.length],
                        ].join(' ')}
                        style={{ width: `${d.pct}%` }}
                      />
                    </div>
                  </div>
                ))
              )}
            </div>
            <div className="pointer-events-none absolute -bottom-10 -right-10 h-40 w-40 rounded-full bg-[#7d5141]/5 blur-3xl" />
          </div>

          <div className="space-y-3">
            <h3 className="font-display text-lg font-semibold text-aura-anthracite">
              Insights do mês
            </h3>
            {insights.length === 0 ? (
              <p className="text-sm text-aura-muted">Nenhum insight no momento.</p>
            ) : (
              insights.map((ins) => (
                <div
                  key={ins.title}
                  className={[
                    'space-y-1 rounded-r-lg border-l-4 p-4',
                    ins.type === 'success' && 'border-[#7d5141] bg-[#ffdbcf]/20',
                    ins.type === 'warning' && 'border-amber-500 bg-amber-50/80',
                    ins.type === 'danger' && 'border-red-600 bg-red-50/80',
                  ].join(' ')}
                >
                  <p
                    className={[
                      'text-sm font-bold',
                      ins.type === 'success' && 'text-[#7d5141]',
                      ins.type === 'warning' && 'text-amber-800',
                      ins.type === 'danger' && 'text-red-700',
                    ].join(' ')}
                  >
                    {ins.title}
                  </p>
                  <p className="text-xs leading-relaxed text-aura-muted">{ins.body}</p>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      <SuperAdminFooter />
    </SuperAdminLayout>
  )
}
