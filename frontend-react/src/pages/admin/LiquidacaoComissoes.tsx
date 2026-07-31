import { useMemo, useState } from 'react'
import { Banknote } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader } from '../../components/dona/DonaLayout'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { useAuth } from '../../contexts/AuthContext'
import type { Lancamento } from '../../types'
import { getDb, liquidarComissoesPendentes } from '../../utils/mockDb'
import { formatBRL } from '../../utils/format'

interface Props {
  embedded?: boolean
}

type ComissaoRow = Lancamento & { profissional_nome: string }

function snapshotDb() {
  return structuredClone(getDb())
}

export function LiquidacaoComissoes({ embedded }: Props) {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(snapshotDb)
  const [msg, setMsg] = useState('')

  /** Clone: getDb() mutates in place; same ref + setState is a no-op for React. */
  const refresh = () => setDb(snapshotDb())

  const pendentesList = useMemo(() => {
    return db.lancamentos.filter(
      (l) =>
        l.tenant_id === tenantId &&
        l.tipo === 'CUSTO_VARIAVEL' &&
        l.status_pagamento === 'PENDENTE',
    )
  }, [db, tenantId])

  const pendentes = useMemo(
    () => pendentesList.reduce((s, l) => s + l.valor, 0),
    [pendentesList],
  )

  const byProf = useMemo(() => {
    return db.profissionais
      .filter((p) => p.tenant_id === tenantId)
      .map((prof) => {
        const items = pendentesList.filter((l) => l.profissional_id === prof.id)
        const total = items.reduce((s, l) => s + l.valor, 0)
        return { prof, items, total }
      })
      .filter((g) => g.total > 0)
  }, [db.profissionais, pendentesList, tenantId])

  const rows: ComissaoRow[] = useMemo(() => {
    const nomeById = new Map(
      db.profissionais
        .filter((p) => p.tenant_id === tenantId)
        .map((p) => [p.id, p.nome] as const),
    )
    return pendentesList.map((l) => ({
      ...l,
      profissional_nome: nomeById.get(l.profissional_id ?? '') ?? '—',
    }))
  }, [db.profissionais, pendentesList, tenantId])

  const liquidarTudo = () => {
    const n = liquidarComissoesPendentes(tenantId)
    refresh()
    setMsg(n > 0 ? `${n} comissão(ões) liquidada(s)!` : 'Nenhuma comissão pendente.')
  }

  const liquidarProf = (profId: string) => {
    const n = liquidarComissoesPendentes(tenantId, profId)
    refresh()
    setMsg(n > 0 ? 'Comissões quitadas para este profissional.' : 'Sem pendências.')
  }

  const columns: EntityColumn<ComissaoRow>[] = [
    {
      header: 'Profissional',
      cell: (l) => <span className="font-medium">{l.profissional_nome}</span>,
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
    {
      header: 'Status',
      cell: () => <Badge variant="warning">PENDENTE</Badge>,
    },
  ]

  const renderCard = (l: ComissaoRow) => (
    <div className="rounded-xl border border-aura-border/60 bg-aura-surface/40 p-4">
      <div className="min-w-0">
        <p className="font-semibold text-aura-anthracite">{l.descricao}</p>
        <p className="mt-0.5 text-xs text-aura-muted">{l.profissional_nome}</p>
      </div>
      <div className="mt-3 flex items-center justify-between gap-3">
        <Badge variant="warning">PENDENTE</Badge>
        <p className="shrink-0 whitespace-nowrap font-semibold">{formatBRL(l.valor)}</p>
      </div>
    </div>
  )

  const content = (
    <div className={embedded ? 'px-6' : ''}>
      <ToastFeedback message={msg || null} onDismiss={() => setMsg('')} />

      <Card className="mb-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm text-aura-muted">Total pendente</p>
            <p className="font-display text-2xl font-semibold text-aura-primary whitespace-nowrap">
              {formatBRL(pendentes)}
            </p>
          </div>
          <Button onClick={liquidarTudo} disabled={pendentes === 0}>
            <Banknote className="h-4 w-4" />
            Quitar tudo em lote
          </Button>
        </div>
      </Card>

      {byProf.length > 0 && (
        <div className="mb-4 flex flex-wrap gap-2">
          {byProf.map(({ prof, total }) => (
            <Button
              key={prof.id}
              variant="secondary"
              onClick={() => liquidarProf(prof.id)}
              className="min-h-touch-min"
            >
              Quitar {prof.nome}
              <span className="ml-1 whitespace-nowrap font-semibold">{formatBRL(total)}</span>
            </Button>
          ))}
        </div>
      )}

      <ResponsiveEntityList
        items={rows}
        getKey={(l) => l.id}
        renderCard={renderCard}
        columns={columns}
        emptyTitle="Nenhuma comissão pendente."
        tableFrom="md"
        cardListClassName="space-y-3"
      />
    </div>
  )

  if (embedded) return content

  return (
    <DonaLayout searchPlaceholder="Buscar comissões…">
      <PageHeader title="Liquidação de Comissões" subtitle="Quite comissões pendentes da equipe." />
      {content}
      <DonaFooter />
    </DonaLayout>
  )
}
