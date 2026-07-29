import { useState } from 'react'

import { Link, useNavigate, useParams } from 'react-router-dom'

import { Alert } from '../../components/ui/Alert'

import { Button } from '../../components/ui/Button'

import { Input } from '../../components/ui/Input'

import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'

import { useAuth } from '../../contexts/AuthContext'

import {

  ClienteAuthError,

  ensureClienteNoTenant,

  getTenantBySlug,

  lookupPerfilGlobal,

} from '../../utils/mockDb'



export function ClienteCadastro() {

  const { slug } = useParams<{ slug: string }>()

  const navigate = useNavigate()

  const { registerOrLoginCliente } = useAuth()

  const tenant = slug ? getTenantBySlug(slug) : undefined



  const [nome, setNome] = useState('')

  const [telefone, setTelefone] = useState('')

  const [email, setEmail] = useState('')

  const [info, setInfo] = useState('')

  const [error, setError] = useState('')

  const [loading, setLoading] = useState(false)



  if (!tenant || tenant.status !== 'ATIVO') {

    return (

      <div className="flex min-h-screen items-center justify-center bg-[#faf9f8] p-4">

        <p className="text-[#514440]">Salão não encontrado ou indisponível.</p>

      </div>

    )

  }



  const handleTelefoneBlur = () => {

    const perfil = lookupPerfilGlobal(telefone)

    if (!perfil) {

      setInfo('')

      return

    }

    setNome(perfil.nome)

    if (perfil.email) setEmail(perfil.email)

    setInfo(

      perfil.hasConta

        ? 'Bem-vinda de volta! Confirme seus dados e continue para agendar.'

        : 'Encontramos seu cadastro em outro salão — dados preenchidos automaticamente.',

    )

  }



  const handleSubmit = async (e: React.FormEvent) => {

    e.preventDefault()

    setError('')

    setLoading(true)

    try {

      const redirect = await registerOrLoginCliente({ nome, telefone, email }, slug!)

      ensureClienteNoTenant(tenant.id, { nome, telefone, email })

      navigate(redirect, { replace: true })

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



        <form onSubmit={handleSubmit} className="mt-6 space-y-4">

          {error && <Alert variant="error">{error}</Alert>}

          {info && <Alert variant="info">{info}</Alert>}



          <Input

            label="WhatsApp (com DDD)"

            value={telefone}

            onChange={(e) => setTelefone(e.target.value)}

            onBlur={handleTelefoneBlur}

            placeholder="(11) 99999-9999"

            required

          />

          <Input label="Nome completo" value={nome} onChange={(e) => setNome(e.target.value)} required />

          <Input

            label="E-mail"

            type="email"

            value={email}

            onChange={(e) => setEmail(e.target.value)}

            required

          />



          <Button type="submit" fullWidth loading={loading} className="bg-[#7d5141] hover:bg-[#996958]">

            Continuar para agendar

          </Button>

        </form>



        <p className="mt-4 text-center text-sm text-[#514440]">

          Já tem senha de acesso?{' '}

          <Link to={`/${slug}/login`} className="font-semibold text-[#7d5141] hover:underline">

            Entrar com e-mail

          </Link>

        </p>

      </div>

    </ClienteSalaoLayout>

  )

}


