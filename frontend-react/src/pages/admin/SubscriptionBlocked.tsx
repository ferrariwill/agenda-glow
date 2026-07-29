import { Lock } from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { useAuth } from '../../contexts/AuthContext'

export function SubscriptionBlocked() {
  const { logout } = useAuth()

  return (
    <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
      <Card className="max-w-md text-center" padding="lg">
        <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-100">
          <Lock className="h-8 w-8 text-red-600" />
        </div>
        <p className="text-xs font-medium uppercase tracking-wider text-aura-muted">HTTP 402</p>
        <h1 className="mt-2 font-display text-2xl font-semibold text-aura-anthracite">
          Assinatura Suspensa
        </h1>
        <p className="mt-3 text-sm text-aura-muted">
          Sua assinatura está vencida ou suspensa. Renove seu plano para continuar gerenciando
          agenda, equipe e financeiro.
        </p>
        <div className="mt-6 flex flex-col gap-2">
          <Button onClick={() => window.open('mailto:suporte@agendaglow.com', '_blank')}>
            Falar com suporte
          </Button>
          <Button variant="ghost" onClick={logout}>
            Sair da conta
          </Button>
        </div>
      </Card>
    </div>
  )
}
