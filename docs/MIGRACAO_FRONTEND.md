# Migração: protótipo → sistema final

O **frontend principal** do AgendaGlow é o app React em `frontend-react/` (design Aura Beauty).  
Os arquivos `tmp_*.html` na raiz e os templates Go em `frontend/templates/` são **legado** — serão substituídos tela a tela pelo React conectado à API.

## Arquitetura alvo

```
┌─────────────────────┐     proxy /api      ┌─────────────────────┐
│  frontend-react     │ ──────────────────► │  backend/cmd/api    │
│  porta 8082 (UI)    │   credentials       │  porta 8081 (API)   │
└─────────────────────┘                     └──────────┬──────────┘
                                                     │
                                                     ▼
                                              PostgreSQL
```

| Camada | Responsabilidade |
|--------|------------------|
| **React (8082)** | Todas as telas (dona, profissional, secretaria, cliente, super admin) |
| **Go API (8081)** | JSON, auth JWT, regras de negócio, Postgres |
| **Templates Go** | Manutenção até paridade; depois removidos ou redirecionam para React |

## Fonte de dados no React

Variável `VITE_DATA_SOURCE` em `frontend-react/.env`:

| Valor | Comportamento |
|-------|----------------|
| `mock` (padrão) | `mockDb.ts` + LocalStorage — desenvolvimento de UI sem banco |
| `api` | `fetch` com cookie JWT para `POST /api/v1/auth/login` e demais endpoints |

```bash
cd frontend-react
cp .env.example .env
# Edite: VITE_DATA_SOURCE=api
npm run dev
```

API deve estar rodando (`go run ./backend/cmd/api` ou docker-compose).

## Status da migração

| Módulo | UI React | Dados mock | API real |
|--------|----------|------------|----------|
| Login equipe (dona, prof, secretaria, super) | ✅ | ✅ | 🟡 auth |
| Dashboard dona | ✅ | ✅ | ❌ |
| Equipe / serviços / financeiro | ✅ | ✅ | ❌ |
| Profissional / secretaria | ✅ | ✅ | ❌ |
| Cliente público (`/{slug}`) | ✅ | ✅ | 🟡 slots |
| Templates Go HTMX | legado | — | — |

Legenda: ✅ pronto · 🟡 em andamento · ❌ pendente

## Próximos passos (ordem sugerida)

1. **Auth** — concluir login API para todos os perfis (incl. secretaria no backend).
2. **Serviços / equipe** — `GET/POST /api/v1/services`, `/api/v1/professionals`.
3. **Agenda** — endpoints de agendamento + substituir `createAgendamento` no mock.
4. **Financeiro** — `/api/v1/finance/*`.
5. **Cliente público** — `GET /api/v1/public/{slug}/slots` + `POST` reserva.
6. **Produção** — servir `frontend-react/dist` pelo Go ou nginx; desligar rotas HTML antigas.

## Arquivos de referência visual

| Origem | Destino canônico |
|--------|------------------|
| `frontend-react/src/pages/**` | Telas oficiais |
| `tmp_*.html` (raiz) | Snapshots antigos — não usar em produção |
| `frontend/templates/*.html` | Legado Go até migração |

## Desenvolvimento local

```bash
# Terminal 1 — API
go run ./backend/cmd/api

# Terminal 2 — UI (protótipo)
cd frontend-react && npm run dev

# UI + API
cd frontend-react && VITE_DATA_SOURCE=api npm run dev
```

Documentação relacionada: [FRONTEND.md](./FRONTEND.md) · [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md)
