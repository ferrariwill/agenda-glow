import { useEffect, useState, type ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { Calendar, LayoutDashboard, Menu } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { initials } from '../../utils/format'
import { ProfissionalSidebar } from './ProfissionalSidebar'

const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white/70 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md'

const mobileNav = [
  { to: '/profissional/dashboard', label: 'Início', icon: LayoutDashboard },
  { to: '/profissional/agenda', label: 'Agenda', icon: Calendar },
]

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
  const name = session?.user.nome ?? 'Profissional'

  useEffect(() => {
    if (!sidebarOpen) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [sidebarOpen])

  return (
    <div className="min-h-screen bg-[#faf9f8] lg:flex">
      {sidebarOpen && (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-[#1a1c1c]/40 lg:hidden"
          onClick={() => setSidebarOpen(false)}
          aria-label="Fechar menu"
        />
      )}
      <ProfissionalSidebar
        mobileOpen={sidebarOpen}
        onNavigate={() => setSidebarOpen(false)}
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 border-b border-[#e5d3c8]/30 bg-white px-4 py-3 sm:px-6">
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <button
                type="button"
                onClick={() => setSidebarOpen(true)}
                className="rounded-lg p-2 text-[#514440] hover:bg-[#f4f3f2] lg:hidden"
                aria-label="Abrir menu"
              >
                <Menu className="h-5 w-5" />
              </button>
              <div className="min-w-0">
                {title && (
                  <h1 className="truncate font-display text-lg font-semibold text-[#7d5141] sm:text-xl">
                    {title}
                  </h1>
                )}
                {subtitle && (
                  <p className="truncate text-xs text-[#514440] sm:text-sm">{subtitle}</p>
                )}
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-3">
              {headerAction}
              <div className="hidden text-right sm:block">
                <p className="text-sm font-medium text-[#1a1c1c]">{name}</p>
                <p className="text-xs text-[#514440]">Profissional</p>
              </div>
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-[#ffdbcf] text-xs font-semibold text-[#7d5141]">
                {initials(name)}
              </div>
            </div>
          </div>
        </header>
        <main className="flex-1 px-4 py-5 pb-24 sm:px-6 sm:py-6 lg:px-8 lg:pb-8">
          {children}
        </main>
      </div>
      <nav className="fixed bottom-0 left-0 right-0 z-40 border-t border-[#e5d3c8]/30 bg-white pb-[max(0.5rem,env(safe-area-inset-bottom))] lg:hidden">
        <div className="flex justify-around px-1 pt-1">
          {mobileNav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                [
                  'flex min-w-0 flex-1 flex-col items-center gap-0.5 rounded-lg px-1 py-1.5 text-[10px] font-medium',
                  isActive ? 'text-[#7d5141]' : 'text-[#514440]',
                ].join(' ')
              }
            >
              <Icon className="h-5 w-5 shrink-0" />
              <span className="truncate">{label}</span>
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  )
}

export { GLASS as ProfissionalGLASS }
