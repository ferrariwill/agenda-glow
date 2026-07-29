import { useEffect, useState, type ReactNode } from 'react'
import { BottomNav } from '../layout/BottomNav'
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
  const searchValue = externalSearch ?? internalSearch
  const onSearchChange = externalOnSearch ?? setInternalSearch
  const closeSidebar = () => setSidebarOpen(false)

  useEffect(() => {
    if (!sidebarOpen) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [sidebarOpen])

  return (
    <div className="min-h-screen bg-aura-surface lg:flex">
      {sidebarOpen && (
        <button
          type="button"
          className="fixed inset-0 z-40 bg-aura-anthracite/40 lg:hidden"
          onClick={closeSidebar}
          aria-label="Fechar menu"
        />
      )}
      <SuperAdminSidebar mobileOpen={sidebarOpen} onNavigate={closeSidebar} />
      <div className="flex min-w-0 flex-1 flex-col">
        <SuperAdminTopBar
          searchPlaceholder={searchPlaceholder}
          searchValue={searchValue}
          onSearchChange={onSearchChange}
          onRenewAll={onRenewAll}
          renewLoading={renewLoading}
          onMenuClick={() => setSidebarOpen(true)}
        />
        <main className="flex-1 px-4 py-5 pb-24 sm:px-6 sm:py-6 lg:px-8 lg:pb-8">
          {children}
        </main>
      </div>
      <BottomNav variant="superadmin" />
    </div>
  )
}

export { SuperAdminFooter, PageHeader, MetricCard, TablePagination } from './SuperAdminUI'
