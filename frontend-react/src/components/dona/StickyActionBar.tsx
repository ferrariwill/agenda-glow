import { Loader2 } from 'lucide-react'

export interface StickyActionBarProps {
  onSave: () => void
  onCancel: () => void
  isSaving?: boolean
  isEdit?: boolean
  disabled?: boolean
  hasUnsavedChanges?: boolean
}

export function StickyActionBar({
  onSave,
  onCancel,
  isSaving = false,
  isEdit = false,
  disabled = false,
  hasUnsavedChanges = false,
}: StickyActionBarProps) {
  const saveLabel = isEdit ? 'Salvar alterações' : 'Salvar Profissional'

  return (
    <footer
      role="region"
      aria-label="Ações do formulário"
      aria-busy={isSaving}
      className={[
        'sticky z-30 w-full border-t border-[#e5d3c8]/50 bg-white/95 backdrop-blur-md',
        'shadow-[0_-4px_20px_rgba(0,0,0,0.06)]',
        /* Acima da BottomNav no mobile; encostado no rodapé em lg+ (já dentro da coluna do layout) */
        'bottom-[calc(4rem+env(safe-area-inset-bottom,0px))] lg:bottom-0',
      ].join(' ')}
    >
      <div className="flex items-center justify-between gap-4 px-4 py-4 sm:px-6">
        <div className="min-w-0 text-sm text-[#514440]">
          {hasUnsavedChanges ? (
            <span className="inline-flex items-center gap-2 font-medium">
              <span className="h-2 w-2 shrink-0 rounded-full bg-[#7d5141]" aria-hidden />
              Modificações pendentes
            </span>
          ) : (
            <span className="text-[#83746f]">Sem alterações pendentes</span>
          )}
        </div>
        <div className="flex shrink-0 flex-wrap items-center justify-end gap-3">
          <button
            type="button"
            onClick={onCancel}
            disabled={isSaving}
            className="rounded-lg border border-[#d6c2bd] px-5 py-2.5 text-sm font-semibold text-[#514440] transition-colors hover:bg-[#f4f3f2] disabled:opacity-50"
          >
            Cancelar
          </button>
          <button
            type="button"
            onClick={onSave}
            disabled={disabled || isSaving}
            aria-busy={isSaving}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-[#7d5141] px-6 py-2.5 text-sm font-semibold text-white shadow-md transition-all hover:opacity-90 active:scale-[0.98] disabled:opacity-50"
          >
            {isSaving && <Loader2 className="h-4 w-4 animate-spin" aria-hidden />}
            {isSaving ? `${saveLabel}...` : saveLabel}
          </button>
        </div>
      </div>
    </footer>
  )
}
