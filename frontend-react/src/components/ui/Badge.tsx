import type { ConfirmacaoCliente, TenantStatus } from '../../types'

type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'muted'

const styles: Record<BadgeVariant, string> = {
  default: 'bg-aura-primary/15 text-aura-primary-dark',
  success: 'bg-emerald-100 text-emerald-800',
  warning: 'bg-aura-warning text-amber-900 border border-aura-warning-border',
  danger: 'bg-red-100 text-red-800',
  muted: 'bg-aura-surface text-aura-muted border border-aura-border',
}

interface BadgeProps {
  children: React.ReactNode
  variant?: BadgeVariant
}

export function Badge({ children, variant = 'default' }: BadgeProps) {
  return (
    <span
      className={[
        'inline-flex items-center rounded-lg px-2.5 py-0.5 text-caption font-medium',
        styles[variant],
      ].join(' ')}
    >
      {children}
    </span>
  )
}

export function statusTenantBadge(status: TenantStatus): BadgeVariant {
  switch (status) {
    case 'ATIVO':
      return 'success'
    case 'VENCIDO':
      return 'danger'
    case 'SUSPENSO':
      return 'warning'
  }
}

export function statusAgendamentoBadge(
  status: string,
): BadgeVariant {
  switch (status) {
    case 'CONFIRMADO':
    case 'CONCLUIDO':
      return 'success'
    case 'EM_APROVACAO':
      return 'warning'
    case 'CANCELADO':
      return 'danger'
    default:
      return 'muted'
  }
}

// eslint-disable-next-line react-refresh/only-export-components -- helper visual compartilhado pelos badges
export function confirmacaoClienteBadge(
  status: ConfirmacaoCliente = 'PENDENTE',
): { variant: BadgeVariant; label: string } {
  switch (status) {
    case 'CONFIRMADO_CLIENTE':
      return { variant: 'success', label: 'Confirmou' }
    case 'CANCELADO_CLIENTE':
      return { variant: 'danger', label: 'Cancelou' }
    case 'PENDENTE':
      return { variant: 'warning', label: 'Pendente' }
  }
}
