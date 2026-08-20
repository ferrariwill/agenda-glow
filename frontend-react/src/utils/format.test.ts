import { describe, expect, it } from 'vitest'
import {
  maskPhoneBRInput,
  phoneDigitsToMaskInput,
  toWhatsAppDigits,
} from './format'

describe('maskPhoneBRInput', () => {
  it('ignora letras e formata progressivamente', () => {
    expect(maskPhoneBRInput('11')).toBe('(11')
    expect(maskPhoneBRInput('(11) 9')).toBe('(11) 9')
    expect(maskPhoneBRInput('119999')).toBe('(11) 9999')
    expect(maskPhoneBRInput('1199999abc')).toBe('(11) 9999-9')
    expect(maskPhoneBRInput('1199999999')).toBe('(11) 9999-9999')
    expect(maskPhoneBRInput('11999999999')).toBe('(11) 99999-9999')
  })

  it('descarta prefixo 55 ao colar valor completo', () => {
    expect(maskPhoneBRInput('5511999999999')).toBe('(11) 99999-9999')
    expect(maskPhoneBRInput('551199999999')).toBe('(11) 9999-9999')
  })

  it('limita a 11 dígitos locais', () => {
    expect(maskPhoneBRInput('119999999991234')).toBe('(11) 99999-9999')
  })
})

describe('toWhatsAppDigits', () => {
  it('prefixa 55 quando ausente', () => {
    expect(toWhatsAppDigits('(11) 99999-9999')).toBe('5511999999999')
    expect(toWhatsAppDigits('(11) 9999-9999')).toBe('551199999999')
  })

  it('mantém 55 já presente', () => {
    expect(toWhatsAppDigits('5511999999999')).toBe('5511999999999')
  })

  it('retorna vazio para entrada sem dígitos', () => {
    expect(toWhatsAppDigits('')).toBe('')
    expect(toWhatsAppDigits('abc')).toBe('')
  })
})

describe('phoneDigitsToMaskInput round-trip', () => {
  it('mask → digits → mask', () => {
    const masked = maskPhoneBRInput('11987654321')
    const digits = toWhatsAppDigits(masked)
    expect(digits).toBe('5511987654321')
    expect(phoneDigitsToMaskInput(digits)).toBe('(11) 98765-4321')
  })
})
