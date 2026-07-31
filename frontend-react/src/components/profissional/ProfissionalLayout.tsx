import { useCallback, useRef, useState, type ReactNode } from 'react'
import { Menu } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useMobileDrawer } from '../../hooks/useMobileDrawer'
import { initials } from '../../utils/format'
import { BottomNav } from '../layout/BottomNav'
import { ProfissionalSidebar } from './ProfissionalSidebar'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white/70 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md'

interface ProfissionalLayoutProps {
  children: ReactNode
  title?: string
  subtitle?: string
  headerAction?: ReactNode
}

export function ProfissionalLayout({
  children,
  title,
  subtitle,
  headerAction,
}: ProfissionalLayoutProps) {
  const { session } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const panelRef = useRef<HTMLElement>(null)
  const name = session?.user.nome ?? 'Profissional'
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
      <ProfissionalSidebar
        ref={panelRef}
        mobileOpen={sidebarOpen}
        onNavigate={closeSidebar}
      />
      <div className="flex min-w-0 flex-1 flex-col" inert={sidebarOpen || undefined}>
        <header className="sticky top-0 z-30 border-b border-aura-border bg-white px-4 py-3 sm:px-6">
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <button
                type="button"
                onClick={() => setSidebarOpen(true)}
                className="inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface lg:hidden"
                aria-label="Abrir menu"
                aria-expanded={sidebarOpen}
                aria-controls="profissional-mobile-drawer"
              >
                <Menu className="h-5 w-5" />
              </button>
              <div className="min-w-0">
                {title && (
                  <h1 className="truncate font-display text-lg font-semibold text-aura-primary-dark sm:text-xl">
                    {title}
                  </h1>
                )}
                {subtitle && (
                  <p className="truncate text-xs text-aura-muted sm:text-sm">{subtitle}</p>
                )}
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-3">
              {headerAction}
              <div className="hidden text-right sm:block">
                <p className="text-sm font-medium text-aura-anthracite">{name}</p>
                <p className="text-xs text-aura-muted">Profissional</p>
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
      <BottomNav variant="profissional" drawerOpen={sidebarOpen} />
    </div>
  )
}

export { GLASS as ProfissionalGLASS }
