import { useCallback, useRef, useState, type ReactNode } from 'react'
import { BottomNav } from '../layout/BottomNav'
import { useMobileDrawer } from '../../hooks/useMobileDrawer'
import { SuperAdminSidebar } from './SuperAdminSidebar'
import { SuperAdminTopBar } from './SuperAdminTopBar'

interface SuperAdminLayoutProps {
  children: ReactNode
  searchPlaceholder?: string
  searchValue?: string
  onSearchChange?: (v: string) => void
  onRenewAll?: () => void
  renewLoading?: boolean
}

export function SuperAdminLayout({
  children,
  searchPlaceholder,
  searchValue: externalSearch,
  onSearchChange: externalOnSearch,
  onRenewAll,
  renewLoading,
}: SuperAdminLayoutProps) {
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
      <SuperAdminSidebar
        ref={panelRef}
        mobileOpen={sidebarOpen}
        onNavigate={closeSidebar}
      />
      <div className="flex min-w-0 flex-1 flex-col" inert={sidebarOpen || undefined}>
        <SuperAdminTopBar
          searchPlaceholder={searchPlaceholder}
          searchValue={searchValue}
          onSearchChange={onSearchChange}
          onRenewAll={onRenewAll}
          renewLoading={renewLoading}
          onMenuClick={() => setSidebarOpen(true)}
          menuOpen={sidebarOpen}
        />
        <main className="flex-1 px-4 py-5 pb-24 sm:px-6 sm:py-6 lg:px-8 lg:pb-8">
          {children}
        </main>
      </div>
      <BottomNav variant="superadmin" drawerOpen={sidebarOpen} />
    </div>
  )
}

export { SuperAdminFooter, PageHeader, MetricCard, TablePagination } from './SuperAdminUI'
