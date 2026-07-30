import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError } from '../../lib/api'
import {
  acceptEarlySlotOffer,
  declineEarlySlotOffer,
  getEarlySlotOffer,
  type EarlySlotAcceptResponse,
  type EarlySlotOfferDetail,
} from '../../services/earlySlotOfferService'

type ViewState =
  | 'loading'
  | 'available'
  | 'submitting_accept'
  | 'submitting_decline'
  | 'accepted'
  | 'declined'
  | 'expired'
  | 'unavailable'
  | 'ineligible'
  | 'not_found'
  | 'error'

const terminalState = (status: EarlySlotOfferDetail['status']): ViewState => {
  switch (status) {
    case 'ACEITA':
      return 'accepted'
    case 'RECUSADA':
      return 'declined'
    case 'EXPIRADA':
      return 'expired'
    case 'INVALIDADA':
      return 'unavailable'
    default:
      return 'available'
  }
}

const errorState = (error: unknown): ViewState => {
  if (!(error instanceof ApiError)) return 'error'
  if (error.status === 404 || error.code === 'offer_not_found') return 'not_found'
  if (error.status === 410 || error.code === 'offer_expired') return 'expired'
  if (error.status === 409 || error.code === 'offer_no_longer_available') return 'unavailable'
  if (error.status === 422 || error.code === 'appointment_no_longer_eligible') return 'ineligible'
  return 'error'
}

