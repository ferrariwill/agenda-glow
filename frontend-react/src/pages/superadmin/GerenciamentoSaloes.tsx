import { useMemo, useState } from 'react'
import { Plus } from 'lucide-react'
import {
  SuperAdminLayout,
  SuperAdminFooter,
  PageHeader,
  MetricCard,
  TablePagination,
} from '../../components/superadmin/SuperAdminLayout'
import { NovoSalaoModal } from '../../components/superadmin/NovoSalaoModal'
import { TenantActionsMenu } from '../../components/superadmin/TenantActionsMenu'
import { TenantIdentityForm } from '../../components/superadmin/TenantIdentityForm'
import { Alert } from '../../components/ui/Alert'
import { Badge, statusTenantBadge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import {
  ResponsiveEntityList,
  type EntityColumn,
} from '../../components/ui/ResponsiveEntityList'
import { ToastFeedback } from '../../components/ui/ToastFeedback'
import { ApiError } from '../../lib/api'
import type { Tenant } from '../../types'
import {
  SLUG_REGEX,
  activateTenant,
  assignPlanToTenant,
  createDonaForTenant,
  getDb,
  getPlano,
  getSuperAdminStats,
  renewAllExpired,
  renewTenant,
  suspendTenant,
  updateTenantIdentity,
} from '../../utils/mockDb'
import { formatDateBR, initials } from '../../utils/format'
import { matchesTenantStatusFilter, type TenantStatusFilter } from '../../utils/tenantStatus'

const MAX_LOGO_BYTES = 2 * 1024 * 1024
const PAGE_SIZE = 5

export function GerenciamentoSaloes() {
  const [db, setDb] = useState(getDb())
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<TenantStatusFilter>('TODOS')
  const [page, setPage] = useState(1)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [renewLoading, setRenewLoading] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState<Tenant | null>(null)
  const [assignOpen, setAssignOpen] = useState<Tenant | null>(null)
  const [donaOpen, setDonaOpen] = useState<Tenant | null>(null)
  const [assignPlanoId, setAssignPlanoId] = useState('')
  const [donaNome, setDonaNome] = useState('')
  const [donaEmail, setDonaEmail] = useState('')
  const [donaSubmitting, setDonaSubmitting] = useState(false)

  const [nome, setNome] = useState('')
  const [slug, setSlug] = useState('')
  const [logoPreview, setLogoPreview] = useState<string>()
  const [logoChanged, setLogoChanged] = useState(false)

  const stats = useMemo(() => getSuperAdminStats(), [db])
  const refresh = () => setDb(getDb())

  const filtered = useMemo(() => {
    let list = [...db.tenants]
    list = list.filter((t) => matchesTenantStatusFilter(t.status, statusFilter))
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

  const resetIdentityForm = () => {
    setNome('')
    setSlug('')
    setLogoPreview(undefined)
    setLogoChanged(false)
  }

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
    reader.onload = () => {
      setLogoPreview(reader.result as string)
      setLogoChanged(true)
    }
    reader.readAsDataURL(file)
  }

  const openEdit = (t: Tenant) => {
    setError('')
    setNome(t.nome)
    setSlug(t.slug)
    setLogoPreview(t.logo_url)
    setLogoChanged(false)
    setEditOpen(t)
  }

  const handleUpdate = async () => {
    if (!editOpen) return
    setError('')
    if (!SLUG_REGEX.test(slug)) {
      setError('Slug inválido. Use apenas letras minúsculas, números e hífens.')
      return
    }
    if (db.tenants.some((t) => t.slug === slug && t.id !== editOpen.id)) {
      setError('Este slug já está em uso')
      return
    }
    try {
      const patch: Pick<Tenant, 'nome' | 'slug'> & { logo_url?: string } = {
        nome,
        slug,
      }
      if (logoChanged) {
        patch.logo_url = logoPreview
      }
      await updateTenantIdentity(editOpen.id, patch)
      refresh()
      setEditOpen(null)
      resetIdentityForm()
      setSuccess('Salão atualizado com sucesso!')
    } catch (err) {
      setError(mapTenantError(err))
    }
  }

  const handleRenewAll = async () => {
    setRenewLoading(true)
    const n = renewAllExpired()
    refresh()
    setRenewLoading(false)
    setSuccess(n > 0 ? `${n} assinatura(s) renovada(s).` : 'Nenhuma assinatura vencida.')
  }

  const openCreate = () => {
    setError('')
    setCreateOpen(true)
  }

  const renderActions = (t: Tenant) => (
    <TenantActionsMenu
      status={t.status}
      slug={t.slug}
      onEdit={() => openEdit(t)}
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
  )

  const columns: EntityColumn<Tenant>[] = [
    {
      header: 'Salão',
      cell: (t) => (
        <div className="flex items-center gap-3">
          {t.logo_url ? (
            <img src={t.logo_url} alt="" className="h-9 w-9 rounded-full object-cover" />
          ) : (
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary">
              {initials(t.nome)}
            </div>
          )}
          <div className="min-w-0">
            <p className="font-medium">{t.nome}</p>
            <p className="text-xs text-aura-muted">/{t.slug}</p>
          </div>
        </div>
      ),
    },
    {
      header: 'Dona',
      cell: (t) => (
        <>
          <p className="text-aura-anthracite">{t.dona_nome ?? '—'}</p>
          <p className="text-xs text-aura-muted">{t.dona_email ?? 'Sem dona'}</p>
        </>
      ),
      priority: 'secondary',
    },
    {
      header: 'Plano',
      cell: (t) => (
        <span className="text-aura-muted">{getPlano(t.plano_id)?.nome ?? '—'}</span>
      ),
      priority: 'secondary',
    },
    {
      header: 'Vencimento',
      cell: (t) => (
        <span className="whitespace-nowrap text-aura-muted">
          {formatDateBR(t.data_vencimento)}
        </span>
      ),
    },
    {
      header: 'Status',
      cell: (t) => <Badge variant={statusTenantBadge(t.status)}>{t.status}</Badge>,
    },
    {
      header: 'Ações',
      cell: (t) => renderActions(t),
      align: 'right',
    },
  ]

  const renderCard = (t: Tenant) => (
    <div className="rounded-xl border border-[#efdcd1]/30 bg-[#faf9f8]/50 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          {t.logo_url ? (
            <img src={t.logo_url} alt="" className="h-10 w-10 shrink-0 rounded-full object-cover" />
          ) : (
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-aura-primary/15 text-xs font-semibold text-aura-primary">
              {initials(t.nome)}
            </div>
          )}
          <div className="min-w-0">
            <p className="font-semibold text-aura-anthracite">{t.nome}</p>
            <p className="truncate text-xs text-aura-muted">/{t.slug}</p>
          </div>
        </div>
        {renderActions(t)}
      </div>
      <div className="mt-3 flex items-center justify-between gap-3">
        <Badge variant={statusTenantBadge(t.status)}>{t.status}</Badge>
        <span className="shrink-0 whitespace-nowrap text-sm font-medium text-aura-anthracite">
          {formatDateBR(t.data_vencimento)}
        </span>
      </div>
    </div>
  )

  const identityFormProps = {
    nome,
    slug,
    logoPreview,
    onNomeChange: setNome,
    onSlugChange: setSlug,
    onLogoFile: handleLogo,
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
      {error && !createOpen && !editOpen && !donaOpen && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}
      <ToastFeedback message={success || null} onDismiss={() => setSuccess('')} />

      <PageHeader
        title="Gestão de Salões"
        subtitle="Visualize e gerencie os tenants da plataforma Aura Beauty."
        action={
          <Button onClick={openCreate}>
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
          className="min-h-11 rounded-lg border border-aura-border bg-white px-3 py-2 text-sm"
        >
          <option value="TODOS">Todos os Salões</option>
          <option value="ATIVO">Ativos</option>
          <option value="INATIVOS">Vencidos / Suspensos</option>
        </select>
        <button
          type="button"
          onClick={() => {
            setStatusFilter('TODOS')
            setSearch('')
            setPage(1)
          }}
          className="min-h-11 text-sm text-aura-primary hover:underline"
        >
          Limpar Filtros
        </button>
      </div>

      <div className="overflow-hidden rounded-lg border border-aura-border bg-white shadow-sm">
        <ResponsiveEntityList
          items={slice}
          getKey={(t) => t.id}
          renderCard={renderCard}
          columns={columns}
          emptyTitle="Nenhum salão encontrado."
          emptyAction={
            <Button onClick={openCreate}>
              <Plus className="h-4 w-4" />
              Novo Salão
            </Button>
          }
        />
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

      <NovoSalaoModal
        open={createOpen}
        onClose={() => {
          setCreateOpen(false)
          setError('')
        }}
        planos={db.planos.map((p) => ({ id: p.id, nome: p.nome }))}
        existingSlugs={db.tenants.map((t) => t.slug)}
        onSuccess={() => {
          refresh()
          setSuccess('Salão cadastrado com sucesso! Senha da dona: AgendaGlow@2026')
        }}
        onError={setError}
      />

      <Modal
        open={!!editOpen}
        onClose={() => {
          setEditOpen(null)
          resetIdentityForm()
          setError('')
        }}
        title="Editar Salão"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => {
                setEditOpen(null)
                resetIdentityForm()
                setError('')
              }}
            >
              Cancelar
            </Button>
            <Button onClick={handleUpdate}>Salvar</Button>
          </>
        }
      >
        {error && editOpen && (
          <p className="mb-3 text-sm text-red-600" role="alert">
            {error}
          </p>
        )}
        <TenantIdentityForm {...identityFormProps} showPlano={false} />
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
        onClose={() => {
          if (donaSubmitting) return
          setDonaOpen(null)
        }}
        title="Criar dona do salão"
        footer={
          <>
            <Button
              variant="secondary"
              disabled={donaSubmitting}
              onClick={() => setDonaOpen(null)}
            >
              Cancelar
            </Button>
            <Button
              loading={donaSubmitting}
              disabled={donaSubmitting}
              onClick={async () => {
                if (!donaOpen) return
                setError('')
                setDonaSubmitting(true)
                try {
                  await createDonaForTenant(donaOpen.id, donaNome, donaEmail)
                  refresh()
                  setDonaOpen(null)
                  setSuccess('Dona criada. Senha padrão: AgendaGlow@2026')
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Erro ao criar dona')
                } finally {
                  setDonaSubmitting(false)
                }
              }}
            >
              Criar
            </Button>
          </>
        }
      >
        {error && donaOpen && (
          <p className="mb-3 text-sm text-red-600" role="alert">
            {error}
          </p>
        )}
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

function mapTenantError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 409 || err.code === 'slug_already_exists') {
      return 'Este slug já está em uso'
    }
    if (err.code === 'invalid_slug') {
      return 'Slug inválido. Use apenas letras minúsculas, números e hífens.'
    }
    if (err.code === 'missing_nome_comercial') {
      return 'Informe o nome do salão.'
    }
  }
  if (err instanceof Error && err.message) return err.message
  return 'Não foi possível salvar o salão. Tente novamente.'
}
