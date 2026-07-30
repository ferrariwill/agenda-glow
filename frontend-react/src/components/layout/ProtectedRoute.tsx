import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { useBootstrapState } from '../../data/BootstrapContext'
import { IS_MOCK } from '../../lib/config'
import type { UserRole } from '../../types'
import { getDb } from '../../utils/mockDb'
import { resolveSubscriptionGate } from '../../utils/subscriptionGate'
import { SubscriptionBlocked } from '../../pages/admin/SubscriptionBlocked'
import { DataUnavailable } from './DataUnavailable'

interface ProtectedRouteProps {
  allowedRoles: UserRole[]
  checkSubscription?: boolean
}

export function ProtectedRoute({ allowedRoles, checkSubscription }: ProtectedRouteProps) {
  const { session, isAuthenticated } = useAuth()
  const { phase } = useBootstrapState()

  if (!isAuthenticated || !session) {
    return <Navigate to="/login" replace />
  }

  if (!allowedRoles.includes(session.user.role)) {
    return <Navigate to="/login" replace />
  }

  if (checkSubscription) {
    const tenantId = session.user.tenant_id
    const tenant = tenantId ? getDb().tenants.find((t) => t.id === tenantId) : undefined
    const decision = resolveSubscriptionGate({
      phase,
      tenantStatus: tenant?.status,
      mock: IS_MOCK,
    })
    if (decision === 'blocked') return <SubscriptionBlocked />
    if (decision === 'unavailable') return <DataUnavailable />
  }

  return <Outlet />
}
