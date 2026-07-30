import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { EarlySlotQueueInactiveBanner } from './EarlySlotQueueInactiveBanner'

function render(active?: boolean, canReconnectWhatsApp = false): string {
  return renderToStaticMarkup(
    <MemoryRouter>
      <EarlySlotQueueInactiveBanner
        active={active}
        canReconnectWhatsApp={canReconnectWhatsApp}
      />
    </MemoryRouter>,
  )
}

describe('EarlySlotQueueInactiveBanner', () => {
  it('só mostra o link de reconexão para quem pode abrir /admin/whatsapp', () => {
    const dona = render(false, true)
    const secretaria = render(false, false)

    expect(dona).toContain('href="/admin/whatsapp"')
    expect(secretaria).not.toContain('href="/admin/whatsapp"')
    expect(secretaria).toContain('dona ou administradora')
  })

  it('não trata campo ausente de backend antigo como fila inativa', () => {
    expect(render(undefined)).toBe('')
    expect(render(true)).toBe('')
  })
})
