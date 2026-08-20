import { useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { ClienteIdentificacaoForm } from '../../components/public/ClienteIdentificacaoForm'
import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'
import { useAuth } from '../../contexts/AuthContext'
import { ClienteAuthError, ensureClienteNoTenant, getTenantBySlug } from '../../utils/mockDb'

export function ClienteCadastro() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { registerOrLoginCliente } = useAuth()
  const tenant = slug ? getTenantBySlug(slug) : undefined

  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const from = (location.state as { from?: string } | null)?.from

  if (!tenant || tenant.status !== 'ATIVO') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8] p-4">
        <p className="text-[#514440]">Salão não encontrado ou indisponível.</p>
      </div>
    )
  }

  const handleSubmit = async (values: { nome: string; telefone: string; email: string }) => {
    setError('')
    setLoading(true)
    try {
      const fallback = await registerOrLoginCliente(values, slug!)
      ensureClienteNoTenant(tenant.id, values)
      navigate(from ?? fallback, { replace: true })
    } catch (err) {
      if (err instanceof ClienteAuthError) setError(err.message)
      else setError(err instanceof Error ? err.message : 'Erro ao cadastrar')
    } finally {
      setLoading(false)
    }
  }

  return (
    <ClienteSalaoLayout tenant={tenant} backTo={`/${slug}`} backLabel="Voltar ao salão" showAccount={false}>
      <div className="rounded-xl border border-[#e5d3c8]/40 bg-white/90 p-6 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md">
        <h1 className="font-display text-2xl font-semibold text-[#1a1c1c]">Seus dados</h1>
        <p className="mt-1 text-sm text-[#514440]">
          Informe nome, telefone e e-mail para agendar em {tenant.nome}.
        </p>

        <ClienteIdentificacaoForm
          tenantId={tenant.id}
          tenantNome={tenant.nome}
          loginTo={`/${slug}/login`}
          loginState={from ? { from } : undefined}
          loading={loading}
          error={error}
          onSubmit={handleSubmit}
        />
      </div>
    </ClienteSalaoLayout>
  )
}
