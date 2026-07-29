import { useMemo, useState } from 'react'
import { Group, Pencil, Plus, Store, Trash2 } from 'lucide-react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
  MetricCard,
} from '../../components/superadmin/SuperAdminLayout'
import { Alert } from '../../components/ui/Alert'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { ConfirmModal, Modal } from '../../components/ui/Modal'
import type { PlanoSaas } from '../../types'
import {
  createPlanoSaas,
  deletePlanoSaas,
  getDb,
  updatePlanoSaas,
} from '../../utils/mockDb'
import { formatBRL, parseBRLInput } from '../../utils/format'

const FEATURED_PLAN_ID = 'plano-profissional'

const TIER_BADGE: Record<string, { label: string; className: string }> = {
  'plano-essencial': {
    label: 'Entrada',
    className: 'bg-[#efdcd1] text-[#6d6057]',
  },
  'plano-profissional': {
    label: 'Mais popular',
    className: 'bg-white/20 text-white backdrop-blur-md',
  },
  'plano-elite': {
    label: 'Premium',
    className: 'bg-[#ebe0dd] text-[#4b4543]',
  },
}

function defaultBadge(ativo: boolean) {
  return ativo
    ? { label: 'Ativo', className: 'bg-[#efdcd1] text-[#6d6057]' }
    : { label: 'Inativo', className: 'bg-aura-surface text-aura-muted' }
}

const emptyForm = (): Omit<PlanoSaas, 'id'> => ({
  nome: '',
  preco_mensal: 0,
  limite_profissionais: 3,
  ativo: true,
})

