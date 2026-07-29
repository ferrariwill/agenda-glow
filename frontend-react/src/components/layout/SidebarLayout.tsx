import { BottomNav } from './BottomNav'
import { AppHeader } from './AppHeader'
import { SideNavBar } from './SideNavBar'
import { useAuth } from '../../contexts/AuthContext'

interface SidebarLayoutProps {
  children: React.ReactNode
  title?: string
}

export function SidebarLayout({ children, title }: SidebarLayoutProps) {
  const { session } = useAuth()
  const variant = session?.user.role === 'SUPER_ADMIN' ? 'superadmin' : 'dona'

  return (
    <div className="min-h-screen bg-aura-surface">
      <SideNavBar />
      <div className="lg:pl-64">
        <AppHeader title={title} />
        <main className="px-4 py-6 pb-24 lg:px-8 lg:pb-8">{children}</main>
      </div>
      <BottomNav variant={variant} />
    </div>
  )
}
