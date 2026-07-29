import { useMemo, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  AlertTriangle,
  Boxes,
  ChevronLeft,
  ChevronRight,
  Edit,
  Filter,
  Plus,
  ShoppingCart,
  Sparkles,
  TrendingUp,
  Wallet,
  Warehouse,
} from 'lucide-react'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import type { InsumoCategoria, InsumoEstoque } from '../../types'
import {
  ajustarEstoqueInsumo,
  getDb,
  getInsumoNivelEstoque,
  getInsumosStats,
} from '../../utils/mockDb'
import { formatBRL, formatCompactBRL } from '../../utils/format'

const PAGE_SIZE = 10
const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]'

const DEFAULT_IMG =
  'https://images.unsplash.com/photo-1560066984-138dadb4c035?w=400&q=80'

const CATEGORIA_LABEL: Record<InsumoCategoria, string> = {
  CUIDADOS_CAPILARES: 'Cuidados Capilares',
  TECNICA_UNHAS: 'Técnica em Unhas',
  ESTETICA: 'Estética',
  GERAL: 'Geral',
}

const CATEGORIA_TAB: { key: InsumoCategoria | 'all'; label: string }[] = [
  { key: 'all', label: 'Todos' },
  { key: 'CUIDADOS_CAPILARES', label: 'Cuidados Capilares' },
  { key: 'TECNICA_UNHAS', label: 'Técnica em Unhas' },
  { key: 'ESTETICA', label: 'Estética' },
]

function stockBar(nivel: ReturnType<typeof getInsumoNivelEstoque>) {
  switch (nivel) {
    case 'critico':
      return { bar: 'bg-red-500', track: 'bg-red-100', label: 'Estoque crítico', labelClass: 'text-red-600' }
    case 'atencao':
      return { bar: 'bg-amber-400', track: 'bg-amber-100', label: 'Atenção', labelClass: 'text-amber-600' }
    default:
      return { bar: 'bg-green-500', track: 'bg-green-100', label: '', labelClass: '' }
  }
}

