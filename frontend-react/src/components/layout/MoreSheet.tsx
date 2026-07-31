import { NavLink } from 'react-router-dom'
import type { LucideIcon } from 'lucide-react'
import { Modal } from '../ui/Modal'

export interface MoreSheetItem {
  to: string
  label: string
  icon: LucideIcon
}

interface MoreSheetProps {
  open: boolean
  onClose: () => void
  items: MoreSheetItem[]
  title?: string
}

export function MoreSheet({
  open,
  onClose,
  items,
  title = 'Mais',
}: MoreSheetProps) {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      <nav aria-label="Destinos adicionais" className="flex flex-col gap-1">
        {items.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            onClick={onClose}
            className={({ isActive }) =>
              [
                'flex min-h-touch-min touch-manipulation items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-aura-primary/8 text-aura-primary-dark'
                  : 'text-aura-anthracite hover:bg-aura-surface',
              ].join(' ')
            }
          >
            <Icon className="h-5 w-5 shrink-0" aria-hidden />
            {label}
          </NavLink>
        ))}
      </nav>
    </Modal>
  )
}
