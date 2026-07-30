import { Link } from 'react-router-dom'
import { Alert } from '../ui/Alert'

interface EarlySlotQueueInactiveBannerProps {
  active?: boolean
  className?: string
}

export function EarlySlotQueueInactiveBanner({
  active,
  className = 'mb-4',
}: EarlySlotQueueInactiveBannerProps) {
  if (active !== false) return null

  return (
    <Alert variant="warning" className={className}>
      A fila de antecipação está inativa porque o WhatsApp do salão está desconectado. Reconecte em{' '}
      <Link to="/admin/whatsapp" className="font-semibold underline">
        Integração WhatsApp
      </Link>
      .
    </Alert>
  )
}
