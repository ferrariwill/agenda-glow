import { useState } from 'react'
import { KeyRound } from 'lucide-react'
import { ChangePasswordModal } from './ChangePasswordModal'
import { ToastFeedback } from '../ui/ToastFeedback'

interface ChangePasswordEntryProps {
  /** Fecha o drawer mobile ao abrir o modal, se o layout passar o callback. */
  onNavigate?: () => void
  className?: string
}

export function ChangePasswordEntry({ onNavigate, className = '' }: ChangePasswordEntryProps) {
  const [open, setOpen] = useState(false)
  const [toast, setToast] = useState<string | null>(null)

  return (
    <>
      <button
        type="button"
        onClick={() => {
          setOpen(true)
          onNavigate?.()
        }}
        className={[
          'flex min-h-touch-min w-full touch-manipulation items-center gap-3 rounded-lg px-3 py-2 text-sm text-aura-muted hover:bg-aura-surface',
          className,
        ]
          .filter(Boolean)
          .join(' ')}
      >
        <KeyRound className="h-[18px] w-[18px]" />
        Alterar senha
      </button>
      <ChangePasswordModal
        open={open}
        onClose={() => setOpen(false)}
        onSuccess={() => setToast('Senha alterada com sucesso')}
      />
      <ToastFeedback
        message={toast}
        onDismiss={() => setToast(null)}
        variant="success"
      />
    </>
  )
}
