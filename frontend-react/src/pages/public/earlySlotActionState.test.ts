import { describe, expect, it } from 'vitest'
import { softReloadResult, visibleOfferError } from './earlySlotActionState'

describe('estado de erro da ação de antecipação', () => {
  it('preserva o erro do submit quando o soft GET conclui sem erro', () => {
    const result = softReloadResult('Este horário não está mais disponível.', '')

    expect(result.actionError).toBe('Este horário não está mais disponível.')
    expect(visibleOfferError(result.loadError, result.actionError)).toBe(
      'Este horário não está mais disponível.',
    )
  })

  it('prioriza erro da ação sobre erro de recarga', () => {
    expect(visibleOfferError('Falha ao recarregar.', 'Oferta expirada.')).toBe(
      'Oferta expirada.',
    )
  })
})
