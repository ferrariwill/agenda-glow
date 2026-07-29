// Fixture capturada de GET /api/v1/admin/bootstrap na API Go (8081) durante o
// reteste do Stage 4: os tres estados reais de assinatura do Super Admin.
export const adminBootstrapLive = {
  "tenants": [
    {
      "id": "13ce3850-19d6-4691-9400-f603abd68d7d",
      "nome_comercial": "QA Retest suspenso",
      "slug": "qa-retest-suspenso",
      "ativo": false,
      "data_cadastro": "2026-07-29T17:58:22.31166Z",
      "plano_id": "0160634a-80b4-4519-b1bf-316934fae2e4",
      "plano_nome": "QA Retest Plano 1785347902",
      "data_vencimento": "2027-07-29T00:00:00Z",
      "status_assinatura": "SUSPENSO"
    },
    {
      "id": "e1f3188a-5c2a-473c-82b5-7279bd71f991",
      "nome_comercial": "QA Retest vencido",
      "slug": "qa-retest-vencido",
      "ativo": true,
      "data_cadastro": "2026-07-29T17:58:22.30546Z",
      "plano_id": "0160634a-80b4-4519-b1bf-316934fae2e4",
      "plano_nome": "QA Retest Plano 1785347902",
      "data_vencimento": "2026-07-22T00:00:00Z",
      "status_assinatura": "VENCIDO"
    },
    {
      "id": "84be9ece-9fd1-4967-84ac-dc58c41ce4ea",
      "nome_comercial": "QA Retest ativo",
      "slug": "qa-retest-ativo",
      "ativo": true,
      "data_cadastro": "2026-07-29T17:58:22.299562Z",
      "plano_id": "0160634a-80b4-4519-b1bf-316934fae2e4",
      "plano_nome": "QA Retest Plano 1785347902",
      "data_vencimento": "2027-07-29T00:00:00Z",
      "status_assinatura": "ATIVO"
    }
  ],
  "planos": []
} as const
