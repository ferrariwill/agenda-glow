import { useState } from 'react'
import { Bell, Menu, Search, X } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { initials } from '../../utils/format'

interface DonaTopBarProps {
  searchPlaceholder?: string
  searchValue: string
  onSearchChange: (v: string) => void
  onMenuClick?: () => void
  menuOpen?: boolean
  action?: React.ReactNode
}

export function DonaTopBar({
  searchPlaceholder = 'Busque por clientes…',
  searchValue,
  onSearchChange,
  onMenuClick,
  menuOpen = false,
  action,
}: DonaTopBarProps) {
  const { session } = useAuth()
  const name = session?.user.nome ?? 'Dona'
  const [searchOpen, setSearchOpen] = useState(false)

  return (
    <div className="sticky top-0 z-30 border-b border-aura-border bg-white px-4 py-3 sm:px-6">
      <div className="flex items-center gap-2 sm:gap-4">
        <button
          type="button"
          onClick={onMenuClick}
          className="inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface lg:hidden"
          aria-label="Abrir menu"
          aria-expanded={menuOpen}
          aria-controls="dona-mobile-drawer"
        >
          <Menu className="h-5 w-5" />
        </button>
        <div
          className={[
            'relative min-w-0 flex-1 max-w-xl',
            searchOpen ? 'flex' : 'hidden sm:flex',
          ].join(' ')}
        >
          <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-aura-muted" />
          <input
            type="search"
            value={searchValue}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={searchPlaceholder}
            className="w-full rounded-lg border border-aura-border bg-aura-surface py-2.5 pl-10 pr-4 text-sm focus:border-aura-primary focus:outline-none focus:ring-2 focus:ring-aura-primary/15"
          />
        </div>
        <button
          type="button"
          onClick={() => setSearchOpen((v) => !v)}
          className="rounded-lg p-2 text-aura-muted hover:bg-aura-surface sm:hidden"
          aria-label={searchOpen ? 'Fechar busca' : 'Buscar'}
        >
          {searchOpen ? <X className="h-5 w-5" /> : <Search className="h-5 w-5" />}
        </button>
        {action && <div className="hidden shrink-0 sm:block">{action}</div>}
      <button type="button" className="relative rounded-lg p-2 text-aura-muted hover:bg-aura-surface">
        <Bell className="h-5 w-5" />
        <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500" />
      </button>
        <div className="hidden text-right sm:block">
          <p className="text-sm font-medium text-aura-anthracite">{name}</p>
          <p className="text-xs text-aura-muted">Dona do Salão</p>
        </div>
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary-dark">
          {initials(name)}
        </div>
      </div>
    </div>
  )
}
