import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { CalendarClock, CheckCircle2, Loader2, Timer } from 'lucide-react'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { EarlySlotOfferSummary } from '../../components/public/EarlySlotOfferSummary'
import { useOfferCountdown } from '../../hooks/useOfferCountdown'
import {
  acceptOffer,
  declineOffer,
  earlySlotErrorMessage,
  getOffer,
  isEarlySlotTerminalError,
} from '../../services/earlySlotOfferApi'
import type { EarlySlotAcceptResult, EarlySlotOffer } from '../../types/earlySlot'
import { formatTimestampBR, formatTimestampTimeBR } from '../../utils/format'
import { visibleOfferError } from './earlySlotActionState'

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
      <Card className="w-full max-w-md" padding="lg">
        {children}
      </Card>
    </div>
  )
}

function statusFinalMessage(status: string): { titulo: string; texto: string } {
  switch (status) {
    case 'RECUSADA':
      return {
        titulo: 'Oferta recusada',
        texto: 'Seu horário original foi mantido. Obrigado por responder!',
      }
    case 'EXPIRADA':
      return {
        titulo: 'Esta oferta expirou',
        texto: 'O prazo de 5 minutos terminou e o horário foi oferecido a outro cliente.',
      }
    case 'INVALIDADA':
      return {
        titulo: 'Oferta não está mais disponível',
        texto: 'Este horário deixou de estar disponível para antecipação.',
      }
    default:
      return {
        titulo: 'Oferta encerrada',
        texto: 'Não há mais ações disponíveis para esta oferta.',
      }
  }
}

