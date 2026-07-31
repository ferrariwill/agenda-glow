import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { BottomNav } from './BottomNav'

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    session: {
      user: {
        role: 'DONA',
        nome: 'Dona',
        email: 'dona@test.com',
        profissional_id: null,
      },
    },
  }),
}))

function render(variant: 'dona' | 'secretaria' | 'profissional' | 'superadmin', drawerOpen = false) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <BottomNav variant={variant} drawerOpen={drawerOpen} />
    </MemoryRouter>,
  )
}

describe('BottomNav', () => {
  it('dona: 4 primários + Mais (não link direto para Configurações)', () => {
    const html = render('dona')
    expect(html).toContain('aria-label="Navegação principal — painel da dona"')
    expect(html).toContain('Início')
    expect(html).toContain('Agenda')
    expect(html).toContain('Clientes')
    expect(html).toContain('Caixa')
    expect(html).toContain('Mais')
    expect(html).toContain('min-h-touch-min')
    expect(html).toContain('text-xs')
    expect(html).toContain('touch-manipulation')
    // Mais is a button, not a NavLink to configuracoes
    expect(html).not.toMatch(/href="\/admin\/configuracoes"[^>]*>[\s\S]*Mais/)
  })

  it('secretaria/profissional: sem sheet Mais', () => {
    expect(render('secretaria')).not.toContain('>Mais<')
    expect(render('profissional')).not.toContain('>Mais<')
    expect(render('secretaria')).toContain('Agenda')
    expect(render('secretaria')).toContain('Clientes')
    expect(render('profissional')).toContain('Início')
  })

  it('superadmin: 5 destinos sem Mais', () => {
    const html = render('superadmin')
    expect(html).toContain('Salões')
    expect(html).toContain('Planos')
    expect(html).not.toContain('>Mais<')
  })

  it('drawer aberto marca nav como inert', () => {
    const html = render('dona', true)
    expect(html).toContain('inert')
  })
})
