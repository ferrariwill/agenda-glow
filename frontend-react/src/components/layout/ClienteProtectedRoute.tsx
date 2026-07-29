import { Navigate, Outlet, useLocation, useParams } from 'react-router-dom'
import { useAuth } from '../../contexts/AuthContext'
import { getTenantBySlug } from '../../utils/mockDb'

export function ClienteProtectedRoute() {
  const { slug } = useParams<{ slug: string }>()
  const { session, isAuthenticated } = useAuth()
  const location = useLocation()
  const tenant = slug ? getTenantBySlug(slug) : undefined

  if (!tenant || tenant.status !== 'ATIVO') {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <p className="text-aura-muted">Salão não encontrado ou indisponível.</p>
      </div>
    )
  }

  if (!isAuthenticated || !session || session.user.role !== 'CLIENTE') {
    return (
      <Navigate
        to={`/${slug}/cadastro`}
        replace
        state={{ from: location.pathname }}
      />
    )
  }

  return <Outlet />
}
