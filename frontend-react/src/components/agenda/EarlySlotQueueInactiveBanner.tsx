import { Link } from 'react-router-dom'
import { Alert } from '../ui/Alert'

/**
 * Banner persistente quando a API reporta fila inativa.
 * Ausente/undefined = backend antigo — não tratar como inativo.
 */
export function EarlySlotQueueInactiveBanner({
  queueActive,
}: {
  queueActive?: boolean
}) {
  if (queueActive !== false) return null

  return (
    <Alert variant="warning" className="mb-4">
      A fila de antecipação está inativa porque o WhatsApp do salão está desconectado. Reconecte em{' '}
      <Link to="/admin/whatsapp" className="font-semibold underline">
        Integração WhatsApp
      </Link>
      .
    </Alert>
  )
}
