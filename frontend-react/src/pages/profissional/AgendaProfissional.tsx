import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, Navigate } from 'react-router-dom'
import {
  ArrowLeft,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Edit,
  Plus,
} from 'lucide-react'
import { AceitaAntecipacaoBadge } from '../../components/agenda/AceitaAntecipacaoBadge'
import { EarlySlotRoundIndicator } from '../../components/agenda/EarlySlotRoundIndicator'
import { EarlySlotQueueInactiveBanner } from '../../components/agenda/EarlySlotQueueInactiveBanner'
import { NovoAgendamentoModal } from '../../components/dona/NovoAgendamentoModal'
import { ProfissionalLayout, ProfissionalGLASS } from '../../components/profissional/ProfissionalLayout'
import { Alert } from '../../components/ui/Alert'
import { Badge, confirmacaoClienteBadge, statusAgendamentoBadge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { useAuth } from '../../contexts/AuthContext'
import { subscribeStore } from '../../data/store'
import { refreshAfterMutation } from '../../data/sync'
import { useSilentTenantBootstrapRefresh } from '../../hooks/useSilentTenantBootstrapRefresh'
import { IS_MOCK } from '../../lib/config'
import { useEarlySlotAgendaExpiry } from '../../hooks/useEarlySlotAgendaExpiry'
import type { Agendamento } from '../../types'
import {
  getAgendamentoDuration,
  getAgendamentoServicosNomes,
  getDb,
  registrarComissaoAgendamento,
  updateAgendamentoStatus,
} from '../../utils/mockDb'
import {
  addDaysISO,
  formatTimeBR,
  formatWeekdayDateBR,
  todayISO,
} from '../../utils/format'
import { EditarAgendamentoProfModal } from './EditarAgendamentoProfModal'

function weekStart(iso: string): string {
  const d = new Date(`${iso}T12:00:00`)
  const day = d.getDay()
  const monday = new Date(d)
  monday.setDate(d.getDate() - ((day + 6) % 7))
  return monday.toISOString().slice(0, 10)
}

function weekDays(start: string): string[] {
  return Array.from({ length: 7 }, (_, i) => addDaysISO(start, i))
}

export function AgendaProfissional() {
  const { session } = useAuth()
  const profId = session?.user.profissional_id ?? ''
  const tenantId = session?.user.tenant_id ?? ''
  const isProf = session?.user.role === 'PROFISSIONAL'
  const isDonaAgenda = session?.user.role === 'DONA'
  useSilentTenantBootstrapRefresh(session?.user.role)

  const [db, setDb] = useState(getDb())
  const [weekAnchor, setWeekAnchor] = useState(weekStart(todayISO()))
  const [selectedDay, setSelectedDay] = useState(todayISO())
  const [agModalOpen, setAgModalOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Agendamento | null>(null)
  const [success, setSuccess] = useState('')

  const days = useMemo(() => weekDays(weekAnchor), [weekAnchor])
  const hoje = todayISO()
  const tenant = db.tenants.find((item) => item.id === tenantId)

  const refresh = useCallback(() => setDb(getDb()), [])

  useEffect(() => subscribeStore(refresh), [refresh])

  // Oferta de antecipação expirada: refetch silencioso, sem reload da página.
  const handleOfferExpired = useCallback(() => {
    if (IS_MOCK) {
      refresh()
      return
    }
    refreshAfterMutation(session?.user.role)
      .then(refresh)
      .catch(() => undefined)
  }, [refresh, session?.user.role])


  const agendamentos = db.agendamentos.filter(
    (a) =>
      a.profissional_id === profId &&
      a.tenant_id === tenantId &&
      days.includes(a.data) &&
      a.status !== 'CANCELADO',
  )

  useEarlySlotAgendaExpiry(
    agendamentos
      .filter((ag) => ag.early_slot_offer?.offer_status === 'PENDENTE')
      .map((ag) => ag.early_slot_offer?.expires_at),
    handleOfferExpired,
  )

  if (isDonaAgenda && !profId) {
    return <Navigate to="/admin/configuracoes" replace />
  }

  const dayAgs = agendamentos
    .filter((a) => a.data === selectedDay)
    .sort((a, b) => a.hora_inicio.localeCompare(b.hora_inicio))

  const concluir = async (ag: Agendamento) => {
    await updateAgendamentoStatus(ag.id, 'CONCLUIDO', { role: 'PROFISSIONAL' })
    if (IS_MOCK) {
      registrarComissaoAgendamento(ag.id, profId, tenantId)
    }
    refresh()
    setSuccess('Atendimento concluído!')
  }

  const novoBtn = (
    <button
      type="button"
      onClick={() => setAgModalOpen(true)}
      className="flex items-center gap-2 rounded-full bg-[#7d5141] px-4 py-2 text-xs font-bold uppercase tracking-wider text-white hover:opacity-90"
    >
      <Plus className="h-4 w-4" />
      Novo
    </button>
  )

  const agendaContent = (
  <>
      {isDonaAgenda && (
        <Link
          to="/admin/dashboard"
          className="mb-4 inline-flex items-center gap-1 text-sm text-[#7d5141] hover:underline"
        >
          <ArrowLeft className="h-4 w-4" />
          Voltar ao painel da dona
        </Link>
      )}

      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}
      <EarlySlotQueueInactiveBanner
        queueActive={tenant?.early_slot_queue_active}
        canReconnectWhatsApp={session?.user.role === 'DONA'}
      />

      {/* Navegação semanal */}
      <div className={`mb-6 flex flex-wrap items-center justify-between gap-4 p-4 ${ProfissionalGLASS}`}>
        <button
          type="button"
          onClick={() => setWeekAnchor(addDaysISO(weekAnchor, -7))}
          className="rounded-lg p-2 text-[#514440] hover:bg-[#efdcd1]/30"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
        <div className="flex flex-1 flex-wrap justify-center gap-2">
          {days.map((d) => {
            const isSelected = d === selectedDay
            const isToday = d === hoje
            const count = agendamentos.filter((a) => a.data === d).length
            return (
              <button
                key={d}
                type="button"
                onClick={() => setSelectedDay(d)}
                className={[
                  'min-w-[3.5rem] rounded-xl px-2 py-2 text-center text-xs transition-all',
                  isSelected
                    ? 'bg-[#7d5141] text-white shadow-md'
                    : 'bg-[#f4f3f2] text-[#514440] hover:bg-[#efdcd1]/40',
                  isToday && !isSelected ? 'ring-2 ring-[#7d5141]/30' : '',
                ].join(' ')}
              >
                <span className="block font-bold uppercase">
                  {new Date(`${d}T12:00:00`).toLocaleDateString('pt-BR', { weekday: 'short' })}
                </span>
                <span className="block text-lg font-semibold">
                  {new Date(`${d}T12:00:00`).getDate()}
                </span>
                {count > 0 && (
                  <span className="mt-0.5 block text-[10px] opacity-80">{count} ag.</span>
                )}
              </button>
            )
          })}
        </div>
        <button
          type="button"
          onClick={() => setWeekAnchor(addDaysISO(weekAnchor, 7))}
          className="rounded-lg p-2 text-[#514440] hover:bg-[#efdcd1]/30"
        >
          <ChevronRight className="h-5 w-5" />
        </button>
      </div>

      <div className="mb-4 flex items-center justify-between">
        <h2 className="font-display text-lg font-semibold capitalize text-[#7d5141]">
          {formatWeekdayDateBR(selectedDay)}
        </h2>
        {!isProf && (
          <Button onClick={() => setAgModalOpen(true)}>
            <Plus className="h-4 w-4" />
            Novo agendamento
          </Button>
        )}
      </div>

      {dayAgs.length === 0 ? (
        <div className={`py-16 text-center ${ProfissionalGLASS}`}>
          <p className="text-[#514440]">Nenhum agendamento neste dia.</p>
          <button
            type="button"
            onClick={() => setAgModalOpen(true)}
            className="mt-4 text-sm font-semibold text-[#7d5141] hover:underline"
          >
            Criar agendamento
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {dayAgs.map((ag) => {
            const servicoLabel = getAgendamentoServicosNomes(ag, db)
            const dur = getAgendamentoDuration(ag, db)
            const podeConcluir =
              ag.status === 'CONFIRMADO' || ag.status === 'AGENDADO'
            const podeEditar = ag.status !== 'CONCLUIDO'
            const confirmation = confirmacaoClienteBadge(ag.confirmacao_cliente)
            const confirmationHint = ag.ultimo_lembrete_enviado_em
              ? `Lembrete enviado em ${new Date(ag.ultimo_lembrete_enviado_em).toLocaleString('pt-BR')}`
              : (ag.confirmacao_cliente ?? 'PENDENTE') === 'PENDENTE'
                ? 'Aguardando resposta'
                : undefined

            return (
              <div
                key={ag.id}
                className={`p-4 sm:p-5 ${ProfissionalGLASS} transition-colors hover:bg-[#996958]/5`}
              >
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-lg font-semibold text-[#7d5141]">
                        {formatTimeBR(ag.hora_inicio)}
                      </span>
                      <Badge variant={statusAgendamentoBadge(ag.status)}>
                        {ag.status}
                      </Badge>
                      <AceitaAntecipacaoBadge aceitaAdiantar={ag.aceita_adiantar} />
                      <EarlySlotRoundIndicator
                        offer={ag.early_slot_offer}
                      />
                      <span title={confirmationHint}>
                        <Badge variant={confirmation.variant}>{confirmation.label}</Badge>
                      </span>
                    </div>
                    <p className="mt-1 font-medium text-[#1a1c1c]">{ag.cliente_nome}</p>
                    <p className="text-sm text-[#514440]">
                      {servicoLabel} · {dur} min
                    </p>
                    {ag.observacoes && (
                      <p className="mt-1 text-xs italic text-[#514440]/80">{ag.observacoes}</p>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {podeEditar && (
                      <button
                        type="button"
                        onClick={() => setEditTarget(ag)}
                        className="flex items-center gap-1 rounded-lg border border-[#d6c2bd] px-3 py-2 text-sm text-[#514440] hover:border-[#7d5141] hover:text-[#7d5141]"
                      >
                        <Edit className="h-4 w-4" />
                        Editar
                      </button>
                    )}
                    {podeConcluir && (
                      <button
                        type="button"
                        onClick={() => concluir(ag)}
                        className="flex items-center gap-1 rounded-lg bg-[#7d5141] px-3 py-2 text-sm font-semibold text-white hover:opacity-90"
                      >
                        <CheckCircle2 className="h-4 w-4" />
                        Concluir
                      </button>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <NovoAgendamentoModal
        open={agModalOpen}
        onClose={() => setAgModalOpen(false)}
        tenantId={tenantId}
        profissionalFixo={profId}
        preset={{ data: selectedDay }}
        cadastrarClienteNovo
        onSuccess={(msg) => {
          setSuccess(msg)
          refresh()
        }}
      />

      <EditarAgendamentoProfModal
        open={Boolean(editTarget)}
        agendamento={editTarget}
        profId={profId}
        tenantId={tenantId}
        onClose={() => setEditTarget(null)}
        onSuccess={(msg) => {
          setSuccess(msg)
          refresh()
        }}
      />
  </>
  )

  if (isProf) {
    return (
      <ProfissionalLayout
        title="Minha Agenda"
        subtitle="Somente seus agendamentos"
        headerAction={novoBtn}
      >
        {agendaContent}
      </ProfissionalLayout>
    )
  }

  return (
    <div className="min-h-screen bg-[#faf9f8] p-4">
      <div className="mx-auto max-w-3xl">{agendaContent}</div>
    </div>
  )
}
