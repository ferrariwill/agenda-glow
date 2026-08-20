import { useId, useRef } from 'react'
import { Upload, X } from 'lucide-react'
import { Button } from '../ui/Button'

const MAX_LOGO_BYTES = 2 * 1024 * 1024
const ACCEPT = 'image/png,image/jpeg,.png,.jpg,.jpeg'

export interface LogoUploadFieldProps {
  file: File | null
  preview?: string
  onChange: (file: File | null, preview: string | undefined) => void
  onError?: (message: string) => void
  disabled?: boolean
}

export function LogoUploadField({
  file,
  preview,
  onChange,
  onError,
  disabled = false,
}: LogoUploadFieldProps) {
  const inputId = useId()
  const inputRef = useRef<HTMLInputElement>(null)

  const clear = () => {
    if (inputRef.current) inputRef.current.value = ''
    onChange(null, undefined)
  }

  const applyFile = (next: File | null) => {
    if (!next) {
      clear()
      return
    }
    if (!['image/png', 'image/jpeg'].includes(next.type)) {
      onError?.('Logo deve ser .png ou .jpg')
      if (inputRef.current) inputRef.current.value = ''
      return
    }
    if (next.size > MAX_LOGO_BYTES) {
      onError?.('Logo deve ter menos de 2MB')
      if (inputRef.current) inputRef.current.value = ''
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      onChange(next, reader.result as string)
    }
    reader.readAsDataURL(next)
  }

  return (
    <div className="space-y-3">
      <p className="text-sm font-medium text-aura-anthracite">Logo do salão</p>
      <label
        htmlFor={inputId}
        className={[
          'flex min-h-[7.5rem] cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-aura-border bg-aura-surface/40 px-4 py-5 text-center transition-colors',
          disabled ? 'pointer-events-none opacity-60' : 'hover:border-aura-primary/50 hover:bg-aura-surface',
        ].join(' ')}
      >
        {preview ? (
          <img
            src={preview}
            alt=""
            className="h-16 w-16 rounded-full object-cover ring-2 ring-white"
          />
        ) : (
          <Upload className="h-6 w-6 text-aura-muted" aria-hidden />
        )}
        <span className="text-sm text-aura-anthracite">
          Enviar logo do computador ou celular
        </span>
        <span className="text-xs text-aura-muted">PNG ou JPG, máx. 2MB</span>
        <input
          ref={inputRef}
          id={inputId}
          type="file"
          accept={ACCEPT}
          className="sr-only"
          disabled={disabled}
          aria-label="Enviar logo do computador ou celular"
          onChange={(e) => applyFile(e.target.files?.[0] ?? null)}
        />
      </label>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="secondary"
          disabled={disabled}
          onClick={() => inputRef.current?.click()}
        >
          Escolher arquivo
        </Button>
        {(file || preview) && (
          <>
            <span className="min-w-0 flex-1 truncate text-sm text-aura-muted">
              {file?.name ?? 'Logo selecionada'}
            </span>
            <button
              type="button"
              disabled={disabled}
              onClick={clear}
              className="inline-flex min-h-touch-min min-w-touch-min items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface hover:text-aura-anthracite"
              aria-label="Remover logo"
            >
              <X className="h-4 w-4" />
            </button>
          </>
        )}
      </div>
    </div>
  )
}
