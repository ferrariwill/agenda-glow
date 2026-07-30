import { Alert } from '../../ui/Alert'
import { Button } from '../../ui/Button'
import { Input } from '../../ui/Input'
import { Modal } from '../../ui/Modal'

interface GestaoCancelarDialogProps {
  open: boolean
  reasonRequired: boolean
  motivo: string
  error?: string
  loading: boolean
  onMotivoChange: (value: string) => void
  onClose: () => void
  onConfirm: () => void
}

export function GestaoCancelarDialog({
  open,
  reasonRequired,
  motivo,
  error,
  loading,
  onMotivoChange,
  onClose,
  onConfirm,
}: GestaoCancelarDialogProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Cancelar este horário?"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Voltar</Button>
          <Button
            variant="danger"
            loading={loading}
            disabled={reasonRequired && !motivo.trim()}
            onClick={onConfirm}
          >
            Confirmar cancelamento
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <p className="text-sm text-aura-muted">
          Ao confirmar, o horário será liberado para outra pessoa.
        </p>
        {reasonRequired && (
          <Input
            label="Motivo do cancelamento"
            value={motivo}
            onChange={(event) => onMotivoChange(event.target.value)}
            placeholder="Conte brevemente o motivo"
            required
          />
        )}
        {error && <Alert variant="error">{error}</Alert>}
      </div>
    </Modal>
  )
}
