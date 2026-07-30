import { describe, expect, it } from 'vitest'
import { ApiError } from '../lib/api'
import { earlySlotErrorMessage, isEarlySlotTerminalError } from './earlySlotOfferApi'

describe('earlySlotErrorMessage', () => {
  it('traduz os códigos estáveis do contrato', () => {
    expect(earlySlotErrorMessage(new ApiError('x', 404, 'offer_not_found'))).toBe(
      'Oferta não encontrada.',
    )
    expect(earlySlotErrorMessage(new ApiError('x', 410, 'offer_expired'))).toBe(
      'Esta oferta expirou.',
    )
    expect(earlySlotErrorMessage(new ApiError('x', 409, 'offer_no_longer_available'))).toBe(
      'Este horário não está mais disponível.',
    )
    expect(
      earlySlotErrorMessage(new ApiError('x', 422, 'appointment_no_longer_eligible')),
    ).toBe('Este agendamento não está mais elegível para antecipação.')
  })

  it('usa o status quando a API não manda code', () => {
    expect(earlySlotErrorMessage(new ApiError('x', 410))).toBe('Esta oferta expirou.')
  })

  it('não vaza mensagem técnica em erro genérico', () => {
    const generico = earlySlotErrorMessage(new ApiError('pq: duplicate key value', 500))
    expect(generico).toBe('Não foi possível carregar a oferta. Tente novamente em instantes.')
    expect(earlySlotErrorMessage(new Error('boom'))).toBe(generico)
  })
})

describe('isEarlySlotTerminalError', () => {
  it('reconhece os estados finais do contrato', () => {
    expect(isEarlySlotTerminalError(new ApiError('x', 404, 'offer_not_found'))).toBe(true)
    expect(isEarlySlotTerminalError(new ApiError('x', 409, 'offer_no_longer_available'))).toBe(true)
  })

  it('não trata falha genérica como estado final', () => {
    expect(isEarlySlotTerminalError(new ApiError('x', 500))).toBe(false)
    expect(isEarlySlotTerminalError(new Error('offline'))).toBe(false)
  })
})