export function AntecipacaoOferta() {
  const { token } = useParams<{ token: string }>()

  const [offer, setOffer] = useState<EarlySlotOffer | null>(null)
  const [accepted, setAccepted] = useState<EarlySlotAcceptResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [revalidating, setRevalidating] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [actionError, setActionError] = useState('')
  const [actionTerminal, setActionTerminal] = useState(false)
  const mountedRef = useRef(true)

  const carregar = useCallback(async (opts?: { soft?: boolean }) => {
    if (!token) {
      setLoadError('Oferta não encontrada.')
      setLoading(false)
      return
    }
    if (!opts?.soft) setLoading(true)
    else setRevalidating(true)
    try {
      const data = await getOffer(token)
      if (!mountedRef.current) return
      setOffer(data)
      setLoadError('')
    } catch (err) {
      if (!mountedRef.current) return
      setOffer(null)
      setLoadError(earlySlotErrorMessage(err))
    } finally {
      if (mountedRef.current) {
        setLoading(false)
        setRevalidating(false)
      }
    }
  }, [token])

  useEffect(() => {
    mountedRef.current = true
    // Sync da página com o recurso indicado pela rota.
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetch-on-mount alinhado ao restante do app
    void carregar()
    return () => {
      mountedRef.current = false
    }
  }, [carregar])

  const onCountdownZero = useCallback(() => {
    void carregar({ soft: true })
  }, [carregar])

  const isPendente = offer?.status === 'PENDENTE'
  const { label: countdownLabel, secondsRemaining, expired } = useOfferCountdown(
    isPendente ? offer?.expires_at : undefined,
    onCountdownZero,
    isPendente ? offer?.seconds_remaining : undefined,
  )

  const acoes = offer?.actions_allowed ?? []
  const actionsBlocked =
    submitting || revalidating || expired || secondsRemaining === 0 || actionTerminal
  const podeAceitar = isPendente && acoes.includes('accept') && !actionsBlocked
  const podeRecusar = isPendente && acoes.includes('decline') && !actionsBlocked

  const aceitar = async () => {
    if (!token || !podeAceitar) return
    setSubmitting(true)
    setActionError('')
    setActionTerminal(false)
    try {
      setAccepted(await acceptOffer(token))
    } catch (err) {
      setActionError(earlySlotErrorMessage(err))
      setActionTerminal(isEarlySlotTerminalError(err))
      await carregar({ soft: true })
    } finally {
      setSubmitting(false)
    }
  }

  const recusar = async () => {
    if (!token || !podeRecusar) return
    setSubmitting(true)
    setActionError('')
    setActionTerminal(false)
    try {
      const result = await declineOffer(token)
      setOffer((prev) => (prev ? { ...prev, ...result, actions_allowed: [] } : prev))
    } catch (err) {
      setActionError(earlySlotErrorMessage(err))
      setActionTerminal(isEarlySlotTerminalError(err))
      await carregar({ soft: true })
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <Shell>
        <div className="flex flex-col items-center gap-3 py-6 text-aura-muted">
          <Loader2 className="h-6 w-6 animate-spin" />
          <p className="text-sm">Carregando oferta…</p>
        </div>
      </Shell>
    )
  }

  if (accepted) {
    const confirmedStart = accepted.new_start
    const confirmedEnd = accepted.new_end
    return (
      <Shell>
        <div className="text-center" aria-live="polite">
          <CheckCircle2 className="mx-auto mb-3 h-10 w-10 text-emerald-600" />
          <h1 className="font-display text-2xl font-semibold text-aura-anthracite">
            Horário antecipado!
          </h1>
          {confirmedStart ? (
            <>
              <p className="mt-2 text-sm text-aura-muted">Seu atendimento foi movido para:</p>
              <p className="mt-3 font-display text-xl font-semibold text-aura-primary-dark">
                {formatTimestampBR(confirmedStart)}
                {confirmedEnd && ` – ${formatTimestampTimeBR(confirmedEnd)}`}
              </p>
            </>
          ) : (
            <p className="mt-2 text-sm text-aura-muted">
              Seu atendimento foi antecipado. Confira o novo horário na confirmação do salão.
            </p>
          )}
          <p className="mt-4 text-sm text-aura-muted">
            Se o WhatsApp do salão estiver ativo, você receberá a confirmação do novo horário.
          </p>
        </div>
      </Shell>
    )
  }

  if (!offer) {
    return (
      <Shell>
        <div className="text-center" role="alert">
          <h1 className="font-display text-xl font-semibold text-aura-anthracite">
            Oferta indisponível
          </h1>
          <p className="mt-2 text-sm text-aura-muted">
            {visibleOfferError(loadError, actionError) || 'Oferta não encontrada.'}
          </p>
        </div>
      </Shell>
    )
  }

  if (!isPendente) {
    const final = statusFinalMessage(offer.status)
    return (
      <Shell>
        <div className="text-center" aria-live="polite">
          <h1 className="font-display text-xl font-semibold text-aura-anthracite">
            {final.titulo}
          </h1>
          <p className="mt-2 text-sm text-aura-muted">{final.texto}</p>
        </div>
        {offer.offered_start && (
          <div className="mt-5">
            <EarlySlotOfferSummary
              currentStart={offer.current_start}
              currentEnd={offer.current_end}
              offeredStart={offer.offered_start}
              offeredEnd={offer.offered_end}
            />
          </div>
        )}
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="flex items-center gap-2 text-aura-primary-dark">
        <CalendarClock className="h-5 w-5" />
        <p className="text-[11px] font-bold uppercase tracking-widest">Horário mais cedo</p>
      </div>
      <h1 className="mt-2 font-display text-2xl font-semibold text-aura-anthracite">
        {offer.salon_name ?? 'Seu salão'}
      </h1>
      <p className="mt-1 text-sm text-aura-muted">
        {[offer.service_name, offer.professional_name].filter(Boolean).join(' · ') ||
          'Surgiu um horário mais cedo para o seu atendimento.'}
      </p>

      {visibleOfferError(loadError, actionError) && (
        <Alert
          variant="error"
          className="mt-4"
          onDismiss={() => {
            setLoadError('')
            setActionError('')
          }}
        >
          {visibleOfferError(loadError, actionError)}
        </Alert>
      )}

      <div className="mt-5">
        <EarlySlotOfferSummary
          currentStart={offer.current_start}
          currentEnd={offer.current_end}
          offeredStart={offer.offered_start}
          offeredEnd={offer.offered_end}
        />
      </div>

      {secondsRemaining !== null && (
        <div className="mt-5 flex items-center justify-center gap-2 rounded-xl border border-aura-warning-border bg-aura-warning px-4 py-3">
          <Timer className="h-4 w-4 text-amber-900" />
          <p className="text-sm text-amber-950">
            {revalidating || expired
              ? 'Revalidando oferta no servidor…'
              : (
                <>
                  Oferta exclusiva por mais{' '}
                  <strong className="font-display text-base tabular-nums">{countdownLabel}</strong>
                </>
              )}
          </p>
        </div>
      )}

      <div className="mt-5 space-y-2">
        <Button fullWidth onClick={aceitar} loading={submitting} disabled={!podeAceitar}>
          Aceitar novo horário
        </Button>
        <Button
          variant="secondary"
          fullWidth
          onClick={recusar}
          disabled={!podeRecusar}
        >
          Recusar
        </Button>
      </div>

      <p className="mt-4 text-center text-xs text-aura-muted">
        Se você não responder, seu horário atual continua valendo.
      </p>
    </Shell>
  )
}
