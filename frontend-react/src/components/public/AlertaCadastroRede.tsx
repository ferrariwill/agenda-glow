import { Alert } from '../ui/Alert'
import type { SalaoRedeResumo } from '../../utils/mockDb'

interface AlertaCadastroRedeProps {
  saloesOutros: SalaoRedeResumo[]
  salaoAtualNome: string
}

export function AlertaCadastroRede({ saloesOutros, salaoAtualNome }: AlertaCadastroRedeProps) {
  if (saloesOutros.length === 0) return null

  const primeiro = saloesOutros[0].nome
  const extras = saloesOutros.length > 1 ? ` e outros ${saloesOutros.length - 1}` : ''

  return (
    <Alert variant="warning" title="Cadastro ativo em outro salão">
      Você já possui cadastro ativo em {primeiro}
      {extras}. Ao continuar, vamos vincular sua conta também a {salaoAtualNome}.
    </Alert>
  )
}
