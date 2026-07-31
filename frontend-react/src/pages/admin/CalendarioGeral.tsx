import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Banknote,
  CalendarPlus,
  ChevronLeft,
  ChevronRight,
  Clock,
  Lock,
  Pencil,
  Plus,
  TrendingUp,
  XCircle,
} from 'lucide-react'
import { AceitaAntecipacaoBadge } from '../../components/agenda/AceitaAntecipacaoBadge'
import { EarlySlotRoundIndicator } from '../../components/agenda/EarlySlotRoundIndicator'
import { EarlySlotQueueInactiveBanner } from '../../components/agenda/EarlySlotQueueInactiveBanner'
import { DonaLayout, DonaFooter } from '../../components/dona/DonaLayout'
import { SecretariaLayout } from '../../components/secretaria/SecretariaLayout'
import { CobrancaAgendamentoModal } from '../../components/secretaria/CobrancaAgendamentoModal'
import { NovoAgendamentoModal, type NovoAgendamentoPreset } from '../../components/dona/NovoAgendamentoModal'
import { EditarAgendamentoProfModal } from '../profissional/EditarAgendamentoProfModal'
import { Alert } from '../../components/ui/Alert'
import { Badge, confirmacaoClienteBadge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { ConfirmModal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import { refreshAfterMutation } from '../../data/sync'
import { subscribeStore } from '../../data/store'
import { IS_MOCK } from '../../lib/config'
import { useEarlySlotAgendaExpiry } from '../../hooks/useEarlySlotAgendaExpiry'
import { useSilentTenantBootstrapRefresh } from '../../hooks/useSilentTenantBootstrapRefresh'
import { enviarOfertaVagaFila } from '../../services/whatsappService'
import type { Agendamento } from '../../types'
import {
  cancelAgendamento,
  getAgendamentoDuration,
  getAgendamentoServicosNomes,
  getDb,
  getFilaAguardando,
  markFilaNotificada,
  minutesToTime,
  timeToMinutes,
} from '../../utils/mockDb'
import {
  addDaysISO,
  formatDateShortBR,
  formatMonthYearBR,
  formatTimeBR,
  getCalendarGrid,
  initials,
  todayISO,
  WEEKDAYS_SHORT_PT,
} from '../../utils/format'

const GRID_START = 8 * 60
const GRID_END = 20 * 60
const ROW_HEIGHT = 64

type ViewMode = 'hoje' | 'semana' | 'mes'
type CalendarioVariant = 'dona' | 'secretaria'

interface CalendarioGeralProps {
  variant?: CalendarioVariant
}

function hourLabels(): string[] {
  const labels: string[] = []
  for (let h = 8; h <= 20; h++) labels.push(`${String(h).padStart(2, '0')}:00`)
  return labels
}

function calcOcupacao(
  data: string,
  profissionais: ReturnType<typeof getDb>['profissionais'],
  agendamentos: Agendamento[],
  db: ReturnType<typeof getDb>,
): number {
  const day = new Date(`${data}T12:00:00`).getDay()
  let totalAvailable = 0
  let totalBooked = 0
  for (const prof of profissionais) {
    const exp = prof.expedientes.find((e) => e.dia_semana === day)
    if (!exp) continue
    const work = timeToMinutes(exp.horario_saida) - timeToMinutes(exp.horario_entrada)
    const lunch =
      exp.inicio_almoco && exp.fim_almoco
        ? timeToMinutes(exp.fim_almoco) - timeToMinutes(exp.inicio_almoco)
        : 0
    totalAvailable += Math.max(0, work - lunch)
    const ags = agendamentos.filter((a) => a.profissional_id === prof.id)
    totalBooked += ags.reduce((s, a) => s + getAgendamentoDuration(a, db), 0)
  }
  return totalAvailable > 0 ? Math.round((totalBooked / totalAvailable) * 100) : 0
}

function appointmentEndTime(ag: Agendamento, db: ReturnType<typeof getDb>) {
  const start = timeToMinutes(ag.hora_inicio)
  const end = start + getAgendamentoDuration(ag, db)
  return `${formatTimeBR(ag.hora_inicio)} - ${formatTimeBR(minutesToTime(end))}`
}

function confirmationHint(ag: Agendamento) {
  if (ag.ultimo_lembrete_enviado_em) {
    return `Lembrete enviado em ${new Date(ag.ultimo_lembrete_enviado_em).toLocaleString('pt-BR')}`
  }
  return (ag.confirmacao_cliente ?? 'PENDENTE') === 'PENDENTE'
    ? 'Aguardando resposta'
    : undefined
}

function statusDisplay(ag: Agendamento, data: string) {
  const now = new Date()
  const isToday = data === todayISO()
  const start = timeToMinutes(ag.hora_inicio)
  const end = start + 30
  const nowMin = now.getHours() * 60 + now.getMinutes()

  if (ag.status === 'EM_APROVACAO') {
    return { label: 'Encaixe pendente', dot: 'bg-amber-400', card: 'conflict' as const }
  }
  if (ag.status === 'AGENDADO') {
    return { label: 'Aguardando cliente', dot: 'bg-[#615b58]', card: 'muted' as const }
  }
  if (isToday && nowMin >= start && nowMin < end && ag.status === 'CONFIRMADO') {
    return { label: 'Em atendimento', dot: 'bg-amber-400', card: 'active' as const }
  }
  if (ag.status === 'CONCLUIDO') {
    return { label: 'Concluído', dot: 'bg-[#615b58]', card: 'muted' as const }
  }
  return { label: 'Confirmado', dot: 'bg-emerald-500', card: 'confirmed' as const }
}

function cardClasses(kind: ReturnType<typeof statusDisplay>['card']) {
  switch (kind) {
    case 'confirmed':
      return 'border-l-4 border-[#7d5141] bg-[#efdcd1]/40'
    case 'active':
      return 'border-l-4 border-[#7d5141] bg-[#7d5141]/10'
    case 'conflict':
      return 'border-2 border-dashed border-red-400/40 bg-red-50/80'
    case 'muted':
      return 'border-l-4 border-[#83746f] bg-[#e3e2e1]'
    default:
      return 'border-l-4 border-[#7d5141] bg-[#efdcd1]/40'
  }
}

export function CalendarioGeral({ variant = 'dona' }: CalendarioGeralProps) {
  const isSecretaria = variant === 'secretaria'
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''
  useSilentTenantBootstrapRefresh(session?.user.role)
  const [data, setData] = useState(todayISO())
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('hoje')
  const [calYear, setCalYear] = useState(() => new Date().getFullYear())
  const [calMonth, setCalMonth] = useState(() => new Date().getMonth() + 1)
  const [profMobile, setProfMobile] = useState('')
  const [soAguardandoConfirmacao, setSoAguardandoConfirmacao] = useState(false)

  const [filaModal, setFilaModal] = useState<{
    cliente: string
    telefone: string
    hora: string
    profissionalId: string
    filaId: string
  } | null>(null)
  const [sending, setSending] = useState(false)
  const [agModalOpen, setAgModalOpen] = useState(false)
  const [agPreset, setAgPreset] = useState<NovoAgendamentoPreset | undefined>()
  const [success, setSuccess] = useState('')
  const [editAg, setEditAg] = useState<Agendamento | null>(null)
  const [cobrancaAg, setCobrancaAg] = useState<Agendamento | null>(null)

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
  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)
  const especialidades = db.especialidades.filter((e) => e.tenant_id === tenantId)
  const tenant = db.tenants.find((item) => item.id === tenantId)
  const hours = hourLabels()
  const dayOfWeek = new Date(`${data}T12:00:00`).getDay()
  const profMobileId = profMobile || profissionais[0]?.id || ''

  const agendamentosDia = useMemo(() => {
    const q = search.trim().toLowerCase()
    return db.agendamentos
      .filter(
        (a) =>
          a.tenant_id === tenantId &&
          a.data === data &&
          a.status !== 'CANCELADO' &&
          (!soAguardandoConfirmacao ||
            ((a.confirmacao_cliente ?? 'PENDENTE') === 'PENDENTE' && a.status !== 'CONCLUIDO')) &&
          (!q ||
            a.cliente_nome.toLowerCase().includes(q) ||
            getAgendamentoServicosNomes(a, db).toLowerCase().includes(q)),
      )
      .sort((a, b) => a.hora_inicio.localeCompare(b.hora_inicio))
  }, [db, tenantId, data, search, soAguardandoConfirmacao])

  useEarlySlotAgendaExpiry(
    agendamentosDia
      .filter((ag) => ag.early_slot_offer?.offer_status === 'PENDENTE')
      .map((ag) => ag.early_slot_offer?.expires_at),
    handleOfferExpired,
  )

  const totalHoje = agendamentosDia.length
  const ocupacao = calcOcupacao(data, profissionais, agendamentosDia, db)

  const proximos = useMemo(() => {
    const nowMin =
      data === todayISO()
        ? new Date().getHours() * 60 + new Date().getMinutes()
        : 0
    return agendamentosDia
      .filter((a) => data !== todayISO() || timeToMinutes(a.hora_inicio) >= nowMin)
      .slice(0, 5)
  }, [agendamentosDia, data])

  const nowLineTop = useMemo(() => {
    if (data !== todayISO()) return null
    const nowMin = new Date().getHours() * 60 + new Date().getMinutes()
    if (nowMin < GRID_START || nowMin > GRID_END) return null
    return ((nowMin - GRID_START) / 60) * ROW_HEIGHT
  }, [data])

  const gridHeight = ((GRID_END - GRID_START) / 60) * ROW_HEIGHT

  const openNovoAgendamento = (preset?: NovoAgendamentoPreset) => {
    setAgPreset({ data, ...preset })
    setAgModalOpen(true)
  }

  const handleCancel = async (ag: Agendamento) => {
    await cancelAgendamento(ag.id)
    refresh()
    const fila = getFilaAguardando(ag.profissional_id)
    if (fila) {
      setFilaModal({
        cliente: fila.cliente_nome,
        telefone: fila.cliente_telefone,
        hora: ag.hora_inicio,
        profissionalId: ag.profissional_id,
        filaId: fila.id,
      })
    }
  }

  const confirmFila = async () => {
    if (!filaModal) return
    setSending(true)
    const prof = db.profissionais.find((p) => p.id === filaModal.profissionalId)
    await enviarOfertaVagaFila({
      telefone: filaModal.telefone,
      clienteNome: filaModal.cliente,
      profissional: prof?.nome ?? 'Profissional',
      data,
      hora: filaModal.hora,
      linkConfirmacao: `${window.location.origin}/${db.tenants.find((t) => t.id === tenantId)?.slug}/agendar?hora=${filaModal.hora}`,
    })
    markFilaNotificada(filaModal.filaId)
    setSending(false)
    setFilaModal(null)
    refresh()
  }

  const getLunchBlock = (profId: string) => {
    const prof = profissionais.find((p) => p.id === profId)
    const exp = prof?.expedientes.find((e) => e.dia_semana === dayOfWeek)
    if (!exp?.inicio_almoco || !exp?.fim_almoco) return null
    const start = timeToMinutes(exp.inicio_almoco)
    const end = timeToMinutes(exp.fim_almoco)
    return {
      top: ((start - GRID_START) / 60) * ROW_HEIGHT,
      height: ((end - start) / 60) * ROW_HEIGHT,
    }
  }

  const renderAppointment = (ag: Agendamento) => {
    const startMin = timeToMinutes(ag.hora_inicio)
    const duration = getAgendamentoDuration(ag, db)
    const top = ((startMin - GRID_START) / 60) * ROW_HEIGHT
    const height = Math.max(((duration / 60) * ROW_HEIGHT) - 4, 36)
    const servicoLabel = getAgendamentoServicosNomes(ag, db)
    const st = statusDisplay(ag, data)
    const confirmation = confirmacaoClienteBadge(ag.confirmacao_cliente)
    const hint = confirmationHint(ag)

    return (
      <div
        key={ag.id}
        role="button"
        tabIndex={0}
        onClick={() => isSecretaria && setEditAg(ag)}
        onKeyDown={(e) => {
          if (isSecretaria && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault()
            setEditAg(ag)
          }
        }}
        className={[
          'group absolute left-2 right-2 cursor-pointer overflow-hidden rounded-xl p-3 shadow-sm transition-shadow hover:shadow-md',
          cardClasses(st.card),
        ].join(' ')}
        style={{ top, height }}
      >
        <div className="flex items-start justify-between gap-1">
          <div className="min-w-0">
            <div className="flex items-center justify-between gap-1">
              <p className="truncate text-[10px] font-bold uppercase text-[#7d5141]">
                {appointmentEndTime(ag, db)}
              </p>
              <span className="flex shrink-0 items-center gap-1">
                <AceitaAntecipacaoBadge aceitaAdiantar={ag.aceita_adiantar} compact />
                <EarlySlotRoundIndicator offer={ag.early_slot_offer} compact />
              </span>
            </div>
            <p className="truncate text-sm font-semibold text-aura-anthracite">{ag.cliente_nome}</p>
            <p className="truncate text-xs text-aura-muted">{servicoLabel}</p>
            <div className="mt-1.5 flex flex-wrap items-center gap-1" title={hint}>
              <span className={`h-2 w-2 rounded-full ${st.dot}`} />
              <span className="text-[10px] text-aura-muted">{st.label}</span>
              <Badge variant={confirmation.variant}>{confirmation.label}</Badge>
              {ag.cobrado_em && (
                <span className="text-[10px] font-medium text-emerald-700">· Pago</span>
              )}
            </div>
          </div>
          <div className="flex shrink-0 flex-col gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
            {isSecretaria && !ag.cobrado_em && ag.status !== 'CANCELADO' && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  setCobrancaAg(ag)
                }}
                className="rounded p-1 hover:bg-white/50"
                aria-label="Cobrar"
              >
                <Banknote className="h-3.5 w-3.5 text-[#7d5141]" />
              </button>
            )}
            {isSecretaria && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  setEditAg(ag)
                }}
                className="rounded p-1 hover:bg-white/50"
                aria-label="Editar"
              >
                <Pencil className="h-3.5 w-3.5 text-aura-muted" />
              </button>
            )}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                handleCancel(ag)
              }}
              className="rounded p-1 hover:bg-white/50"
              aria-label="Cancelar"
            >
              <XCircle className="h-3.5 w-3.5 text-aura-muted" />
            </button>
          </div>
        </div>
      </div>
    )
  }

  const calendarDays = getCalendarGrid(calYear, calMonth)
  const shiftMonth = (delta: number) => {
    let m = calMonth + delta
    let y = calYear
    if (m < 1) {
      m = 12
      y -= 1
    } else if (m > 12) {
      m = 1
      y += 1
    }
    setCalMonth(m)
    setCalYear(y)
  }

  const sidebarPanel = (
    <aside className="flex w-full flex-col gap-6 lg:w-80 lg:shrink-0">
      <div className="rounded-xl border border-[#efdcd1]/20 bg-white p-4 shadow-sm">
        <div className="mb-3 flex items-center justify-between px-1">
          <p className="font-semibold text-[#7d5141]">
            {formatMonthYearBR(`${calYear}-${String(calMonth).padStart(2, '0')}`)}
          </p>
          <div className="flex gap-1">
            <button type="button" onClick={() => shiftMonth(-1)} className="rounded p-1 text-aura-muted hover:text-[#7d5141]">
              <ChevronLeft className="h-4 w-4" />
            </button>
            <button type="button" onClick={() => shiftMonth(1)} className="rounded p-1 text-aura-muted hover:text-[#7d5141]">
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
        <div className="mb-2 grid grid-cols-7 text-center text-[10px] font-bold uppercase text-aura-muted">
          {WEEKDAYS_SHORT_PT.map((d) => (
            <div key={d}>{d.charAt(0)}</div>
          ))}
        </div>
        <div className="grid grid-cols-7 gap-1 text-center">
          {calendarDays.map((cell, i) => {
            if (!cell.iso) return <div key={`e-${i}`} />
            const selected = cell.iso === data
            const isPast = cell.iso < todayISO()
            return (
              <button
                key={cell.iso}
                type="button"
                disabled={isPast}
                onClick={() => setData(cell.iso!)}
                className={[
                  'rounded-full p-1 text-xs transition-colors',
                  selected ? 'bg-[#7d5141] font-bold text-white' : isPast ? 'text-aura-muted/30' : 'hover:bg-[#efdcd1]',
                ].join(' ')}
              >
                {cell.day}
              </button>
            )
          })}
        </div>
      </div>

      <div className="grid gap-3">
        <div className="rounded-xl border border-[#7d5141]/10 bg-[#7d5141]/5 p-4">
          <p className="text-[11px] font-bold uppercase tracking-widest text-aura-muted">Total hoje</p>
          <div className="mt-1 flex items-center justify-between">
            <p className="font-display text-2xl font-bold text-[#7d5141]">{totalHoje}</p>
            <CalendarPlus className="h-6 w-6 text-[#7d5141]/40" />
          </div>
        </div>
        <div className="rounded-xl border border-[#efdcd1]/20 bg-[#efdcd1]/30 p-4">
          <p className="text-[11px] font-bold uppercase tracking-widest text-aura-muted">Ocupação</p>
          <div className="mt-1 flex items-center justify-between">
            <p className="font-display text-2xl font-bold text-[#695c53]">{ocupacao}%</p>
            <TrendingUp className="h-6 w-6 text-[#695c53]/40" />
          </div>
        </div>
      </div>

      <div>
        <h3 className="mb-3 font-semibold text-aura-anthracite">Próximos atendimentos</h3>
        <div className="flex flex-col gap-3">
          {proximos.length === 0 ? (
            <p className="text-sm text-aura-muted">Nenhum agendamento restante.</p>
          ) : (
            proximos.map((ag) => {
              const prof = profissionais.find((p) => p.id === ag.profissional_id)
              return (
                <div key={ag.id} className="flex items-start gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#e9e8e7]">
                    <p className="text-[10px] font-bold text-[#7d5141]">
                      {formatTimeBR(ag.hora_inicio).slice(0, 5)}
                    </p>
                  </div>
                  <div className="flex-1 border-b border-[#efdcd1]/40 pb-3">
                    <p className="text-sm font-bold text-aura-anthracite">{ag.cliente_nome}</p>
                    <p className="text-xs text-aura-muted">
                      {getAgendamentoServicosNomes(ag, db)} · {prof?.nome ?? '—'}
                    </p>
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>
    </aside>
  )

  const layoutSearch = {
    searchPlaceholder: 'Buscar agendamento ou cliente…',
    searchValue: search,
    onSearchChange: setSearch,
  }

  const pageContent = (
    <>
      <div className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 className="font-display text-2xl font-semibold text-aura-anthracite sm:text-3xl">
            {isSecretaria ? 'Agenda do salão' : 'Agenda do salão'}
          </h1>
          <p className="mt-1 text-sm text-aura-muted">
            {isSecretaria
              ? 'Visualize e altere os agendamentos de todas as profissionais.'
              : 'Gerencie a rotina da sua equipe com precisão.'}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex rounded-lg bg-[#e9e8e7] p-1">
            {(
              [
                ['hoje', 'Hoje'],
                ['semana', 'Semana'],
                ['mes', 'Mês'],
              ] as const
            ).map(([key, label]) => (
              <button
                key={key}
                type="button"
                onClick={() => setViewMode(key)}
                className={[
                  'rounded-md px-4 py-1.5 text-sm font-medium transition-colors',
                  viewMode === key
                    ? 'bg-white font-bold text-[#7d5141] shadow-sm'
                    : 'text-aura-muted hover:text-[#7d5141]',
                ].join(' ')}
              >
                {label}
              </button>
            ))}
          </div>
          <button
            type="button"
            aria-pressed={soAguardandoConfirmacao}
            onClick={() => setSoAguardandoConfirmacao((active) => !active)}
            className={[
              'rounded-full border px-3 py-1.5 text-sm font-medium transition-colors',
              soAguardandoConfirmacao
                ? 'border-[#7d5141] bg-[#efdcd1] text-[#7d5141]'
                : 'border-[#d6c2bd] text-aura-muted hover:border-[#7d5141] hover:text-[#7d5141]',
            ].join(' ')}
          >
            Aguardando confirmação
          </button>
          <Button
            className="bg-[#7d5141] hover:bg-[#996958] shadow-lg shadow-[#7d5141]/20"
            onClick={() => openNovoAgendamento()}
          >
            <Plus className="h-4 w-4" />
            Novo agendamento
          </Button>
        </div>
      </div>

      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}
      <EarlySlotQueueInactiveBanner
        queueActive={tenant?.early_slot_queue_active}
        canReconnectWhatsApp={session?.user.role === 'DONA'}
      />

      {viewMode !== 'hoje' && (
        <div className="mb-4 rounded-xl border border-[#efdcd1]/40 bg-[#faf9f8] px-4 py-3 text-sm text-aura-muted">
          Visão <strong>{viewMode === 'semana' ? 'semanal' : 'mensal'}</strong> em breve. Exibindo o dia{' '}
          {formatDateShortBR(data)}.
        </div>
      )}

      <div className="flex flex-col gap-6 lg:flex-row">
        {/* Calendar main */}
        <section className="min-w-0 flex-1">
          {/* Mobile list */}
          <div className="space-y-3 lg:hidden">
            <div className="flex items-center justify-between gap-2">
              <Button variant="ghost" onClick={() => setData(addDaysISO(data, -1))}>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <p className="text-center text-sm font-semibold capitalize">{formatDateShortBR(data)}</p>
              <Button variant="ghost" onClick={() => setData(addDaysISO(data, 1))}>
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
            <select
              value={profMobileId}
              onChange={(e) => setProfMobile(e.target.value)}
              className="w-full rounded-lg border border-[#d6c2bd] bg-white px-3 py-2.5 text-sm"
            >
              {profissionais.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome}
                </option>
              ))}
            </select>
            {agendamentosDia
              .filter((a) => a.profissional_id === profMobileId)
              .map((ag) => {
                const st = statusDisplay(ag, data)
                const confirmation = confirmacaoClienteBadge(ag.confirmacao_cliente)
                const hint = confirmationHint(ag)
                return (
                  <div
                    key={ag.id}
                    className={['rounded-xl p-4 shadow-sm', cardClasses(st.card)].join(' ')}
                  >
                    <div className="flex justify-between gap-2">
                      <button
                        type="button"
                        className="min-w-0 flex-1 text-left"
                        onClick={() => isSecretaria && setEditAg(ag)}
                      >
                        <p className="text-xs font-bold uppercase text-[#7d5141]">
                          {appointmentEndTime(ag, db)}
                        </p>
                        <p className="font-semibold">{ag.cliente_nome}</p>
                        <p className="text-sm text-aura-muted">
                          {getAgendamentoServicosNomes(ag, db)}
                        </p>
                        <div className="mt-1 flex flex-wrap items-center gap-1" title={hint}>
                          <span className={`h-2 w-2 rounded-full ${st.dot}`} />
                          <span className="text-xs text-aura-muted">{st.label}</span>
                          <Badge variant={confirmation.variant}>{confirmation.label}</Badge>
                          {ag.cobrado_em && (
                            <span className="text-xs font-medium text-emerald-700">· Pago</span>
                          )}
                        </div>
                        {(ag.aceita_adiantar || ag.early_slot_offer) && (
                          <div className="mt-2 flex flex-wrap items-center gap-1.5">
                            <AceitaAntecipacaoBadge aceitaAdiantar={ag.aceita_adiantar} />
                            <EarlySlotRoundIndicator
                              offer={ag.early_slot_offer}
                            />
                          </div>
                        )}
                      </button>
                      <div className="flex shrink-0 flex-col gap-1">
                        {isSecretaria && !ag.cobrado_em && ag.status !== 'CANCELADO' && (
                          <button
                            type="button"
                            onClick={() => setCobrancaAg(ag)}
                            aria-label="Cobrar"
                          >
                            <Banknote className="h-4 w-4 text-[#7d5141]" />
                          </button>
                        )}
                        {isSecretaria && (
                          <button type="button" onClick={() => setEditAg(ag)} aria-label="Editar">
                            <Pencil className="h-4 w-4 text-aura-muted" />
                          </button>
                        )}
                        <button type="button" onClick={() => handleCancel(ag)} aria-label="Cancelar">
                          <XCircle className="h-4 w-4 text-aura-muted" />
                        </button>
                      </div>
                    </div>
                  </div>
                )
              })}
            {agendamentosDia.filter((a) => a.profissional_id === profMobileId).length === 0 && (
              <p className="py-6 text-center text-sm text-aura-muted">Nenhum agendamento neste dia.</p>
            )}
            {sidebarPanel}
          </div>

          {/* Desktop grid */}
          <div className="hidden overflow-hidden rounded-xl border border-[#efdcd1]/20 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)] lg:flex lg:flex-col lg:h-[calc(100vh-220px)]">
            <div
              className="sticky top-0 z-20 grid border-b border-[#efdcd1]/30 bg-[#f4f3f2]/50"
              style={{ gridTemplateColumns: `80px repeat(${profissionais.length}, 1fr)` }}
            >
              <div className="flex items-center justify-center border-r border-[#efdcd1]/20 p-4">
                <Clock className="h-5 w-5 text-aura-muted" />
              </div>
              {profissionais.map((prof) => {
                const esp = especialidades.find((e) => e.id === prof.especialidade_id)?.nome ?? 'Profissional'
                return (
                  <div
                    key={prof.id}
                    className="border-r border-[#efdcd1]/20 p-4 text-center last:border-r-0"
                  >
                    <div className="mx-auto mb-1 flex h-8 w-8 items-center justify-center rounded-full bg-[#f1dfd4] text-xs font-bold text-[#7d5141]">
                      {initials(prof.nome)}
                    </div>
                    <p className="text-sm font-semibold text-[#7d5141]">{prof.nome}</p>
                    <p className="text-[10px] font-bold uppercase tracking-widest text-aura-muted">{esp}</p>
                  </div>
                )
              })}
            </div>

            <div className="relative flex-1 overflow-y-auto">
              {nowLineTop !== null && (
                <div
                  className="pointer-events-none absolute left-0 right-0 z-10 border-t-2 border-[#7d5141]/40"
                  style={{ top: nowLineTop }}
                >
                  <span className="absolute -left-0 -top-3 rounded bg-[#7d5141] px-1 text-[10px] text-white">
                    AGORA
                  </span>
                </div>
              )}

              <div
                className="grid relative"
                style={{
                  gridTemplateColumns: `80px repeat(${profissionais.length}, 1fr)`,
                  minHeight: gridHeight,
                }}
              >
                <div className="border-r border-[#efdcd1]/20 bg-[#f4f3f2]/20">
                  {hours.slice(0, -1).map((h) => (
                    <div
                      key={h}
                      className="border-b border-[#efdcd1]/50 p-4 text-right text-sm text-aura-muted"
                      style={{ height: ROW_HEIGHT }}
                    >
                      {h}
                    </div>
                  ))}
                </div>

                {profissionais.map((prof) => {
                  const lunch = getLunchBlock(prof.id)
                  const profAgs = agendamentosDia.filter((a) => a.profissional_id === prof.id)
                  return (
                    <div
                      key={prof.id}
                      className="relative border-r border-[#efdcd1]/20 last:border-r-0"
                    >
                      {hours.slice(0, -1).map((h) => (
                        <button
                          key={h}
                          type="button"
                          onClick={() =>
                            openNovoAgendamento({ profissional_id: prof.id, hora_inicio: h })
                          }
                          className="w-full border-b border-[#efdcd1]/50 transition-colors hover:bg-[#7d5141]/5"
                          style={{ height: ROW_HEIGHT }}
                          aria-label={`Agendar com ${prof.nome} às ${h}`}
                        />
                      ))}
                      {lunch && (
                        <div
                          className="absolute left-2 right-2 flex items-center justify-between rounded-xl border-l-4 border-[#615b58] bg-[#7a7371]/10 p-3"
                          style={{ top: lunch.top, height: Math.max(lunch.height, 40) }}
                        >
                          <div>
                            <p className="text-[10px] font-bold uppercase text-aura-muted">Bloqueado</p>
                            <p className="text-xs font-bold">Intervalo almoço</p>
                          </div>
                          <Lock className="h-4 w-4 text-aura-muted/40" />
                        </div>
                      )}
                      {profAgs.map((ag) => renderAppointment(ag))}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        </section>

        <div className="hidden lg:block">
          {sidebarPanel}
        </div>
      </div>

      <button
        type="button"
        onClick={() => openNovoAgendamento()}
        className="fixed bottom-24 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-[#7d5141] text-white shadow-2xl transition-transform hover:rotate-90 lg:hidden"
        aria-label="Novo agendamento"
      >
        <Plus className="h-6 w-6" />
      </button>

      {!isSecretaria && <DonaFooter />}

      <ConfirmModal
        open={!!filaModal}
        onClose={() => setFilaModal(null)}
        onConfirm={confirmFila}
        title="Oferecer vaga na fila"
        message={
          filaModal ? `Deseja oferecer esta vaga para ${filaModal.cliente} via WhatsApp?` : ''
        }
        confirmLabel="Enviar WhatsApp"
        loading={sending}
      />

      <NovoAgendamentoModal
        open={agModalOpen}
        onClose={() => setAgModalOpen(false)}
        tenantId={tenantId}
        preset={agPreset}
        onSuccess={(msg) => {
          setSuccess(msg)
          refresh()
        }}
      />

      {isSecretaria && (
        <>
          <EditarAgendamentoProfModal
            open={!!editAg}
            agendamento={editAg}
            profId={editAg?.profissional_id ?? ''}
            tenantId={tenantId}
            allowProfissionalChange
            onCobrar={(ag) => {
              setEditAg(null)
              setCobrancaAg(ag)
            }}
            onClose={() => setEditAg(null)}
            onSuccess={(msg) => {
              setSuccess(msg)
              refresh()
              setEditAg(null)
            }}
          />
          <CobrancaAgendamentoModal
            open={!!cobrancaAg}
            agendamento={cobrancaAg}
            onClose={() => setCobrancaAg(null)}
            onSuccess={(msg) => {
              setSuccess(msg)
              refresh()
              setCobrancaAg(null)
            }}
          />
        </>
      )}
    </>
  )

  if (isSecretaria) {
    return <SecretariaLayout {...layoutSearch}>{pageContent}</SecretariaLayout>
  }

  return <DonaLayout {...layoutSearch}>{pageContent}</DonaLayout>
}
