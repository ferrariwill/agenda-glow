import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { Lock, Mail } from 'lucide-react'
import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { useAuth } from '../../contexts/AuthContext'
import { ClienteAuthError, ensureClienteNoTenant, findContaClienteByEmail, getTenantBySlug } from '../../utils/mockDb'

export function ClienteLogin() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { loginCliente, session } = useAuth()
  const tenant = slug ? getTenantBySlug(slug) : undefined

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
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
      <div className="flex min-h-screen items-center justify-center p-4">
        <p className="text-aura-muted">Salão não encontrado ou indisponível.</p>
      </div>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await loginCliente(email, password, slug!)
      const conta = findContaClienteByEmail(email)
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
      <div className="rounded-xl border border-aura-border bg-white p-6 shadow-sm">
        <h1 className="font-display text-2xl font-semibold text-aura-anthracite">Entrar</h1>
        <p className="mt-1 text-sm text-aura-muted">
          Acesse sua conta para agendar em {tenant.nome}.
        </p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          {error && <Alert variant="error">{error}</Alert>}

          <div>
            <label className="mb-1.5 block text-sm font-medium">E-mail</label>
            <div className="relative">
              <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-aura-muted" />
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="w-full rounded-lg border border-aura-border py-2.5 pl-10 pr-3 text-sm"
                placeholder="seu@email.com"
              />
            </div>
          </div>

          <div>
            <label className="mb-1.5 block text-sm font-medium">Senha</label>
            <div className="relative">
              <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-aura-muted" />
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-aura-border py-2.5 pl-10 pr-3 text-sm"
                placeholder="••••••••"
              />
            </div>
          </div>

          <Button type="submit" fullWidth loading={loading}>
            Entrar e agendar
          </Button>
        </form>

        <p className="mt-6 text-center text-sm text-aura-muted">
          Primeira vez aqui?{' '}
          <Link to={`/${slug}/cadastro`} className="font-semibold text-aura-primary hover:underline">
            Criar conta
          </Link>
        </p>

        <p className="mt-3 rounded-lg bg-aura-surface px-3 py-2 text-xs text-aura-muted">
          Demo: mariana@email.com / AgendaGlow@2026 (já cliente em outro salão)
        </p>
      </div>
    </ClienteSalaoLayout>
  )
}
