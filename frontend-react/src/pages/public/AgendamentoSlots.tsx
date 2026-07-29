import { useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { DateInput } from '../../components/ui/DateInput'
import { enviarConfirmacaoAgendamento } from '../../services/whatsappService'
import {
  getDb,
  getSlotsLivres,
  getTenantBySlug,
  minutesToTime,
  persistDb,
} from '../../utils/mockDb'
import { formatDateTimeBR, formatTimeBR, todayISO } from '../../utils/format'

const SLOT_MINUTES = 30
const GRID_START = 8 * 60
const GRID_END = 20 * 60

export function AgendamentoSlots() {
  const { slug } = useParams<{ slug: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const tenant = slug ? getTenantBySlug(slug) : undefined
  const db = getDb()

  const [profId, setProfId] = useState('')
  const [servicoId, setServicoId] = useState('')
  const [data, setData] = useState(todayISO())
  const [clienteNome, setClienteNome] = useState('')
  const [clienteTel, setClienteTel] = useState('')
  const [selectedSlot, setSelectedSlot] = useState(searchParams.get('hora') ?? '')
  const [loading, setLoading] = useState(false)
  const [done, setDone] = useState(false)

  const profissionais = tenant
    ? db.profissionais.filter((p) => p.tenant_id === tenant.id && p.ativo)
    : []
  const servicos = tenant ? db.servicos.filter((s) => s.tenant_id === tenant.id && s.ativo) : []

  const slots = useMemo(() => {
    if (!profId) return []
    const livres = getSlotsLivres(profId, data, 20)
    const list: string[] = []
    for (let m = GRID_START; m < GRID_END; m += SLOT_MINUTES) {
      list.push(minutesToTime(m))
    }
    const ocupados = db.agendamentos
      .filter(
        (a) =>
          a.profissional_id === profId &&
          a.data === data &&
          a.status !== 'CANCELADO',
      )
      .map((a) => a.hora_inicio)
    return list.filter((s) => !ocupados.includes(s) || livres.some((l) => l.hora === s))
  }, [profId, data, db.agendamentos])

  const confirmar = async () => {
    if (!tenant || !profId || !servicoId || !selectedSlot || !clienteNome || !clienteTel) return
    setLoading(true)
    const ag = {
      id: crypto.randomUUID(),
      tenant_id: tenant.id,
      profissional_id: profId,
      servico_id: servicoId,
      adicional_ids: [] as string[],
      cliente_nome: clienteNome,
      cliente_telefone: clienteTel,
      data,
      hora_inicio: selectedSlot,
      status: 'CONFIRMADO' as const,
    }
    persistDb({ ...db, agendamentos: [...db.agendamentos, ag] })
    const servico = servicos.find((s) => s.id === servicoId)
    const prof = profissionais.find((p) => p.id === profId)
    await enviarConfirmacaoAgendamento({
      telefone: clienteTel,
      clienteNome,
      servico: servico?.nome ?? '',
      profissional: prof?.nome ?? '',
      data,
      hora: selectedSlot,
    })
    setLoading(false)
    setDone(true)
  }

  if (!tenant) {
    return <p className="p-8 text-center text-aura-muted">Salão não encontrado.</p>
  }

  if (done) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
        <Card className="max-w-md text-center">
          <h1 className="font-display text-2xl font-semibold">Agendamento confirmado!</h1>
          <p className="mt-2 text-sm text-aura-muted">
            Enviamos a confirmação por WhatsApp. Até breve!
          </p>
          <Button className="mt-4" onClick={() => navigate(`/${slug}`)}>
            Voltar ao catálogo
          </Button>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-aura-surface px-4 py-6">
      <button
        type="button"
        onClick={() => navigate(`/${slug}`)}
        className="mb-4 flex items-center gap-1 text-sm text-aura-primary"
      >
        <ArrowLeft className="h-4 w-4" />
        Voltar
      </button>

      <h1 className="mb-6 font-display text-2xl font-semibold">Escolha seu horário</h1>

      <div className="mx-auto max-w-lg space-y-4">
        <Card className="space-y-3">
          <input
            placeholder="Seu nome"
            value={clienteNome}
            onChange={(e) => setClienteNome(e.target.value)}
            className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
          />
          <input
            placeholder="WhatsApp (com DDD)"
            value={clienteTel}
            onChange={(e) => setClienteTel(e.target.value)}
            className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
          />
          <select
            value={servicoId}
            onChange={(e) => setServicoId(e.target.value)}
            className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
          >
            <option value="">Serviço</option>
            {servicos.map((s) => (
              <option key={s.id} value={s.id}>
                {s.nome}
              </option>
            ))}
          </select>
          <select
            value={profId}
            onChange={(e) => setProfId(e.target.value)}
            className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
          >
            <option value="">Profissional</option>
            {profissionais.map((p) => (
              <option key={p.id} value={p.id}>
                {p.nome}
              </option>
            ))}
          </select>
          <DateInput label="Data do atendimento" value={data} onChange={setData} />
        </Card>

        {profId && (
          <Card>
            <p className="mb-3 text-sm font-medium">Horários disponíveis (30 min)</p>
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
              {slots.map((slot) => (
                <button
                  key={slot}
                  type="button"
                  onClick={() => setSelectedSlot(slot)}
                  className={[
                    'rounded-lg border py-2.5 text-sm font-medium transition-colors',
                    selectedSlot === slot
                      ? 'border-aura-primary bg-aura-primary text-white'
                      : 'border-aura-border bg-white hover:border-aura-primary',
                  ].join(' ')}
                >
                  {formatTimeBR(slot)}
                </button>
              ))}
            </div>
          </Card>
        )}

        {selectedSlot && (
          <Alert variant="info">
            Horário selecionado: <strong>{formatDateTimeBR(data, selectedSlot)}</strong>
          </Alert>
        )}

        <Button fullWidth loading={loading} onClick={confirmar} disabled={!selectedSlot}>
          Confirmar agendamento
        </Button>
      </div>
    </div>
  )
}
