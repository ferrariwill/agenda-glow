import type { TenantStatus } from '../types'

export const TENANT_STATUSES: readonly TenantStatus[] = ['ATIVO', 'VENCIDO', 'SUSPENSO']

const STATUS_CONHECIDOS = new Set<string>(TENANT_STATUSES)

const STATUS_INATIVOS: readonly TenantStatus[] = ['VENCIDO', 'SUSPENSO']

/**
 * Única normalização de `status_assinatura` na fronteira com a API.
 * O campo `ativo` nunca participa da decisão: status desconhecido cai em
 * VENCIDO para falhar fechado em vez de liberar acesso indevidamente.
 */
export function normalizeTenantStatus(status: string | null | undefined): TenantStatus {
  if (status && STATUS_CONHECIDOS.has(status)) return status as TenantStatus
  return 'VENCIDO'
}

export type TenantStatusFilter = 'TODOS' | 'ATIVO' | 'INATIVOS'

export function matchesTenantStatusFilter(
  status: TenantStatus,
  filter: TenantStatusFilter,
): boolean {
  if (filter === 'TODOS') return true
  if (filter === 'INATIVOS') return STATUS_INATIVOS.includes(status)
  return status === filter
}

export function isTenantAtivo(status: TenantStatus): boolean {
  return status === 'ATIVO'
}
