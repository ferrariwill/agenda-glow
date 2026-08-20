import { useEffect, useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { CreditCard, MapPin, Scissors, Upload } from 'lucide-react'
import { DonaLayout, DonaFooter, PageHeader } from '../../components/dona/DonaLayout'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../contexts/AuthContext'
import { PlanLimitExceededError } from '../../types'
import {
  assignPlanToTenant,
  getDb,
  getDonaProfissional,
  getPlano,
  renewTenant,
  setDonaComoProfissional,
  unsetDonaComoProfissional,
  updateTenant,
} from '../../utils/mockDb'
import { formatBRL, formatDateBR } from '../../utils/format'
import { ExpedienteForm, validateAllExpedientes } from './ExpedienteForm'

const defaultExpedientes = [1, 2, 3, 4, 5].map((dia) => ({
  dia_semana: dia,
  horario_entrada: '09:00',
  horario_saida: '18:00',
  inicio_almoco: '12:00',
  fim_almoco: '13:00',
}))

export function ConfiguracoesSalao() {
  const { session, refreshSession } = useAuth()
  const location = useLocation()
  const tenantId = session?.user.tenant_id ?? ''
  const userId = session?.user.id ?? ''
  const [db, setDb] = useState(getDb())
  const tenant = db.tenants.find((t) => t.id === tenantId)
  const plano = tenant ? getPlano(tenant.plano_id) : undefined
  const especialidades = db.especialidades.filter((e) => e.tenant_id === tenantId)
  const donaProf = getDonaProfissional(tenantId)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')
  const [planoModalOpen, setPlanoModalOpen] = useState(false)
  const [selectedPlanoId, setSelectedPlanoId] = useState('')
  const [savingPlano, setSavingPlano] = useState(false)

  const [atuaComoProf, setAtuaComoProf] = useState(!!tenant?.dona_atua_como_profissional)
  const [espId, setEspId] = useState(donaProf?.especialidade_id ?? especialidades[0]?.id ?? '')
  const [comissao, setComissao] = useState(donaProf?.comissao_percent ?? 50)
  const [expedientes, setExpedientes] = useState(donaProf?.expedientes ?? defaultExpedientes)

  const [form, setForm] = useState({
    nome: tenant?.nome ?? '',
    email_contato: tenant?.email_contato ?? '',
    bio: tenant?.bio ?? '',
    cep: tenant?.cep ?? '',
    logradouro: tenant?.logradouro ?? '',
    cidade: tenant?.cidade ?? '',
    uf: tenant?.uf ?? '',
    logo_url: tenant?.logo_url ?? '',
  })

  const planosDisponiveis = useMemo(
    () => db.planos.filter((p) => p.ativo && p.id !== tenant?.plano_id),
    [db.planos, tenant?.plano_id],
  )

  useEffect(() => {
    const t = getDb().tenants.find((x) => x.id === tenantId)
    const prof = getDonaProfissional(tenantId)
    setAtuaComoProf(!!t?.dona_atua_como_profissional)
    if (prof) {
      setEspId(prof.especialidade_id)
      setComissao(prof.comissao_percent)
      setExpedientes(prof.expedientes)
    }
  }, [tenantId, db])

  useEffect(() => {
    if (location.hash !== '#assinatura') return
    const el = document.getElementById('assinatura')
    if (!el) return
    const timer = window.setTimeout(() => {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 50)
    return () => window.clearTimeout(timer)
  }, [location.hash, location.pathname])

  const refresh = () => setDb(getDb())

  const openPlanoModal = () => {
    setError('')
    setSelectedPlanoId(planosDisponiveis[0]?.id ?? '')
    setPlanoModalOpen(true)
  }

  const confirmAlterarPlano = async () => {
    if (!tenantId || !selectedPlanoId) return
    setSavingPlano(true)
    setError('')
    try {
      await assignPlanToTenant(tenantId, selectedPlanoId)
      refresh()
      setPlanoModalOpen(false)
      setMsg('Plano alterado com sucesso!')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Não foi possível alterar o plano.')
    } finally {
      setSavingPlano(false)
    }
  }

  const save = () => {
    if (!tenantId) return
    setError('')
    updateTenant(tenantId, form)
    refresh()
    setMsg('Configurações salvas com sucesso!')
  }

  const savePerfilProfissional = () => {
    if (!tenantId || !userId) return
    setError('')
    setMsg('')
    try {
      if (atuaComoProf) {
        const expErr = validateAllExpedientes(expedientes)
        if (expErr) {
          setError(expErr)
          return
        }
        if (!espId) {
          setError('Selecione uma especialidade')
          return
        }
        if (comissao < 0 || comissao > 100) {
          setError('Comissão deve estar entre 0% e 100%')
          return
        }
        setDonaComoProfissional(tenantId, userId, {
          especialidade_id: espId,
          comissao_percent: comissao,
          expedientes,
        })
        refreshSession()
        setMsg('Você agora aparece na agenda como profissional. Acesse "Minha agenda" no menu.')
      } else {
        unsetDonaComoProfissional(tenantId, userId)
        refreshSession()
        setMsg('Perfil de atendimento desativado — você não aparece mais na grade da agenda.')
      }
      refresh()
    } catch (err) {
      if (err instanceof PlanLimitExceededError) {
        setError('Limite de profissionais do plano atingido. Desative outro profissional ou faça upgrade.')
      } else {
        setError(err instanceof Error ? err.message : 'Erro ao salvar perfil profissional')
      }
    }
  }

  const handleLogo = (file: File | null) => {
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => setForm({ ...form, logo_url: reader.result as string })
    reader.readAsDataURL(file)
  }

  return (
    <DonaLayout searchPlaceholder="Buscar configurações…">
      <PageHeader
        title="Configurações do Salão"
        subtitle="Identidade visual, endereço e assinatura do seu estabelecimento."
      />

      {msg && (
        <Alert variant="info" className="mb-4" onDismiss={() => setMsg('')}>
          {msg}
        </Alert>
      )}
      {error && (
        <Alert variant="error" className="mb-4" onDismiss={() => setError('')}>
          {error}
        </Alert>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <div className="rounded-lg border border-aura-border bg-white p-6 shadow-sm">
            <h2 className="mb-4 font-display text-lg font-semibold">Informações gerais</h2>
            <div className="mb-4 flex items-center gap-4">
              {form.logo_url ? (
                <img src={form.logo_url} alt="" className="h-20 w-20 rounded-lg object-cover" />
              ) : (
                <div className="flex h-20 w-20 items-center justify-center rounded-lg border border-dashed border-aura-border bg-aura-surface">
                  <Upload className="h-6 w-6 text-aura-muted" />
                </div>
              )}
              <label className="cursor-pointer text-sm text-aura-primary hover:underline">
                Alterar logo
                <input type="file" accept=".png,.jpg,.jpeg" className="hidden" onChange={(e) => handleLogo(e.target.files?.[0] ?? null)} />
              </label>
            </div>
            <div className="space-y-4">
              <Input label="Nome do Salão" value={form.nome} onChange={(e) => setForm({ ...form, nome: e.target.value })} />
              <Input label="E-mail de Contato" type="email" value={form.email_contato} onChange={(e) => setForm({ ...form, email_contato: e.target.value })} />
              <div>
                <label className="mb-1.5 block text-sm font-medium">Bio pública</label>
                <textarea
                  value={form.bio}
                  onChange={(e) => setForm({ ...form, bio: e.target.value })}
                  rows={3}
                  className="w-full rounded-lg border border-aura-border px-3 py-2 text-sm"
                />
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-aura-border bg-white p-6 shadow-sm">
            <h2 className="mb-2 flex items-center gap-2 font-display text-lg font-semibold">
              <Scissors className="h-5 w-5 text-aura-primary" />
              Eu também realizo atendimentos
            </h2>
            <p className="mb-4 text-sm text-aura-muted">
              Ative para aparecer na agenda geral com sua própria coluna — como qualquer profissional da equipe.
              Desative se você só gerencia o salão e não atende clientes.
            </p>
            <label className="mb-4 flex cursor-pointer items-center gap-3">
              <input
                type="checkbox"
                checked={atuaComoProf}
                onChange={(e) => setAtuaComoProf(e.target.checked)}
                className="h-4 w-4 rounded border-aura-border text-aura-primary"
              />
              <span className="text-sm font-medium">Atuo como profissional na agenda</span>
            </label>
            {atuaComoProf && (
              <div className="space-y-4 border-t border-aura-border pt-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className="mb-1.5 block text-sm font-medium">Especialidade</label>
                    <select
                      value={espId}
                      onChange={(e) => setEspId(e.target.value)}
                      className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
                    >
                      {especialidades.map((e) => (
                        <option key={e.id} value={e.id}>
                          {e.nome}
                        </option>
                      ))}
                    </select>
                  </div>
                  <Input
                    label="Comissão (%)"
                    type="number"
                    min={0}
                    max={100}
                    value={comissao}
                    onChange={(e) => setComissao(Number(e.target.value))}
                  />
                </div>
                <ExpedienteForm value={expedientes} onChange={setExpedientes} />
              </div>
            )}
            <Button className="mt-4" variant="secondary" onClick={savePerfilProfissional}>
              Salvar perfil de atendimento
            </Button>
          </div>

          <div className="rounded-lg border border-aura-border bg-white p-6 shadow-sm">
            <h2 className="mb-4 flex items-center gap-2 font-display text-lg font-semibold">
              <MapPin className="h-5 w-5 text-aura-primary" />
              Endereço
            </h2>
            <div className="grid gap-4 sm:grid-cols-2">
              <Input label="CEP" value={form.cep} onChange={(e) => setForm({ ...form, cep: e.target.value })} />
              <Input label="UF" value={form.uf} onChange={(e) => setForm({ ...form, uf: e.target.value })} maxLength={2} />
              <div className="sm:col-span-2">
                <Input label="Logradouro" value={form.logradouro} onChange={(e) => setForm({ ...form, logradouro: e.target.value })} />
              </div>
              <Input label="Cidade" value={form.cidade} onChange={(e) => setForm({ ...form, cidade: e.target.value })} />
            </div>
            <div className="mt-4 flex h-40 items-center justify-center rounded-lg bg-aura-surface text-sm text-aura-muted">
              Mapa — {form.cidade || 'Cidade'}, {form.uf || 'UF'}
            </div>
          </div>

          <Button onClick={save}>Salvar alterações</Button>
        </div>

        <div className="space-y-4">
          <div
            id="assinatura"
            className="scroll-mt-24 rounded-lg border border-aura-border bg-white p-6 shadow-sm"
          >
            <h2 className="mb-4 font-display text-lg font-semibold">Assinatura e Perfil</h2>
            <p className="text-sm text-aura-muted">Plano atual</p>
            <p className="font-display text-xl font-semibold text-aura-primary">{plano?.nome ?? '—'}</p>
            <p className="mt-2 text-2xl font-semibold">{formatBRL(plano?.preco_mensal ?? 0)}<span className="text-sm font-normal text-aura-muted">/mês</span></p>
            {tenant && (
              <p className="mt-2 text-sm text-aura-muted">
                Vencimento: {formatDateBR(tenant.data_vencimento)}
              </p>
            )}
            <div className="mt-4 space-y-2">
              <Button
                fullWidth
                variant="secondary"
                onClick={() => {
                  if (tenantId) {
                    renewTenant(tenantId)
                    refresh()
                    setMsg('Assinatura renovada por +12 meses!')
                  }
                }}
              >
                Renovar Assinatura
              </Button>
              <Button fullWidth variant="ghost" onClick={openPlanoModal}>
                Alterar Plano
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-aura-border bg-white p-6 shadow-sm">
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
              <CreditCard className="h-4 w-4 text-aura-primary" />
              Método de Pagamento
            </h3>
            <p className="rounded-lg bg-aura-surface px-3 py-2 font-mono text-sm">•••• •••• •••• 4242</p>
            <p className="mt-1 text-xs text-aura-muted">Visa · Expira 12/28</p>
          </div>
        </div>
      </div>

      <Modal
        open={planoModalOpen}
        onClose={() => setPlanoModalOpen(false)}
        title="Alterar plano"
        footer={
          <>
            <Button variant="secondary" onClick={() => setPlanoModalOpen(false)} disabled={savingPlano}>
              Cancelar
            </Button>
            <Button
              onClick={confirmAlterarPlano}
              disabled={savingPlano || !selectedPlanoId || planosDisponiveis.length === 0}
            >
              {savingPlano ? 'Salvando…' : 'Confirmar'}
            </Button>
          </>
        }
      >
        {planosDisponiveis.length === 0 ? (
          <p className="text-sm text-aura-muted">Não há outros planos ativos disponíveis no momento.</p>
        ) : (
          <>
            <p className="mb-3 text-sm text-aura-muted">
              Plano atual: <strong>{plano?.nome ?? '—'}</strong>
            </p>
            <label className="mb-1.5 block text-sm font-medium" htmlFor="alterar-plano-select">
              Novo plano
            </label>
            <select
              id="alterar-plano-select"
              value={selectedPlanoId}
              onChange={(e) => setSelectedPlanoId(e.target.value)}
              className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
            >
              {planosDisponiveis.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome} — {formatBRL(p.preco_mensal)}/mês · {p.limite_profissionais} profissionais
                </option>
              ))}
            </select>
          </>
        )}
      </Modal>

      <DonaFooter />
    </DonaLayout>
  )
}
