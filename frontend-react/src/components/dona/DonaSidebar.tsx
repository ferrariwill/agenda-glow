import { forwardRef } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import {
  Calendar,
  CalendarDays,
  CreditCard,
  LayoutDashboard,
  LogOut,
  Package,
  Scissors,
  Settings,
  Sparkles,
  Users,
  UserCircle,
  Wallet,
  MessageCircle,
} from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useLogout } from '../../hooks/useLogout'

const baseNav = [
  { to: '/admin/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/admin/clientes', label: 'Clientes', icon: UserCircle },
  { to: '/admin/calendario', label: 'Agenda', icon: Calendar },
  { to: '/admin/insumos', label: 'Insumos', icon: Package },
  { to: '/admin/servicos', label: 'Serviços', icon: Scissors },
  { to: '/admin/equipe', label: 'Equipe', icon: Users },
  { to: '/admin/especialidades', label: 'Especialidades', icon: Sparkles },
  { to: '/admin/financeiro', label: 'Financeiro', icon: Wallet },
  { to: '/admin/whatsapp', label: 'WhatsApp', icon: MessageCircle },
  { to: '/admin/configuracoes', label: 'Configurações', icon: Settings },
]

interface DonaSidebarProps {
  mobileOpen?: boolean
  onNavigate?: () => void
}

export const DonaSidebar = forwardRef<HTMLElement, DonaSidebarProps>(
  function DonaSidebar({ mobileOpen = false, onNavigate }, ref) {
    const { session } = useAuth()
    const handleLogout = useLogout()
    const location = useLocation()

    const nav = session?.user.profissional_id
      ? [
          ...baseNav.slice(0, 3),
          { to: '/admin/minha-agenda', label: 'Minha agenda', icon: CalendarDays },
          ...baseNav.slice(3),
        ]
      : baseNav

    return (
      <aside
        ref={ref}
        id="dona-mobile-drawer"
        className={[
          'fixed inset-y-0 left-0 z-50 flex h-screen w-64 flex-col border-r border-aura-border bg-white transition-transform duration-200 ease-out',
          'lg:static lg:z-auto lg:shrink-0 lg:translate-x-0',
          mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        ].join(' ')}
      >
        <div className="border-b border-aura-border px-6 py-6">
          <p className="font-display text-xl font-semibold text-aura-anthracite">Aura Beauty</p>
          <p className="mt-0.5 text-[11px] font-medium uppercase tracking-wider text-aura-muted">
            Painel da Dona
          </p>
        </div>

        <nav className="flex-1 space-y-0.5 overflow-y-auto p-3" aria-label="Menu lateral">
          {nav.map(({ to, label, icon: Icon }) => {
            const active = location.pathname === to || location.pathname.startsWith(to + '/')
            return (
              <NavLink
                key={to}
                to={to}
                onClick={onNavigate}
                className={[
                  'relative flex min-h-touch-min touch-manipulation items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
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
            to="/admin/configuracoes#assinatura"
            onClick={onNavigate}
            className="flex min-h-touch-min w-full touch-manipulation items-center justify-center rounded-lg bg-aura-primary px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-aura-primary-dark"
          >
            <CreditCard className="mr-2 h-4 w-4" />
            Upgrade de plano
          </NavLink>
          <button
            type="button"
            onClick={handleLogout}
            className="flex min-h-touch-min w-full touch-manipulation items-center gap-3 rounded-lg px-3 py-2 text-sm text-aura-muted hover:bg-aura-surface"
          >
            <LogOut className="h-[18px] w-[18px]" />
            Sair
          </button>
          <p className="truncate px-3 text-[10px] text-aura-muted">{session?.user.email}</p>
        </div>
      </aside>
    )
  },
)
