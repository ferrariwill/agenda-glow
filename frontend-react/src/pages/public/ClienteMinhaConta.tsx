import { useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Heart } from 'lucide-react'
import { ClienteSalaoLayout } from '../../components/public/ClienteSalaoLayout'
import { Badge, statusAgendamentoBadge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { useAuth } from '../../contexts/AuthContext'
import {
  getAgendamentosCliente,
  getDb,
  getProfissionalFavorito,
  getTenantBySlug,
  setProfissionalFavorito,
} from '../../utils/mockDb'
import { formatDateTimeBR } from '../../utils/format'

export function ClienteMinhaConta() {
  const { slug } = useParams<{ slug: string }>()
  const { session } = useAuth()
  const tenant = slug ? getTenantBySlug(slug) : undefined
  const [tick, setTick] = useState(0)

  const db = getDb()
  const telefone = session?.user.telefone ?? ''

  const agendamentosSalao = useMemo(
    () => (tenant ? getAgendamentosCliente(telefone, tenant.id) : []),
    [telefone, tenant?.id, tick],
  )
  const agendamentosTodos = useMemo(
    () => getAgendamentosCliente(telefone),
    [telefone, tick],
  )

  const profissionais = tenant
    ? db.profissionais.filter((p) => p.tenant_id === tenant.id && p.ativo)
    : []
  const favoritoId = session && tenant ? getProfissionalFavorito(session.user.id, tenant.id) : undefined

  if (!tenant) {
    return <p className="p-8 text-center text-aura-muted">Salão não encontrado.</p>
  }

  const setFavorito = (profId: string) => {
    if (!session) return
    setProfissionalFavorito(session.user.id, tenant.id, profId)
    setTick((t) => t + 1)
  }

  return (
    <ClienteSalaoLayout tenant={tenant} backTo={`/${slug}`}>
      <h1 className="mb-4 font-display text-2xl font-semibold">Minha conta</h1>

      <Card className="mb-4">
        <p className="text-sm font-medium">{session?.user.nome}</p>
        <p className="text-sm text-aura-muted">{session?.user.email}</p>
        <p className="text-sm text-aura-muted">{telefone}</p>
        <Button className="mt-4" fullWidth onClick={() => window.location.assign(`/${slug}/agendar`)}>
          Novo agendamento
        </Button>
      </Card>

      <Card className="mb-4">
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold">
          <Heart className="h-4 w-4 text-aura-primary" />
          Profissional favorito — {tenant.nome}
        </h2>
        <div className="flex flex-wrap gap-2">
          {profissionais.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => setFavorito(p.id)}
              className={[
                'rounded-full border px-3 py-1.5 text-xs font-medium transition-colors',
                favoritoId === p.id
                  ? 'border-aura-primary bg-aura-primary text-white'
                  : 'border-aura-border hover:border-aura-primary',
              ].join(' ')}
            >
              {p.nome}
            </button>
          ))}
        </div>
      </Card>

      <Card className="mb-4">
        <h2 className="mb-3 text-sm font-semibold">Agendamentos neste salão</h2>
        {agendamentosSalao.length === 0 ? (
          <p className="text-sm text-aura-muted">Nenhum agendamento ainda.</p>
        ) : (
          <ul className="space-y-2">
            {agendamentosSalao.map((a) => (
              <li
                key={a.id}
                className="rounded-lg border border-aura-border bg-aura-surface px-3 py-2 text-sm"
              >
                <p className="font-medium">{formatDateTimeBR(a.data, a.hora_inicio)}</p>
                <p className="text-aura-muted">
                  {a.servico_nome} · {a.profissional_nome}
                </p>
                <Badge variant={statusAgendamentoBadge(a.status)}>{a.status}</Badge>
              </li>
            ))}
          </ul>
        )}
      </Card>

      {agendamentosTodos.length > agendamentosSalao.length && (
        <Card>
          <h2 className="mb-3 text-sm font-semibold">Outros salões</h2>
          <ul className="space-y-2">
            {agendamentosTodos
              .filter((a) => a.tenant_id !== tenant.id)
              .map((a) => (
                <li
                  key={a.id}
                  className="rounded-lg border border-aura-border px-3 py-2 text-sm"
                >
                  <p className="font-medium">{a.tenant_nome}</p>
                  <p className="text-aura-muted">
                    {formatDateTimeBR(a.data, a.hora_inicio)} · {a.servico_nome}
                  </p>
                  {a.tenant_slug && (
                    <Link
                      to={`/${a.tenant_slug}`}
                      className="text-xs text-aura-primary hover:underline"
                    >
                      Ir para o salão
                    </Link>
                  )}
                </li>
              ))}
          </ul>
        </Card>
      )}
    </ClienteSalaoLayout>
  )
}
