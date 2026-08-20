import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError } from '../../lib/api'
import { PlanLimitExceededError } from '../../types'
import { createEspecialidade } from '../../utils/mockDb'
import { Alert } from '../ui/Alert'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Modal } from '../ui/Modal'

export interface QuickSpecialtyModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: (newSpecialty: { id: string; nome: string }) => void
  tenantId: string
}

export function QuickSpecialtyModal({
  isOpen,
  onClose,
  onSuccess,
  tenantId,
}: QuickSpecialtyModalProps) {
  const [nome, setNome] = useState('')
  const [fieldError, setFieldError] = useState('')
  const [submitError, setSubmitError] = useState('')
  const [planLimit, setPlanLimit] = useState(false)
  const [isSaving, setIsSaving] = useState(false)

  useEffect(() => {
    if (!isOpen) return
    setNome('')
    setFieldError('')
    setSubmitError('')
    setPlanLimit(false)
    setIsSaving(false)
  }, [isOpen])

  const handleClose = () => {
    if (isSaving) return
    onClose()
  }

  const handleSubmit = async () => {
    const trimmed = nome.trim()
    if (!trimmed) {
      setFieldError('Informe o nome da especialidade.')
      setSubmitError('')
      setPlanLimit(false)
      return
    }

    setFieldError('')
    setSubmitError('')
    setPlanLimit(false)
    setIsSaving(true)
    try {
      const created = await createEspecialidade(tenantId, trimmed)
      onSuccess({ id: created.id, nome: created.nome })
      setNome('')
      onClose()
    } catch (err) {
      if (
        err instanceof PlanLimitExceededError ||
        (err instanceof ApiError && (err.status === 403 || err.status === 429))
      ) {
        setPlanLimit(true)
        setSubmitError(
          'Limite do plano atingido. Faça upgrade para cadastrar mais especialidades.',
        )
      } else {
        setSubmitError(
          'Não foi possível salvar os dados. Verifique sua conexão e tente novamente.',
        )
      }
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Modal
      open={isOpen}
      onClose={handleClose}
      title="Nova Especialidade"
      footer={
        <>
          <Button variant="secondary" onClick={handleClose} disabled={isSaving}>
            Cancelar
          </Button>
          <Button onClick={handleSubmit} loading={isSaving} aria-busy={isSaving}>
            Salvar
          </Button>
        </>
      }
    >
      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          void handleSubmit()
        }}
      >
        {submitError && (
          <Alert variant="error" onDismiss={() => setSubmitError('')}>
            {submitError}
            {planLimit && (
              <>
                {' '}
                <Link
                  to="/admin/configuracoes"
                  className="font-semibold text-red-900 underline underline-offset-2"
                >
                  Ir para upgrade do plano
                </Link>
              </>
            )}
          </Alert>
        )}
        <Input
          label="Nome da Especialidade"
          value={nome}
          onChange={(e) => {
            setNome(e.target.value)
            if (fieldError) setFieldError('')
          }}
          placeholder="Ex: Barbearia Premium"
          error={fieldError || undefined}
          disabled={isSaving}
          autoComplete="off"
          autoFocus
        />
      </form>
    </Modal>
  )
}
