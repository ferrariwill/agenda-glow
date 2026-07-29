import type { Lancamento, LancamentoTipo, TransacaoCategoria } from '../../types'

export const GLASS =
  'rounded-xl border border-[#e5d3c8]/30 bg-white/80 shadow-[0px_4px_20px_rgba(183,132,114,0.08)] backdrop-blur-md'

export const CATEGORIA_LABEL: Record<TransacaoCategoria, string> = {
  SERVICO: 'Serviços',
  PRODUTO: 'Produtos',
  ALUGUEL: 'Aluguel',
  SALARIOS: 'Salários',
  SUPRIMENTOS: 'Suprimentos',
  MARKETING: 'Marketing',
  OUTRO: 'Outro',
}

export const CATEGORIA_BADGE: Record<TransacaoCategoria, string> = {
  SERVICO: 'bg-[#efdcd1]/50 text-[#695c53]',
  PRODUTO: 'bg-[#996958]/10 text-[#7d5141]',
  ALUGUEL: 'bg-[#e3e2e1] text-[#615b58]',
  SALARIOS: 'bg-[#e3e2e1] text-[#615b58]',
  SUPRIMENTOS: 'bg-[#f4f3f2] text-[#514440]',
  MARKETING: 'bg-[#ffdbcf]/40 text-[#653d2e]',
  OUTRO: 'bg-[#f4f3f2] text-[#514440]',
}

export const CATEGORIAS_RECEITA: TransacaoCategoria[] = ['SERVICO', 'PRODUTO']
export const CATEGORIAS_DESPESA: TransacaoCategoria[] = [
  'ALUGUEL',
  'SUPRIMENTOS',
  'MARKETING',
  'SALARIOS',
  'PRODUTO',
  'OUTRO',
]

export const FORNECEDORES_MOCK = [
  "Distribuidora L'Oréal",
  'Papelaria Central',
  'Wella Professionals',
]

export const FIELD_INPUT =
  'w-full border-0 border-b border-[#695c53]/30 bg-transparent px-0 py-2 text-base transition-all focus:border-[#7d5141] focus:ring-0'

export function isReceitaLanc(l: Lancamento): boolean {
  return l.tipo === 'ENTRADA'
}

export function categoriaToTipo(
  cat: TransacaoCategoria,
  flow: 'receita' | 'despesa',
  natureza: 'FIXO' | 'VARIAVEL',
): LancamentoTipo {
  if (flow === 'receita') return 'ENTRADA'
  if (natureza === 'VARIAVEL' || cat === 'PRODUTO' || cat === 'SUPRIMENTOS') {
    return 'CUSTO_VARIAVEL'
  }
  return 'CUSTO_FIXO'
}

export function naturezaFromLanc(l: Lancamento): 'FIXO' | 'VARIAVEL' {
  if (l.natureza) return l.natureza
  return l.tipo === 'CUSTO_VARIAVEL' ? 'VARIAVEL' : 'FIXO'
}
