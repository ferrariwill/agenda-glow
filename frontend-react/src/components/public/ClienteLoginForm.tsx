import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { AlertaCadastroRede } from './AlertaCadastroRede'
import {
  findContaClienteByEmail,
  listSaloesAtivosDaConta,
  type SalaoRedeResumo,
} from '../../utils/mockDb'

export interface ClienteLoginValues {
  email: string
  password: string
}

interface ClienteLoginFormProps {
  tenantId: string
  tenantNome: string
  cadastroTo: string
  cadastroState?: { from?: string }
  loading?: boolean
  error?: string
  onSubmit: (values: ClienteLoginValues) => void | Promise<void>
}

export function ClienteLoginForm({
  tenantId,
  tenantNome,
  cadastroTo,
  cadastroState,
  loading = false,
  error,
  onSubmit,
}: ClienteLoginFormProps) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [saloesOutros, setSaloesOutros] = useState<SalaoRedeResumo[]>([])

  const refreshRedeAlert = (nextEmail: string) => {
    const conta = findContaClienteByEmail(nextEmail)
    if (!conta) {
      setSaloesOutros([])
      return
    }
    setSaloesOutros(listSaloesAtivosDaConta(conta.id, tenantId))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await onSubmit({ email, password })
  }

  return (
    <form onSubmit={handleSubmit} className="mt-6 space-y-4">
      {error && <Alert variant="error">{error}</Alert>}
      <AlertaCadastroRede saloesOutros={saloesOutros} salaoAtualNome={tenantNome} />

      <Input
        label="E-mail"
        type="email"
        value={email}
        onChange={(e) => {
          setEmail(e.target.value)
          setSaloesOutros([])
        }}
        onBlur={() => refreshRedeAlert(email)}
        autoComplete="email"
        placeholder="seu@email.com"
        required
      />
      <Input
        label="Senha"
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        autoComplete="current-password"
        placeholder="••••••••"
        required
      />

      <Button type="submit" fullWidth loading={loading} className="bg-[#7d5141] hover:bg-[#996958]">
        Entrar e agendar
      </Button>

      <p className="text-center text-sm text-[#514440]">
        Primeira vez aqui?{' '}
        <Link
          to={cadastroTo}
          state={cadastroState}
          className="inline-flex min-h-touch-min items-center font-semibold text-[#7d5141] hover:underline"
        >
          Identificar-se
        </Link>
      </p>
    </form>
  )
}
