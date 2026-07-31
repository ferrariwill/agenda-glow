import { forwardRef } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Calendar, LayoutDashboard, LogOut } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useLogout } from '../../hooks/useLogout'

const nav = [
  { to: '/profissional/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/profissional/agenda', label: 'Minha Agenda', icon: Calendar },
]

interface Props {
  mobileOpen?: boolean
  onNavigate?: () => void
}

export const ProfissionalSidebar = forwardRef<HTMLElement, Props>(
  function ProfissionalSidebar({ mobileOpen = false, onNavigate }, ref) {
    const { session } = useAuth()
    const handleLogout = useLogout()
    const location = useLocation()

    return (
      <aside
        ref={ref}
        id="profissional-mobile-drawer"
        className={[
          'fixed inset-y-0 left-0 z-50 flex h-screen w-64 flex-col border-r border-aura-border bg-white shadow-md transition-transform duration-200 ease-out',
          'lg:static lg:z-auto lg:shrink-0 lg:translate-x-0',
          mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        ].join(' ')}
      >
        <div className="border-b border-aura-border px-6 py-6">
          <p className="font-display text-xl font-semibold text-aura-primary-dark">AgendaGlow</p>
          <p className="mt-0.5 text-[11px] font-medium uppercase tracking-wider text-aura-muted">
            Área do Profissional
          </p>
        </div>

        <nav className="flex-1 space-y-0.5 overflow-y-auto p-3" aria-label="Menu lateral">
          {nav.map(({ to, label, icon: Icon }) => {
            const active = location.pathname === to || location.pathname.startsWith(`${to}/`)
            return (
              <NavLink
                key={to}
                to={to}
                onClick={onNavigate}
                className={[
                  'relative flex min-h-touch-min touch-manipulation items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                  active
                    ? 'border-r-4 border-aura-primary bg-aura-primary/10 font-semibold text-aura-primary-dark'
                    : 'text-aura-muted hover:bg-aura-surface',
                ].join(' ')}
              >
                <Icon className="h-5 w-5 shrink-0" />
                {label}
              </NavLink>
            )
          })}
        </nav>

        <div className="border-t border-aura-border p-3">
          <p className="mb-2 truncate px-3 text-xs text-aura-muted">{session?.user.nome}</p>
          <button
            type="button"
            onClick={handleLogout}
            className="flex min-h-touch-min w-full touch-manipulation items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-aura-muted transition-colors hover:bg-aura-surface"
          >
            <LogOut className="h-5 w-5" />
            Sair
          </button>
        </div>
      </aside>
    )
  },
)
