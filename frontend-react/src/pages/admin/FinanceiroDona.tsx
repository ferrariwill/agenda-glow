import { useMemo, useState } from 'react'
import {
  ArrowDown,
  ArrowUp,
  ChevronLeft,
  ChevronRight,
  Download,
  Filter,
  PlusCircle,
  Printer,
  Search,
  Trash2,
  TrendingUp,
  Wallet,
  FileEdit,
  Banknote,
} from 'lucide-react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { ConfirmModal, Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import type { Lancamento, TransacaoCategoria, TransacaoStatus } from '../../types'
import {
  deleteLancamento,
  getDb,
  getFinanceiroStats,
  getTransacoesFiltradas,
  isReceita,
  lancamentoCategoria,
  lancamentoStatus,
} from '../../utils/mockDb'
import {
  currentYearMonth,
  formatBRL,
  formatMonthYearBR,
  formatTableDateBR,
  recentYearMonths,
} from '../../utils/format'
import { LiquidacaoComissoes } from './LiquidacaoComissoes'
import {
  CATEGORIA_BADGE,
  CATEGORIA_LABEL,
  GLASS,
} from './financeiroShared'

const PAGE_SIZE = 10

function statusBadge(status: TransacaoStatus) {
  return status === 'PAGO'
    ? 'bg-[#E1F5FE] text-[#01579B]'
    : 'bg-[#efdcd1]/50 text-[#50443c]'
}

