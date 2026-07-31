import { useEffect, useId, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { Button } from './Button'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

function getFocusable(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
    (el) => !el.hasAttribute('disabled') && el.getAttribute('aria-hidden') !== 'true',
  )
}

interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
  footer?: ReactNode
  /** Largura máxima do painel (padrão: max-w-md). */
  maxWidthClass?: string
}

export function Modal({
  open,
  onClose,
  title,
  children,
  footer,
  maxWidthClass = 'max-w-md',
}: ModalProps) {
  const panelRef = useRef<HTMLDivElement>(null)
  const closeBtnRef = useRef<HTMLButtonElement>(null)
  const restoreFocusRef = useRef<HTMLElement | null>(null)
  const titleId = useId()

  useEffect(() => {
    if (!open) return

    const scrollY = window.scrollY
    const { style } = document.body
    const prevOverflow = style.overflow
    const prevPosition = style.position
    const prevTop = style.top
    const prevWidth = style.width

    style.overflow = 'hidden'
    style.position = 'fixed'
    style.top = `-${scrollY}px`
    style.width = '100%'

    return () => {
      style.overflow = prevOverflow
      style.position = prevPosition
      style.top = prevTop
      style.width = prevWidth
      window.scrollTo(0, scrollY)
    }
  }, [open])

  useEffect(() => {
    if (!open) return

    restoreFocusRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null

    const panel = panelRef.current
    const focusables = panel ? getFocusable(panel) : []
    const initial = focusables[0] ?? closeBtnRef.current
    initial?.focus()

    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
        return
      }
      if (e.key !== 'Tab' || !panel) return

      const nodes = getFocusable(panel)
      if (nodes.length === 0) {
        e.preventDefault()
        closeBtnRef.current?.focus()
        return
      }

      const first = nodes[0]
      const last = nodes[nodes.length - 1]
      const active = document.activeElement

      if (e.shiftKey) {
        if (active === first || !panel.contains(active)) {
          e.preventDefault()
          last.focus()
        }
      } else if (active === last || !panel.contains(active)) {
        e.preventDefault()
        first.focus()
      }
    }

    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      restoreFocusRef.current?.focus()
      restoreFocusRef.current = null
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
    >
      <div
        className="absolute inset-0 bg-aura-anthracite/40 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden
      />
      <div
        ref={panelRef}
        className={[
          'relative z-10 flex max-h-[min(92dvh,100%)] w-full flex-col rounded-t-xl border border-aura-border bg-white shadow-xl sm:max-h-[min(90dvh,100%)] sm:rounded-lg',
          maxWidthClass,
        ].join(' ')}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex shrink-0 items-center justify-between border-b border-aura-border px-4 py-4 sm:px-6">
          <h2 id={titleId} className="font-display text-lg font-semibold text-aura-anthracite">
            {title}
          </h2>
          <button
            ref={closeBtnRef}
            type="button"
            onClick={onClose}
            className="inline-flex min-h-touch-min min-w-touch-min touch-manipulation items-center justify-center rounded-lg text-aura-muted hover:bg-aura-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-aura-primary/40"
            aria-label="Fechar"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-4 sm:px-6">
          {children}
        </div>

        {footer && (
          <div className="flex shrink-0 flex-col-reverse gap-2 border-t border-aura-border px-4 py-4 sm:flex-row sm:justify-end sm:px-6 [&>*]:w-full sm:[&>*]:w-auto">
            {footer}
          </div>
        )}
      </div>
    </div>
  )
}

interface ConfirmModalProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmLabel?: string
  loading?: boolean
}

export function ConfirmModal({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Confirmar',
  loading,
}: ConfirmModalProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button onClick={onConfirm} loading={loading}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      <p className="text-sm text-aura-muted">{message}</p>
    </Modal>
  )
}
