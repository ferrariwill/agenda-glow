import { NavLink } from 'react-router-dom'
import {
  Building2,
  Calendar,
  CreditCard,
  LayoutDashboard,
  LogOut,
  Scissors,
  Sparkles,
  Users,
  Wallet,
} from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useLogout } from '../../hooks/useLogout'

const donaLinks = [
  { to: '/admin/calendario', label: 'Agenda', icon: Calendar },
  { to: '/admin/equipe', label: 'Equipe', icon: Users },
  { to: '/admin/financeiro', label: 'Financeiro', icon: Wallet },
  { to: '/admin/comissoes', label: 'Comissões', icon: CreditCard },
]

const superAdminLinks = [
  { to: '/superadmin/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/superadmin/saloes', label: 'Salões', icon: Building2 },
  { to: '/superadmin/planos', label: 'Planos SaaS', icon: Sparkles },
]

export function SideNavBar() {
  const { session } = useAuth()
  const handleLogout = useLogout()
  const role = session?.user.role
  const links = role === 'SUPER_ADMIN' ? superAdminLinks : donaLinks

  if (role === 'PROFISSIONAL') return null

  return (
    <aside className="fixed left-0 top-0 z-40 hidden h-full w-64 flex-col border-r border-aura-border bg-white lg:flex">
      <div className="border-b border-aura-border px-6 py-6">
        <p className="font-display text-xl font-semibold text-aura-primary">AgendaGlow</p>
        <p className="mt-1 text-xs text-aura-muted">Aura Beauty</p>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {links.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              [
                'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-aura-primary/10 text-aura-primary-dark'
                  : 'text-aura-muted hover:bg-aura-surface hover:text-aura-anthracite',
              ].join(' ')
            }
          >
            <Icon className="h-5 w-5" />
            {label}
          </NavLink>
        ))}
        {role === 'DONA' && (
          <NavLink
            to="/admin/servicos"
            className={({ isActive }) =>
              [
                'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-aura-primary/10 text-aura-primary-dark'
                  : 'text-aura-muted hover:bg-aura-surface hover:text-aura-anthracite',
              ].join(' ')
            }
          >
            <Scissors className="h-5 w-5" />
            Serviços
          </NavLink>
        )}
      </nav>
      <div className="border-t border-aura-border p-4">
        <p className="mb-2 truncate text-xs text-aura-muted">{session?.user.email}</p>
        <button
          type="button"
          onClick={handleLogout}
          className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-aura-muted hover:bg-aura-surface"
        >
          <LogOut className="h-4 w-4" />
          Sair
        </button>
      </div>
    </aside>
  )
}
