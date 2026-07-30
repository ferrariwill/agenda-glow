import { Timer } from 'lucide-react'
import type { EarlySlotOfferResumo } from '../../types/earlySlot'
import { formatTimestampTimeBR } from '../../utils/format'

interface EarlySlotRoundIndicatorProps {
  offer?: EarlySlotOfferResumo | null
  compact?: boolean
}

/**
 * Leitura da rodada de antecipação no card da agenda. Sem ações — aceitar,
 * pular ou forçar candidato é responsabilidade exclusiva do backend.
 */
export function EarlySlotRoundIndicator({
  offer,
  compact = false,
}: EarlySlotRoundIndicatorProps) {
  const pendente = offer?.offer_status === 'PENDENTE'

  if (!pendente) return null

  const prazo = offer?.expires_at ? `até ${formatTimestampTimeBR(offer.expires_at)}` : ''

  if (compact) {
    return (
      <span
        className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded bg-aura-warning text-amber-900"
        title={`Oferta ativa${prazo ? ` · ${prazo}` : ''}`}
        aria-label={`Oferta ativa${prazo ? `, ${prazo}` : ''}`}
      >
        <Timer className="h-3 w-3" />
      </span>
    )
  }

  return (
    <span
      className={[
        'inline-flex items-center gap-1 rounded-lg bg-aura-warning px-2 py-0.5 font-medium text-amber-900',
        'text-xs',
      ].join(' ')}
      title={offer?.posicao ? `Candidato ${offer.posicao} da fila` : undefined}
    >
      <Timer className="h-3.5 w-3.5" />
      Oferta ativa{prazo && ` · ${prazo}`}
    </span>
  )
}
