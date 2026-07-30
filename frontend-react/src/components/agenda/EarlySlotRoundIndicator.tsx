import { Timer } from 'lucide-react'
import { useOfferCountdown } from '../../hooks/useOfferCountdown'
import type { EarlySlotOfferResumo } from '../../types/earlySlot'
import { formatTimestampTimeBR } from '../../utils/format'

interface EarlySlotRoundIndicatorProps {
  offer?: EarlySlotOfferResumo | null
  /** Recarrega a agenda quando a oferta pendente expira (sem reload da página). */
  onExpire?: () => void
  compact?: boolean
}

/**
 * Leitura da rodada de antecipação no card da agenda. Sem ações — aceitar,
 * pular ou forçar candidato é responsabilidade exclusiva do backend.
 */
export function EarlySlotRoundIndicator({
  offer,
  onExpire,
  compact = false,
}: EarlySlotRoundIndicatorProps) {
  const pendente = offer?.offer_status === 'PENDENTE'
  const { label, secondsRemaining } = useOfferCountdown(
    pendente ? offer?.expires_at : undefined,
    onExpire,
  )

  if (!pendente) return null

  const prazo =
    secondsRemaining !== null
      ? label
      : offer?.expires_at
        ? `até ${formatTimestampTimeBR(offer.expires_at)}`
        : ''

  return (
    <span
      className={[
        'inline-flex items-center gap-1 rounded-lg bg-aura-warning px-2 py-0.5 font-medium text-amber-900',
        compact ? 'text-[10px]' : 'text-xs',
      ].join(' ')}
      title={offer?.posicao ? `Candidato ${offer.posicao} da fila` : undefined}
    >
      <Timer className={compact ? 'h-3 w-3' : 'h-3.5 w-3.5'} />
      Oferta ativa{prazo && ` · ${prazo}`}
    </span>
  )
}
