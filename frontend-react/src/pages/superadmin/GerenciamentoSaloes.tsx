import { useMemo, useState } from 'react'
import { Plus, Upload } from 'lucide-react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
  MetricCard,
  TablePagination,
} from '../../components/superadmin/SuperAdminLayout'
import { TenantActionsMenu } from '../../components/superadmin/TenantActionsMenu'
import { Alert } from '../../components/ui/Alert'
import { Badge, statusTenantBadge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import type { Tenant } from '../../types'
import {
  SLUG_REGEX,
  activateTenant,
  assignPlanToTenant,
  createDonaForTenant,
  createTenant,
  getDb,
  getPlano,
  getSuperAdminStats,
  renewAllExpired,
  renewTenant,
  suspendTenant,
} from '../../utils/mockDb'
import { formatDateBR, initials, todayISO } from '../../utils/format'

const MAX_LOGO_BYTES = 2 * 1024 * 1024
const PAGE_SIZE = 5

export function GerenciamentoSaloes() {
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'TODOS' | 'ATIVO' | 'VENCIDO'>('TODOS')
  const [page, setPage] = useState(1)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [renewLoading, setRenewLoading] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [assignOpen, setAssignOpen] = useState<Tenant | null>(null)
  const [donaOpen, setDonaOpen] = useState<Tenant | null>(null)
  const [assignPlanoId, setAssignPlanoId] = useState('')
  const [donaNome, setDonaNome] = useState('')
  const [donaEmail, setDonaEmail] = useState('')

  const [nome, setNome] = useState('')
  const [slug, setSlug] = useState('')
  const [planoId, setPlanoId] = useState(db.planos[0]?.id ?? '')
  const [logoPreview, setLogoPreview] = useState<string>()

  const stats = useMemo(() => getSuperAdminStats(), [db])
  const refresh = () => setDb(getDb())

  const filtered = useMemo(() => {
    let list = [...db.tenants]
    if (statusFilter !== 'TODOS') list = list.filter((t) => t.status === statusFilter)
    if (search.trim()) {
      const q = search.toLowerCase()
      list = list.filter(
        (t) =>
          t.nome.toLowerCase().includes(q) ||
          t.slug.includes(q) ||
          t.dona_email?.toLowerCase().includes(q) ||
          t.dona_nome?.toLowerCase().includes(q),
      )
    }
    return list.sort((a, b) => a.nome.localeCompare(b.nome))
  }, [db.tenants, search, statusFilter])

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const slice = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  const handleLogo = (file: File | null) => {
    if (!file) return
    if (!['image/png', 'image/jpeg'].includes(file.type)) {
      setError('Logo deve ser .png ou .jpg')
      return
    }
    if (file.size > MAX_LOGO_BYTES) {
      setError('Logo deve ter menos de 2MB')
      return
    }
    const reader = new FileReader()
    reader.onload = () => setLogoPreview(reader.result as string)
    reader.readAsDataURL(file)
  }

  const handleCreate = async () => {
    setError('')
    if (!SLUG_REGEX.test(slug)) {
      setError('Slug inválido. Use apenas letras minúsculas, números e hífens.')
      return
    }
    if (db.tenants.some((t) => t.slug === slug)) {
      setError('Este slug já está em uso')
      return
    }
    const today = todayISO()
    await createTenant({
      nome,
      slug,
      status: 'ATIVO',
      plano_id: planoId,
      logo_url: logoPreview,
      bio: '',
      data_vencimento: new Date(Date.now() + 365 * 86400000).toISOString().slice(0, 10),
      criado_em: today,
    })
    refresh()
    setCreateOpen(false)
    setNome('')
    setSlug('')
    setLogoPreview(undefined)
    setSuccess('Salão cadastrado com sucesso!')
  }

  const handleRenewAll = async () => {
    setRenewLoading(true)
    const n = renewAllExpired()
    refresh()
    setRenewLoading(false)
    setSuccess(n > 0 ? `${n} assinatura(s) renovada(s).` : 'Nenhuma assinatura vencida.')
  }

  return (
    <SuperAdminLayout
      searchPlaceholder="Buscar salões…"
      searchValue={search}
      onSearchChange={(v) => {
        setSearch(v)
        setPage(1)
      }}
      onRenewAll={handleRenewAll}
      renewLoading={renewLoading}
    >
      {error && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert variant="info" className="mb-4" onDismiss={() => setSuccess('')}>
          {success}
        </Alert>
      )}

      <PageHeader
        title="Gestão de Salões"
        subtitle="Visualize e gerencie os tenants da plataforma Aura Beauty."
        action={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            Novo Salão
          </Button>
        }
      />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Total de Salões" value={String(db.tenants.length)} />
        <MetricCard label="Ativos" value={String(stats.ativos)} highlight="success" />
        <MetricCard
          label="Vencidos"
          value={String(stats.vencidos)}
          highlight={stats.vencidos > 0 ? 'danger' : 'default'}
        />
        <MetricCard label="Novos este Mês" value={`+${stats.novosMes}`} highlight="success" />
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value as typeof statusFilter)
            setPage(1)
          }}
          className="rounded-lg border border-aura-border bg-white px-3 py-2 text-sm"
        >
          <option value="TODOS">Todos os Salões</option>
          <option value="ATIVO">Ativos</option>
          <option value="VENCIDO">Vencidos / Suspensos</option>
        </select>
        <button
          type="button"
          onClick={() => {
            setStatusFilter('TODOS')
            setSearch('')
            setPage(1)
          }}
          className="text-sm text-aura-primary hover:underline"
        >
          Limpar Filtros
        </button>
      </div>

      <div className="overflow-hidden rounded-lg border border-aura-border bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-aura-border bg-aura-surface/50 text-left text-aura-muted">
                <th className="px-4 py-3 font-medium">Salão</th>
                <th className="px-4 py-3 font-medium">Dona</th>
                <th className="px-4 py-3 font-medium">Plano</th>
                <th className="px-4 py-3 font-medium">Vencimento</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium text-right">Ações</th>
              </tr>
            </thead>
            <tbody>
              {slice.map((t) => {
                const plano = getPlano(t.plano_id)
                return (
                  <tr key={t.id} className="border-b border-aura-border/60 hover:bg-aura-surface/30">
                    <td className="px-4 py-3.5">
                      <div className="flex items-center gap-3">
                        {t.logo_url ? (
                          <img src={t.logo_url} alt="" className="h-9 w-9 rounded-full object-cover" />
                        ) : (
                          <div className="flex h-9 w-9 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary">
                            {initials(t.nome)}
                          </div>
                        )}
                        <div>
                          <p className="font-medium">{t.nome}</p>
                          <p className="text-xs text-aura-muted">/{t.slug}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3.5">
                      <p className="text-aura-anthracite">{t.dona_nome ?? '—'}</p>
                      <p className="text-xs text-aura-muted">{t.dona_email ?? 'Sem dona'}</p>
                    </td>
                    <td className="px-4 py-3.5 text-aura-muted">{plano?.nome ?? '—'}</td>
                    <td className="px-4 py-3.5 text-aura-muted">{formatDateBR(t.data_vencimento)}</td>
                    <td className="px-4 py-3.5">
                      <Badge variant={statusTenantBadge(t.status)}>{t.status}</Badge>
                    </td>
                    <td className="px-4 py-3.5 text-right">
                      <TenantActionsMenu
                        status={t.status}
                        slug={t.slug}
                        onAssignPlan={() => {
                          setAssignPlanoId(t.plano_id)
                          setAssignOpen(t)
                        }}
                        onRenew={async () => {
                          await renewTenant(t.id)
                          refresh()
                          setSuccess(`Assinatura de ${t.nome} renovada por 12 meses.`)
                        }}
                        onToggleStatus={async () => {
                          if (t.status === 'ATIVO') await suspendTenant(t.id)
                          else await activateTenant(t.id)
                          refresh()
                        }}
                        onCreateDona={() => {
                          setDonaNome(t.dona_nome ?? '')
                          setDonaEmail(t.dona_email ?? '')
                          setDonaOpen(t)
                        }}
                      />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
        <TablePagination
          page={page}
          totalPages={totalPages}
          total={filtered.length}
          pageSize={PAGE_SIZE}
          onPageChange={setPage}
          label="salões"
        />
      </div>

      <SuperAdminFooter />

      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Novo Salão"
        footer={
          <>
            <Button variant="secondary" onClick={() => setCreateOpen(false)}>
              Cancelar
            </Button>
            <Button onClick={handleCreate}>Cadastrar</Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input label="Nome do salão" value={nome} onChange={(e) => setNome(e.target.value)} />
          <Input
            label="Slug público"
            value={slug}
            onChange={(e) => setSlug(e.target.value.toLowerCase())}
            placeholder="meu-salao-luxo"
          />
          <div>
            <label className="mb-1.5 block text-sm font-medium">Plano SaaS</label>
            <select
              value={planoId}
              onChange={(e) => setPlanoId(e.target.value)}
              className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
            >
              {db.planos.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome}
                </option>
              ))}
            </select>
          </div>
          <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-aura-border p-4">
            <Upload className="h-5 w-5 text-aura-muted" />
            <span className="text-sm text-aura-muted">Logo PNG/JPG (&lt; 2MB)</span>
            <input
              type="file"
              accept=".png,.jpg,.jpeg"
              className="hidden"
              onChange={(e) => handleLogo(e.target.files?.[0] ?? null)}
            />
          </label>
        </div>
      </Modal>

      <Modal
        open={!!assignOpen}
        onClose={() => setAssignOpen(null)}
        title="Atribuir plano"
        footer={
          <>
            <Button variant="secondary" onClick={() => setAssignOpen(null)}>
              Cancelar
            </Button>
            <Button
              onClick={async () => {
                if (!assignOpen) return
                await assignPlanToTenant(assignOpen.id, assignPlanoId)
                refresh()
                setAssignOpen(null)
                setSuccess('Plano atribuído e vencimento estendido por 12 meses.')
              }}
            >
              Salvar
            </Button>
          </>
        }
      >
        <p className="mb-3 text-sm text-aura-muted">
          Salão: <strong>{assignOpen?.nome}</strong>
        </p>
        <select
          value={assignPlanoId}
          onChange={(e) => setAssignPlanoId(e.target.value)}
          className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
        >
          {db.planos.map((p) => (
            <option key={p.id} value={p.id}>
              {p.nome} — {p.limite_profissionais} profissionais
            </option>
          ))}
        </select>
      </Modal>

      <Modal
        open={!!donaOpen}
        onClose={() => setDonaOpen(null)}
        title="Criar dona do salão"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDonaOpen(null)}>
              Cancelar
            </Button>
            <Button
              onClick={() => {
                if (!donaOpen) return
                try {
                  createDonaForTenant(donaOpen.id, donaNome, donaEmail)
                  refresh()
                  setDonaOpen(null)
                  setSuccess('Dona criada. Senha padrão: AgendaGlow@2026')
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Erro ao criar dona')
                }
              }}
            >
              Criar
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Input label="Nome" value={donaNome} onChange={(e) => setDonaNome(e.target.value)} />
          <Input
            label="E-mail"
            type="email"
            value={donaEmail}
            onChange={(e) => setDonaEmail(e.target.value)}
          />
        </div>
      </Modal>
    </SuperAdminLayout>
  )
}
