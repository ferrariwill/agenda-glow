import { NavLink } from 'react-router-dom'
import {
  Building2,
  Calendar,
  LayoutDashboard,
  Settings,
  Sparkles,
  UserCircle,
  Users,
  Wallet,
} from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'

const donaMobile = [
  { to: '/admin/dashboard', label: 'Início', icon: LayoutDashboard },
  { to: '/admin/calendario', label: 'Agenda', icon: Calendar },
  { to: '/admin/clientes', label: 'Clientes', icon: UserCircle },
  { to: '/admin/financeiro', label: 'Caixa', icon: Wallet },
  { to: '/admin/configuracoes', label: 'Mais', icon: Settings },
]

const superMobile = [
  { to: '/superadmin/dashboard', label: 'Início', icon: LayoutDashboard },
  { to: '/superadmin/saloes', label: 'Salões', icon: Building2 },
  { to: '/superadmin/clientes', label: 'Clientes', icon: Users },
  { to: '/superadmin/planos', label: 'Planos', icon: Sparkles },
  { to: '/superadmin/financeiro', label: 'Caixa', icon: Wallet },
]

interface BottomNavProps {
  variant: 'dona' | 'superadmin'
}

export function BottomNav({ variant }: BottomNavProps) {
  const { session } = useAuth()
  const role = session?.user.role
  if (role === 'PROFISSIONAL' || role === 'CLIENTE') return null

  const links = variant === 'superadmin' ? superMobile : donaMobile

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 border-t border-aura-border bg-white pb-[max(0.5rem,env(safe-area-inset-bottom))] lg:hidden">
      <div className="flex justify-around px-1 pt-1">
        {links.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              [
                'flex min-w-0 flex-1 flex-col items-center gap-0.5 rounded-lg px-1 py-1.5 text-[10px] font-medium',
                isActive ? 'text-aura-primary' : 'text-aura-muted',
              ].join(' ')
            }
          >
            <Icon className="h-5 w-5 shrink-0" />
            <span className="truncate">{label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