function formatDateTime(value: string): string {
  return new Intl.DateTimeFormat('pt-BR', {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(new Date(value))
}

function formatCountdown(seconds: number): string {
  const safe = Math.max(0, seconds)
  return `${String(Math.floor(safe / 60)).padStart(2, '0')}:${String(safe % 60).padStart(2, '0')}`
}

export function OfertaAntecipacaoPublica() {
  const { token = '' } = useParams()
  const [offer, setOffer] = useState<EarlySlotOfferDetail | null>(null)
  const [accepted, setAccepted] = useState<EarlySlotAcceptResponse | null>(null)
  const [view, setView] = useState<ViewState>('loading')
  const [seconds, setSeconds] = useState(0)
  const deadlineRef = useRef(0)
  const expiryRefreshRef = useRef(false)
  const mountedRef = useRef(true)

  const load = useCallback(async (signal?: AbortSignal) => {
    if (!token) {
      setView('not_found')
      return
    }
    try {
      const detail = await getEarlySlotOffer(token, signal)
      if (!mountedRef.current) return
      setOffer(detail)
      const serverSeconds = Math.max(0, detail.seconds_remaining)
      const expiresIn = Math.max(0, Math.floor((new Date(detail.expires_at).getTime() - Date.now()) / 1000))
      const reconciledSeconds = Math.min(serverSeconds, expiresIn)
      deadlineRef.current = Date.now() + reconciledSeconds * 1000
      setSeconds(reconciledSeconds)
      expiryRefreshRef.current = detail.status !== 'PENDENTE'
      setView(terminalState(detail.status))
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') return
      if (mountedRef.current) setView(errorState(error))
    }
  }, [token])

  useEffect(() => {
    mountedRef.current = true
    const controller = new AbortController()
    // A carga é assíncrona e sincroniza a página com o recurso indicado pela rota.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load(controller.signal)
    return () => {
      mountedRef.current = false
      controller.abort()
    }
  }, [load])

  useEffect(() => {
    if (view !== 'available') return
    const tick = () => {
      const remaining = Math.max(0, Math.ceil((deadlineRef.current - Date.now()) / 1000))
      setSeconds(remaining)
      if (remaining === 0 && !expiryRefreshRef.current) {
        expiryRefreshRef.current = true
        void load()
      }
    }
    tick()
    const timer = window.setInterval(tick, 1000)
    const onVisibility = () => {
      if (document.visibilityState === 'visible' && deadlineRef.current <= Date.now()) tick()
    }
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [load, view])

  const submit = async (action: 'accept' | 'decline') => {
    if (!offer || view !== 'available' || !offer.actions_allowed.includes(action)) return
    setView(action === 'accept' ? 'submitting_accept' : 'submitting_decline')
    try {
      if (action === 'accept') {
        const response = await acceptEarlySlotOffer(token)
        setAccepted(response)
        setView('accepted')
      } else {
        await declineEarlySlotOffer(token)
        setView('declined')
      }
    } catch (error) {
      const next = errorState(error)
      if (next === 'unavailable') {
        setView('unavailable')
        await load()
      } else {
        setView(next)
      }
    }
  }

  const retry = () => {
    setView('loading')
    void load()
  }

  const stateCopy: Partial<Record<ViewState, { title: string; body: string }>> = {
    accepted: {
      title: 'Novo horário confirmado',
      body: accepted
        ? `Seu atendimento foi antecipado para ${formatDateTime(accepted.new_start)}.`
        : 'Esta oferta já foi aceita.',
    },
    declined: { title: 'Oferta recusada', body: 'Seu agendamento atual não foi alterado.' },
    expired: { title: 'Oferta expirada', body: 'O prazo desta oferta terminou.' },
    unavailable: { title: 'Oferta indisponível', body: 'Esta oferta não está mais disponível.' },
    ineligible: { title: 'Atendimento não elegível', body: 'Este atendimento não pode mais ser antecipado.' },
    not_found: { title: 'Oferta não encontrada', body: 'Confira se o link recebido está completo.' },
  }

  return (
    <main className="min-h-screen bg-[#faf9f8] px-4 py-10 text-[#1a1c1c]">
      <section className="mx-auto max-w-lg rounded-3xl border border-[#efdcd1] bg-white p-6 shadow-xl sm:p-8">
        <p className="text-xs font-bold uppercase tracking-[0.2em] text-[#7d5141]">AgendaGlow</p>
        <div className="mt-4" aria-live="polite">
          {view === 'loading' && <p className="py-16 text-center text-[#514440]">Carregando oferta…</p>}

          {offer && ['available', 'submitting_accept', 'submitting_decline'].includes(view) && (
            <>
              <h1 className="font-display text-2xl font-semibold">Um horário mais cedo ficou disponível</h1>
              <p className="mt-2 text-sm text-[#514440]">
                {offer.salon_name} · {offer.professional_name} · {offer.service_name}
              </p>
              <dl className="mt-6 grid gap-3 rounded-2xl bg-[#f8f3f0] p-4">
                <div>
                  <dt className="text-xs font-bold uppercase text-[#83746f]">Horário atual</dt>
                  <dd className="font-medium">{formatDateTime(offer.current_start)}</dd>
                </div>
                <div>
                  <dt className="text-xs font-bold uppercase text-[#83746f]">Novo horário</dt>
                  <dd className="text-lg font-semibold text-[#7d5141]">{formatDateTime(offer.offered_start)}</dd>
                </div>
              </dl>
              <p className="mt-5 text-center text-sm text-[#514440]">
                Tempo restante: <strong className="tabular-nums text-[#7d5141]">{formatCountdown(seconds)}</strong>
              </p>
              <div className="mt-6 grid gap-3">
                <button
                  type="button"
                  disabled={view !== 'available' || !offer.actions_allowed.includes('accept') || seconds === 0}
                  onClick={() => void submit('accept')}
                  className="rounded-xl bg-[#7d5141] px-4 py-3 font-semibold text-white disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {view === 'submitting_accept' ? 'Confirmando…' : 'Aceitar novo horário'}
                </button>
                <button
                  type="button"
                  disabled={view !== 'available' || !offer.actions_allowed.includes('decline') || seconds === 0}
                  onClick={() => void submit('decline')}
                  className="rounded-xl border border-[#d6c2bd] px-4 py-3 font-semibold text-[#514440] disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {view === 'submitting_decline' ? 'Recusando…' : 'Recusar'}
                </button>
              </div>
            </>
          )}

          {stateCopy[view] && (
            <div className="py-12 text-center" tabIndex={-1}>
              <h1 className="font-display text-2xl font-semibold">{stateCopy[view]?.title}</h1>
              <p className="mt-3 text-[#514440]">{stateCopy[view]?.body}</p>
            </div>
          )}

          {view === 'error' && (
            <div className="py-12 text-center" role="alert">
              <h1 className="font-display text-2xl font-semibold">Não foi possível carregar a oferta</h1>
              <button type="button" onClick={retry} className="mt-5 rounded-xl bg-[#7d5141] px-4 py-2 text-white">
                Tentar novamente
              </button>
            </div>
          )}
        </div>
      </section>
    </main>
  )
}
