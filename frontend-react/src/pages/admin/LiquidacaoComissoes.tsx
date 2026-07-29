import { useState } from 'react'
import { Banknote } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { useAuth } from '../../contexts/AuthContext'
import { calcLucroLiquido, getDb, liquidarComissoesPendentes } from '../../utils/mockDb'
import { formatBRL } from '../../utils/format'

interface Props {
  embedded?: boolean
}

export function LiquidacaoComissoes({ embedded }: Props) {
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  const [db, setDb] = useState(getDb())
  const [msg, setMsg] = useState('')

  const refresh = () => setDb(getDb())
  const { pendentes } = calcLucroLiquido(tenantId)

  const pendentesList = db.lancamentos.filter(
    (l) =>
      l.tenant_id === tenantId &&
      l.tipo === 'CUSTO_VARIAVEL' &&
      l.status_pagamento === 'PENDENTE',
  )

  const byProf = db.profissionais
    .filter((p) => p.tenant_id === tenantId)
    .map((prof) => {
      const items = pendentesList.filter((l) => l.profissional_id === prof.id)
      const total = items.reduce((s, l) => s + l.valor, 0)
      return { prof, items, total }
    })
    .filter((g) => g.total > 0)

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

  const content = (
    <div className={embedded ? 'px-6' : ''}>
      {msg && <Alert variant="info" className="mb-4" onDismiss={() => setMsg('')}>{msg}</Alert>}

      <Card className="mb-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm text-aura-muted">Total pendente</p>
            <p className="font-display text-2xl font-semibold text-aura-primary">
              {formatBRL(pendentes)}
            </p>
          </div>
          <Button onClick={liquidarTudo} disabled={pendentes === 0}>
            <Banknote className="h-4 w-4" />
            Quitar tudo em lote
          </Button>
        </div>
      </Card>

      <div className="space-y-4">
        {byProf.map(({ prof, items, total }) => (
          <Card key={prof.id}>
            <div className="mb-3 flex items-center justify-between">
              <h3 className="font-medium">{prof.nome}</h3>
              <div className="flex items-center gap-2">
                <span className="font-semibold">{formatBRL(total)}</span>
                <Button variant="secondary" onClick={() => liquidarProf(prof.id)}>
                  Quitar
                </Button>
              </div>
            </div>
            <ul className="space-y-2">
              {items.map((l) => (
                <li
                  key={l.id}
                  className="flex items-center justify-between rounded-lg bg-aura-surface px-3 py-2 text-sm"
                >
                  <span className="text-aura-muted">{l.descricao}</span>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{formatBRL(l.valor)}</span>
                    <Badge variant="warning">PENDENTE</Badge>
                  </div>
                </li>
              ))}
            </ul>
          </Card>
        ))}
        {byProf.length === 0 && (
          <p className="text-center text-sm text-aura-muted">Nenhuma comissão pendente.</p>
        )}
      </div>
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
