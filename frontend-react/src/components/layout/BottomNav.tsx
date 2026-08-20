import { useMemo, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import {
  Building2,
  Calendar,
  CalendarDays,
  LayoutDashboard,
  MessageCircle,
  MoreHorizontal,
  Package,
  Scissors,
  Settings,
  Sparkles,
  UserCircle,
  Users,
  Wallet,
  type LucideIcon,
} from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { MoreSheet, type MoreSheetItem } from './MoreSheet'

export type BottomNavVariant = 'dona' | 'secretaria' | 'profissional' | 'superadmin'

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
}

const donaPrimary: NavItem[] = [
  { to: '/admin/dashboard', label: 'Início', icon: LayoutDashboard },
  { to: '/admin/calendario', label: 'Agenda', icon: Calendar },
  { to: '/admin/clientes', label: 'Clientes', icon: UserCircle },
  { to: '/admin/financeiro', label: 'Caixa', icon: Wallet },
]

const donaMoreBase: MoreSheetItem[] = [
  { to: '/admin/servicos', label: 'Serviços', icon: Scissors },
  { to: '/admin/equipe', label: 'Equipe', icon: Users },
  { to: '/admin/insumos', label: 'Insumos', icon: Package },
  { to: '/admin/especialidades', label: 'Especialidades', icon: Sparkles },
  { to: '/admin/whatsapp', label: 'WhatsApp', icon: MessageCircle },
  { to: '/admin/configuracoes', label: 'Configurações', icon: Settings },
]

const secretariaPrimary: NavItem[] = [
  { to: '/secretaria/agenda', label: 'Agenda', icon: Calendar },
  { to: '/secretaria/clientes', label: 'Clientes', icon: Users },
]

const profissionalPrimary: NavItem[] = [
  { to: '/profissional/dashboard', label: 'Início', icon: LayoutDashboard },
  { to: '/profissional/agenda', label: 'Agenda', icon: Calendar },
]

const superPrimary: NavItem[] = [
  { to: '/superadmin/dashboard', label: 'Início', icon: LayoutDashboard },
  { to: '/superadmin/saloes', label: 'Salões', icon: Building2 },
  { to: '/superadmin/planos', label: 'Planos', icon: Sparkles },
  { to: '/superadmin/financeiro', label: 'Caixa', icon: Wallet },
]

const ariaLabelByVariant: Record<BottomNavVariant, string> = {
  dona: 'Navegação principal — painel da dona',
  secretaria: 'Navegação principal — secretaria',
  profissional: 'Navegação principal — profissional',
  superadmin: 'Navegação principal — super admin',
}

function pathMatches(pathname: string, to: string) {
  return pathname === to || pathname.startsWith(`${to}/`)
}

interface BottomNavProps {
  variant: BottomNavVariant
  /** When the hamburger drawer is open, nav becomes inert / non-interactive. */
  drawerOpen?: boolean
}

export function BottomNav({ variant, drawerOpen = false }: BottomNavProps) {
  const { session } = useAuth()
  const role = session?.user.role
  const location = useLocation()
  const [moreOpen, setMoreOpen] = useState(false)

  const moreItems = useMemo((): MoreSheetItem[] => {
    if (variant !== 'dona') return []
    if (session?.user.profissional_id) {
      return [
        { to: '/admin/minha-agenda', label: 'Minha agenda', icon: CalendarDays },
        ...donaMoreBase,
      ]
    }
    return donaMoreBase
  }, [variant, session?.user.profissional_id])

  if (role === 'CLIENTE') return null

  const primary =
    variant === 'superadmin'
      ? superPrimary
      : variant === 'secretaria'
        ? secretariaPrimary
        : variant === 'profissional'
          ? profissionalPrimary
          : donaPrimary

  const showMore = moreItems.length > 0
  const moreActive = moreItems.some((item) => pathMatches(location.pathname, item.to))

  return (
    <>
      <nav
        className="fixed bottom-0 left-0 right-0 z-40 border-t border-aura-border bg-white pb-[max(0.5rem,env(safe-area-inset-bottom))] lg:hidden"
        aria-label={ariaLabelByVariant[variant]}
        inert={drawerOpen || undefined}
        aria-hidden={drawerOpen || undefined}
      >
        <div className="flex justify-around px-1 pt-1">
          {primary.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                [
                  'flex min-h-touch-min min-w-touch-min flex-1 touch-manipulation flex-col items-center justify-center gap-0.5 rounded-lg px-1 text-xs font-medium',
                  isActive ? 'text-aura-primary' : 'text-aura-muted',
                ].join(' ')
              }
            >
              <Icon className="h-5 w-5 shrink-0" aria-hidden />
              <span className="truncate">{label}</span>
            </NavLink>
          ))}

          {showMore && (
            <button
              type="button"
              onClick={() => setMoreOpen(true)}
              className={[
                'flex min-h-touch-min min-w-touch-min flex-1 touch-manipulation flex-col items-center justify-center gap-0.5 rounded-lg px-1 text-xs font-medium',
                moreActive || moreOpen ? 'text-aura-primary' : 'text-aura-muted',
              ].join(' ')}
              aria-haspopup="dialog"
              aria-expanded={moreOpen}
              aria-current={moreActive ? 'page' : undefined}
            >
              <MoreHorizontal className="h-5 w-5 shrink-0" aria-hidden />
              <span className="truncate">Mais</span>
            </button>
          )}
        </div>
      </nav>

      {showMore && (
        <MoreSheet open={moreOpen} onClose={() => setMoreOpen(false)} items={moreItems} />
      )}
    </>
  )
}
