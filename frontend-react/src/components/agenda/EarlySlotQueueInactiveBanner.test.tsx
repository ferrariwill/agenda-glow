import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { EarlySlotQueueInactiveBanner } from './EarlySlotQueueInactiveBanner'

function render(canReconnectWhatsApp: boolean) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <EarlySlotQueueInactiveBanner
        queueActive={false}
        canReconnectWhatsApp={canReconnectWhatsApp}
      />
    </MemoryRouter>,
  )
}

describe('EarlySlotQueueInactiveBanner', () => {
  it('mostra link administrativo somente para quem pode reconectar', () => {
    expect(render(true)).toContain('href="/admin/whatsapp"')
    expect(render(false)).not.toContain('href="/admin/whatsapp"')
    expect(render(false)).toContain('Peça à dona ou administradora')
  })
})
