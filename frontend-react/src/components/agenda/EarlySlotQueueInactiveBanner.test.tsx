import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { EarlySlotQueueInactiveBanner } from './EarlySlotQueueInactiveBanner'

function render(active?: boolean): string {
  return renderToStaticMarkup(
    <MemoryRouter>
      <EarlySlotQueueInactiveBanner active={active} />
    </MemoryRouter>,
  )
}

describe('EarlySlotQueueInactiveBanner', () => {
  it('exibe aviso e reconexão somente quando o sinal canônico é false', () => {
    const html = render(false)

    expect(html).toContain('fila de antecipação está inativa')
    expect(html).toContain('href="/admin/whatsapp"')
  })

  it('não trata campo ausente de backend antigo como fila inativa', () => {
    expect(render(undefined)).toBe('')
    expect(render(true)).toBe('')
  })
})