export function PlanosSaaS() {
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<PlanoSaas | null>(null)
  const [editing, setEditing] = useState<PlanoSaas | null>(null)
  const [form, setForm] = useState(emptyForm())
  const [precoStr, setPrecoStr] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const refresh = () => setDb(getDb())

  const tenantsPorPlano = useMemo(() => {
    const map = new Map<string, number>()
    for (const t of db.tenants) {
      map.set(t.plano_id, (map.get(t.plano_id) ?? 0) + 1)
    }
    return map
  }, [db.tenants])

  const planosOrdenados = useMemo(() => {
    const q = search.trim().toLowerCase()
    return [...db.planos]
      .filter((p) => !q || p.nome.toLowerCase().includes(q))
      .sort((a, b) => a.preco_mensal - b.preco_mensal)
  }, [db.planos, search])

  const mrrPotencial = db.planos.reduce(
    (s, p) => s + p.preco_mensal * (tenantsPorPlano.get(p.id) ?? 0),
    0,
  )

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm())
    setPrecoStr('')
    setError('')
    setModalOpen(true)
  }

  const openEdit = (plano: PlanoSaas) => {
    setEditing(plano)
    setForm({
      nome: plano.nome,
      preco_mensal: plano.preco_mensal,
      limite_profissionais: plano.limite_profissionais,
      ativo: plano.ativo,
    })
    setPrecoStr(String(plano.preco_mensal).replace('.', ','))
    setError('')
    setModalOpen(true)
  }

  const save = async () => {
    setError('')
    const preco = parseBRLInput(precoStr)
    if (Number.isNaN(preco) || preco < 0) {
      setError('Informe um preço mensal válido')
      return
    }
    const payload = { ...form, preco_mensal: preco }
    try {
      if (editing) {
        await updatePlanoSaas(editing.id, payload)
        setSuccess(`Plano "${payload.nome}" atualizado.`)
      } else {
        await createPlanoSaas(payload)
        setSuccess(`Plano "${payload.nome}" criado.`)
      }
      refresh()
      setModalOpen(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro ao salvar plano')
    }
  }

  const confirmDelete = () => {
    if (!deleteTarget) return
    setError('')
    try {
      deletePlanoSaas(deleteTarget.id)
      setSuccess(`Plano "${deleteTarget.nome}" removido.`)
      refresh()
      setDeleteTarget(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Erro ao excluir plano')
      setDeleteTarget(null)
    }
  }

  return (
    <SuperAdminLayout
      searchPlaceholder="Buscar planos…"
      searchValue={search}
      onSearchChange={setSearch}
    >
      <PageHeader
        title="Gerenciamento de Planos"
        subtitle="Configure os tiers de assinatura e limites operacionais da plataforma."
        action={
          <Button
            fullWidth
            className="sm:w-auto bg-[#7d5141] hover:bg-[#996958]"
            onClick={openCreate}
          >
            <Plus className="h-4 w-4" />
            Criar novo plano
          </Button>
        }
      />

      {error && !modalOpen && !deleteTarget && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      <div className="mb-8 grid gap-4 sm:grid-cols-3">
        <MetricCard label="Planos cadastrados" value={String(db.planos.length)} />
        <MetricCard
          label="Planos ativos"
          value={String(db.planos.filter((p) => p.ativo).length)}
          highlight="success"
        />
        <MetricCard label="MRR dos planos" value={formatBRL(mrrPotencial)} />
      </div>

      <div className="grid grid-cols-1 gap-6 md:grid-cols-3 md:items-stretch">
        {planosOrdenados.length === 0 && (
          <p className="col-span-full rounded-xl border border-dashed border-aura-border py-12 text-center text-sm text-aura-muted">
            {search.trim()
              ? 'Nenhum plano encontrado para esta busca.'
              : 'Nenhum plano cadastrado.'}
          </p>
        )}
        {planosOrdenados.map((plano) => {
          const featured = plano.id === FEATURED_PLAN_ID
          const tier = TIER_BADGE[plano.id] ?? defaultBadge(plano.ativo)
          const assinantes = tenantsPorPlano.get(plano.id) ?? 0

          return (
            <div
              key={plano.id}
              className={[
                'group relative flex flex-col rounded-xl p-6 transition-all duration-300',
                featured
                  ? 'z-10 scale-100 bg-[#7d5141] text-white shadow-xl md:scale-105'
                  : 'border border-[#e5d3c8]/50 bg-white/80 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-sm hover:-translate-y-1 hover:border-[#7d5141]/30',
                !plano.ativo && !featured ? 'opacity-75' : '',
              ].join(' ')}
            >
              {featured && (
                <div className="pointer-events-none absolute -right-10 -top-10 h-40 w-40 rounded-full bg-white/10 blur-3xl transition-all group-hover:bg-white/20" />
              )}

              <div className="relative z-10 mb-4 flex items-start justify-between">
                <span
                  className={[
                    'rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-wider',
                    tier.className,
                  ].join(' ')}
                >
                  {tier.label}
                </span>
                <div className="flex gap-1 opacity-100 transition-opacity md:opacity-0 md:group-hover:opacity-100">
                  <button
                    type="button"
                    onClick={() => openEdit(plano)}
                    className={[
                      'rounded-lg p-1.5 transition-colors',
                      featured ? 'text-white/70 hover:text-white' : 'text-aura-muted hover:text-[#7d5141]',
                    ].join(' ')}
                    aria-label={`Editar ${plano.nome}`}
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    type="button"
                    onClick={() => setDeleteTarget(plano)}
                    className={[
                      'rounded-lg p-1.5 transition-colors',
                      featured ? 'text-white/70 hover:text-red-200' : 'text-aura-muted hover:text-red-600',
                    ].join(' ')}
                    aria-label={`Excluir ${plano.nome}`}
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              <div className="relative z-10 flex items-center gap-2">
                <h3
                  className={[
                    'font-display text-xl font-semibold',
                    featured ? 'text-white' : 'text-aura-anthracite',
                  ].join(' ')}
                >
                  {plano.nome}
                </h3>
                {!plano.ativo && (
                  <Badge variant="muted">{featured ? 'Inativo' : 'Inativo'}</Badge>
                )}
              </div>

              <div className="relative z-10 mt-2 flex items-baseline gap-1">
                <span
                  className={[
                    'font-display text-3xl font-bold',
                    featured ? 'text-white' : 'text-[#7d5141]',
                  ].join(' ')}
                >
                  {formatBRL(plano.preco_mensal).replace(/\s/g, ' ')}
                </span>
                <span className={featured ? 'text-sm text-white/70' : 'text-sm text-aura-muted'}>
                  /mês
                </span>
              </div>

              <div className="relative z-10 mt-6 flex-1 space-y-4">
                <div className="flex items-center gap-4">
                  <div
                    className={[
                      'flex h-10 w-10 items-center justify-center rounded-lg',
                      featured ? 'bg-white/10' : 'bg-[#e9e8e7]',
                    ].join(' ')}
                  >
                    <Group className={featured ? 'h-5 w-5 text-white' : 'h-5 w-5 text-[#7d5141]'} />
                  </div>
                  <div>
                    <p className={featured ? 'text-sm text-white/70' : 'text-sm text-aura-muted'}>
                      Profissionais
                    </p>
                    <p className={featured ? 'font-semibold' : 'font-semibold text-aura-anthracite'}>
                      Até {plano.limite_profissionais}{' '}
                      {plano.limite_profissionais === 1 ? 'profissional' : 'profissionais'}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div
                    className={[
                      'flex h-10 w-10 items-center justify-center rounded-lg',
                      featured ? 'bg-white/10' : 'bg-[#e9e8e7]',
                    ].join(' ')}
                  >
                    <Store className={featured ? 'h-5 w-5 text-white' : 'h-5 w-5 text-[#7d5141]'} />
                  </div>
                  <div>
                    <p className={featured ? 'text-sm text-white/70' : 'text-sm text-aura-muted'}>
                      Agenda & catálogo
                    </p>
                    <p className={featured ? 'font-semibold' : 'font-semibold text-aura-anthracite'}>
                      Incluso no plano
                    </p>
                  </div>
                </div>
              </div>

              <div
                className={[
                  'relative z-10 mt-6 flex items-center justify-between border-t pt-4',
                  featured ? 'border-white/10' : 'border-[#e5d3c8]/40',
                ].join(' ')}
              >
                <div>
                  <p
                    className={[
                      'text-[10px] font-bold uppercase tracking-wider',
                      featured ? 'text-white/70' : 'text-aura-muted',
                    ].join(' ')}
                  >
                    Salões neste plano
                  </p>
                  <p
                    className={[
                      'font-semibold',
                      featured ? 'text-white' : 'text-[#7d5141]',
                    ].join(' ')}
                  >
                    {assinantes} {assinantes === 1 ? 'salão' : 'salões'}
                  </p>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      <SuperAdminFooter />

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editing ? 'Editar plano' : 'Criar novo plano'}
        maxWidthClass="max-w-md"
        footer={
          <>
            <Button variant="secondary" onClick={() => setModalOpen(false)}>
              Cancelar
            </Button>
            <Button className="bg-[#7d5141] hover:bg-[#996958]" onClick={save}>
              {editing ? 'Salvar alterações' : 'Criar plano'}
            </Button>
          </>
        }
      >
        {error && modalOpen && <p className="mb-3 text-sm text-red-600">{error}</p>}
        <div className="space-y-4">
          <Input
            label="Nome do plano"
            value={form.nome}
            onChange={(e) => setForm({ ...form, nome: e.target.value })}
            placeholder="Essencial, Profissional, Elite…"
          />
          <Input
            label="Preço mensal (R$)"
            value={precoStr}
            onChange={(e) => setPrecoStr(e.target.value)}
            placeholder="149,90"
            inputMode="decimal"
          />
          <Input
            label="Limite de profissionais"
            type="number"
            min={1}
            value={form.limite_profissionais}
            onChange={(e) =>
              setForm({ ...form, limite_profissionais: Number(e.target.value) })
            }
          />
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.ativo}
              onChange={(e) => setForm({ ...form, ativo: e.target.checked })}
              className="rounded border-aura-border text-[#7d5141] focus:ring-[#7d5141]/20"
            />
            Plano disponível para novos salões
          </label>
          <p className="text-xs text-aura-muted">
            Nome único na plataforma. Preço ≥ 0. Limite mínimo: 1 profissional.
          </p>
        </div>
      </Modal>

      <ConfirmModal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={confirmDelete}
        title="Excluir plano"
        message={
          deleteTarget
            ? `Deseja excluir o plano "${deleteTarget.nome}"? Só é possível se nenhum salão estiver vinculado.`
            : ''
        }
        confirmLabel="Excluir"
      />
    </SuperAdminLayout>
  )
}
