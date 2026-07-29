import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { enviarConfirmacaoAgendamento } from '../../services/whatsappService'
import { cancelAgendamento, getDb, getSlotsLivres, remarcarAgendamento } from '../../utils/mockDb'
import { addDaysISO, formatDateTimeBR } from '../../utils/format'

export function CancelamentoRecuperacao() {
  const { id } = useParams<{ id: string }>()
  const [db] = useState(getDb())
  const [remarcado, setRemarcado] = useState(false)
  const [loading, setLoading] = useState(false)

  const ag = db.agendamentos.find((a) => a.id === id)
  const prof = ag ? db.profissionais.find((p) => p.id === ag.profissional_id) : undefined
  const servico = ag ? db.servicos.find((s) => s.id === ag.servico_id) : undefined
  const tenant = ag ? db.tenants.find((t) => t.id === ag.tenant_id) : undefined

  const vagas =
    ag && prof
      ? getSlotsLivres(ag.profissional_id, ag.data, 3).length > 0
        ? getSlotsLivres(ag.profissional_id, ag.data, 3)
        : getSlotsLivres(ag.profissional_id, addDaysISO(ag.data, 1), 3)
      : []

  const confirmarCancelamento = async () => {
    if (!ag) return
    await cancelAgendamento(ag.id)
  }

  const remarcar = async (hora: string, data: string) => {
    if (!ag) return
    setLoading(true)
    remarcarAgendamento(ag.id, data, hora)
    await enviarConfirmacaoAgendamento({
      telefone: ag.cliente_telefone,
      clienteNome: ag.cliente_nome,
      servico: servico?.nome ?? '',
      profissional: prof?.nome ?? '',
      data,
      hora,
    })
    setLoading(false)
    setRemarcado(true)
  }

  if (!ag) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <p className="text-aura-muted">Agendamento não encontrado.</p>
      </div>
    )
  }

  if (remarcado) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
        <Card className="max-w-md text-center">
          <h1 className="font-display text-2xl font-semibold text-aura-primary">Remarcado!</h1>
          <p className="mt-2 text-sm text-aura-muted">Novo horário confirmado. Até breve!</p>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
      <Card className="w-full max-w-md" padding="lg">
        <p className="text-4xl">😢</p>
        <h1 className="mt-2 font-display text-2xl font-semibold">Que pena que você não poderá vir…</h1>
        <p className="mt-2 text-sm text-aura-muted">
          Sentiremos sua falta! Mas temos ótimas alternativas com {prof?.nome}:
        </p>

        <div className="mt-6 space-y-2">
          {vagas.length === 0 ? (
            <p className="text-sm text-aura-muted">Nenhuma vaga próxima disponível no momento.</p>
          ) : (
            vagas.map((v) => (
              <Button
                key={`${v.data}-${v.hora}`}
                variant="secondary"
                fullWidth
                loading={loading}
                onClick={() => remarcar(v.hora, v.data)}
              >
                Remarcar: {formatDateTimeBR(v.data, v.hora)}
              </Button>
            ))
          )}
        </div>

        <div className="mt-6 border-t border-aura-border pt-4">
          <Button variant="ghost" fullWidth onClick={confirmarCancelamento}>
            Confirmar cancelamento sem remarcar
          </Button>
        </div>

        {tenant && (
          <p className="mt-4 text-center text-xs text-aura-muted">{tenant.nome}</p>
        )}
      </Card>
    </div>
  )
}
