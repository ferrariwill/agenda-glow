import { useMemo, useState } from 'react'
import { Plus } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { DateInput } from '../../components/ui/DateInput'
import { Input } from '../../components/ui/Input'
import { MoneyInput } from '../../components/ui/MoneyInput'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { useAuth } from '../../contexts/AuthContext'
import type { Lancamento, LancamentoTipo } from '../../types'
import {
  calcLucroLiquido,
  createLancamento,
  getDb,
  LancamentoValidationError,
} from '../../utils/mockDb'
import { formatBRL, formatDateBR, parseBRLInput, todayISO } from '../../utils/format'

const TIPO_OPCOES: { value: LancamentoTipo; label: string; hint: string }[] = [
  {
    value: 'CUSTO_FIXO',
    label: 'Custo fixo',
    hint: 'Aluguel, água, luz, internet…',
  },
  {
    value: 'CUSTO_VARIAVEL',
    label: 'Custo variável',
    hint: 'Insumos, esmaltes, materiais…',
  },
  {
    value: 'ENTRADA',
    label: 'Entrada extra',
    hint: 'Receita fora da agenda automática',
  },
]

function tipoLabel(tipo: LancamentoTipo): string {
  switch (tipo) {
    case 'ENTRADA':
      return 'Entrada'
    case 'CUSTO_FIXO':
      return 'Custo fixo'
    case 'CUSTO_VARIAVEL':
      return 'Custo variável'
  }
}

interface Props {
  embedded?: boolean
  onLancamentoCreated?: () => void
}

