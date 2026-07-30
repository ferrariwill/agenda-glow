import { describe, expect, it } from 'vitest'
import { ApiError, SUBSCRIPTION_BLOCKED_CODE } from '../lib/api'
import { phaseFromError } from './bootstrapPhase'

describe('phaseFromError', () => {
  it('identifica o 402 específico de assinatura bloqueada', () => {
    const error = new ApiError('Assinatura suspensa', 402, SUBSCRIPTION_BLOCKED_CODE)

    expect(phaseFromError(error)).toBe('blocked')
  })

  it('não confunde outros erros da API com bloqueio de assinatura', () => {
    expect(phaseFromError(new ApiError('internal_error', 500))).toBe('error')
  })

  it('trata falha de transporte como indisponibilidade', () => {
    expect(phaseFromError(new TypeError('Failed to fetch'))).toBe('error')
  })
})
