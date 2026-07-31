import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { DatePicker } from './DatePicker'

describe('DatePicker touch (DEV-114)', () => {
  it('preserva hit area 44×44 + gap-2 sem comprimir no viewport 360', () => {
    const html = renderToStaticMarkup(
      <DatePicker value="2026-07-15" onChange={() => {}} />,
    )
    expect(html).toContain('overflow-x-auto')
    expect(html).toContain('w-[382px]')
    expect(html).toContain('gap-2')
    expect(html).toContain('size-touch-min')
  })
})
