import { describe, expect, it } from 'vitest'
import {
  isTenantAtivo,
  matchesTenantStatusFilter,
  normalizeTenantStatus,
} from './tenantStatus'
import type { TenantStatus } from '../types'

describe('normalizeTenantStatus', () => {
  it('preserva os três estados do contrato', () => {
    expect(normalizeTenantStatus('ATIVO')).toBe('ATIVO')
    expect(normalizeTenantStatus('VENCIDO')).toBe('VENCIDO')
    expect(normalizeTenantStatus('SUSPENSO')).toBe('SUSPENSO')
  })

  it('trata status ausente ou desconhecido como VENCIDO', () => {
    expect(normalizeTenantStatus(undefined)).toBe('VENCIDO')
    expect(normalizeTenantStatus(null)).toBe('VENCIDO')
    expect(normalizeTenantStatus('')).toBe('VENCIDO')
    expect(normalizeTenantStatus('PENDENTE')).toBe('VENCIDO')
    expect(normalizeTenantStatus('ativo')).toBe('VENCIDO')
  })
})

describe('matchesTenantStatusFilter', () => {
  const todos: TenantStatus[] = ['ATIVO', 'VENCIDO', 'SUSPENSO']

  it('TODOS aceita os três estados', () => {
    expect(todos.filter((s) => matchesTenantStatusFilter(s, 'TODOS'))).toEqual(todos)
  })

  it('ATIVO seleciona apenas assinaturas ativas', () => {
    expect(todos.filter((s) => matchesTenantStatusFilter(s, 'ATIVO'))).toEqual(['ATIVO'])
  })

  it('INATIVOS agrupa VENCIDO e SUSPENSO', () => {
    expect(todos.filter((s) => matchesTenantStatusFilter(s, 'INATIVOS'))).toEqual([
      'VENCIDO',
      'SUSPENSO',
    ])
  })
})

describe('isTenantAtivo', () => {
  it('bloqueia qualquer assinatura diferente de ATIVO', () => {
    expect(isTenantAtivo('ATIVO')).toBe(true)
    expect(isTenantAtivo('VENCIDO')).toBe(false)
    expect(isTenantAtivo('SUSPENSO')).toBe(false)
  })
})
