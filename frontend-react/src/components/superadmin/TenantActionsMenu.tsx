import { ExternalLink, MoreHorizontal, Pencil, RefreshCw, Shield, UserPlus, Wallet } from 'lucide-react'
import { ActionsDropdown } from '../ui/ActionsDropdown'
import type { TenantStatus } from '../../types'

interface TenantActionsMenuProps {
  status: TenantStatus
  slug: string
  onEdit: () => void
  onAssignPlan: () => void
  onRenew: () => void
  onToggleStatus: () => void
  onCreateDona: () => void
}

export function TenantActionsMenu({
  status,
  slug,
  onEdit,
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
        { label: 'Editar', icon: Pencil, onClick: onEdit },
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
