import { useEffect, useState, type ReactNode } from 'react'
import { BottomNav } from '../layout/BottomNav'
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
      <DonaSidebar mobileOpen={sidebarOpen} onNavigate={closeSidebar} />
      <div className="flex min-w-0 flex-1 flex-col">
        <DonaTopBar
          searchPlaceholder={searchPlaceholder}
          searchValue={searchValue}
          onSearchChange={onSearchChange}
          onMenuClick={() => setSidebarOpen(true)}
          action={headerAction}
        />
        <main className="flex-1 px-4 py-5 pb-24 sm:px-6 sm:py-6 lg:px-8 lg:pb-8">
          {children}
        </main>
      </div>
      <BottomNav variant="dona" />
    </div>
  )
}

export { PageHeader, MetricCard, TablePagination, SuperAdminFooter as DonaFooter } from '../superadmin/SuperAdminLayout'
