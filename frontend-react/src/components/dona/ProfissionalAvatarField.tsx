import { useEffect, useRef, useState } from 'react'
import { Edit3, ImagePlus, Loader2 } from 'lucide-react'
import { ApiError } from '../../lib/api'
import { uploadProfissionalFoto } from '../../utils/mockDb'

const STOCK_AVATARS = [
  'https://images.unsplash.com/photo-1595476108010-b4d1f102b1b1?w=400&q=80',
  'https://images.unsplash.com/photo-1503951914875-452162b0f3f1?w=400&q=80',
  'https://images.unsplash.com/photo-1604654894610-df63bc536371?w=400&q=80',
  'https://images.unsplash.com/photo-1570172619644-dfd03ed5d881?w=400&q=80',
]

const ACCEPT = 'image/png,image/jpeg,image/webp'
const MAX_BYTES = 2 * 1024 * 1024
const ALLOWED_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])

export interface ProfissionalAvatarFieldProps {
  fotoUrl: string
  onFotoUrlChange: (url: string) => void
  /** Em modo criar, o arquivo fica pendente até existir id. */
  onPendingFileChange?: (file: File | null) => void
  profissionalId?: string
  disabled?: boolean
  onError?: (msg: string) => void
}

function isAllowedImage(file: File): boolean {
  if (ALLOWED_TYPES.has(file.type)) return true
  const lower = file.name.toLowerCase()
  return lower.endsWith('.png') || lower.endsWith('.jpg') || lower.endsWith('.jpeg') || lower.endsWith('.webp')
}

function mapUploadError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code === 'invalid_image' || err.status === 400) {
      return 'Imagem inválida ou maior que 2MB. Use PNG, JPG ou WebP.'
    }
    if (err.code === 'professional_not_found' || err.status === 404) {
      return 'Profissional não encontrado.'
    }
    if (err.status === 502) {
      return 'Não foi possível enviar a foto. Tente de novo.'
    }
  }
  return err instanceof Error ? err.message : 'Não foi possível enviar a foto.'
}

export function ProfissionalAvatarField({
  fotoUrl,
  onFotoUrlChange,
  onPendingFileChange,
  profissionalId,
  disabled = false,
  onError,
}: ProfissionalAvatarFieldProps) {
  const fileRef = useRef<HTMLInputElement>(null)
  const [pickerOpen, setPickerOpen] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [objectPreview, setObjectPreview] = useState<string | null>(null)

  useEffect(() => {
    return () => {
      if (objectPreview) URL.revokeObjectURL(objectPreview)
    }
  }, [objectPreview])

  const displayUrl = objectPreview || fotoUrl

  const clearObjectPreview = () => {
    setObjectPreview((prev) => {
      if (prev) URL.revokeObjectURL(prev)
      return null
    })
  }

  const applyFile = async (file: File | null) => {
    if (!file) return
    if (!isAllowedImage(file)) {
      onError?.('Imagem inválida. Use PNG, JPG ou WebP.')
      if (fileRef.current) fileRef.current.value = ''
      return
    }
    if (file.size > MAX_BYTES) {
      onError?.('Imagem inválida ou maior que 2MB. Use PNG, JPG ou WebP.')
      if (fileRef.current) fileRef.current.value = ''
      return
    }

    const preview = URL.createObjectURL(file)
    setObjectPreview((prev) => {
      if (prev) URL.revokeObjectURL(prev)
      return preview
    })

    if (!profissionalId) {
      onPendingFileChange?.(file)
      onFotoUrlChange(preview)
      setPickerOpen(false)
      return
    }

    setUploading(true)
    try {
      const res = await uploadProfissionalFoto(profissionalId, file)
      clearObjectPreview()
      onPendingFileChange?.(null)
      onFotoUrlChange(res.foto_url)
      setPickerOpen(false)
    } catch (err) {
      clearObjectPreview()
      onError?.(mapUploadError(err))
    } finally {
      setUploading(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const selectStock = (url: string) => {
    clearObjectPreview()
    onPendingFileChange?.(null)
    onFotoUrlChange(url)
    setPickerOpen(false)
  }

  const removeFoto = () => {
    clearObjectPreview()
    onPendingFileChange?.(null)
    onFotoUrlChange('')
    setPickerOpen(false)
  }

  return (
    <div className="col-span-2 mb-2 space-y-4">
      <div className="flex flex-col items-start gap-6 sm:flex-row sm:items-center">
        <button
          type="button"
          disabled={disabled || uploading}
          onClick={() => setPickerOpen((o) => !o)}
          className="group relative shrink-0"
          aria-busy={uploading}
          aria-label="Alterar foto do perfil"
        >
          <div className="flex h-24 w-24 items-center justify-center overflow-hidden rounded-full border-2 border-dashed border-[#d6c2bd] bg-[#f4f3f2] transition-colors group-hover:border-[#7d5141]">
            {displayUrl ? (
              <img src={displayUrl} alt="" className="h-full w-full object-cover" />
            ) : (
              <ImagePlus className="h-10 w-10 text-[#83746f] group-hover:text-[#7d5141]" />
            )}
          </div>
          <span className="absolute bottom-0 right-0 rounded-full border-2 border-white bg-[#7d5141] p-1 text-white shadow-sm">
            {uploading ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Edit3 className="h-3.5 w-3.5" />
            )}
          </span>
        </button>

        <div className="flex-1">
          <p className="text-xs font-bold uppercase tracking-widest text-[#514440]">
            Foto do Perfil
          </p>
          <p className="mt-1 text-sm text-[#514440]">
            JPG, PNG ou WebP · máx. 2MB · min. 400×400 recomendado
          </p>
          <div className="mt-2 flex flex-wrap gap-3">
            <button
              type="button"
              disabled={disabled || uploading}
              onClick={() => fileRef.current?.click()}
              className="text-sm font-bold text-[#7d5141] hover:underline disabled:opacity-60"
            >
              Enviar do dispositivo
            </button>
            <button
              type="button"
              disabled={disabled || uploading}
              onClick={() => setPickerOpen((o) => !o)}
              className="text-sm font-bold text-[#7d5141] hover:underline disabled:opacity-60"
            >
              {pickerOpen ? 'Fechar opções' : 'Avatares prontos'}
            </button>
          </div>
        </div>
      </div>

      <input
        ref={fileRef}
        type="file"
        accept={ACCEPT}
        className="sr-only"
        disabled={disabled || uploading}
        aria-label="Enviar foto do dispositivo"
        onChange={(e) => void applyFile(e.target.files?.[0] ?? null)}
      />

      {pickerOpen && (
        <div className="flex flex-wrap gap-2">
          {STOCK_AVATARS.map((url) => (
            <button
              key={url}
              type="button"
              disabled={disabled || uploading}
              onClick={() => selectStock(url)}
              className={[
                'h-14 w-14 overflow-hidden rounded-full border-2',
                fotoUrl === url && !objectPreview ? 'border-[#7d5141]' : 'border-transparent',
              ].join(' ')}
            >
              <img src={url} alt="" className="h-full w-full object-cover" />
            </button>
          ))}
          <button
            type="button"
            disabled={disabled || uploading}
            onClick={removeFoto}
            className="rounded-lg border border-dashed border-[#d6c2bd] px-3 py-1 text-xs text-[#514440]"
          >
            Remover
          </button>
        </div>
      )}
    </div>
  )
}
