import { AlertTriangle, Clock3, SearchX } from 'lucide-react'
import { Card } from '../../ui/Card'

type ErrorKind = 'not_found' | 'expired' | 'window_closed' | 'terminal' | 'unexpected'

const content: Record<ErrorKind, { title: string; message: string }> = {
  not_found: {
    title: 'Link não encontrado',
    message: 'Confira se o link recebido está completo ou fale com o salão.',
  },
  expired: {
    title: 'Este link expirou',
    message: 'Por segurança, solicite um novo link diretamente ao salão.',
  },
  window_closed: {
    title: 'Prazo de cancelamento encerrado',
    message: 'O horário está muito próximo para ser cancelado por este link.',
  },
  terminal: {
    title: 'Este horário não pode ser alterado',
    message: 'O agendamento já foi cancelado ou concluído.',
  },
  unexpected: {
    title: 'Não foi possível abrir seu horário',
    message: 'Tente novamente em alguns instantes.',
  },
}

export function GestaoEstadoErro({
  kind,
  message,
  contactPhone,
}: {
  kind: ErrorKind
  message?: string
  contactPhone?: string | null
}) {
  const copy = content[kind]
  const Icon = kind === 'not_found' ? SearchX : kind === 'window_closed' ? Clock3 : AlertTriangle

  return (
    <Card className="text-center">
      <Icon className="mx-auto h-10 w-10 text-aura-primary" />
      <h1 className="mt-4 font-display text-2xl font-semibold text-aura-anthracite">
        {copy.title}
      </h1>
      <p className="mt-2 text-sm text-aura-muted">{message || copy.message}</p>
      {contactPhone && (
        <a
          className="mt-5 inline-flex rounded-lg bg-aura-primary px-4 py-2.5 text-sm font-medium text-white"
          href={`tel:${contactPhone}`}
        >
          Ligar para o salão
        </a>
      )}
    </Card>
  )
}
