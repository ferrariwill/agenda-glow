import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { TimeInput } from '../../components/ui/TimeInput'
import { getDb, persistDb } from '../../utils/mockDb'
import { formatDateTimeBR, formatTimeBR } from '../../utils/format'

export function AprovacaoPublica() {
  const { id } = useParams<{ id: string }>()
  const [db, setDb] = useState(getDb())
  const [novaHora, setNovaHora] = useState('')
  const [msg, setMsg] = useState('')

  const ag = db.agendamentos.find((a) => a.id === id)
  const servico = ag ? db.servicos.find((s) => s.id === ag.servico_id) : undefined
  const prof = ag ? db.profissionais.find((p) => p.id === ag.profissional_id) : undefined

  const aprovar = () => {
    if (!ag) return
    const idx = db.agendamentos.findIndex((a) => a.id === ag.id)
    const next = { ...db }
    next.agendamentos[idx] = { ...ag, status: 'CONFIRMADO' }
    persistDb(next)
    setDb(getDb())
    setMsg('Horário aprovado! Aguardamos você.')
  }

  const reagendar = () => {
    if (!ag || !novaHora) return
    const idx = db.agendamentos.findIndex((a) => a.id === ag.id)
    const next = { ...db }
    next.agendamentos[idx] = { ...ag, hora_inicio: novaHora, status: 'CONFIRMADO', minutos_invadidos: 0 }
    persistDb(next)
    setDb(getDb())
    setMsg(`Reagendado para ${formatTimeBR(novaHora)}.`)
  }

  if (!ag || ag.status !== 'EM_APROVACAO') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
        <Card className="max-w-md text-center">
          <p className="text-aura-muted">Solicitação não encontrada ou já processada.</p>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
      <Card className="w-full max-w-md" padding="lg">
        <h1 className="font-display text-2xl font-semibold">Aprovação de encaixe</h1>
        <p className="mt-2 text-sm text-aura-muted">
          Olá, <strong>{ag.cliente_nome}</strong>! Seu horário de {servico?.nome} com{' '}
          {prof?.nome} precisa de confirmação.
        </p>

        {ag.minutos_invadidos && (
          <Alert variant="warning" className="mt-4">
            Este encaixe utiliza <strong>{ag.minutos_invadidos} minutos</strong> além do slot
            padrão.
          </Alert>
        )}

        {msg && <Alert variant="info" className="mt-4">{msg}</Alert>}

        <div className="mt-6 space-y-3">
          <p className="text-sm">
            📅 {formatDateTimeBR(ag.data, ag.hora_inicio)}
          </p>
          <Button fullWidth onClick={aprovar}>
            Aprovar horário
          </Button>
          <div className="border-t border-aura-border pt-4">
            <p className="mb-2 text-sm font-medium">Ou reagendar no mesmo dia:</p>
            <TimeInput label="Novo horário" value={novaHora} onChange={setNovaHora} />
            <Button variant="secondary" fullWidth onClick={reagendar} disabled={!novaHora}>
              Reagendar
            </Button>
          </div>
        </div>
      </Card>
    </div>
  )
}
