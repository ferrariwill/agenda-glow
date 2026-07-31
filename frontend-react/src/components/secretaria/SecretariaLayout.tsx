import { useCallback, useRef, useState, type ReactNode } from 'react'
import { Menu, Search } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useMobileDrawer } from '../../hooks/useMobileDrawer'
import { initials } from '../../utils/format'
import { BottomNav } from '../layout/BottomNav'
import { SecretariaSidebar } from './SecretariaSidebar'

interface SecretariaLayoutProps {
  children: ReactNode
  searchPlaceholder?: string
  searchValue?: string
  onSearchChange?: (v: string) => void
  headerAction?: ReactNode
}

export function SecretariaLayout({
  children,
  searchPlaceholder = 'Buscar…',
  searchValue = '',
  onSearchChange,
  headerAction,
}: SecretariaLayoutProps) {
  const { session } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const panelRef = useRef<HTMLElement>(null)
  const name = session?.user.nome ?? 'Secretaria'
  const closeSidebar = useCallback(() => setSidebarOpen(false), [])

  useMobileDrawer(sidebarOpen, closeSidebar, panelRef)

  return (
    <div className="min-h-screen bg-aura-surface lg:flex">
      {sidebarOpen && (
        <button
          type="button"
          className="fixed inset-0 z-[45] bg-aura-anthracite/40 lg:hidden"
          onClick={closeSidebar}
          aria-label="Fechar menu"
        />
      )}
      <SecretariaSidebar
        ref={panelRef}
        mobileOpen={sidebarOpen}
        onNavigate={closeSidebar}
      />
      <div className="flex min-w-0 flex-1 flex-col" inert={sidebarOpen || undefined}>
        <header className="sticky top-0 z-30 border-b border-aura-border bg-white px-4 py-3 sm:px-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 items-center gap-3">
              <button
                type="button"
                onClick={() => setSidebarOpen(true)}
                className="inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface lg:hidden"
                aria-label="Abrir menu"
                aria-expanded={sidebarOpen}
                aria-controls="secretaria-mobile-drawer"
              >
                <Menu className="h-5 w-5" />
              </button>
              {onSearchChange && (
                <div className="relative min-w-0 flex-1 sm:max-w-md">
                  <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-aura-muted/50" />
                  <input
                    type="search"
                    value={searchValue}
                    onChange={(e) => onSearchChange(e.target.value)}
                    placeholder={searchPlaceholder}
                    className="min-h-touch-min w-full touch-manipulation rounded-lg border border-aura-border bg-aura-surface py-2 pl-10 pr-3 text-control text-aura-anthracite placeholder:text-aura-muted/50 focus:border-aura-primary focus:outline-none focus:ring-2 focus:ring-aura-primary/20"
                  />
                </div>
              )}
            </div>
            <div className="flex shrink-0 items-center justify-end gap-3">
              {headerAction}
              <div className="hidden text-right sm:block">
                <p className="text-sm font-medium text-aura-anthracite">{name}</p>
                <p className="text-xs text-aura-muted">Secretaria</p>
              </div>
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary-dark">
                {initials(name)}
              </div>
            </div>
          </div>
        </header>
        <main className="flex-1 px-4 py-5 pb-24 sm:px-6 sm:py-6 lg:px-8 lg:pb-8">
          {children}
        </main>
      </div>
      <BottomNav variant="secretaria" drawerOpen={sidebarOpen} />
    </div>
  )
}
