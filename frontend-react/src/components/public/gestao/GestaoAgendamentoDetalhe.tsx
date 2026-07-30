import { CalendarDays, Clock, Scissors, UserRound } from 'lucide-react'
import { Badge, confirmacaoClienteBadge } from '../../ui/Badge'
import { Card } from '../../ui/Card'
import type { PublicAppointmentManageResponse } from '../../../types'

export function GestaoAgendamentoDetalhe({
  data,
}: {
  data: PublicAppointmentManageResponse
}) {
  const startsAt = new Date(data.appointment.starts_at)
  const confirmation = confirmacaoClienteBadge(data.appointment.customer_confirmation)

  return (
    <Card className="space-y-5">
      <div>
        <p className="text-xs font-bold uppercase tracking-widest text-aura-muted">Seu horário em</p>
        <h1 className="mt-1 font-display text-2xl font-semibold text-aura-anthracite">
          {data.establishment.name}
        </h1>
      </div>

      <dl className="space-y-3 text-sm">
        <div className="flex items-center gap-3">
          <Scissors className="h-4 w-4 text-aura-primary" />
          <div><dt className="sr-only">Serviço</dt><dd>{data.appointment.service}</dd></div>
        </div>
        <div className="flex items-center gap-3">
          <UserRound className="h-4 w-4 text-aura-primary" />
          <div><dt className="sr-only">Profissional</dt><dd>{data.appointment.professional}</dd></div>
        </div>
        <div className="flex items-center gap-3">
          <CalendarDays className="h-4 w-4 text-aura-primary" />
          <div>
            <dt className="sr-only">Data</dt>
            <dd>{startsAt.toLocaleDateString('pt-BR', { dateStyle: 'long' })}</dd>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Clock className="h-4 w-4 text-aura-primary" />
          <div>
            <dt className="sr-only">Horário</dt>
            <dd>{startsAt.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}</dd>
          </div>
        </div>
      </dl>

      <div className="flex flex-wrap items-center gap-2 border-t border-aura-border pt-4">
        <Badge variant={data.appointment.status === 'CANCELADO' ? 'danger' : 'muted'}>
          {data.appointment.status}
        </Badge>
        <Badge variant={confirmation.variant}>{confirmation.label}</Badge>
      </div>
    </Card>
  )
}
