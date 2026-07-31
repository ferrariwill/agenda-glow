import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { ResponsiveEntityList } from './ResponsiveEntityList'

type Row = { id: string; nome: string; valor: number }

const items: Row[] = [
  { id: '1', nome: 'Ana', valor: 120 },
  { id: '2', nome: 'Bia', valor: 80 },
]

describe('ResponsiveEntityList', () => {
  it('mostra cards em md:hidden e tabela em hidden md:block por padrão', () => {
    const html = renderToStaticMarkup(
      <ResponsiveEntityList
        items={items}
        getKey={(r) => r.id}
        renderCard={(r) => <span data-card={r.id}>{r.nome}</span>}
        columns={[
          { header: 'Nome', cell: (r) => r.nome },
          { header: 'Valor', cell: (r) => String(r.valor), align: 'right' },
        ]}
        emptyTitle="Vazio"
      />,
    )
    expect(html).toContain('md:hidden')
    expect(html).toContain('hidden md:block')
    expect(html).toContain('data-card="1"')
    expect(html).toContain('<th')
    expect(html).not.toContain('overflow-x-auto')
  })

  it('respeita tableFrom=lg', () => {
    const html = renderToStaticMarkup(
      <ResponsiveEntityList
        items={items}
        getKey={(r) => r.id}
        renderCard={(r) => r.nome}
        columns={[{ header: 'Nome', cell: (r) => r.nome }]}
        emptyTitle="Vazio"
        tableFrom="lg"
      />,
    )
    expect(html).toContain('lg:hidden')
    expect(html).toContain('hidden lg:block')
  })

  it('renderiza empty state sem overflow-x', () => {
    const html = renderToStaticMarkup(
      <ResponsiveEntityList
        items={[]}
        getKey={(r: Row) => r.id}
        renderCard={() => null}
        columns={[{ header: 'Nome', cell: () => null }]}
        emptyTitle="Nenhum cliente encontrado."
        emptyAction={<button type="button">Novo</button>}
      />,
    )
    expect(html).toContain('Nenhum cliente encontrado.')
    expect(html).toContain('Novo')
    expect(html).not.toContain('overflow-x-auto')
  })

  it('esconde colunas secondary até lg', () => {
    const html = renderToStaticMarkup(
      <ResponsiveEntityList
        items={items}
        getKey={(r) => r.id}
        renderCard={(r) => r.nome}
        columns={[
          { header: 'Nome', cell: (r) => r.nome },
          { header: 'Extra', cell: (r) => r.nome, priority: 'secondary' },
        ]}
        emptyTitle="Vazio"
      />,
    )
    expect(html).toContain('hidden lg:table-cell')
  })

  it('mostra skeleton quando loading', () => {
    const html = renderToStaticMarkup(
      <ResponsiveEntityList
        items={[]}
        getKey={(r: Row) => r.id}
        renderCard={() => null}
        columns={[{ header: 'Nome', cell: () => null }]}
        emptyTitle="Vazio"
        loading
      />,
    )
    expect(html).toContain('animate-pulse')
    expect(html).not.toContain('Nenhum')
  })
})
