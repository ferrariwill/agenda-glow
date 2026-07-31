import { MoreVertical, Pencil, Power, Trash2 } from 'lucide-react'
import { ActionsDropdown } from '../ui/ActionsDropdown'

interface ServicoActionsMenuProps {
  ativo: boolean
  onEdit: () => void
  onToggleAtivo: () => void
  onDelete: () => void
  /** Always show trigger (cards/mobile). Default: reveal on row hover. */
  alwaysVisible?: boolean
}

export function ServicoActionsMenu({
  ativo,
  onEdit,
  onToggleAtivo,
  onDelete,
  alwaysVisible = false,
}: ServicoActionsMenuProps) {
  return (
    <ActionsDropdown
      icon={MoreVertical}
      iconClassName="h-5 w-5"
      ariaLabel="Ações do serviço"
      triggerClassName={[
        'inline-flex h-11 w-11 items-center justify-center rounded-full text-[#615b58] transition-all hover:bg-[#e9e8e7]',
        alwaysVisible ? '' : 'opacity-0 group-hover:opacity-100',
      ]
        .filter(Boolean)
        .join(' ')}
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
