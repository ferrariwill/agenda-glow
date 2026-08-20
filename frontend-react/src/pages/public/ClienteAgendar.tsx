import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Clock, Sparkles, WifiOff } from 'lucide-react'
import { BookingStickyBar, BOOKING_STICKY_BAR_SPACE } from '../../components/public/BookingStickyBar'
import { BookingStepper } from '../../components/public/BookingStepper'
import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'
import { EarlySlotPreferenceToggle } from '../../components/public/EarlySlotPreferenceToggle'
import { SlotChip } from '../../components/public/SlotChip'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { DateInput } from '../../components/ui/DateInput'
import { useAuth } from '../../contexts/AuthContext'
import { IS_MOCK } from '../../lib/config'
import { ApiError } from '../../lib/api'
import { useStoreDb } from '../../data/store'
import { fetchPublicSlots, syncPublicCatalog } from '../../data/sync'
import { enviarConfirmacaoAgendamento } from '../../services/whatsappService'
import {
  AgendaConflitoError,
  createAgendamento,
  ensureClienteNoTenant,
  getHorariosDisponiveis,
  getServicosDoProfissional,
  getTenantBySlug,
} from '../../utils/mockDb'
import { formatBRL, formatDateTimeBR, formatTimeBR, initials, todayISO } from '../../utils/format'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white/80 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md'

type SlotsStatus = 'idle' | 'loading' | 'empty' | 'error' | 'ready'
type Step = 1 | 2 | 3 | 4

const CATALOG_STALE_CODES = new Set([
  'professional_not_found',
  'service_not_found',
  'invalid_additionals',
])

function slotsErrorMessage(err: unknown): { message: string; catalogStale: boolean } {
  if (err instanceof ApiError) {
    if (err.code && CATALOG_STALE_CODES.has(err.code)) {
      return {
        message: 'Catálogo desatualizado — recarregue e tente de novo.',
        catalogStale: true,
      }
    }
    if (err.code === 'missing_query_params' || err.code === 'invalid_date') {
      return {
        message: 'Não foi possível montar a busca de horários. Ajuste data, profissional e serviço.',
        catalogStale: false,
      }
    }
    if (err.code === 'internal_error' || err.status >= 500) {
      return {
        message: 'O servidor não conseguiu carregar os horários agora. Tente de novo em instantes.',
        catalogStale: false,
      }
    }
  }
  return {
    message:
      'Não foi possível carregar os horários. Sua seleção de serviço, profissional e data foi mantida.',
    catalogStale: false,
  }
}

function useOnlineStatus() {
  const [online, setOnline] = useState(
    typeof navigator === 'undefined' ? true : navigator.onLine,
  )
  useEffect(() => {
    const on = () => setOnline(true)
    const off = () => setOnline(false)
    window.addEventListener('online', on)
    window.addEventListener('offline', off)
    return () => {
      window.removeEventListener('online', on)
      window.removeEventListener('offline', off)
    }
  }, [])
  return online
}

function SlotsSkeleton() {
  return (
    <div
      className="grid max-h-[min(40vh,280px)] grid-cols-3 gap-2 overflow-y-auto sm:grid-cols-4"
      aria-hidden
      aria-busy="true"
    >
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="min-h-touch-min animate-pulse rounded-lg bg-[#efdcd1]/50"
        />
      ))}
    </div>
  )
}

