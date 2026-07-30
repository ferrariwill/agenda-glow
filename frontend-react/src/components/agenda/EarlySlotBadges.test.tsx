import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { EarlySlotOfferIndicator, EarlySlotOptInBadge } from './EarlySlotBadges'

describe('EarlySlotBadges compact', () => {
  it('renderiza sinais compactos sem badge de texto longo', () => {
    const optIn = renderToStaticMarkup(<EarlySlotOptInBadge enabled compact />)
    expect(optIn).toContain('Aceita antecipação')
    expect(optIn).not.toContain('Aceita antecipação</span>')

    const offer = renderToStaticMarkup(
      <EarlySlotOfferIndicator
        compact
        offer={{
          round_id: 'r1',
          offer_status: 'PENDENTE',
          expires_at: '2026-08-01T13:05:00-03:00',
        }}
      />,
    )
    expect(offer).toContain('Oferta')
    expect(offer).toContain('aria-label')
  })
})
