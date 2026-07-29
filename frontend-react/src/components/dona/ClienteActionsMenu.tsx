import { Calendar, CalendarPlus, MoreHorizontal, Pencil, UserX } from 'lucide-react'
import { ActionsDropdown } from '../ui/ActionsDropdown'

interface ClienteActionsMenuProps {
  ativo: boolean
  onEdit: () => void
  onHistorico: () => void
  onAgendar: () => void
  onToggleAtivo: () => void
}

export function ClienteActionsMenu({
  ativo,
  onEdit,
  onHistorico,
  onAgendar,
  onToggleAtivo,
}: ClienteActionsMenuProps) {
  return (
    <ActionsDropdown
      icon={MoreHorizontal}
      ariaLabel="Ações do cliente"
      items={[
        { label: 'Agendar', icon: CalendarPlus, onClick: onAgendar, disabled: !ativo },
        { label: 'Ver histórico', icon: Calendar, onClick: onHistorico },
        { label: 'Editar cliente', icon: Pencil, onClick: onEdit },
        {
          label: ativo ? 'Desativar' : 'Reativar',
          icon: UserX,
          onClick: onToggleAtivo,
        },
      ]}
    />
  )
}
