import { useEffect, useState, type ReactNode } from 'react'
import {
  ArrowDown,
  ArrowUp,
  Save,
  Sparkles,
} from 'lucide-react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { useAuth } from '../../contexts/AuthContext'
import type {
  FrequenciaRecorrencia,
  MetodoPagamento,
  NaturezaValor,
  TransacaoCategoria,
  TransacaoStatus,
} from '../../types'
import { PlanLimitExceededError } from '../../types'
import {
  createLancamento,
  getDb,
  getLancamentoById,
  isReceita,
  lancamentoCategoria,
  LancamentoValidationError,
  updateLancamento,
} from '../../utils/mockDb'
import { parseBRLInput, todayISO } from '../../utils/format'
import {
  CATEGORIAS_DESPESA,
  CATEGORIAS_RECEITA,
  categoriaToTipo,
  FIELD_INPUT,
  FORNECEDORES_MOCK,
  GLASS,
  naturezaFromLanc,
  CATEGORIA_LABEL,
} from './financeiroShared'

const METODOS: { value: MetodoPagamento; label: string }[] = [
  { value: 'PIX', label: 'PIX' },
  { value: 'CARTAO_CREDITO', label: 'Cartão crédito' },
  { value: 'CARTAO_DEBITO', label: 'Cartão débito' },
  { value: 'CARTAO', label: 'Cartão (legado)' },
  { value: 'DINHEIRO', label: 'Dinheiro' },
  { value: 'TRANSFERENCIA', label: 'Transferência' },
]

function FieldLabel({ children }: { children: ReactNode }) {
  return (
    <label className="mb-1 block text-xs font-bold uppercase tracking-widest text-[#514440]">
      {children}
    </label>
  )
}

