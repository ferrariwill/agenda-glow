import { ExternalLink, MoreHorizontal, RefreshCw, Shield, UserPlus, Wallet } from 'lucide-react'
import { ActionsDropdown } from '../ui/ActionsDropdown'

interface TenantActionsMenuProps {
  status: 'ATIVO' | 'VENCIDO'
  slug: string
  onAssignPlan: () => void
  onRenew: () => void
  onToggleStatus: () => void
  onCreateDona: () => void
}

export function TenantActionsMenu({
  status,
  slug,
  onAssignPlan,
  onRenew,
  onToggleStatus,
  onCreateDona,
}: TenantActionsMenuProps) {
  return (
    <ActionsDropdown
      icon={MoreHorizontal}
      minWidth={192}
      items={[
        { label: 'Atribuir plano', icon: Wallet, onClick: onAssignPlan },
        { label: 'Renovar +12 meses', icon: RefreshCw, onClick: onRenew },
        {
          label: status === 'ATIVO' ? 'Suspender' : 'Ativar',
          icon: Shield,
          onClick: onToggleStatus,
        },
        { label: 'Criar dona', icon: UserPlus, onClick: onCreateDona },
        {
          label: 'Ver catálogo',
          icon: ExternalLink,
          onClick: () => window.open(`/${slug}`, '_blank'),
        },
      ]}
    />
  )
}
