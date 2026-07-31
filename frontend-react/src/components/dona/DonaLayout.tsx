import { useCallback, useRef, useState, type ReactNode } from 'react'
import { BottomNav } from '../layout/BottomNav'
import { useMobileDrawer } from '../../hooks/useMobileDrawer'
import { DonaSidebar } from './DonaSidebar'
import { DonaTopBar } from './DonaTopBar'

interface DonaLayoutProps {
  children: ReactNode
  searchPlaceholder?: string
  searchValue?: string
  onSearchChange?: (v: string) => void
  headerAction?: ReactNode
}

export function DonaLayout({
  children,
  searchPlaceholder,
  searchValue: externalSearch,
  onSearchChange: externalOnSearch,
  headerAction,
}: DonaLayoutProps) {
  const [internalSearch, setInternalSearch] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const panelRef = useRef<HTMLElement>(null)
  const searchValue = externalSearch ?? internalSearch
  const onSearchChange = externalOnSearch ?? setInternalSearch
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
      <DonaSidebar
        ref={panelRef}
        mobileOpen={sidebarOpen}
        onNavigate={closeSidebar}
      />
      <div className="flex min-w-0 flex-1 flex-col" inert={sidebarOpen || undefined}>
        <DonaTopBar
          searchPlaceholder={searchPlaceholder}
          searchValue={searchValue}
          onSearchChange={onSearchChange}
          onMenuClick={() => setSidebarOpen(true)}
          menuOpen={sidebarOpen}
          action={headerAction}
        />
        <main className="flex-1 px-4 py-5 pb-24 sm:px-6 sm:py-6 lg:px-8 lg:pb-8">
          {children}
        </main>
      </div>
      <BottomNav variant="dona" drawerOpen={sidebarOpen} />
    </div>
  )
}

export { PageHeader, MetricCard, TablePagination, SuperAdminFooter as DonaFooter } from '../superadmin/SuperAdminLayout'
