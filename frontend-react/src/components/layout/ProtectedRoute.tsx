import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import type { UserRole } from '../../types'
import { getDb } from '../../utils/mockDb'
import { SubscriptionBlocked } from '../../pages/admin/SubscriptionBlocked'

interface ProtectedRouteProps {
  allowedRoles: UserRole[]
  checkSubscription?: boolean
}

export function ProtectedRoute({ allowedRoles, checkSubscription }: ProtectedRouteProps) {
  const { session, isAuthenticated } = useAuth()

  if (!isAuthenticated || !session) {
    return <Navigate to="/login" replace />
  }

  if (!allowedRoles.includes(session.user.role)) {
    return <Navigate to="/login" replace />
  }

  if (checkSubscription && session.user.tenant_id) {
    const tenant = getDb().tenants.find((t) => t.id === session.user.tenant_id)
    if (tenant?.status === 'VENCIDO') {
      return <SubscriptionBlocked />
    }
  }

  return <Outlet />
}
