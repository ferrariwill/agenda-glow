import { useEffect, useState } from 'react'
import type { MockDatabase } from '../types'

let apiDb: MockDatabase = emptyDb()
const listeners = new Set<() => void>()

function emptyDb(): MockDatabase {
  return {
    tenants: [],
    planos: [],
    users: [],
    categorias: [],
    especialidades: [],
    profissionais: [],
    servicos: [],
    adicionais: [],
    agendamentos: [],
    fila_espera: [],
    lancamentos: [],
    clientes: [],
    insumos: [],
    contas_cliente: [],
    cliente_preferencias: [],
    cliente_notas: [],
    cliente_galeria: [],
    faturas_saas: [],
  }
}

export function getStoreDb(): MockDatabase {
  return apiDb
}

export function setStoreDb(db: MockDatabase): void {
  apiDb = db
  listeners.forEach((l) => l())
}

export function subscribeStore(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function mergeStoreDb(patch: Partial<MockDatabase>): void {
  setStoreDb({ ...apiDb, ...patch })
}

/** Re-render quando o store em memória (modo API) for atualizado. */
export function useStoreDb(): MockDatabase {
  const [db, setDb] = useState(getStoreDb)
  useEffect(() => subscribeStore(() => setDb(getStoreDb())), [])
  return db
}
