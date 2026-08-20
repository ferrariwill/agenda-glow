import { describe, expect, it } from 'vitest'
import {
  formatBRLInputValue,
  maskBRLInput,
  maskPhoneBRInput,
  parseBRLInput,
  phoneDigitsToMaskInput,
  toWhatsAppDigits,
} from './format'

describe('maskBRLInput', () => {
  it('retorna vazio sem dígitos', () => {
    expect(maskBRLInput('')).toBe('')
    expect(maskBRLInput('abc')).toBe('')
    expect(maskBRLInput('R$')).toBe('')
  })

  it('formata centavos da direita', () => {
    expect(maskBRLInput('1')).toBe('0,01')
    expect(maskBRLInput('12')).toBe('0,12')
    expect(maskBRLInput('123')).toBe('1,23')
    expect(maskBRLInput('123456')).toBe('1.234,56')
  })

  it('digitar 100000 exibe 1.000,00 e parseia 1000', () => {
    expect(maskBRLInput('100000')).toBe('1.000,00')
    expect(parseBRLInput(maskBRLInput('100000'))).toBe(1000)
  })

  it('ignora não-dígitos e round-trip com parseBRLInput', () => {
    expect(maskBRLInput('R$ 1.234,56')).toBe('1.234,56')
    expect(parseBRLInput(maskBRLInput('123456'))).toBe(1234.56)
  })

  it('limita a 12 dígitos', () => {
    expect(maskBRLInput('1234567890123')).toBe('1.234.567.890,12')
  })
})

describe('parseBRLInput', () => {
  it('aceita milhar BR sem centavos', () => {
    expect(parseBRLInput('1.000')).toBe(1000)
    expect(parseBRLInput('10.000')).toBe(10000)
    expect(parseBRLInput('1.234.567')).toBe(1234567)
  })

  it('aceita vírgula decimal e milhar+centavos', () => {
    expect(parseBRLInput('1.000,00')).toBe(1000)
    expect(parseBRLInput('180,50')).toBe(180.5)
    expect(parseBRLInput('R$ 1.234,56')).toBe(1234.56)
  })

  it('mantém decimal com ponto quando não é milhar BR', () => {
    expect(parseBRLInput('1.5')).toBe(1.5)
    expect(parseBRLInput('180.50')).toBe(180.5)
  })

  it('parseia inteiros e strings mascaradas típicas', () => {
    expect(parseBRLInput('1000')).toBe(1000)
    expect(parseBRLInput('0,00')).toBe(0)
  })
})

describe('formatBRLInputValue', () => {
  it('hidrata input a partir de reais', () => {
    expect(formatBRLInputValue(1000)).toBe('1.000,00')
    expect(formatBRLInputValue(14.2)).toBe('14,20')
  })
})

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

  it('mantém 55 já presente (E.164 completo)', () => {
    expect(toWhatsAppDigits('5511999999999')).toBe('5511999999999')
  })

  it('trata DDD local 55 como local, não como país', () => {
    expect(toWhatsAppDigits('(55) 99999-9999')).toBe('5555999999999')
    expect(toWhatsAppDigits('55999999999')).toBe('5555999999999')
    expect(toWhatsAppDigits('(55) 9999-9999')).toBe('555599999999')
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

  it('round-trip com DDD 55', () => {
    const masked = maskPhoneBRInput('55987654321')
    const digits = toWhatsAppDigits(masked)
    expect(digits).toBe('5555987654321')
    expect(phoneDigitsToMaskInput(digits)).toBe('(55) 98765-4321')
  })
})