export function ClienteAgendar() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const { session } = useAuth()
  const tenant = slug ? getTenantBySlug(slug) : undefined
  const db = useStoreDb()
  const online = useOnlineStatus()

  const [step, setStep] = useState<Step>(1)
  const [profId, setProfId] = useState('')
  const [servicoIds, setServicoIds] = useState<string[]>([])
  const [adicionalIds, setAdicionalIds] = useState<string[]>([])
  const [data, setData] = useState(todayISO())
  const [hora, setHora] = useState('')
  const [aceitaAdiantar, setAceitaAdiantar] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const [managementUrl, setManagementUrl] = useState('')
  const [horarios, setHorarios] = useState<string[]>([])
  const [slotsStatus, setSlotsStatus] = useState<SlotsStatus>('idle')
  const [slotsErrorMsg, setSlotsErrorMsg] = useState('')
  const [slotsCatalogStale, setSlotsCatalogStale] = useState(false)
  const [slotsRetry, setSlotsRetry] = useState(0)
  const [catalogRefreshing, setCatalogRefreshing] = useState(false)

  const profissionais = tenant
    ? db.profissionais.filter((p) => p.tenant_id === tenant.id && p.ativo)
    : []
  const allServicos = tenant ? db.servicos.filter((s) => s.tenant_id === tenant.id) : []

  const servicosProf = useMemo(
    () => (profId && tenant ? getServicosDoProfissional(tenant.id, profId, allServicos) : []),
    [profId, tenant?.id, allServicos],
  )

  const procedimentoId = servicoIds[0] ?? ''

  const adicionaisDoServico = useMemo(() => {
    if (!procedimentoId) return []
    return db.adicionais.filter((a) => a.servico_id === procedimentoId)
  }, [db.adicionais, procedimentoId])

  const adicionaisSelecionados = useMemo(
    () => adicionaisDoServico.filter((a) => adicionalIds.includes(a.id)),
    [adicionaisDoServico, adicionalIds],
  )

  useEffect(() => {
    if (!profId && profissionais[0]) {
      setProfId(profissionais[0].id)
    }
  }, [profissionais, profId])

  useEffect(() => {
    if (servicosProf.length === 0) {
      setServicoIds([])
      return
    }
    setServicoIds((prev) => {
      if (!IS_MOCK) {
        const stillValid = prev[0] && servicosProf.some((s) => s.id === prev[0])
        return stillValid ? [prev[0]] : [servicosProf[0].id]
      }
      const valid = prev.filter((id) => servicosProf.some((s) => s.id === id))
      if (valid.length > 0) return valid
      return [servicosProf[0].id]
    })
  }, [servicosProf])

  useEffect(() => {
    setAdicionalIds((prev) => prev.filter((id) => adicionaisDoServico.some((a) => a.id === id)))
  }, [adicionaisDoServico])

  const servicosSelecionados = useMemo(
    () => servicosProf.filter((s) => servicoIds.includes(s.id)),
    [servicosProf, servicoIds],
  )
  const totalDuracao =
    servicosSelecionados.reduce((s, x) => s + x.duracao_minutos, 0) +
    adicionaisSelecionados.reduce((s, x) => s + x.duracao_minutos, 0)
  const totalPreco =
    servicosSelecionados.reduce((s, x) => s + x.preco, 0) +
    adicionaisSelecionados.reduce((s, x) => s + x.preco, 0)

  useEffect(() => {
    if (!profId || servicoIds.length === 0 || totalDuracao === 0) {
      setHorarios([])
      setSlotsStatus('idle')
      setSlotsErrorMsg('')
      setSlotsCatalogStale(false)
      return
    }
    if (IS_MOCK) {
      const slots = getHorariosDisponiveis(profId, data, totalDuracao)
      setHorarios(slots)
      setSlotsStatus(slots.length === 0 ? 'empty' : 'ready')
      setSlotsErrorMsg('')
      setSlotsCatalogStale(false)
      return
    }
    if (!slug || !procedimentoId) {
      setHorarios([])
      setSlotsStatus('idle')
      return
    }
    let cancelled = false
    setSlotsStatus('loading')
    setHorarios([])
    setSlotsErrorMsg('')
    setSlotsCatalogStale(false)
    fetchPublicSlots(slug, profId, data, procedimentoId, adicionalIds)
      .then((slots) => {
        if (cancelled) return
        setHorarios(slots)
        setSlotsStatus(slots.length === 0 ? 'empty' : 'ready')
      })
      .catch((err) => {
        if (cancelled) return
        const mapped = slotsErrorMessage(err)
        setHorarios([])
        setSlotsStatus('error')
        setSlotsErrorMsg(mapped.message)
        setSlotsCatalogStale(mapped.catalogStale)
      })
    return () => {
      cancelled = true
    }
  }, [
    profId,
    data,
    totalDuracao,
    servicoIds,
    adicionalIds,
    procedimentoId,
    slug,
    db.agendamentos,
    slotsRetry,
  ])

  const maxReachable: Step = (() => {
    if (!hora) {
      if (servicoIds.length === 0 || !profId) return profId ? 2 : 1
      return 3
    }
    return 4
  })()

  const selectServico = (id: string) => {
    setServicoIds([id])
    setAdicionalIds([])
    setHora('')
  }

  const toggleServico = (id: string) => {
    setServicoIds((prev) => {
      if (prev.includes(id)) {
        const next = prev.filter((x) => x !== id)
        return next.length > 0 ? next : prev
      }
      return [...prev, id]
    })
    setHora('')
  }

  const toggleAdicional = (id: string) => {
    setAdicionalIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    )
    setHora('')
  }

  const selectProf = (id: string) => {
    setProfId(id)
    setHora('')
  }

  const refreshSlots = () => {
    setHora('')
    setSlotsRetry((n) => n + 1)
  }

  const retrySlotsWithCatalog = async () => {
    if (!slug || IS_MOCK) {
      refreshSlots()
      return
    }
    setCatalogRefreshing(true)
    try {
      await syncPublicCatalog(slug)
    } catch {
      /* UI de slots permanece em error; usuário pode tentar de novo */
    } finally {
      setCatalogRefreshing(false)
      refreshSlots()
    }
  }

  const confirmar = async () => {
    if (!tenant || !session?.user || !profId || !hora || servicoIds.length === 0) return
    if (!online) {
      setError('Você está offline. Conecte-se para confirmar o agendamento.')
      return
    }
    setError('')
    setLoading(true)
    try {
      ensureClienteNoTenant(tenant.id, {
        nome: session.user.nome,
        telefone: session.user.telefone ?? '',
        email: session.user.email,
      })
      const ag = await createAgendamento(
        {
          tenant_id: tenant.id,
          profissional_id: profId,
          servico_ids: IS_MOCK ? servicoIds : [procedimentoId],
          adicional_ids: adicionalIds,
          cliente_nome: session.user.nome,
          cliente_telefone: session.user.telefone ?? '',
          data,
          hora_inicio: hora,
          aceita_adiantar: aceitaAdiantar,
        },
        { publicSlug: slug },
      )

      const prof = profissionais.find((p) => p.id === profId)
      await enviarConfirmacaoAgendamento({
        telefone: session.user.telefone ?? '',
        clienteNome: session.user.nome,
        servico: [
          ...servicosSelecionados.map((s) => s.nome),
          ...adicionaisSelecionados.map((a) => a.nome),
        ].join(' + '),
        profissional: prof?.nome ?? '',
        data,
        hora,
      })

      if (ag.status === 'EM_APROVACAO') {
        setError('Horário sujeito a aprovação — você receberá um link no WhatsApp.')
      }
      setManagementUrl(ag.management_url ?? '')
      setDone(true)
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate(`/${slug}/login`, {
          replace: true,
          state: { from: `/${slug}/agendar` },
        })
        return
      }
      if (
        err instanceof AgendaConflitoError ||
        (err instanceof ApiError && err.status === 409)
      ) {
        setError(
          'Esse horário acabou de ser ocupado. Escolha outro horário disponível.',
        )
        setHora('')
        setStep(3)
        refreshSlots()
      } else if (err instanceof ApiError && (err.status === 429 || err.status >= 500)) {
        setError('Não foi possível confirmar agora. Tente de novo — sua seleção foi mantida.')
      } else {
        setError(err instanceof Error ? err.message : 'Erro ao agendar')
      }
    } finally {
      setLoading(false)
    }
  }

  if (!tenant) {
    return <p className="p-8 text-center text-[#514440]">Salão não encontrado.</p>
  }

  if (done) {
    return (
      <ClienteSalaoLayout tenant={tenant} backTo={`/${slug}`}>
        <div className={[GLASS, 'p-6 text-center'].join(' ')} aria-live="polite">
          <Sparkles className="mx-auto mb-3 h-8 w-8 text-[#7d5141]" />
          <h1 className="font-display text-2xl font-semibold text-[#7d5141]">
            Agendamento confirmado!
          </h1>
          <p className="mt-2 text-sm text-[#514440]">
            {formatDateTimeBR(data, hora)} ·{' '}
            {[
              ...servicosSelecionados.map((s) => s.nome),
              ...adicionaisSelecionados.map((a) => a.nome),
            ].join(' + ')}
          </p>
          {aceitaAdiantar && (
            <p className="mt-2 text-sm text-[#514440]">
              {tenant?.early_slot_notifications_available === false
                ? 'Preferência de antecipação salva. Os avisos por WhatsApp começarão quando o salão reativar o canal.'
                : 'Se surgir um horário mais cedo com este profissional, avisamos pelo WhatsApp. A oferta vale por 5 minutos e seu horário atual só muda se você aceitar.'}
            </p>
          )}
          <p className="mt-1 text-sm text-[#514440]">
            {tenant?.early_slot_notifications_available === false
              ? 'A confirmação por WhatsApp depende do canal do salão estar ativo.'
              : 'Enviamos a confirmação por WhatsApp.'}
          </p>
          {managementUrl && (
            <a
              href={managementUrl}
              className="mt-4 inline-flex min-h-touch-min items-center text-sm font-semibold text-[#7d5141] underline underline-offset-4"
            >
              Gerenciar / cancelar meu horário
            </a>
          )}
          <div className="mt-5 flex flex-col gap-2">
            <Button
              onClick={() => navigate(`/${slug}/conta`)}
              className="bg-[#7d5141] hover:bg-[#996958]"
            >
              Ver meus agendamentos
            </Button>
            <Button variant="secondary" onClick={() => navigate(`/${slug}`)}>
              Voltar ao salão
            </Button>
          </div>
        </div>
      </ClienteSalaoLayout>
    )
  }

  const showSticky = step === 4 && Boolean(hora)
  const profNome = profissionais.find((p) => p.id === profId)?.nome

  return (
    <ClienteSalaoLayout
      tenant={tenant}
      backTo={`/${slug}`}
      contentClassName={showSticky ? BOOKING_STICKY_BAR_SPACE : ''}
    >
      <h1 className="mb-1 font-display text-2xl font-semibold text-[#1a1c1c]">
        Agendar horário
      </h1>
      <p className="mb-4 text-sm text-[#514440]">
        {IS_MOCK
          ? 'Escolha a profissional, os serviços e o melhor horário para você.'
          : 'Escolha a profissional, o serviço e o melhor horário para você.'}
      </p>

      <BookingStepper
        current={step}
        maxReachable={maxReachable}
        onGoTo={(s) => setStep(s)}
      />

      {!online && (
        <Alert variant="warning" className="mb-4" role="status">
          <span className="inline-flex items-center gap-2">
            <WifiOff className="h-4 w-4 shrink-0" aria-hidden />
            Sem conexão. Você pode montar o agendamento, mas a confirmação só funciona online.
          </span>
        </Alert>
      )}

      {error && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <div className="space-y-4" aria-live="polite">
        {step === 1 && (
          <section className={GLASS + ' p-4'}>
            <h2 className="mb-3 text-sm font-bold uppercase tracking-widest text-[#514440]">
              1 · Profissional
            </h2>
            <div className="space-y-3">
              {profissionais.map((p) => {
                const esp = db.especialidades.find((e) => e.id === p.especialidade_id)
                const selected = p.id === profId
                return (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => selectProf(p.id)}
                    aria-pressed={selected}
                    className={[
                      'flex min-h-touch-min w-full touch-manipulation items-center gap-3 rounded-xl border px-4 py-3 text-left transition-colors',
                      selected
                        ? 'border-[#7d5141] bg-[#efdcd1]/20'
                        : 'border-[#efdcd1]/40 bg-[#faf9f8]/50',
                    ].join(' ')}
                  >
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[#f1dfd4] text-xs font-bold text-[#7d5141]">
                      {initials(p.nome)}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="font-semibold text-[#1a1c1c]">{p.nome}</p>
                      <p className="text-xs text-[#514440]">{esp?.nome}</p>
                    </div>
                    <span
                      className={[
                        'h-4 w-4 shrink-0 rounded-full border-2',
                        selected ? 'border-[#7d5141] bg-[#7d5141]' : 'border-[#d6c2bd]',
                      ].join(' ')}
                      aria-hidden
                    />
                  </button>
                )
              })}
            </div>
            <Button
              fullWidth
              className="mt-4 bg-[#7d5141] hover:bg-[#996958]"
              disabled={!profId}
              onClick={() => setStep(2)}
            >
              Continuar
            </Button>
          </section>
        )}

        {step === 2 && (
          <section className={GLASS + ' p-4'}>
            <h2 className="mb-3 text-sm font-bold uppercase tracking-widest text-[#514440]">
              2 · {IS_MOCK ? 'Serviços' : 'Serviço'}
            </h2>
            <p className="mb-2 text-xs text-[#514440]">Com {profNome}</p>
            {!IS_MOCK && (
              <p className="mb-3 text-xs text-[#514440]">
                Selecione um serviço principal. Opcionalmente, marque adicionais do catálogo.
              </p>
            )}
            {servicosProf.length === 0 ? (
              <p className="text-sm text-[#514440]">
                Nenhum serviço online para esta profissional.
              </p>
            ) : (
              <div className="space-y-1">
                {servicosProf.map((s) => {
                  const selected = IS_MOCK
                    ? servicoIds.includes(s.id)
                    : procedimentoId === s.id
                  return (
                    <label
                      key={s.id}
                      className={[
                        'flex min-h-touch-min cursor-pointer touch-manipulation items-center gap-3 rounded-lg px-2 py-2 text-sm',
                        selected ? 'bg-[#efdcd1]/40' : 'hover:bg-white/60',
                      ].join(' ')}
                    >
                      <input
                        type={IS_MOCK ? 'checkbox' : 'radio'}
                        name="servico-publico"
                        checked={selected}
                        onChange={() => (IS_MOCK ? toggleServico(s.id) : selectServico(s.id))}
                        className="h-5 w-5 rounded border-[#d6c2bd] text-[#7d5141]"
                      />
                      <span className="flex-1 font-medium text-[#1a1c1c]">{s.nome}</span>
                      <span className="text-xs text-[#514440]">
                        {s.duracao_minutos} min · {formatBRL(s.preco)}
                      </span>
                    </label>
                  )
                })}
              </div>
            )}
            {!IS_MOCK && adicionaisDoServico.length > 0 && (
              <div className="mt-4 space-y-1 border-t border-[#efdcd1]/50 pt-3">
                <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-[#514440]">
                  Adicionais
                </p>
                {adicionaisDoServico.map((a) => {
                  const checked = adicionalIds.includes(a.id)
                  return (
                    <label
                      key={a.id}
                      className={[
                        'flex min-h-touch-min cursor-pointer touch-manipulation items-center gap-3 rounded-lg px-2 py-2 text-sm',
                        checked ? 'bg-[#efdcd1]/40' : 'hover:bg-white/60',
                      ].join(' ')}
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleAdicional(a.id)}
                        className="h-5 w-5 rounded border-[#d6c2bd] text-[#7d5141]"
                      />
                      <span className="flex-1 font-medium text-[#1a1c1c]">{a.nome}</span>
                      <span className="text-xs text-[#514440]">
                        +{a.duracao_minutos} min · {formatBRL(a.preco)}
                      </span>
                    </label>
                  )
                })}
              </div>
            )}
            {servicosSelecionados.length > 0 && (
              <p className="mt-3 text-sm font-medium text-[#7d5141]">
                Total: {totalDuracao} min · {formatBRL(totalPreco)}
              </p>
            )}
            <div className="mt-4 flex gap-2">
              <Button variant="secondary" fullWidth onClick={() => setStep(1)}>
                Voltar
              </Button>
              <Button
                fullWidth
                className="bg-[#7d5141] hover:bg-[#996958]"
                disabled={servicoIds.length === 0}
                onClick={() => setStep(3)}
              >
                Continuar
              </Button>
            </div>
          </section>
        )}

        {step === 3 && (
          <>
            <section className={GLASS + ' p-4'}>
              <h2 className="mb-3 text-sm font-bold uppercase tracking-widest text-[#514440]">
                3 · Data
              </h2>
              <DateInput
                label=""
                value={data}
                onChange={(d) => {
                  setData(d)
                  setHora('')
                }}
                minDate={todayISO()}
              />
            </section>

            <section className={GLASS + ' p-4'}>
              <div className="mb-3 flex items-center gap-2">
                <Clock className="h-4 w-4 text-[#7d5141]" aria-hidden />
                <p className="text-sm font-bold uppercase tracking-widest text-[#514440]">
                  Horários disponíveis
                </p>
              </div>

              {slotsStatus === 'loading' && (
                <>
                  <p className="sr-only">Carregando horários</p>
                  <SlotsSkeleton />
                </>
              )}

              {slotsStatus === 'error' && (
                <div className="space-y-3">
                  <p className="text-sm text-[#514440]">
                    {slotsErrorMsg ||
                      'Não foi possível carregar os horários. Sua seleção de serviço, profissional e data foi mantida.'}
                  </p>
                  <Button
                    variant="secondary"
                    fullWidth
                    loading={catalogRefreshing}
                    onClick={slotsCatalogStale ? retrySlotsWithCatalog : refreshSlots}
                  >
                    {slotsCatalogStale ? 'Recarregar catálogo e horários' : 'Tentar horários de novo'}
                  </Button>
                </div>
              )}

              {slotsStatus === 'empty' && (
                <div className="space-y-3">
                  <p className="text-sm text-[#514440]">Nenhum horário livre nesta data.</p>
                  <p className="text-xs text-[#514440]">
                    Escolha outra data acima ou volte e ajuste o serviço.
                  </p>
                </div>
              )}

              {slotsStatus === 'ready' && (
                <div
                  role="listbox"
                  aria-label="Horários disponíveis"
                  className="grid max-h-[min(40vh,280px)] grid-cols-3 gap-2 overflow-y-auto overscroll-contain sm:grid-cols-4"
                >
                  {horarios.map((slot) => (
                    <SlotChip
                      key={slot}
                      value={slot}
                      label={formatTimeBR(slot)}
                      selected={hora === slot}
                      onSelect={(v) => {
                        setHora(v)
                        setStep(4)
                      }}
                    />
                  ))}
                </div>
              )}
            </section>

            <Button variant="secondary" fullWidth onClick={() => setStep(2)}>
              Voltar
            </Button>
          </>
        )}

        {step === 4 && hora && (
          <>
            <section className={GLASS + ' p-4'}>
              <h2 className="mb-3 text-sm font-bold uppercase tracking-widest text-[#514440]">
                4 · Confirmar
              </h2>
              <dl className="space-y-2 text-sm text-[#514440]">
                <div className="flex justify-between gap-2">
                  <dt>Quando</dt>
                  <dd className="font-medium text-[#1a1c1c]">{formatDateTimeBR(data, hora)}</dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt>Profissional</dt>
                  <dd className="font-medium text-[#1a1c1c]">{profNome}</dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt>{IS_MOCK ? 'Serviços' : 'Serviço'}</dt>
                  <dd className="text-right font-medium text-[#1a1c1c]">
                    {[
                      ...servicosSelecionados.map((s) => s.nome),
                      ...adicionaisSelecionados.map((a) => a.nome),
                    ].join(' + ')}
                  </dd>
                </div>
                <div className="flex justify-between gap-2">
                  <dt>Total</dt>
                  <dd className="font-medium text-[#7d5141]">
                    {totalDuracao} min · {formatBRL(totalPreco)}
                  </dd>
                </div>
              </dl>
            </section>

            <section className={GLASS + ' p-4'}>
              <EarlySlotPreferenceToggle
                checked={aceitaAdiantar}
                onChange={setAceitaAdiantar}
                profissionalNome={profNome}
                notificationsAvailable={tenant?.early_slot_notifications_available}
              />
            </section>

            <Button variant="secondary" fullWidth onClick={() => setStep(3)}>
              Alterar horário
            </Button>

            <p className="pb-2 text-center text-xs text-[#514440]">
              {session?.user.nome} · {session?.user.telefone}{' '}
              <Link to={`/${slug}/conta`} className="text-[#7d5141] hover:underline">
                Minha conta
              </Link>
            </p>
          </>
        )}
      </div>

      {showSticky && (
        <BookingStickyBar
          summary={
            <>
              <strong>{formatDateTimeBR(data, hora)}</strong>
              {totalPreco > 0 ? ` · ${formatBRL(totalPreco)}` : ''}
              {aceitaAdiantar ? ' · aviso de horário mais cedo' : ''}
            </>
          }
          onConfirm={confirmar}
          loading={loading}
          disabled={!hora || servicoIds.length === 0 || !online || slotsStatus === 'loading'}
          validationMessage={
            !online ? 'Conecte-se à internet para confirmar.' : undefined
          }
        />
      )}
    </ClienteSalaoLayout>
  )
}
