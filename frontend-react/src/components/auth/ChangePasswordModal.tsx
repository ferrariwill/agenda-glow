import { useEffect, useState } from 'react'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Modal } from '../ui/Modal'
import {
  changePasswordErrorMessage,
  changePasswordWithApi,
} from '../../services/authApi'

interface ChangePasswordModalProps {
  open: boolean
  onClose: () => void
  onSuccess?: () => void
}

type FieldErrors = {
  current_password?: string
  new_password?: string
  confirm_password?: string
}

function validate(
  current_password: string,
  new_password: string,
  confirm_password: string,
): FieldErrors | null {
  const errors: FieldErrors = {}

  if (!current_password) errors.current_password = 'Informe a senha atual'
  if (!new_password) errors.new_password = 'Informe a nova senha'
  if (!confirm_password) errors.confirm_password = 'Confirme a nova senha'

  if (new_password && new_password.length < 8) {
    errors.new_password = 'A nova senha deve ter pelo menos 8 caracteres'
  }
  if (new_password && confirm_password && new_password !== confirm_password) {
    errors.confirm_password = 'A confirmação não confere com a nova senha'
  }
  if (current_password && new_password && new_password === current_password) {
    errors.new_password = 'A nova senha deve ser diferente da atual'
  }

  return Object.keys(errors).length > 0 ? errors : null
}

export function ChangePasswordModal({ open, onClose, onSuccess }: ChangePasswordModalProps) {
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [apiError, setApiError] = useState('')
  const [loading, setLoading] = useState(false)

  const resetForm = () => {
    setCurrentPassword('')
    setNewPassword('')
    setConfirmPassword('')
    setFieldErrors({})
    setApiError('')
    setLoading(false)
  }

  useEffect(() => {
    if (open) resetForm()
  }, [open])

  const handleClose = () => {
    if (loading) return
    resetForm()
    onClose()
  }

  const handleSubmit = async () => {
    setApiError('')
    const errors = validate(currentPassword, newPassword, confirmPassword)
    if (errors) {
      setFieldErrors(errors)
      return
    }
    setFieldErrors({})
    setLoading(true)
    try {
      await changePasswordWithApi({
        current_password: currentPassword,
        new_password: newPassword,
        confirm_password: confirmPassword,
      })
      resetForm()
      onClose()
      onSuccess?.()
    } catch (err) {
      setApiError(changePasswordErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Alterar senha"
      footer={
        <>
          <Button variant="secondary" onClick={handleClose} disabled={loading}>
            Cancelar
          </Button>
          <Button onClick={handleSubmit} loading={loading}>
            Salvar
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {apiError && (
          <Alert variant="error" onDismiss={() => setApiError('')}>
            {apiError}
          </Alert>
        )}
        <Input
          label="Senha atual"
          type="password"
          autoComplete="current-password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          error={fieldErrors.current_password}
          disabled={loading}
        />
        <Input
          label="Nova senha"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          error={fieldErrors.new_password}
          disabled={loading}
        />
        <Input
          label="Confirmar nova senha"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          error={fieldErrors.confirm_password}
          disabled={loading}
        />
      </div>
    </Modal>
  )
}
