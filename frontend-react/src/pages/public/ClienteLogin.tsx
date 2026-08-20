import { useEffect, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { ClienteLoginForm } from '../../components/public/ClienteLoginForm'
import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'
import { useAuth } from '../../contexts/AuthContext'
import {
  ClienteAuthError,
  ensureClienteNoTenant,
  findContaClienteByEmail,
  getTenantBySlug,
} from '../../utils/mockDb'

export function ClienteLogin() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { loginCliente, session } = useAuth()
  const tenant = slug ? getTenantBySlug(slug) : undefined

  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const from = (location.state as { from?: string } | null)?.from ?? `/${slug}/agendar`

  useEffect(() => {
    if (session?.user.role === 'CLIENTE') {
      navigate(from, { replace: true })
    }
  }, [session, from, navigate])

  if (session?.user.role === 'CLIENTE') {
    return null
  }

  if (!tenant || tenant.status !== 'ATIVO') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8] p-4">
        <p className="text-[#514440]">Salão não encontrado ou indisponível.</p>
      </div>
    )
  }

  const handleSubmit = async (values: { email: string; password: string }) => {
    setError('')
    setLoading(true)
    try {
      await loginCliente(values.email, values.password, slug!)
      const conta = findContaClienteByEmail(values.email)
      if (conta) {
        ensureClienteNoTenant(tenant.id, {
          nome: conta.nome,
          telefone: conta.telefone,
          email: conta.email,
        })
      }
      navigate(from, { replace: true })
    } catch (err) {
      if (err instanceof ClienteAuthError) setError(err.message)
      else setError(err instanceof Error ? err.message : 'Falha no login')
    } finally {
      setLoading(false)
    }
  }

  return (
    <ClienteSalaoLayout tenant={tenant} backTo={`/${slug}`} showAccount={false}>
      <div className="rounded-xl border border-[#e5d3c8]/40 bg-white/90 p-6 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md">
        <h1 className="font-display text-2xl font-semibold text-[#1a1c1c]">Entrar</h1>
        <p className="mt-1 text-sm text-[#514440]">
          Acesse sua conta para agendar em {tenant.nome}.
        </p>

        <ClienteLoginForm
          tenantId={tenant.id}
          tenantNome={tenant.nome}
          cadastroTo={`/${slug}/cadastro`}
          cadastroState={{ from }}
          loading={loading}
          error={error}
          onSubmit={handleSubmit}
        />
      </div>
    </ClienteSalaoLayout>
  )
}
