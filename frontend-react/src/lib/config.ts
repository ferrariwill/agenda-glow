/** Fonte de dados: `mock` = LocalStorage (protótipo); `api` = backend Go (8081). */
export const DATA_SOURCE = (import.meta.env.VITE_DATA_SOURCE ?? 'mock') as 'mock' | 'api'

export const IS_MOCK = DATA_SOURCE === 'mock'

export const API_BASE = import.meta.env.VITE_API_BASE ?? ''
