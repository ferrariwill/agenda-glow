import { Badge } from '../ui/Badge'

interface AceitaAntecipacaoBadgeProps {
  aceitaAdiantar?: boolean
}

/** Sinaliza no card que o cliente optou por receber ofertas de horário mais cedo. */
export function AceitaAntecipacaoBadge({ aceitaAdiantar }: AceitaAntecipacaoBadgeProps) {
  if (!aceitaAdiantar) return null
  return <Badge variant="muted">Aceita antecipação</Badge>
}