export function InsumosDona() {
  const navigate = useNavigate()
  const location = useLocation()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [categoria, setCategoria] = useState<InsumoCategoria | 'all'>('all')
  const [page, setPage] = useState(1)
  const [success, setSuccess] = useState(
    (location.state as { success?: string } | null)?.success ?? '',
  )
  const [toast, setToast] = useState('')

  const [estoqueOpen, setEstoqueOpen] = useState<InsumoEstoque | null>(null)
  const [estoqueDelta, setEstoqueDelta] = useState('')

  const refresh = () => setDb(getDb())
  const stats = useMemo(() => getInsumosStats(tenantId), [db, tenantId])

  const insumos = useMemo(() => {
    const q = search.trim().toLowerCase()
    let list = db.insumos.filter((i) => i.tenant_id === tenantId && i.ativo)
    if (categoria !== 'all') list = list.filter((i) => i.categoria === categoria)
    if (q) {
      list = list.filter(
        (i) =>
          i.nome.toLowerCase().includes(q) ||
          i.marca?.toLowerCase().includes(q) ||
          CATEGORIA_LABEL[i.categoria].toLowerCase().includes(q),
      )
    }
    const order = { critico: 0, atencao: 1, ok: 2 }
    return list.sort((a, b) => {
      const na = getInsumoNivelEstoque(a)
      const nb = getInsumoNivelEstoque(b)
      if (order[na] !== order[nb]) return order[na] - order[nb]
      return a.nome.localeCompare(b.nome)
    })
  }, [db.insumos, tenantId, categoria, search])

  const totalPages = Math.max(1, Math.ceil(insumos.length / PAGE_SIZE))
  const pageSafe = Math.min(page, totalPages)
  const pageItems = insumos.slice((pageSafe - 1) * PAGE_SIZE, pageSafe * PAGE_SIZE)
  const rangeStart = insumos.length === 0 ? 0 : (pageSafe - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(pageSafe * PAGE_SIZE, insumos.length)

  const applyEstoque = () => {
    if (!estoqueOpen) return
    const delta = Number(estoqueDelta)
    if (Number.isNaN(delta) || delta === 0) return
    ajustarEstoqueInsumo(estoqueOpen.id, delta)
    refresh()
    setEstoqueOpen(null)
    setEstoqueDelta('')
    setSuccess('Estoque atualizado.')
  }

  const reposicaoRapida = (item: InsumoEstoque) => {
    const delta = Math.max(0, item.estoque_ideal - item.quantidade)
    if (delta <= 0) return
    ajustarEstoqueInsumo(item.id, delta)
    refresh()
    setSuccess(`Reposição de ${delta} ${item.unidade} registrada para ${item.nome}.`)
  }

  const novoInsumoBtn = (
    <Link
      to="/admin/insumos/novo"
      className="hidden items-center gap-2 rounded-lg bg-[#7d5141] px-5 py-2 text-sm font-semibold text-white shadow-sm transition-all hover:opacity-90 active:scale-95 sm:flex"
    >
      <Plus className="h-4 w-4" />
      Novo insumo
    </Link>
  )

  return (
    <DonaLayout
      searchPlaceholder="Pesquisar insumos ou marcas…"
      searchValue={search}
      onSearchChange={(v) => {
        setSearch(v)
        setPage(1)
      }}
      headerAction={novoInsumoBtn}
    >
      <div className="mb-8">
        <h1 className="font-display text-2xl font-semibold text-[#7d5141] sm:text-3xl">
          Gestão de insumos
        </h1>
        <p className="mt-1 text-[#514440]">
          Controle o estoque de produtos de luxo do seu salão em tempo real.
        </p>
      </div>

      {(success || toast) && (
        <Alert
          variant="info"
          className="mb-6"
          onDismiss={() => {
            setSuccess('')
            setToast('')
          }}
        >
          {success || toast}
        </Alert>
      )}

      {/* KPIs */}
      <div className="mb-8 grid gap-6 md:grid-cols-3">
        <div className={`flex items-start justify-between p-6 ${GLASS}`}>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]/60">
              Total de itens
            </p>
            <p className="font-display mt-2 text-3xl font-bold text-[#7d5141] sm:text-4xl">
              {stats.totalSkus}
            </p>
            <p className="mt-1 flex items-center gap-1 text-xs text-[#695c53]">
              <TrendingUp className="h-3.5 w-3.5 text-green-600" />
              <span className="font-bold text-green-600">+{stats.variacaoMes}%</span>
              em relação ao mês anterior
            </p>
          </div>
          <div className="rounded-lg bg-[#ffdbcf] p-2 text-[#653d2e]">
            <Boxes className="h-5 w-5" />
          </div>
        </div>

        <div className={`flex items-start justify-between border-l-4 border-l-[#ba1a1a] p-6 ${GLASS}`}>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]/60">
              Alerta de estoque baixo
            </p>
            <p className="font-display mt-2 text-3xl font-bold text-[#ba1a1a] sm:text-4xl">
              {String(stats.estoqueBaixo).padStart(2, '0')}
            </p>
            <p className="mt-1 text-xs text-[#695c53]">Reposição imediata recomendada</p>
          </div>
          <div className="rounded-lg bg-[#ffdad6] p-2 text-[#93000a]">
            <AlertTriangle className="h-5 w-5 fill-current" />
          </div>
        </div>

        <div className={`flex items-start justify-between p-6 ${GLASS}`}>
          <div className="min-w-0 flex-1">
            <p className="text-xs font-bold uppercase tracking-widest text-[#514440]/60">
              Investimento mensal
            </p>
            <p className="font-display mt-2 text-3xl font-bold text-[#7d5141] sm:text-4xl">
              {formatCompactBRL(stats.investimentoMensal)}
            </p>
            <div className="mt-4 h-1 overflow-hidden rounded-full bg-[#f4f3f2]">
              <div
                className="h-full bg-[#7d5141]"
                style={{ width: `${stats.investimentoPct}%` }}
              />
            </div>
          </div>
          <div className="ml-3 rounded-lg bg-[#ebe0dd] p-2 text-[#1f1b19]">
            <Wallet className="h-5 w-5" />
          </div>
        </div>
      </div>

      {/* Table */}
      <div className={`overflow-hidden ${GLASS}`}>
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[#efdcd1]/20 px-6 py-4">
          <div className="flex flex-wrap gap-6">
            {CATEGORIA_TAB.map(({ key, label }) => (
              <button
                key={key}
                type="button"
                onClick={() => {
                  setCategoria(key)
                  setPage(1)
                }}
                className={[
                  'pb-1 text-sm font-semibold transition-colors',
                  categoria === key
                    ? 'border-b-2 border-[#7d5141] text-[#7d5141]'
                    : 'text-[#514440] hover:text-[#7d5141]',
                ].join(' ')}
              >
                {label}
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={() => setToast('Filtros avançados em breve.')}
            className="flex items-center gap-1 text-sm text-[#514440] hover:text-[#7d5141]"
          >
            <Filter className="h-4 w-4" />
            Filtros avançados
          </button>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-[#efdcd1]/20 bg-[#f4f3f2]">
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-[#514440]/60">
                  Produto
                </th>
                <th className="px-6 py-4 text-xs font-bold uppercase tracking-wider text-[#514440]/60">
                  Categoria
                </th>
                <th className="px-6 py-4 text-center text-xs font-bold uppercase tracking-wider text-[#514440]/60">
                  Estoque
                </th>
                <th className="px-6 py-4 text-right text-xs font-bold uppercase tracking-wider text-[#514440]/60">
                  Valor unit.
                </th>
                <th className="px-6 py-4 text-center text-xs font-bold uppercase tracking-wider text-[#514440]/60">
                  Ações
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#efdcd1]/10">
              {pageItems.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center text-sm text-[#83746f]">
                    Nenhum insumo encontrado.
                  </td>
                </tr>
              ) : (
                pageItems.map((item) => {
                  const nivel = getInsumoNivelEstoque(item)
                  const pct = Math.min(100, (item.quantidade / item.estoque_ideal) * 100)
                  const bar = stockBar(nivel)
                  return (
                    <tr
                      key={item.id}
                      className={[
                        'transition-colors hover:bg-[#f4f3f2]/50',
                        nivel === 'critico' ? 'bg-red-50/40' : '',
                      ].join(' ')}
                    >
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-4">
                          <div className="h-12 w-12 shrink-0 overflow-hidden rounded-lg border border-[#efdcd1]/20 bg-[#eeeeed]">
                            <img
                              src={item.imagem_url ?? DEFAULT_IMG}
                              alt=""
                              className="h-full w-full object-cover"
                            />
                          </div>
                          <div>
                            <p className="text-sm font-semibold">{item.nome}</p>
                            {item.marca && (
                              <p className="text-xs text-[#514440]">{item.marca}</p>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span className="inline-flex rounded-full bg-[#efdcd1] px-2 py-1 text-xs font-semibold text-[#50443c]">
                          {CATEGORIA_LABEL[item.categoria]}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-center">
                        <p
                          className={[
                            'text-sm font-semibold',
                            nivel === 'critico' ? 'font-bold text-[#ba1a1a]' : '',
                          ].join(' ')}
                        >
                          {String(item.quantidade).padStart(2, '0')}{' '}
                          <span className="text-xs font-normal text-[#514440]">{item.unidade}</span>
                        </p>
                        <div
                          className={`mx-auto mt-1 h-1 w-16 overflow-hidden rounded-full ${bar.track}`}
                        >
                          <div className={`h-full ${bar.bar}`} style={{ width: `${pct}%` }} />
                        </div>
                        {bar.label && (
                          <p
                            className={`mt-1 text-[10px] font-bold uppercase tracking-wider ${bar.labelClass}`}
                          >
                            {bar.label}
                          </p>
                        )}
                      </td>
                      <td className="px-6 py-4 text-right text-sm font-semibold">
                        {formatBRL(item.valor_unitario)}
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex justify-center gap-3">
                          {nivel !== 'ok' ? (
                            <button
                              type="button"
                              onClick={() => reposicaoRapida(item)}
                              className="rounded-md bg-[#ffdbcf]/50 p-1.5 text-[#7d5141] transition-transform hover:scale-110"
                              title="Reposição rápida"
                            >
                              <ShoppingCart className="h-4 w-4" />
                            </button>
                          ) : (
                            <button
                              type="button"
                              onClick={() => {
                                setEstoqueOpen(item)
                                setEstoqueDelta('')
                              }}
                              className="rounded-md p-1.5 text-[#514440] transition-colors hover:bg-[#7d5141]/10 hover:text-[#7d5141]"
                              title="Gerenciar estoque"
                            >
                              <Warehouse className="h-4 w-4" />
                            </button>
                          )}
                          <button
                            type="button"
                            onClick={() => navigate(`/admin/insumos/${item.id}/edit`)}
                            className="rounded-md p-1.5 text-[#514440] transition-colors hover:bg-[#7d5141]/10 hover:text-[#7d5141]"
                            title="Editar"
                          >
                            <Edit className="h-4 w-4" />
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

        <div className="flex flex-col items-center justify-between gap-4 border-t border-[#efdcd1]/10 px-6 py-4 sm:flex-row">
          <p className="text-xs text-[#514440]">
            Exibindo {rangeStart}-{rangeEnd} de {insumos.length} produtos
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={pageSafe <= 1}
              onClick={() => setPage((p) => p - 1)}
              className="rounded border border-[#efdcd1]/30 p-2 text-[#514440] transition-colors hover:bg-[#f4f3f2] disabled:opacity-40"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setPage(p)}
                className={[
                  'flex h-8 w-8 items-center justify-center rounded text-xs font-bold',
                  p === pageSafe
                    ? 'bg-[#7d5141] text-white'
                    : 'text-[#514440] hover:bg-[#f4f3f2]',
                ].join(' ')}
              >
                {p}
              </button>
            ))}
            <button
              type="button"
              disabled={pageSafe >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="rounded border border-[#efdcd1]/30 p-2 text-[#514440] transition-colors hover:bg-[#f4f3f2] disabled:opacity-40"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {/* AI banner */}
      <div className="mt-8 flex flex-col items-start justify-between gap-4 rounded-xl bg-gradient-to-r from-[#7d5141] to-[#996958] p-6 text-white shadow-lg sm:flex-row sm:items-center">
        <div className="flex items-start gap-4">
          <div className="rounded-full bg-white/20 p-4">
            <Sparkles className="h-6 w-6" />
          </div>
          <div>
            <h3 className="font-semibold">Previsão inteligente de estoque</h3>
            <p className="mt-1 text-sm opacity-90">
              Ative nossa IA para prever quando seus insumos acabarão com base no fluxo de
              agendamentos.
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setToast('Previsão inteligente será ativada em breve.')}
          className="shrink-0 rounded-lg bg-white px-6 py-2 text-sm font-bold text-[#7d5141] shadow-sm transition-colors hover:bg-[#faf9f8]"
        >
          Ativar agora
        </button>
      </div>

      <Link
        to="/admin/insumos/novo"
        className="fixed bottom-24 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-[#7d5141] text-white shadow-2xl transition-transform hover:scale-110 active:scale-95 lg:hidden"
        aria-label="Novo insumo"
      >
        <Plus className="h-6 w-6" />
      </Link>

      <DonaFooter />

      <Modal
        open={!!estoqueOpen}
        onClose={() => setEstoqueOpen(null)}
        title="Gerenciar estoque"
        footer={
          <>
            <Button variant="secondary" onClick={() => setEstoqueOpen(null)}>
              Cancelar
            </Button>
            <Button onClick={applyEstoque}>Aplicar</Button>
          </>
        }
      >
        {estoqueOpen && (
          <div className="space-y-4">
            <p className="text-sm text-[#514440]">
              <strong>{estoqueOpen.nome}</strong> — estoque atual:{' '}
              <strong>
                {estoqueOpen.quantidade} {estoqueOpen.unidade}
              </strong>
            </p>
            <Input
              label="Ajuste (+ entrar / − sair)"
              type="number"
              value={estoqueDelta}
              onChange={(e) => setEstoqueDelta(e.target.value)}
              placeholder="Ex.: 10 ou -2"
            />
          </div>
        )}
      </Modal>
    </DonaLayout>
  )
}
