import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { DatePicker } from './DatePicker'

describe('DatePicker touch (DEV-114)', () => {
  it('dimensiona o painel e dias para hit area ≥ 44×44 com gap-2', () => {
    const html = renderToStaticMarkup(
      <DatePicker value="2026-07-15" onChange={() => {}} />,
    )
    expect(html).toContain('w-[382px]')
    expect(html).toContain('gap-2')
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('min-w-touch-min')
  })
})
