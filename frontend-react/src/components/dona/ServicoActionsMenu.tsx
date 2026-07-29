import { MoreVertical, Pencil, Power, Trash2 } from 'lucide-react'
import { ActionsDropdown } from '../ui/ActionsDropdown'

interface ServicoActionsMenuProps {
  ativo: boolean
  onEdit: () => void
  onToggleAtivo: () => void
  onDelete: () => void
}

export function ServicoActionsMenu({
  ativo,
  onEdit,
  onToggleAtivo,
  onDelete,
}: ServicoActionsMenuProps) {
  return (
    <ActionsDropdown
      icon={MoreVertical}
      iconClassName="h-5 w-5"
      ariaLabel="Ações do serviço"
      triggerClassName="rounded-full p-2 text-[#615b58] opacity-0 transition-all hover:bg-[#e9e8e7] group-hover:opacity-100"
      menuClassName="border-[#d6c2bd]/40"
      items={[
        { label: 'Editar serviço', icon: Pencil, onClick: onEdit },
        {
          label: ativo ? 'Desativar' : 'Ativar',
          icon: Power,
          onClick: onToggleAtivo,
        },
        { label: 'Excluir', icon: Trash2, onClick: onDelete, danger: true },
      ]}
    />
  )
}
