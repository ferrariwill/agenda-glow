import { useState } from 'react'
import { Bell, Menu, RefreshCw, Search, X } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { initials } from '../../utils/format'

interface SuperAdminTopBarProps {
  searchPlaceholder?: string
  searchValue: string
  onSearchChange: (v: string) => void
  onRenewAll?: () => void
  renewLoading?: boolean
  onMenuClick?: () => void
}

export function SuperAdminTopBar({
  searchPlaceholder = 'Buscar…',
  searchValue,
  onSearchChange,
  onRenewAll,
  renewLoading,
  onMenuClick,
}: SuperAdminTopBarProps) {
  const { session } = useAuth()
  const name = session?.user.nome ?? 'Admin'
  const [searchOpen, setSearchOpen] = useState(false)

  return (
    <div className="sticky top-0 z-30 border-b border-aura-border bg-white px-4 py-3 sm:px-6">
      <div className="flex items-center gap-2 sm:gap-4">
        <button
          type="button"
          onClick={onMenuClick}
          className="rounded-lg p-2 text-aura-muted hover:bg-aura-surface lg:hidden"
          aria-label="Abrir menu"
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
        {onRenewAll && (
          <>
            <button
              type="button"
              onClick={onRenewAll}
              disabled={renewLoading}
              className="inline-flex items-center gap-2 rounded-lg border border-aura-primary px-3 py-2 text-sm font-medium text-aura-primary transition-colors hover:bg-aura-primary/5 disabled:opacity-50 sm:hidden"
              aria-label="Renovar assinatura"
            >
              <RefreshCw className={`h-4 w-4 ${renewLoading ? 'animate-spin' : ''}`} />
            </button>
            <button
              type="button"
              onClick={onRenewAll}
              disabled={renewLoading}
              className="hidden items-center gap-2 rounded-lg border border-aura-primary px-4 py-2 text-sm font-medium text-aura-primary transition-colors hover:bg-aura-primary/5 sm:inline-flex disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${renewLoading ? 'animate-spin' : ''}`} />
              Renovar Assinatura
            </button>
          </>
        )}
        <button
          type="button"
          className="relative rounded-lg p-2 text-aura-muted hover:bg-aura-surface"
          aria-label="Notificações"
        >
          <Bell className="h-5 w-5" />
          <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500" />
        </button>
        <div
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary-dark"
          title={name}
        >
          {initials(name)}
        </div>
      </div>
    </div>
  )
}