export function FinanceiroDona() {
  const navigate = useNavigate()
  const location = useLocation()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [yearMonth, setYearMonth] = useState(currentYearMonth())
  const [categoria, setCategoria] = useState<TransacaoCategoria | 'all'>('all')
  const [tipo, setTipo] = useState<'all' | 'receita' | 'despesa'>('all')
  const [page, setPage] = useState(1)
  const [success, setSuccess] = useState(
    (location.state as { success?: string } | null)?.success ?? '',
  )
  const [error, setError] = useState('')

  const [deleteTarget, setDeleteTarget] = useState<Lancamento | null>(null)
  const [comissoesOpen, setComissoesOpen] = useState(false)

  const refresh = () => setDb(getDb())
  const tenant = db.tenants.find((t) => t.id === tenantId)
  const meses = useMemo(() => recentYearMonths(6), [])

  const stats = useMemo(
    () => getFinanceiroStats(tenantId, yearMonth),
    [db, tenantId, yearMonth],
  )

  const transacoes = useMemo(() => {
    return getTransacoesFiltradas(tenantId, {
      yearMonth,
      categoria,
      tipo,
      search,
    })
  }, [db, tenantId, yearMonth, categoria, tipo, search])

  const totalPages = Math.max(1, Math.ceil(transacoes.length / PAGE_SIZE))
  const pageSafe = Math.min(page, totalPages)
  const pageItems = transacoes.slice((pageSafe - 1) * PAGE_SIZE, pageSafe * PAGE_SIZE)
  const rangeStart = transacoes.length === 0 ? 0 : (pageSafe - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(pageSafe * PAGE_SIZE, transacoes.length)

  const novaBtn = (
    <Link
      to="/admin/financeiro/novo"
      className="flex items-center gap-2 rounded-full bg-[#7d5141] px-6 py-2.5 text-xs font-bold uppercase tracking-wider text-white transition-opacity hover:opacity-90 active:scale-95"
    >
      <PlusCircle className="h-4 w-4" />
      Nova Transação
    </Link>
  )

  return (
    <DonaLayout
      searchPlaceholder="Buscar transações..."
      searchValue={search}
      onSearchChange={(v) => {
        setSearch(v)
        setPage(1)
      }}
      headerAction={novaBtn}
    >
      {/* Header */}
      <header className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="font-display text-2xl font-semibold text-[#7d5141] sm:text-3xl">
            Painel Financeiro
          </h1>
          <p className="text-sm text-[#514440]">
            Visão geral e fluxo de caixa da unidade {tenant?.nome ?? 'do salão'}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={() => setComissoesOpen(true)}
            className="flex items-center gap-2 rounded-full border border-[#d6c2bd] px-4 py-2 text-sm font-semibold text-[#695c53] transition-colors hover:bg-[#f4f3f2]"
          >
            <Banknote className="h-4 w-4" />
            Comissões
          </button>
          <div className="lg:hidden">{novaBtn}</div>
        </div>
      </header>

      {(success || error) && (
        <Alert
          variant={error ? 'error' : 'info'}
          className="mb-6"
          onDismiss={() => {
            setSuccess('')
            setError('')
          }}
        >
          {error || success}
        </Alert>
      )}

      {/* KPIs */}
      <section className="mb-8 grid gap-6 md:grid-cols-3">
        <div className={`p-8 ${GLASS}`}>
          <div className="mb-4 flex items-start justify-between">
            <div className="rounded-lg bg-[#996958]/20 p-2 text-[#7d5141]">
              <Wallet className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              Saldo mensal
            </span>
          </div>
          <p className="font-display text-[32px] font-bold text-[#7d5141]">
            {formatBRL(stats.saldoMensal)}
          </p>
          {stats.variacaoSaldoPct !== null && (
            <p className="mt-1 flex items-center gap-1 text-sm text-[#695c53]">
              <TrendingUp className="h-4 w-4" />
              {stats.variacaoSaldoPct >= 0 ? '+' : ''}
              {stats.variacaoSaldoPct}% em relação a {stats.mesAnteriorLabel}
            </p>
          )}
        </div>

        <div className={`p-8 ${GLASS}`}>
          <div className="mb-4 flex items-start justify-between">
            <div className="rounded-lg bg-[#efdcd1]/40 p-2 text-[#695c53]">
              <ArrowUp className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              Total receitas
            </span>
          </div>
          <p className="font-display text-[32px] font-bold text-[#1a1c1c]">
            {formatBRL(stats.totalReceitas)}
          </p>
          <p className="mt-1 text-sm text-[#514440]">
            {stats.atendimentos} atendimentos realizados
          </p>
        </div>

        <div className={`p-8 ${GLASS}`}>
          <div className="mb-4 flex items-start justify-between">
            <div className="rounded-lg bg-[#ffdad6]/30 p-2 text-[#ba1a1a]">
              <ArrowDown className="h-5 w-5" />
            </div>
            <span className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
              Total despesas
            </span>
          </div>
          <p className="font-display text-[32px] font-bold text-[#1a1c1c]">
            {formatBRL(stats.totalDespesas)}
          </p>
          <p className="mt-1 text-sm text-[#ba1a1a]">
            {stats.faturasPendentes} faturas pendentes
          </p>
        </div>
      </section>

      {/* Tabela */}
      <section className={`overflow-hidden ${GLASS}`}>
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[#d6c2bd]/10 bg-white/50 p-6">
          <div className="flex flex-wrap items-center gap-4">
            <Filter className="h-5 w-5 text-[#514440]" />
            <select
              value={yearMonth}
              onChange={(e) => {
                setYearMonth(e.target.value)
                setPage(1)
              }}
              className="border-b border-[#d6c2bd] bg-transparent px-2 py-1 text-sm outline-none focus:border-[#7d5141]"
            >
              {meses.map((m) => (
                <option key={m} value={m}>
                  {formatMonthYearBR(m)}
                </option>
              ))}
            </select>
            <select
              value={categoria}
              onChange={(e) => {
                setCategoria(e.target.value as TransacaoCategoria | 'all')
                setPage(1)
              }}
              className="border-b border-[#d6c2bd] bg-transparent px-2 py-1 text-sm outline-none focus:border-[#7d5141]"
            >
              <option value="all">Todas Categorias</option>
              {(Object.keys(CATEGORIA_LABEL) as TransacaoCategoria[]).map((c) => (
                <option key={c} value={c}>
                  {CATEGORIA_LABEL[c]}
                </option>
              ))}
            </select>
            <select
              value={tipo}
              onChange={(e) => {
                setTipo(e.target.value as 'all' | 'receita' | 'despesa')
                setPage(1)
              }}
              className="border-b border-[#d6c2bd] bg-transparent px-2 py-1 text-sm outline-none focus:border-[#7d5141]"
            >
              <option value="all">Todos Tipos</option>
              <option value="receita">Receita</option>
              <option value="despesa">Despesa</option>
            </select>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setSuccess('Exportação simulada — CSV em breve.')}
              className="rounded-full p-2 text-[#514440] transition-colors hover:bg-[#efdcd1]/20"
            >
              <Download className="h-5 w-5" />
            </button>
            <button
              type="button"
              onClick={() => window.print()}
              className="rounded-full p-2 text-[#514440] transition-colors hover:bg-[#efdcd1]/20"
            >
              <Printer className="h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="bg-[#f4f3f2]/50">
                <th className="px-6 py-4 text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                  Data
                </th>
                <th className="px-6 py-4 text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                  Descrição
                </th>
                <th className="px-6 py-4 text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                  Categoria
                </th>
                <th className="px-6 py-4 text-right text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                  Valor
                </th>
                <th className="px-6 py-4 text-center text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                  Status
                </th>
                <th className="px-6 py-4 text-right text-[11px] font-bold uppercase tracking-widest text-[#514440]">
                  Ações
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#d6c2bd]/10">
              {pageItems.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-16 text-center text-sm text-[#514440]">
                    <Search className="mx-auto mb-3 h-8 w-8 opacity-30" />
                    Nenhuma transação encontrada para os filtros selecionados.
                  </td>
                </tr>
              ) : (
                pageItems.map((t) => {
                  const cat = lancamentoCategoria(t)
                  const st = lancamentoStatus(t)
                  const receita = isReceita(t)
                  return (
                    <tr
                      key={t.id}
                      className="transition-colors hover:bg-[#996958]/5"
                    >
                      <td className="px-6 py-4 text-sm text-[#514440]">
                        {formatTableDateBR(t.data)}
                      </td>
                      <td className="px-6 py-4">
                        <p className="font-medium text-[#1a1c1c]">{t.descricao}</p>
                        {t.subtitulo && (
                          <p className="text-xs text-[#514440]">{t.subtitulo}</p>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={[
                            'rounded-full px-3 py-1 text-xs font-medium',
                            CATEGORIA_BADGE[cat],
                          ].join(' ')}
                        >
                          {CATEGORIA_LABEL[cat]}
                        </span>
                      </td>
                      <td
                        className={[
                          'px-6 py-4 text-right font-medium',
                          receita ? 'text-[#7d5141]' : 'text-[#ba1a1a]',
                        ].join(' ')}
                      >
                        {receita ? '' : '- '}
                        {formatBRL(t.valor)}
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex justify-center">
                          <span
                            className={[
                              'rounded-full px-3 py-1 text-[11px] font-bold uppercase tracking-wider',
                              statusBadge(st),
                            ].join(' ')}
                          >
                            {st === 'PAGO' ? 'Pago' : 'Pendente'}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex justify-end gap-2">
                          <button
                            type="button"
                            title="Editar"
                            onClick={() => navigate(`/admin/financeiro/${t.id}/edit`)}
                            className="rounded p-1 text-[#514440] transition-colors hover:text-[#7d5141]"
                          >
                            <FileEdit className="h-5 w-5" />
                          </button>
                          <button
                            type="button"
                            title="Excluir"
                            onClick={() => setDeleteTarget(t)}
                            className="rounded p-1 text-[#514440] transition-colors hover:text-[#ba1a1a]"
                          >
                            <Trash2 className="h-5 w-5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        <div className="flex items-center justify-between border-t border-[#d6c2bd]/10 bg-[#f4f3f2]/30 px-6 py-4">
          <span className="text-sm text-[#514440]">
            Exibindo {rangeStart}-{rangeEnd} de {transacoes.length} transações
          </span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={pageSafe <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              className="rounded-lg p-1 text-[#514440] hover:bg-[#efdcd1]/20 disabled:opacity-30"
            >
              <ChevronLeft className="h-5 w-5" />
            </button>
            {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => setPage(n)}
                className={[
                  'flex h-8 w-8 items-center justify-center rounded-lg text-sm font-medium',
                  n === pageSafe
                    ? 'bg-[#7d5141] text-white'
                    : 'text-[#514440] hover:bg-[#efdcd1]/20',
                ].join(' ')}
              >
                {n}
              </button>
            ))}
            <button
              type="button"
              disabled={pageSafe >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              className="rounded-lg p-1 text-[#514440] hover:bg-[#efdcd1]/20 disabled:opacity-30"
            >
              <ChevronRight className="h-5 w-5" />
            </button>
          </div>
        </div>
      </section>

      <DonaFooter />

      <ConfirmModal
        open={Boolean(deleteTarget)}
        title="Excluir transação"
        message={`Remover "${deleteTarget?.descricao}"? Esta ação não pode ser desfeita.`}
        confirmLabel="Excluir"
        onConfirm={() => {
          if (!deleteTarget) return
          try {
            deleteLancamento(deleteTarget.id)
            refresh()
            setDeleteTarget(null)
            setSuccess('Transação removida.')
          } catch (err) {
            setError(err instanceof Error ? err.message : 'Erro ao excluir')
            setDeleteTarget(null)
          }
        }}
        onClose={() => setDeleteTarget(null)}
      />

      {comissoesOpen && (
        <Modal
          open
          title="Liquidação de Comissões"
          onClose={() => setComissoesOpen(false)}
          maxWidthClass="max-w-2xl"
        >
          <LiquidacaoComissoes embedded />
        </Modal>
      )}
    </DonaLayout>
  )
}