export function TransacaoFormDona() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const isEdit = Boolean(id)

  const existing = isEdit ? getLancamentoById(tenantId, id!) : undefined
  const profissionais = getDb().profissionais.filter(
    (p) => p.tenant_id === tenantId && p.ativo,
  )

  const [flow, setFlow] = useState<'receita' | 'despesa'>('receita')
  const [descricao, setDescricao] = useState('')
  const [categoria, setCategoria] = useState<TransacaoCategoria>('SERVICO')
  const [metodo, setMetodo] = useState<MetodoPagamento>('PIX')
  const [data, setData] = useState(todayISO())
  const [valor, setValor] = useState('')
  const [fornecedor, setFornecedor] = useState('')
  const [profissionalId, setProfissionalId] = useState('')
  const [natureza, setNatureza] = useState<NaturezaValor>('FIXO')
  const [recorrente, setRecorrente] = useState(false)
  const [frequencia, setFrequencia] = useState<FrequenciaRecorrencia>('MENSAL')
  const [status, setStatus] = useState<TransacaoStatus>('PAGO')
  const [error, setError] = useState('')
  const [loaded, setLoaded] = useState(!isEdit)

  useEffect(() => {
    if (!isEdit) return
    if (!existing) {
      setLoaded(true)
      return
    }
    setFlow(isReceita(existing) ? 'receita' : 'despesa')
    setDescricao(existing.descricao)
    setCategoria(lancamentoCategoria(existing))
    setMetodo(existing.metodo_pagamento ?? 'PIX')
    setData(existing.data)
    setValor(existing.valor.toFixed(2).replace('.', ','))
    setFornecedor(existing.fornecedor ?? '')
    setProfissionalId(existing.profissional_id ?? '')
    setNatureza(naturezaFromLanc(existing))
    setRecorrente(existing.recorrente ?? false)
    setFrequencia(existing.frequencia_recorrencia ?? 'MENSAL')
    setStatus(existing.status_transacao ?? 'PAGO')
    setLoaded(true)
  }, [isEdit, existing])

  useEffect(() => {
    if (isEdit) return
    const cats = flow === 'receita' ? CATEGORIAS_RECEITA : CATEGORIAS_DESPESA
    if (!cats.includes(categoria)) setCategoria(cats[0])
  }, [flow, isEdit, categoria])

  if (isEdit && !existing && loaded) {
    return <Navigate to="/admin/financeiro" replace />
  }

  const categorias = flow === 'receita' ? CATEGORIAS_RECEITA : CATEGORIAS_DESPESA

  const save = async () => {
    setError('')
    const parsed = parseBRLInput(valor)
    if (!descricao.trim()) {
      setError('Informe a descrição da transação.')
      return
    }
    if (Number.isNaN(parsed) || parsed <= 0) {
      setError('Informe um valor válido.')
      return
    }

    const payload = {
      tipo: categoriaToTipo(categoria, flow, natureza),
      valor: parsed,
      descricao: descricao.trim(),
      data,
      categoria,
      status_transacao: status,
      metodo_pagamento: metodo,
      fornecedor: flow === 'despesa' ? fornecedor || undefined : undefined,
      profissional_id: flow === 'despesa' && profissionalId ? profissionalId : undefined,
      natureza: flow === 'despesa' ? natureza : undefined,
      recorrente,
      frequencia_recorrencia: recorrente ? frequencia : undefined,
    }

    try {
      if (isEdit && existing) {
        updateLancamento(existing.id, payload)
        navigate('/admin/financeiro', { state: { success: 'Transação atualizada.' } })
      } else {
        await createLancamento({ tenant_id: tenantId, ...payload })
        navigate('/admin/financeiro', { state: { success: 'Transação registrada.' } })
      }
    } catch (err) {
      if (err instanceof LancamentoValidationError) {
        setError(err.message)
      } else if (err instanceof PlanLimitExceededError) {
        setError('Limite do plano atingido.')
      } else {
        setError(err instanceof Error ? err.message : 'Erro ao salvar')
      }
    }
  }

  const pageTitle = isEdit ? 'Editar Transação' : 'Registrar Fluxo de Caixa'
  const breadcrumb = isEdit ? descricao || 'Editar' : 'Nova Transação'

  return (
    <DonaLayout searchPlaceholder="Buscar transações...">
      <header className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <nav className="mb-2 flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-[#514440]">
            <Link to="/admin/financeiro" className="hover:text-[#7d5141]">
              Financeiro
            </Link>
            <span>/</span>
            <span className="text-[#7d5141]">{breadcrumb}</span>
          </nav>
          <h1 className="font-display text-2xl font-semibold text-[#7d5141] sm:text-3xl">
            {pageTitle}
          </h1>
        </div>
        <div className="flex flex-wrap gap-3">
          <Link
            to="/admin/financeiro"
            className="rounded-lg border border-[#7d5141] px-6 py-3 text-sm font-semibold text-[#7d5141] transition-colors hover:bg-[#7d5141]/5"
          >
            Cancelar
          </Link>
          <button
            type="button"
            onClick={save}
            className="flex items-center gap-2 rounded-lg bg-[#7d5141] px-6 py-3 text-sm font-semibold text-white transition-all hover:opacity-90 active:scale-95"
          >
            <Save className="h-4 w-4" />
            {isEdit ? 'Salvar alterações' : 'Salvar Transação'}
          </button>
        </div>
      </header>

      {error && (
        <Alert variant="error" className="mb-6" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <div className="relative mx-auto max-w-[1000px]">
        <form
          className="space-y-6"
          onSubmit={(e) => {
            e.preventDefault()
            save()
          }}
        >
          {/* Entrada / Saída */}
          <div className="grid grid-cols-2 gap-4">
            <label className="group cursor-pointer">
              <input
                type="radio"
                name="flow"
                className="peer sr-only"
                checked={flow === 'receita'}
                onChange={() => setFlow('receita')}
              />
              <div className="flex items-center justify-center gap-4 rounded-xl border border-[#d6c2bd]/30 bg-[#faf9f8] p-6 transition-all peer-checked:border-[#7d5141] peer-checked:bg-[#996958]/10 group-hover:border-[#7d5141]/40">
                <ArrowUp className="h-6 w-6 text-[#7d5141]" strokeWidth={2.5} />
                <div className="text-left">
                  <p className="text-lg font-semibold text-[#1a1c1c]">Entrada</p>
                  <p className="text-sm text-[#514440]">Receitas e Vendas</p>
                </div>
              </div>
            </label>
            <label className="group cursor-pointer">
              <input
                type="radio"
                name="flow"
                className="peer sr-only"
                checked={flow === 'despesa'}
                onChange={() => setFlow('despesa')}
              />
              <div className="flex items-center justify-center gap-4 rounded-xl border border-[#d6c2bd]/30 bg-[#faf9f8] p-6 transition-all peer-checked:border-[#ba1a1a] peer-checked:bg-[#ffdad6]/20 group-hover:border-[#ba1a1a]/40">
                <ArrowDown className="h-6 w-6 text-[#ba1a1a]" strokeWidth={2.5} />
                <div className="text-left">
                  <p className="text-lg font-semibold text-[#1a1c1c]">Saída</p>
                  <p className="text-sm text-[#514440]">Custos e Despesas</p>
                </div>
              </div>
            </label>
          </div>

          <div className="grid grid-cols-12 gap-6">
            {/* Coluna principal */}
            <div className="col-span-12 space-y-6 lg:col-span-8">
              <div className={`p-6 sm:p-8 ${GLASS}`}>
                <h2 className="mb-6 border-b border-[#d6c2bd]/20 pb-2 text-lg font-semibold text-[#7d5141]">
                  Detalhes da Transação
                </h2>
                <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                  <div className="md:col-span-2">
                    <FieldLabel>Descrição</FieldLabel>
                    <input
                      type="text"
                      value={descricao}
                      onChange={(e) => setDescricao(e.target.value)}
                      placeholder="Ex: Venda de Pacote Estética Facial"
                      className={FIELD_INPUT}
                      required
                    />
                  </div>
                  <div>
                    <FieldLabel>Categoria</FieldLabel>
                    <select
                      value={categoria}
                      onChange={(e) => setCategoria(e.target.value as TransacaoCategoria)}
                      className={`${FIELD_INPUT} appearance-none`}
                    >
                      {categorias.map((c) => (
                        <option key={c} value={c}>
                          {CATEGORIA_LABEL[c]}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <FieldLabel>Método de Pagamento</FieldLabel>
                    <select
                      value={metodo}
                      onChange={(e) => setMetodo(e.target.value as MetodoPagamento)}
                      className={`${FIELD_INPUT} appearance-none`}
                    >
                      {METODOS.map((m) => (
                        <option key={m.value} value={m.value}>
                          {m.label}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <FieldLabel>Data da Operação</FieldLabel>
                    <input
                      type="date"
                      value={data}
                      onChange={(e) => setData(e.target.value)}
                      className={FIELD_INPUT}
                      required
                    />
                  </div>
                  <div>
                    <FieldLabel>Valor (R$)</FieldLabel>
                    <div className="relative">
                      <span className="absolute left-0 top-1/2 -translate-y-1/2 text-[#514440]">
                        R$
                      </span>
                      <input
                        type="text"
                        inputMode="decimal"
                        value={valor}
                        onChange={(e) => setValor(e.target.value)}
                        placeholder="0,00"
                        className={`${FIELD_INPUT} pl-8`}
                        required
                      />
                    </div>
                  </div>
                  <div>
                    <FieldLabel>Status</FieldLabel>
                    <select
                      value={status}
                      onChange={(e) => setStatus(e.target.value as TransacaoStatus)}
                      className={`${FIELD_INPUT} appearance-none`}
                    >
                      <option value="PAGO">Pago</option>
                      <option value="PENDENTE">Pendente</option>
                    </select>
                  </div>
                </div>
              </div>

              {flow === 'despesa' && (
                <div className={`p-6 sm:p-8 ${GLASS}`}>
                  <h2 className="mb-6 border-b border-[#d6c2bd]/20 pb-2 text-lg font-semibold text-[#7d5141]">
                    Vínculos Adicionais
                  </h2>
                  <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                    <div>
                      <FieldLabel>Fornecedor</FieldLabel>
                      <select
                        value={fornecedor}
                        onChange={(e) => setFornecedor(e.target.value)}
                        className={`${FIELD_INPUT} appearance-none`}
                      >
                        <option value="">Opcional...</option>
                        {FORNECEDORES_MOCK.map((f) => (
                          <option key={f} value={f}>
                            {f}
                          </option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <FieldLabel>Profissional Relacionado</FieldLabel>
                      <select
                        value={profissionalId}
                        onChange={(e) => setProfissionalId(e.target.value)}
                        className={`${FIELD_INPUT} appearance-none`}
                      >
                        <option value="">Opcional...</option>
                        {profissionais.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.nome}
                          </option>
                        ))}
                      </select>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Sidebar */}
            <div className="col-span-12 space-y-6 lg:col-span-4">
              {flow === 'despesa' && (
                <div className={`p-6 sm:p-8 ${GLASS}`}>
                  <FieldLabel>Natureza do Valor</FieldLabel>
                  <div className="mt-3 flex gap-1 rounded-full bg-[#eeeeed] p-1">
                    {(['FIXO', 'VARIAVEL'] as const).map((n) => (
                      <label key={n} className="flex-1">
                        <input
                          type="radio"
                          name="natureza"
                          className="peer sr-only"
                          checked={natureza === n}
                          onChange={() => setNatureza(n)}
                        />
                        <span className="block cursor-pointer rounded-full py-2 text-center text-sm font-semibold text-[#514440] transition-all peer-checked:bg-[#7d5141] peer-checked:text-white">
                          {n === 'FIXO' ? 'Fixo' : 'Variável'}
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              <div className={`p-6 sm:p-8 ${GLASS}`}>
                <div className="mb-4 flex items-center justify-between">
                  <FieldLabel>Recorrência</FieldLabel>
                  <label className="relative inline-flex cursor-pointer items-center">
                    <input
                      type="checkbox"
                      className="peer sr-only"
                      checked={recorrente}
                      onChange={(e) => setRecorrente(e.target.checked)}
                    />
                    <div className="h-6 w-11 rounded-full bg-[#e9e8e7] after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all peer-checked:bg-[#7d5141] peer-checked:after:translate-x-full" />
                  </label>
                </div>
                <div
                  className={[
                    'space-y-3 transition-all duration-300',
                    recorrente ? '' : 'pointer-events-none opacity-30',
                  ].join(' ')}
                >
                  <p className="text-[10px] font-bold uppercase tracking-widest text-[#514440]">
                    Frequência
                  </p>
                  {(
                    [
                      ['SEMANAL', 'Semanal'],
                      ['MENSAL', 'Mensal'],
                      ['ANUAL', 'Anual'],
                    ] as const
                  ).map(([val, label]) => (
                    <label key={val} className="flex cursor-pointer items-center gap-3">
                      <input
                        type="radio"
                        name="freq"
                        checked={frequencia === val}
                        onChange={() => setFrequencia(val)}
                        className="h-4 w-4 text-[#7d5141] focus:ring-[#7d5141]"
                      />
                      <span className="text-sm">{label}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="flex flex-col gap-3 lg:hidden">
                <button
                  type="submit"
                  className="flex w-full items-center justify-center gap-2 rounded-lg bg-[#7d5141] py-3 text-sm font-semibold text-white"
                >
                  <Save className="h-4 w-4" />
                  {isEdit ? 'Salvar alterações' : 'Salvar Transação'}
                </button>
                <Link
                  to="/admin/financeiro"
                  className="w-full rounded-lg border border-[#7d5141] py-3 text-center text-sm font-semibold text-[#7d5141]"
                >
                  Cancelar
                </Link>
              </div>
            </div>
          </div>
        </form>

        <div className="pointer-events-none fixed bottom-0 right-8 opacity-[0.06]">
          <Sparkles className="h-[280px] w-[280px] text-[#7d5141]" strokeWidth={0.5} />
        </div>
      </div>

      <DonaFooter />
    </DonaLayout>
  )
}
