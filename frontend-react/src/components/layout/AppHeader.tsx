import { LogOut } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useLogout } from '../../hooks/useLogout'

interface AppHeaderProps {
  title?: string
}

export function AppHeader({ title }: AppHeaderProps) {
  const { session } = useAuth()
  const handleLogout = useLogout()

  return (
    <header className="sticky top-0 z-30 border-b border-aura-border bg-aura-surface/95 backdrop-blur">
      <div className="flex items-center justify-between gap-4 px-4 py-4 lg:px-8">
        <div className="min-w-0 flex-1">
          {title && (
            <h1 className="truncate font-display text-xl font-semibold text-aura-anthracite sm:text-2xl">
              {title}
            </h1>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-2 sm:gap-3">
          <div className="hidden text-right sm:block">
            <p className="text-sm font-medium text-aura-anthracite">{session?.user.nome}</p>
            <p className="max-w-[180px] truncate text-xs text-aura-muted">{session?.user.email}</p>
          </div>
          <button
            type="button"
            onClick={handleLogout}
            className="inline-flex items-center gap-2 rounded-lg border border-aura-border bg-white px-3 py-2 text-sm font-medium text-aura-anthracite transition-colors hover:border-aura-primary hover:text-aura-primary"
            title="Sair da conta"
          >
            <LogOut className="h-4 w-4" />
            <span>Sair</span>
          </button>
        </div>
      </div>
    </header>
  )
}
