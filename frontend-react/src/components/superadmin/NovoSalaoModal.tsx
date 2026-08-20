import { useEffect, useState } from 'react'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Modal } from '../ui/Modal'
import { ApiError } from '../../lib/api'
import type { Tenant } from '../../types'
import { createTenantWithOwner, SLUG_REGEX } from '../../utils/mockDb'
import { LogoUploadField } from './LogoUploadField'

export interface NovoSalaoModalProps {
  open: boolean
  onClose: () => void
  planos: { id: string; nome: string }[]
  existingSlugs: string[]
  onSuccess: (tenant: Tenant) => void
  onError: (message: string) => void
}

export function NovoSalaoModal({
  open,
  onClose,
  planos,
  existingSlugs,
  onSuccess,
  onError,
}: NovoSalaoModalProps) {
  const [nome, setNome] = useState('')
  const [slug, setSlug] = useState('')
  const [planoId, setPlanoId] = useState(planos[0]?.id ?? '')
  const [donaNome, setDonaNome] = useState('')
  const [donaEmail, setDonaEmail] = useState('')
  const [logoFile, setLogoFile] = useState<File | null>(null)
  const [logoPreview, setLogoPreview] = useState<string>()
  const [submitting, setSubmitting] = useState(false)
  const [localError, setLocalError] = useState('')

  useEffect(() => {
    if (!open) return
    setNome('')
    setSlug('')
    setPlanoId(planos[0]?.id ?? '')
    setDonaNome('')
    setDonaEmail('')
    setLogoFile(null)
    setLogoPreview(undefined)
    setSubmitting(false)
    setLocalError('')
  }, [open, planos])

  const handleSubmit = async () => {
    setLocalError('')
    const nomeTrim = nome.trim()
    const slugTrim = slug.trim().toLowerCase()
    const donaNomeTrim = donaNome.trim()
    const donaEmailTrim = donaEmail.trim()

    if (!nomeTrim) {
      setLocalError('Informe o nome do salão.')
      return
    }
    if (!SLUG_REGEX.test(slugTrim)) {
      setLocalError('Slug inválido. Use apenas letras minúsculas, números e hífens.')
      return
    }
    if (existingSlugs.includes(slugTrim)) {
      setLocalError('Este slug já está em uso')
      return
    }
    if (!donaNomeTrim) {
      setLocalError('Informe o nome da dona.')
      return
    }
    if (!donaEmailTrim) {
      setLocalError('Informe o e-mail da dona.')
      return
    }

    setSubmitting(true)
    try {
      const tenant = await createTenantWithOwner({
        nome: nomeTrim,
        slug: slugTrim,
        plano_id: planoId,
        dona_nome: donaNomeTrim,
        dona_email: donaEmailTrim,
        logoFile,
        logoPreview,
      })
      onSuccess(tenant)
      onClose()
    } catch (err) {
      const message = mapCreateError(err)
      setLocalError(message)
      onError(message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={() => {
        if (submitting) return
        onClose()
      }}
      title="Novo Salão"
      maxWidthClass="max-w-lg"
      footer={
        <>
          <Button
            variant="secondary"
            disabled={submitting}
            onClick={() => {
              if (submitting) return
              onClose()
            }}
          >
            Cancelar
          </Button>
          <Button onClick={handleSubmit} loading={submitting} disabled={submitting}>
            Cadastrar
          </Button>
        </>
      }
    >
      {localError && (
        <Alert variant="error" className="mb-4" onDismiss={() => setLocalError('')}>
          {localError}
        </Alert>
      )}

      <div className="space-y-6">
        <section className="space-y-4">
          <h3 className="text-sm font-semibold text-aura-anthracite">Salão</h3>
          <Input
            label="Nome do salão"
            value={nome}
            onChange={(e) => setNome(e.target.value)}
            disabled={submitting}
            autoComplete="organization"
          />
          <Input
            label="Slug público"
            value={slug}
            onChange={(e) => setSlug(e.target.value.toLowerCase())}
            placeholder="meu-salao-luxo"
            disabled={submitting}
          />
          <div>
            <label className="mb-1.5 block text-sm font-medium text-aura-anthracite">
              Plano SaaS
            </label>
            <select
              value={planoId}
              onChange={(e) => setPlanoId(e.target.value)}
              disabled={submitting}
              className="min-h-touch-min w-full rounded-lg border border-aura-border bg-white px-3 py-2.5 text-sm"
            >
              {planos.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome}
                </option>
              ))}
            </select>
          </div>
        </section>

        <section>
          <h3 className="mb-3 text-sm font-semibold text-aura-anthracite">Identidade</h3>
          <LogoUploadField
            file={logoFile}
            preview={logoPreview}
            disabled={submitting}
            onError={setLocalError}
            onChange={(nextFile, nextPreview) => {
              setLocalError('')
              setLogoFile(nextFile)
              setLogoPreview(nextPreview)
            }}
          />
        </section>

        <section className="space-y-4">
          <div>
            <h3 className="text-sm font-semibold text-aura-anthracite">Dona do salão</h3>
            <p className="mt-1 text-xs text-aura-muted">
              Senha inicial: <code className="font-mono">AgendaGlow@2026</code>
            </p>
          </div>
          <Input
            label="Nome da dona"
            value={donaNome}
            onChange={(e) => setDonaNome(e.target.value)}
            disabled={submitting}
            autoComplete="name"
          />
          <Input
            label="E-mail da dona"
            type="email"
            value={donaEmail}
            onChange={(e) => setDonaEmail(e.target.value)}
            disabled={submitting}
            autoComplete="email"
          />
        </section>
      </div>
    </Modal>
  )
}

function mapCreateError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 409 || err.code === 'slug_already_exists') {
      return 'Este slug já está em uso'
    }
    if (err.code === 'email_already_exists') {
      return 'E-mail da dona já cadastrado'
    }
    if (err.code === 'invalid_slug') {
      return 'Slug inválido. Use apenas letras minúsculas, números e hífens.'
    }
    if (err.code === 'missing_nome_comercial' || err.code === 'missing_nome') {
      return 'Informe o nome do salão.'
    }
    if (err.code === 'missing_email') {
      return 'Informe o e-mail da dona.'
    }
    if (err.code === 'invalid_image_type') {
      return 'Logo deve ser .png ou .jpg'
    }
    if (err.code === 'file_too_large') {
      return 'Logo deve ter menos de 2MB'
    }
    if (err.message) return err.message
  }
  if (err instanceof Error && err.message) return err.message
  return 'Não foi possível cadastrar o salão. Tente novamente.'
}
