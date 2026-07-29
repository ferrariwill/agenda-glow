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

export function ProfissionalSidebar({ mobileOpen = false, onNavigate }: Props) {
  const { session } = useAuth()
  const handleLogout = useLogout()
  const location = useLocation()

  return (
    <aside
      className={[
        'fixed inset-y-0 left-0 z-50 flex h-screen w-64 flex-col border-r border-[#e5d3c8]/30 bg-[#faf9f8] shadow-md transition-transform duration-200 ease-out',
        'lg:static lg:z-auto lg:shrink-0 lg:translate-x-0',
        mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
      ].join(' ')}
    >
      <div className="border-b border-[#e5d3c8]/30 px-6 py-6">
        <p className="font-display text-xl font-semibold text-[#7d5141]">AgendaGlow</p>
        <p className="mt-0.5 text-[11px] font-medium uppercase tracking-wider text-[#514440]">
          Área do Profissional
        </p>
      </div>

      <nav className="flex-1 space-y-0.5 overflow-y-auto p-3">
        {nav.map(({ to, label, icon: Icon }) => {
          const active = location.pathname === to || location.pathname.startsWith(`${to}/`)
          return (
            <NavLink
              key={to}
              to={to}
              onClick={onNavigate}
              className={[
                'relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                active
                  ? 'border-r-4 border-[#7d5141] bg-[#996958]/20 font-semibold text-[#7d5141]'
                  : 'text-[#514440] hover:bg-[#efdcd1]/30',
              ].join(' ')}
            >
              <Icon className="h-5 w-5 shrink-0" />
              {label}
            </NavLink>
          )
        })}
      </nav>

      <div className="border-t border-[#e5d3c8]/30 p-3">
        <p className="mb-2 truncate px-3 text-xs text-[#514440]">{session?.user.nome}</p>
        <button
          type="button"
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm text-[#514440] transition-colors hover:bg-[#efdcd1]/30"
        >
          <LogOut className="h-5 w-5" />
          Sair
        </button>
      </div>
    </aside>
  )
}
