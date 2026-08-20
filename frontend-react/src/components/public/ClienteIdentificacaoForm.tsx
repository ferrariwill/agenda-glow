import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { AlertaCadastroRede } from './AlertaCadastroRede'
import {
  listSaloesAtivosDaRede,
  lookupPerfilGlobal,
  type SalaoRedeResumo,
} from '../../utils/mockDb'

export interface ClienteIdentificacaoValues {
  nome: string
  telefone: string
  email: string
}

interface ClienteIdentificacaoFormProps {
  tenantId: string
  tenantNome: string
  loginTo: string
  loginState?: { from?: string }
  loading?: boolean
  error?: string
  onSubmit: (values: ClienteIdentificacaoValues) => void | Promise<void>
}

export function ClienteIdentificacaoForm({
  tenantId,
  tenantNome,
  loginTo,
  loginState,
  loading = false,
  error,
  onSubmit,
}: ClienteIdentificacaoFormProps) {
  const [nome, setNome] = useState('')
  const [telefone, setTelefone] = useState('')
  const [email, setEmail] = useState('')
  const [info, setInfo] = useState('')
  const [saloesOutros, setSaloesOutros] = useState<SalaoRedeResumo[]>([])

  const handleTelefoneBlur = () => {
    const perfil = lookupPerfilGlobal(telefone)
    const outros = listSaloesAtivosDaRede(telefone, tenantId)
    setSaloesOutros(outros)

    if (!perfil) {
      setInfo('')
      return
    }

    setNome(perfil.nome)
    if (perfil.email) setEmail(perfil.email)

    if (outros.length > 0) {
      setInfo('')
      return
    }

    setInfo(
      perfil.hasConta
        ? 'Bem-vinda de volta! Confirme seus dados e continue para agendar.'
        : 'Encontramos seu cadastro — dados preenchidos automaticamente.',
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await onSubmit({ nome, telefone, email })
  }

  return (
    <form onSubmit={handleSubmit} className="mt-6 space-y-4">
      {error && <Alert variant="error">{error}</Alert>}
      <AlertaCadastroRede saloesOutros={saloesOutros} salaoAtualNome={tenantNome} />
      {info && saloesOutros.length === 0 && <Alert variant="info">{info}</Alert>}

      <Input
        label="WhatsApp (com DDD)"
        value={telefone}
        onChange={(e) => {
          setTelefone(e.target.value)
          setSaloesOutros([])
          setInfo('')
        }}
        onBlur={handleTelefoneBlur}
        placeholder="(11) 99999-9999"
        autoComplete="tel"
        required
      />
      <Input
        label="Nome completo"
        value={nome}
        onChange={(e) => setNome(e.target.value)}
        autoComplete="name"
        required
      />
      <Input
        label="E-mail"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        autoComplete="email"
        required
      />

      <Button type="submit" fullWidth loading={loading} className="bg-[#7d5141] hover:bg-[#996958]">
        Continuar para agendar
      </Button>

      <p className="text-center text-sm text-[#514440]">
        Já tenho senha?{' '}
        <Link
          to={loginTo}
          state={loginState}
          className="inline-flex min-h-touch-min items-center font-semibold text-[#7d5141] hover:underline"
        >
          Entrar
        </Link>
      </p>
    </form>
  )
}
