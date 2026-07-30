import { Link } from 'react-router-dom'
import { Alert } from '../ui/Alert'

interface EarlySlotQueueInactiveBannerProps {
  active?: boolean
  /** Somente DONA acessa `/admin/whatsapp`; demais perfis recebem orientação textual. */
  canReconnectWhatsApp?: boolean
  className?: string
}

export function EarlySlotQueueInactiveBanner({
  active,
  canReconnectWhatsApp = false,
  className = 'mb-4',
}: EarlySlotQueueInactiveBannerProps) {
  if (active !== false) return null

  return (
    <Alert variant="warning" className={className}>
      A fila de antecipação está inativa porque o WhatsApp do salão está desconectado.{' '}
      {canReconnectWhatsApp ? (
        <>
          Reconecte em{' '}
          <Link to="/admin/whatsapp" className="font-semibold underline">
            Integração WhatsApp
          </Link>
          .
        </>
      ) : (
        <>Peça à dona ou administradora do salão para reconectar o WhatsApp.</>
      )}
    </Alert>
  )
}
