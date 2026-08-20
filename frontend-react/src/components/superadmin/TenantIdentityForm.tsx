import { Upload } from 'lucide-react'
import { Input } from '../ui/Input'

export interface TenantIdentityFormProps {
  nome: string
  slug: string
  logoPreview?: string
  onNomeChange: (v: string) => void
  onSlugChange: (v: string) => void
  onLogoFile: (file: File | null) => void
  /** No create: select de plano. No edit: omitir (plano via “Atribuir plano”). */
  planoId?: string
  onPlanoChange?: (v: string) => void
  planos?: { id: string; nome: string }[]
  showPlano?: boolean
  slugReadOnly?: boolean
}

export function TenantIdentityForm({
  nome,
  slug,
  logoPreview,
  onNomeChange,
  onSlugChange,
  onLogoFile,
  planoId,
  onPlanoChange,
  planos = [],
  showPlano = false,
  slugReadOnly = false,
}: TenantIdentityFormProps) {
  return (
    <div className="space-y-4">
      <Input
        label="Nome do salão"
        value={nome}
        onChange={(e) => onNomeChange(e.target.value)}
      />
      <div>
        <Input
          label="Slug público"
          value={slug}
          onChange={(e) => onSlugChange(e.target.value.toLowerCase())}
          placeholder="meu-salao-luxo"
          readOnly={slugReadOnly}
          disabled={slugReadOnly}
        />
        {slugReadOnly && (
          <p className="mt-1 text-xs text-aura-muted">
            O slug não pode ser alterado neste contexto.
          </p>
        )}
      </div>
      {showPlano && (
        <div>
          <label className="mb-1.5 block text-sm font-medium">Plano SaaS</label>
          <select
            value={planoId ?? ''}
            onChange={(e) => onPlanoChange?.(e.target.value)}
            className="w-full rounded-lg border border-aura-border px-3 py-2.5 text-sm"
          >
            {planos.map((p) => (
              <option key={p.id} value={p.id}>
                {p.nome}
              </option>
            ))}
          </select>
        </div>
      )}
      <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-aura-border p-4">
        {logoPreview ? (
          <img
            src={logoPreview}
            alt=""
            className="h-10 w-10 rounded-full object-cover"
          />
        ) : (
          <Upload className="h-5 w-5 text-aura-muted" />
        )}
        <span className="text-sm text-aura-muted">Logo PNG/JPG (&lt; 2MB)</span>
        <input
          type="file"
          accept=".png,.jpg,.jpeg"
          className="hidden"
          onChange={(e) => onLogoFile(e.target.files?.[0] ?? null)}
        />
      </label>
    </div>
  )
}