export function FinanceiroFluxo({ embedded, onLancamentoCreated }: Props) {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [tick, setTick] = useState(0)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [descricao, setDescricao] = useState('')
  const [valor, setValor] = useState('')
  const [tipo, setTipo] = useState<LancamentoTipo>('CUSTO_FIXO')
  const [data, setData] = useState(todayISO())

  const refresh = () => setTick((t) => t + 1)

  const { entradas, custosFixos, comissoes, lucro } = useMemo(
    () => calcLucroLiquido(tenantId),
    [tenantId, tick],
  )

  const lancamentos = useMemo(() => {
    return getDb()
      .lancamentos.filter((l) => l.tenant_id === tenantId)
      .sort((a, b) => b.data.localeCompare(a.data) || b.id.localeCompare(a.id))
  }, [tenantId, tick])

  const handleSubmit = async () => {
    setError('')
    setSuccess('')
    const parsed = parseBRLInput(valor)
    try {
      await createLancamento({
        tenant_id: tenantId,
        tipo,
        valor: parsed,
        descricao,
        data,
      })
      setDescricao('')
      setValor('')
      setTipo('CUSTO_FIXO')
      setData(todayISO())
      setSuccess('Lançamento registrado!')
      refresh()
      onLancamentoCreated?.()
    } catch (err) {
      if (err instanceof LancamentoValidationError) {
        setError(err.message)
      } else {
        setError(err instanceof Error ? err.message : 'Erro ao registrar')
      }
    }
  }

  const formCard = (
    <Card className="lg:sticky lg:top-6">
      <h2 className="mb-1 font-display text-lg font-semibold">Novo lançamento</h2>
      <p className="mb-4 text-sm text-aura-muted">
        Movimentações manuais fora da agenda automática.
      </p>

      {error && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <div className="space-y-4">
        <Input
          label="Descrição"
          value={descricao}
          onChange={(e) => setDescricao(e.target.value)}
          placeholder="Ex: Conta de água"
          required
        />
        <MoneyInput
          label="Valor (R$)"
          value={valor}
          onChange={setValor}
          placeholder="0,00"
          required
        />
        <DateInput
          label="Data"
          value={data}
          onChange={setData}
        />
        <div>
          <label className="mb-1.5 block text-sm font-medium">Tipo</label>
          <select
            value={tipo}
            onChange={(e) => setTipo(e.target.value as LancamentoTipo)}
            className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
          >
            {TIPO_OPCOES.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
          <p className="mt-1 text-xs text-aura-muted">
            {TIPO_OPCOES.find((o) => o.value === tipo)?.hint}
          </p>
        </div>
        {tipo === 'CUSTO_VARIAVEL' && (
          <p className="rounded-lg bg-aura-surface px-3 py-2 text-xs text-aura-muted">
            Custos variáveis manuais (insumos) não entram na liquidação de comissões da equipe.
          </p>
        )}
        <Button fullWidth onClick={handleSubmit}>
          <Plus className="h-4 w-4" />
          Registrar lançamento
        </Button>
      </div>
    </Card>
  )

  const fluxoColumns: EntityColumn<Lancamento>[] = [
    {
      header: 'Data',
      cell: (l) => (
        <span className="whitespace-nowrap">{formatDateBR(l.data)}</span>
      ),
    },
    {
      header: 'Tipo',
      cell: (l) => (
        <span
          className={
            l.tipo === 'ENTRADA'
              ? 'text-emerald-700'
              : l.tipo === 'CUSTO_FIXO'
                ? 'text-aura-muted'
                : 'text-amber-800'
          }
        >
          {tipoLabel(l.tipo)}
          {l.status_pagamento === 'PENDENTE' && ' · comissão'}
        </span>
      ),
    },
    {
      header: 'Descrição',
      cell: (l) => <span className="text-aura-muted">{l.descricao}</span>,
      priority: 'secondary',
    },
    {
      header: 'Valor',
      cell: (l) => (
        <span className="whitespace-nowrap font-medium">{formatBRL(l.valor)}</span>
      ),
      align: 'right',
    },
  ]

  const renderFluxoCard = (l: Lancamento) => (
    <div className="rounded-xl border border-aura-border/60 bg-aura-surface/40 p-4">
      <div className="min-w-0">
        <p className="font-semibold text-aura-anthracite">{l.descricao}</p>
        <p className="mt-0.5 whitespace-nowrap text-xs text-aura-muted">
          {formatDateBR(l.data)}
        </p>
      </div>
      <div className="mt-3 flex items-center justify-between gap-3">
        <span
          className={
            l.tipo === 'ENTRADA'
              ? 'text-sm text-emerald-700'
              : l.tipo === 'CUSTO_FIXO'
                ? 'text-sm text-aura-muted'
                : 'text-sm text-amber-800'
          }
        >
          {tipoLabel(l.tipo)}
          {l.status_pagamento === 'PENDENTE' && ' · comissão'}
        </span>
        <p className="shrink-0 whitespace-nowrap font-semibold text-aura-anthracite">
          {formatBRL(l.valor)}
        </p>
      </div>
    </div>
  )

  const historicoCard = (
    <Card className={embedded ? 'border-0 shadow-none' : ''}>
      <h2 className="mb-4 font-display text-lg font-semibold">Histórico</h2>
      <ResponsiveEntityList
        items={lancamentos}
        getKey={(l) => l.id}
        renderCard={renderFluxoCard}
        columns={fluxoColumns}
        emptyTitle="Nenhum lançamento registrado."
        tableFrom="md"
        cardListClassName="space-y-3"
      />
    </Card>
  )

  const content = (
    <>
      <ToastFeedback message={success || null} onDismiss={() => setSuccess('')} />
      {!embedded && (
        <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card>
            <p className="text-sm text-aura-muted">Entradas</p>
            <p className="font-display text-xl font-semibold text-emerald-700">{formatBRL(entradas)}</p>
          </Card>
          <Card>
            <p className="text-sm text-aura-muted">Custos fixos</p>
            <p className="font-display text-xl font-semibold">{formatBRL(custosFixos)}</p>
          </Card>
          <Card>
            <p className="text-sm text-aura-muted">Comissões (todas)</p>
            <p className="font-display text-xl font-semibold">{formatBRL(comissoes)}</p>
          </Card>
          <Card>
            <p className="text-sm text-aura-muted">Lucro líquido real</p>
            <p className="font-display text-xl font-semibold text-aura-primary">{formatBRL(lucro)}</p>
          </Card>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-1">{formCard}</div>
        <div className="lg:col-span-2">{historicoCard}</div>
      </div>
    </>
  )

  if (embedded) {
    return <div>{content}</div>
  }

  return (
    <DonaLayout searchPlaceholder="Buscar lançamentos…">
      <PageHeader title="Fluxo de Caixa" subtitle="Histórico de entradas e saídas." />
      {content}
      <DonaFooter />
    </DonaLayout>
  )
}
