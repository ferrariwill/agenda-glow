import { useEffect, useMemo, useState } from 'react'
import {
  ArrowLeft,
  BadgeCheck,
  Calendar,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Lock,
  Phone,
  PlusCircle,
  Shuffle,
  User,
} from 'lucide-react'
import { Link, Navigate, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { DonaLayout } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { useAuth } from '../../contexts/AuthContext'
import { enviarAlertaEncaixe, enviarConfirmacaoAgendamento } from '../../services/whatsappService'
import {
  AgendaConflitoError,
  avaliarNovoAgendamento,
  createAgendamento,
  getClienteById,
  getClienteProfileMetrics,
  getDb,
  getHorariosDisponiveis,
} from '../../utils/mockDb'
import {
  formatBRL,
  formatDateBR,
  formatMonthYearBR,
  formatPhoneBR,
  formatTimeBR,
  getCalendarGrid,
  initials,
  parseISOParts,
  todayISO,
  WEEKDAYS_SHORT_PT,
} from '../../utils/format'

const ANY_PROF = '__any__'

type SlotStatus = 'available' | 'occupied' | 'encaixe'

function getSlotsComStatus(
  profissionalId: string,
  data: string,
  duracaoMinutos: number,
): { hora: string; status: SlotStatus }[] {
  const db = getDb()
  const prof = db.profissionais.find((p) => p.id === profissionalId)
  if (!prof) return []

  const day = new Date(`${data}T12:00:00`).getDay()
  const exp = prof.expedientes.find((e) => e.dia_semana === day)
  if (!exp) return []

  const toMin = (t: string) => {
    const [h, m] = t.split(':').map(Number)
    return h * 60 + m
  }
  const toTime = (m: number) =>
    `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`

  const start = toMin(exp.horario_entrada)
  const end = toMin(exp.horario_saida)
  const lunchStart = exp.inicio_almoco ? toMin(exp.inicio_almoco) : null
  const lunchEnd = exp.fim_almoco ? toMin(exp.fim_almoco) : null

  const slots: { hora: string; status: SlotStatus }[] = []
  for (let m = start; m + duracaoMinutos <= end; m += 30) {
    if (lunchStart !== null && lunchEnd !== null) {
      const slotEnd = m + duracaoMinutos
      if (m < lunchEnd && slotEnd > lunchStart) continue
    }
    const hora = toTime(m)
    try {
      const r = avaliarNovoAgendamento(profissionalId, data, hora, duracaoMinutos)
      slots.push({ hora, status: r.status === 'EM_APROVACAO' ? 'encaixe' : 'available' })
    } catch {
      slots.push({ hora, status: 'occupied' })
    }
  }
  return slots
}

function getHorariosAnyProf(tenantId: string, data: string, duracao: number): string[] {
  const db = getDb()
  const set = new Set<string>()
  for (const p of db.profissionais.filter((x) => x.tenant_id === tenantId && x.ativo)) {
    getHorariosDisponiveis(p.id, data, duracao).forEach((h) => set.add(h))
  }
  return [...set].sort()
}

function resolveProfissional(
  tenantId: string,
  profId: string,
  data: string,
  hora: string,
  duracao: number,
): string | null {
  if (profId !== ANY_PROF) return profId
  const db = getDb()
  for (const p of db.profissionais.filter((x) => x.tenant_id === tenantId && x.ativo)) {
    try {
      const r = avaliarNovoAgendamento(p.id, data, hora, duracao)
      if (r.status === 'CONFIRMADO' || r.status === 'EM_APROVACAO') return p.id
    } catch {
      /* próximo profissional */
    }
  }
  return null
}

function StepHeader({ n, title }: { n: number; title: string }) {
  return (
    <div className="flex items-center gap-4 border-b border-[#e3e2e1] bg-[#faf9f8] p-5">
      <span className="flex h-8 w-8 items-center justify-center rounded-full bg-[#7d5141] text-sm font-bold text-white">
        {n}
      </span>
      <h4 className="font-display text-lg font-semibold text-aura-anthracite">{title}</h4>
    </div>
  )
}

export function AgendarClienteDona() {
  const { id } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenantId = session?.user.tenant_id ?? ''

  const [search, setSearch] = useState('')
  const [categoriaId, setCategoriaId] = useState<string>('all')
  const [servicoIds, setServicoIds] = useState<string[]>([])
  const [profId, setProfId] = useState('')
  const [data, setData] = useState(searchParams.get('data') ?? todayISO())
  const [hora, setHora] = useState(searchParams.get('hora') ?? '')
  const [observacoes, setObservacoes] = useState('')
  const [calYear, setCalYear] = useState(() => parseISOParts(data).year)
  const [calMonth, setCalMonth] = useState(() => parseISOParts(data).month)

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [successMsg, setSuccessMsg] = useState('')

  const db = getDb()
  const cliente = id ? getClienteById(tenantId, id) : undefined
  const metrics = cliente ? getClienteProfileMetrics(tenantId, cliente.telefone) : null
  const vip = metrics ? metrics.visitas > 2 && metrics.ltv >= 300 : false

  const categorias = db.categorias.filter((c) => c.tenant_id === tenantId)
  const servicos = useMemo(() => {
    let list = db.servicos.filter((s) => s.tenant_id === tenantId && s.ativo)
    if (categoriaId !== 'all') list = list.filter((s) => s.categoria_id === categoriaId)
    if (search.trim()) {
      const q = search.toLowerCase()
      list = list.filter((s) => s.nome.toLowerCase().includes(q))
    }
    return list
  }, [db.servicos, tenantId, categoriaId, search])

  const profissionais = db.profissionais.filter((p) => p.tenant_id === tenantId && p.ativo)
  const especialidades = db.especialidades.filter((e) => e.tenant_id === tenantId)

  const servicosSelecionados = db.servicos.filter((s) => servicoIds.includes(s.id))
  const totalDuracao = servicosSelecionados.reduce((s, srv) => s + srv.duracao_minutos, 0)
  const totalPreco = servicosSelecionados.reduce((s, srv) => s + srv.preco, 0)

  const profEfetivo = profId || profissionais[0]?.id || ANY_PROF

  const slots = useMemo(() => {
    if (!data || totalDuracao === 0) return []
    if (profEfetivo === ANY_PROF) {
      return getHorariosAnyProf(tenantId, data, totalDuracao).map((h) => ({
        hora: h,
        status: 'available' as SlotStatus,
      }))
    }
    return getSlotsComStatus(profEfetivo, data, totalDuracao)
  }, [profEfetivo, data, totalDuracao, tenantId])

  const profSelecionado = profissionais.find((p) => p.id === profEfetivo)
  const espProf = profSelecionado
    ? especialidades.find((e) => e.id === profSelecionado.especialidade_id)?.nome
    : profEfetivo === ANY_PROF
      ? 'Qualquer disponível'
      : undefined

  useEffect(() => {
    if (servicoIds.length === 0 && servicos.length > 0) {
      setServicoIds([servicos[0].id])
    }
  }, [servicos, servicoIds.length])

  useEffect(() => {
    if (!hora && slots.length > 0) {
      const first = slots.find((s) => s.status === 'available' || s.status === 'encaixe')
      if (first) setHora(first.hora)
    }
  }, [slots, hora])

  useEffect(() => {
    if (!profId && profissionais.length > 0) setProfId(profissionais[0].id)
  }, [profissionais, profId])

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

  const toggleServico = (sid: string) => {
    setServicoIds((prev) => {
      if (prev.includes(sid)) {
        const next = prev.filter((x) => x !== sid)
        return next.length > 0 ? next : prev
      }
      return [...prev, sid]
    })
  }

  const handleConfirm = async () => {
    if (!cliente) return
    setError('')
    if (servicoIds.length === 0 || !data || !hora) {
      setError('Selecione serviço, data e horário')
      return
    }
    const resolvedProf = resolveProfissional(tenantId, profEfetivo, data, hora, totalDuracao)
    if (!resolvedProf) {
      setError('Nenhum profissional disponível neste horário')
      return
    }
    setLoading(true)
    try {
      const ag = await createAgendamento({
        tenant_id: tenantId,
        profissional_id: resolvedProf,
        servico_ids: servicoIds,
        cliente_nome: cliente.nome,
        cliente_telefone: cliente.telefone,
        data,
        hora_inicio: hora,
        observacoes: observacoes.trim() || undefined,
      })
      const nomesServicos = servicosSelecionados.map((s) => s.nome).join(' + ')
      const prof = profissionais.find((p) => p.id === resolvedProf)
      if (ag.status === 'EM_APROVACAO') {
        await enviarAlertaEncaixe({
          telefone: cliente.telefone,
          clienteNome: cliente.nome,
          linkAprovacao: `${window.location.origin}/publico/aprovacao/${ag.id}`,
          minutosInvadidos: ag.minutos_invadidos ?? 1,
        })
        setSuccessMsg(
          `${cliente.nome} foi notificada via WhatsApp para aprovar o encaixe em ${formatDateBR(data)} às ${formatTimeBR(hora)}.`,
        )
      } else {
        await enviarConfirmacaoAgendamento({
          telefone: cliente.telefone,
          clienteNome: cliente.nome,
          servico: nomesServicos,
          profissional: prof?.nome ?? '',
          data,
          hora,
        })
        setSuccessMsg(
          `${cliente.nome} foi notificada via WhatsApp para ${formatDateBR(data)} às ${formatTimeBR(hora)}.`,
        )
      }
      setSuccess(true)
    } catch (err) {
      setError(err instanceof AgendaConflitoError ? err.message : err instanceof Error ? err.message : 'Erro ao agendar')
    } finally {
      setLoading(false)
    }
  }

  if (!id || !cliente) {
    return <Navigate to="/admin/clientes" replace />
  }

  return (
    <DonaLayout searchPlaceholder="Buscar serviços…" searchValue={search} onSearchChange={setSearch}>
      <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <Link
            to={`/admin/clientes/${cliente.id}`}
            className="rounded-lg p-2 text-aura-muted transition-colors hover:bg-[#f4f3f2] hover:text-[#7d5141]"
            aria-label="Voltar"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div className="h-8 w-px bg-[#d6c2bd]" />
          <div>
            <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
              Detalhes do cliente
            </p>
            <h1 className="font-display text-lg font-semibold text-[#7d5141]">Novo agendamento</h1>
          </div>
        </div>
      </header>

      {error && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      {/* Client banner */}
      <section className="mb-6 flex flex-col gap-4 rounded-xl border-l-4 border-[#7d5141] bg-white p-5 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          {cliente.foto_url ? (
            <img
              src={cliente.foto_url}
              alt={cliente.nome}
              className="h-16 w-16 rounded-full object-cover"
            />
          ) : (
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-[#f1dfd4] text-lg font-bold text-[#7d5141]">
              {initials(cliente.nome)}
            </div>
          )}
          <div>
            <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
              Agendamento para
            </p>
            <h2 className="font-display text-xl font-semibold text-[#7d5141]">{cliente.nome}</h2>
            <div className="mt-1 flex flex-wrap gap-4">
              <span className="flex items-center gap-1 text-sm text-aura-muted">
                <Phone className="h-4 w-4" />
                {formatPhoneBR(cliente.telefone)}
              </span>
              {vip && (
                <span className="flex items-center gap-1 text-sm text-[#1e4620]">
                  <BadgeCheck className="h-4 w-4" />
                  Cliente VIP
                </span>
              )}
            </div>
          </div>
        </div>
        <span className="self-start rounded-full bg-[#efdcd1] px-4 py-2 text-[10px] font-bold uppercase tracking-wider text-[#6d6057] sm:self-center">
          Status: Em aberto
        </span>
      </section>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
        {/* Form column */}
        <div className="space-y-6 lg:col-span-8">
          {/* Step 1 — Services */}
          <div className="overflow-hidden rounded-xl bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
            <div className="flex flex-col gap-3 border-b border-[#e3e2e1] bg-[#faf9f8] p-5 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-4">
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-[#7d5141] text-sm font-bold text-white">
                  1
                </span>
                <h4 className="font-display text-lg font-semibold">Seleção de serviço</h4>
              </div>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => setCategoriaId('all')}
                  className={[
                    'rounded-lg border px-3 py-1 text-sm font-medium transition-colors',
                    categoriaId === 'all'
                      ? 'border-[#7d5141]/20 bg-[#996958]/10 text-[#7d5141]'
                      : 'border-[#d6c2bd] text-aura-muted hover:border-[#7d5141]',
                  ].join(' ')}
                >
                  Todos
                </button>
                {categorias.map((cat) => (
                  <button
                    key={cat.id}
                    type="button"
                    onClick={() => setCategoriaId(cat.id)}
                    className={[
                      'rounded-lg border px-3 py-1 text-sm font-medium transition-colors',
                      categoriaId === cat.id
                        ? 'border-[#7d5141]/20 bg-[#996958]/10 text-[#7d5141]'
                        : 'border-[#d6c2bd] text-aura-muted hover:border-[#7d5141]',
                    ].join(' ')}
                  >
                    {cat.nome}
                  </button>
                ))}
              </div>
            </div>
            <div className="grid grid-cols-1 gap-3 p-5 sm:grid-cols-2">
              {servicos.length === 0 ? (
                <p className="col-span-2 py-6 text-center text-sm text-aura-muted">
                  Nenhum serviço encontrado.
                </p>
              ) : (
                servicos.map((s) => {
                  const selected = servicoIds.includes(s.id)
                  return (
                    <button
                      key={s.id}
                      type="button"
                      onClick={() => toggleServico(s.id)}
                      className={[
                        'flex items-center justify-between rounded-lg border p-4 text-left transition-all',
                        selected
                          ? 'border-[#7d5141] bg-[#efdcd1]/20'
                          : 'border-[#d6c2bd] hover:border-[#7d5141]',
                      ].join(' ')}
                    >
                      <div>
                        <p className="font-semibold text-aura-anthracite">{s.nome}</p>
                        <p className="text-sm text-aura-muted">
                          {s.duracao_minutos} min · {formatBRL(s.preco)}
                        </p>
                      </div>
                      {selected ? (
                        <CheckCircle2 className="h-5 w-5 shrink-0 fill-[#7d5141] text-white" />
                      ) : (
                        <PlusCircle className="h-5 w-5 shrink-0 text-[#d6c2bd]" />
                      )}
                    </button>
                  )
                })
              )}
            </div>
          </div>

          {/* Step 2 — Professional */}
          <div className="overflow-hidden rounded-xl bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
            <StepHeader n={2} title="Escolha o profissional" />
            <div className="flex gap-6 overflow-x-auto p-5 pb-6">
              {profissionais.map((p) => {
                const selected = profEfetivo === p.id
                const esp = especialidades.find((e) => e.id === p.especialidade_id)?.nome ?? 'Profissional'
                return (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => setProfId(p.id)}
                    className={[
                      'w-36 shrink-0 text-center transition-opacity',
                      selected ? 'opacity-100' : 'opacity-60 hover:opacity-100',
                    ].join(' ')}
                  >
                    <div
                      className={[
                        'relative mx-auto flex h-24 w-24 items-center justify-center rounded-full text-xl font-bold',
                        selected
                          ? 'bg-[#f1dfd4] text-[#7d5141] ring-2 ring-[#7d5141]'
                          : 'bg-[#f4f3f2] text-aura-muted ring-1 ring-[#d6c2bd]',
                      ].join(' ')}
                    >
                      {initials(p.nome)}
                      {selected && (
                        <span className="absolute bottom-1 right-1 h-4 w-4 rounded-full border-2 border-white bg-emerald-500" />
                      )}
                    </div>
                    <p className="mt-2 font-bold text-[#7d5141]">{p.nome}</p>
                    <p className="text-xs text-aura-muted">{esp}</p>
                  </button>
                )
              })}
              <button
                type="button"
                onClick={() => setProfId(ANY_PROF)}
                className={[
                  'flex w-36 shrink-0 flex-col items-center justify-center gap-2 transition-opacity',
                  profEfetivo === ANY_PROF ? 'opacity-100' : 'opacity-60 hover:opacity-100',
                ].join(' ')}
              >
                <div className="flex h-24 w-24 items-center justify-center rounded-full border-2 border-dashed border-[#d6c2bd] bg-[#eeeeed] hover:border-[#7d5141]">
                  <Shuffle className="h-8 w-8 text-aura-muted" />
                </div>
                <p className="font-bold text-aura-muted">Qualquer um</p>
              </button>
            </div>
          </div>

          {/* Step 3 — Date & time */}
          <div className="overflow-hidden rounded-xl bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
            <StepHeader n={3} title="Data e horário" />
            <div className="grid grid-cols-1 gap-8 p-5 md:grid-cols-2">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <p className="font-bold capitalize">{formatMonthYearBR(`${calYear}-${String(calMonth).padStart(2, '0')}`)}</p>
                  <div className="flex gap-1">
                    <button
                      type="button"
                      onClick={() => shiftMonth(-1)}
                      className="rounded p-1 text-aura-muted hover:text-[#7d5141]"
                    >
                      <ChevronLeft className="h-5 w-5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => shiftMonth(1)}
                      className="rounded p-1 text-aura-muted hover:text-[#7d5141]"
                    >
                      <ChevronRight className="h-5 w-5" />
                    </button>
                  </div>
                </div>
                <div className="grid grid-cols-7 gap-1 text-center">
                  {WEEKDAYS_SHORT_PT.map((d) => (
                    <span key={d} className="py-1 text-[10px] font-bold uppercase text-aura-muted">
                      {d}
                    </span>
                  ))}
                  {calendarDays.map((cell, i) => {
                    if (!cell.iso) return <span key={`e-${i}`} />
                    const isPast = cell.iso < todayISO()
                    const selected = cell.iso === data
                    return (
                      <button
                        key={cell.iso}
                        type="button"
                        disabled={isPast}
                        onClick={() => setData(cell.iso!)}
                        className={[
                          'rounded-lg py-2 text-sm transition-colors',
                          selected
                            ? 'bg-[#7d5141] font-bold text-white shadow-md'
                            : isPast
                              ? 'cursor-not-allowed text-aura-muted/40'
                              : 'hover:bg-[#efdcd1]',
                        ].join(' ')}
                      >
                        {cell.day}
                      </button>
                    )
                  })}
                </div>
              </div>

              <div className="space-y-3">
                <p className="font-bold">Horários disponíveis</p>
                {totalDuracao === 0 ? (
                  <p className="text-sm text-aura-muted">Selecione um serviço primeiro.</p>
                ) : slots.length === 0 ? (
                  <p className="text-sm text-aura-muted">Sem horários neste dia.</p>
                ) : (
                  <div className="grid max-h-48 grid-cols-3 gap-2 overflow-y-auto pr-1">
                    {slots.map(({ hora: h, status }) => {
                      const selected = hora === h
                      const disabled = status === 'occupied'
                      return (
                        <button
                          key={h}
                          type="button"
                          disabled={disabled}
                          onClick={() => setHora(h)}
                          className={[
                            'rounded-lg py-2 text-center text-sm transition-all',
                            disabled
                              ? 'cursor-not-allowed bg-[#eeeeed] text-aura-muted/40 line-through'
                              : selected
                                ? 'bg-[#7d5141] font-bold text-white shadow-md'
                                : 'border border-[#d6c2bd] bg-white hover:border-[#7d5141]',
                          ].join(' ')}
                        >
                          {formatTimeBR(h)}
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Summary sidebar */}
        <div className="lg:col-span-4">
          <div className="sticky top-24 overflow-hidden rounded-xl border border-[#7d5141]/10 bg-white shadow-[0px_4px_20px_rgba(183,132,114,0.08)]">
            <div className="bg-[#7d5141] p-5 text-white">
              <h4 className="font-display text-xl font-semibold">Resumo</h4>
              <p className="text-sm opacity-80">Verifique os detalhes antes de confirmar</p>
            </div>
            <div className="space-y-4 p-5">
              <div className="flex justify-between gap-3">
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                    Serviço selecionado
                  </p>
                  <p className="font-semibold">
                    {servicosSelecionados.map((s) => s.nome).join(' + ') || '—'}
                  </p>
                </div>
                <p className="shrink-0 font-bold text-[#7d5141]">
                  {totalPreco > 0 ? formatBRL(totalPreco) : '—'}
                </p>
              </div>

              <div>
                <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                  Profissional
                </p>
                <div className="mt-1 flex items-center gap-2">
                  <User className="h-4 w-4 text-[#7d5141]" />
                  <p className="font-semibold">
                    {profSelecionado?.nome ?? (profEfetivo === ANY_PROF ? 'Qualquer disponível' : '—')}
                  </p>
                </div>
                {espProf && <p className="text-xs text-aura-muted">{espProf}</p>}
              </div>

              <div className="flex justify-between border-b border-[#e3e2e1] pb-4">
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                    Data e hora
                  </p>
                  <div className="mt-1 flex items-center gap-2">
                    <Calendar className="h-4 w-4 text-[#7d5141]" />
                    <p className="font-semibold">
                      {data && hora ? `${formatDateBR(data)} às ${formatTimeBR(hora)}` : '—'}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                    Duração
                  </p>
                  <p className="font-semibold">{totalDuracao > 0 ? `${totalDuracao} min` : '—'}</p>
                </div>
              </div>

              <div>
                <label className="text-[10px] font-bold uppercase tracking-wider text-aura-muted">
                  Observações profissionais
                </label>
                <textarea
                  value={observacoes}
                  onChange={(e) => setObservacoes(e.target.value)}
                  rows={3}
                  placeholder="Ex: Cliente tem couro cabeludo sensível…"
                  className="mt-2 w-full rounded-lg border-none bg-[#eeeeed] p-3 text-sm italic focus:ring-1 focus:ring-[#7d5141]"
                />
              </div>

              <div className="space-y-2 border-t border-[#e3e2e1] pt-4">
                <div className="flex justify-between text-sm">
                  <span>Subtotal</span>
                  <span>{formatBRL(totalPreco)}</span>
                </div>
                <div className="flex justify-between border-t border-[#e3e2e1] pt-3">
                  <span className="font-display text-lg font-semibold">Total</span>
                  <span className="font-display text-2xl font-bold text-[#7d5141]">
                    {formatBRL(totalPreco)}
                  </span>
                </div>
              </div>

              <div className="flex flex-col gap-2 pt-2">
                <Button
                  className="w-full bg-[#7d5141] hover:bg-[#996958]"
                  loading={loading}
                  onClick={handleConfirm}
                >
                  Confirmar agendamento
                </Button>
                <Button
                  variant="secondary"
                  className="w-full"
                  onClick={() => navigate(`/admin/clientes/${cliente.id}`)}
                >
                  Cancelar
                </Button>
              </div>

              <div className="flex items-center justify-center gap-2 pt-2 text-aura-muted">
                <Lock className="h-3.5 w-3.5" />
                <span className="text-[11px] uppercase tracking-wider">Agendamento seguro Aura</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Success overlay */}
      {success && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-[#2f3130]/40 p-4 backdrop-blur-sm">
          <div className="w-full max-w-sm scale-100 space-y-6 rounded-2xl bg-white p-8 text-center opacity-100 shadow-2xl transition-all">
            <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-[#7d5141]/10 text-[#7d5141]">
              <CheckCircle2 className="h-12 w-12" />
            </div>
            <div>
              <h3 className="font-display text-xl font-semibold">Agendamento realizado!</h3>
              <p className="mt-2 text-sm text-aura-muted">{successMsg}</p>
            </div>
            <Button
              className="w-full bg-[#7d5141] hover:bg-[#996958]"
              onClick={() => navigate(`/admin/clientes/${cliente.id}`)}
            >
              Voltar ao perfil
            </Button>
            <Button variant="secondary" className="w-full" onClick={() => navigate('/admin/calendario')}>
              Ver agenda
            </Button>
          </div>
        </div>
      )}
    </DonaLayout>
  )
}
