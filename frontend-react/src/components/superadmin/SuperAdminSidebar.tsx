import { NavLink, useLocation } from 'react-router-dom'
import {
  Building2,
  CreditCard,
  LayoutDashboard,
  LogOut,
  Settings,
  Sparkles,
  Users,
  Wallet,
} from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useLogout } from '../../hooks/useLogout'

const nav = [
  { to: '/superadmin/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/superadmin/clientes', label: 'Clientes', icon: Users },
  { to: '/superadmin/saloes', label: 'Salões', icon: Building2 },
  { to: '/superadmin/planos', label: 'Planos', icon: Sparkles },
  { to: '/superadmin/financeiro', label: 'Financeiro', icon: Wallet },
]

interface SuperAdminSidebarProps {
  mobileOpen?: boolean
  onNavigate?: () => void
}

export function SuperAdminSidebar({ mobileOpen = false, onNavigate }: SuperAdminSidebarProps) {
  const { session } = useAuth()
  const handleLogout = useLogout()
  const location = useLocation()

  return (
    <aside
      className={[
        'fixed inset-y-0 left-0 z-50 flex h-screen w-64 flex-col border-r border-aura-border bg-white transition-transform duration-200 ease-out',
        'lg:static lg:z-auto lg:shrink-0 lg:translate-x-0',
        mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
      ].join(' ')}
    >
      <div className="border-b border-aura-border px-6 py-6">
        <p className="font-display text-xl font-semibold text-aura-anthracite">Aura Beauty</p>
        <p className="mt-0.5 text-[11px] font-medium uppercase tracking-wider text-aura-muted">
          Admin Console
        </p>
      </div>

      <nav className="flex-1 space-y-0.5 p-3">
        {nav.map(({ to, label, icon: Icon }) => {
          const active = location.pathname === to
          return (
            <NavLink
              key={to}
              to={to}
              onClick={onNavigate}
              className={[
                'relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                active
                  ? 'bg-aura-primary/8 text-aura-primary-dark'
                  : 'text-aura-muted hover:bg-aura-surface hover:text-aura-anthracite',
              ].join(' ')}
            >
              {active && (
                <span className="absolute left-0 top-1/2 h-8 w-1 -translate-y-1/2 rounded-r-full bg-aura-primary" />
              )}
              <Icon className="h-[18px] w-[18px]" />
              {label}
            </NavLink>
          )
        })}
      </nav>

      <div className="space-y-2 border-t border-aura-border p-4">
        <NavLink
          to="/superadmin/planos"
          className="flex w-full items-center justify-center rounded-lg bg-aura-primary px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-aura-primary-dark"
        >
          <CreditCard className="mr-2 h-4 w-4" />
          Upgrade Plan
        </NavLink>
        <button
          type="button"
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-aura-muted hover:bg-aura-surface"
        >
          <Settings className="h-[18px] w-[18px]" />
          Configurações
        </button>
        <button
          type="button"
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-aura-muted hover:bg-aura-surface"
        >
          <LogOut className="h-[18px] w-[18px]" />
          Sair
        </button>
        <p className="truncate px-3 text-[10px] text-aura-muted">{session?.user.email}</p>
      </div>
    </aside>
  )
}
