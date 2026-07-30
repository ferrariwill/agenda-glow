// Evidência QA Stage 4: aplica a derivação de status usada pelo React
// (frontend-react/src/data/mapBootstrap.ts) sobre o payload real de
// GET /api/v1/admin/bootstrap e compara com o status da API.
import { readFileSync } from 'node:fs'

const payload = JSON.parse(readFileSync(process.argv[2], 'utf8'))
const filtro = process.argv[3] ?? ''

// Linha 300 de mapBootstrap.ts (mapAdminBootstrap).
const mapStatusReact = (t) => (t.status_assinatura === 'ATIVO' || t.ativo ? 'ATIVO' : 'VENCIDO')

const rows = payload.tenants
  .filter((t) => !filtro || t.slug.includes(filtro))
  .map((t) => ({
    slug: t.slug,
    ativo: t.ativo,
    api_status: t.status_assinatura,
    react_status: mapStatusReact(t),
    divergente: mapStatusReact(t) !== t.status_assinatura,
  }))

console.table(rows)
const divergentes = rows.filter((r) => r.divergente)
console.log(`divergencias=${divergentes.length}/${rows.length}`)
