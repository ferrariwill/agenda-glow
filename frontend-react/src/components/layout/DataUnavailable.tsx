import { AlertTriangle } from 'lucide-react'
import { useAuth } from '../../contexts/AuthContext'
import { useBootstrapState } from '../../data/BootstrapContext'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'

export function DataUnavailable() {
  const { logout } = useAuth()
  const { errorMessage, retry } = useBootstrapState()

  return (
    <div className="flex min-h-screen items-center justify-center bg-aura-surface p-4">
      <Card className="max-w-md text-center" padding="lg">
        <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-amber-100">
          <AlertTriangle className="h-8 w-8 text-amber-700" />
        </div>
        <h1 className="font-display text-2xl font-semibold text-aura-anthracite">
          Não foi possível carregar os dados do salão
        </h1>
        <p className="mt-3 text-sm text-aura-muted">
          {errorMessage ?? 'O acesso ao painel está temporariamente indisponível.'}
        </p>
        <div className="mt-6 flex flex-col gap-2">
          <Button onClick={retry}>Tentar novamente</Button>
          <Button variant="ghost" onClick={logout}>
            Sair da conta
          </Button>
        </div>
      </Card>
    </div>
  )
}
