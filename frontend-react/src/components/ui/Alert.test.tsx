import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { Alert } from './Alert'

describe('Alert', () => {
  it('emite role="alert" por padrão na variante error', () => {
    const html = renderToStaticMarkup(<Alert variant="error">Falha</Alert>)
    expect(html).toContain('role="alert"')
  })

  it('não força role="alert" em warning/info', () => {
    expect(renderToStaticMarkup(<Alert variant="warning">Aviso</Alert>)).not.toContain(
      'role="alert"',
    )
    expect(renderToStaticMarkup(<Alert variant="info">Info</Alert>)).not.toContain('role="alert"')
  })
})
