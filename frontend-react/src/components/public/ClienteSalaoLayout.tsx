import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, LogOut } from 'lucide-react'
import type { ReactNode } from 'react'
import { useAuth } from '../../contexts/AuthContext'
import { useLogout } from '../../hooks/useLogout'
import type { Tenant } from '../../types'

interface Props {
  tenant: Tenant
  children: ReactNode
  backTo?: string
  backLabel?: string
  showAccount?: boolean
}

export function ClienteSalaoLayout({
  tenant,
  children,
  backTo,
  backLabel = 'Voltar',
  showAccount = true,
}: Props) {
  const { slug } = useParams<{ slug: string }>()
  const { session } = useAuth()
  const handleLogout = useLogout(`/${slug}`)

  return (
    <div className="min-h-screen bg-[#faf9f8]">
      <header className="border-b border-[#e5d3c8]/30 bg-white px-4 py-3">
        <div className="mx-auto flex max-w-lg items-center justify-between gap-3">
          <div className="min-w-0">
            {backTo && (
              <Link
                to={backTo}
                className="mb-1 flex items-center gap-1 text-xs text-[#7d5141]"
              >
                <ArrowLeft className="h-3.5 w-3.5" />
                {backLabel}
              </Link>
            )}
            <p className="truncate font-display text-lg font-semibold text-[#1a1c1c]">
              {tenant.nome}
            </p>
            {session?.user.role === 'CLIENTE' && (
              <p className="truncate text-xs text-aura-muted">Olá, {session.user.nome}</p>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {showAccount && session?.user.role === 'CLIENTE' && (
              <Link
                to={`/${slug}/conta`}
                className="rounded-lg border border-aura-border px-3 py-1.5 text-xs font-medium text-aura-anthracite hover:bg-aura-surface"
              >
                Minha conta
              </Link>
            )}
            {session?.user.role === 'CLIENTE' && (
              <button
                type="button"
                onClick={handleLogout}
                className="rounded-lg p-2 text-aura-muted hover:bg-aura-surface"
                aria-label="Sair"
              >
                <LogOut className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-lg px-4 py-6 pb-[max(1.5rem,env(safe-area-inset-bottom))]">
        {children}
      </main>
    </div>
  )
}
