import { describe, expect, it } from 'vitest'
import { bookingWhatsAppCopy } from './earlySlotAvailabilityCopy'

describe('bookingWhatsAppCopy', () => {
  it('não promete envio imediato quando o sinal canônico está inativo', () => {
    const copy = bookingWhatsAppCopy(false, true)

    expect(copy.earlySlot).toContain('preferência foi salva')
    expect(copy.earlySlot).toContain('reativar o WhatsApp')
    expect(copy.confirmation).not.toContain('Enviamos')
  })

  it('usa a confirmação por WhatsApp somente quando a API retorna true', () => {
    expect(bookingWhatsAppCopy(true, true).confirmation).toBe(
      'Enviamos a confirmação por WhatsApp.',
    )
    expect(bookingWhatsAppCopy(undefined, true).confirmation).toBe(
      'Agendamento salvo com sucesso.',
    )
  })
})
