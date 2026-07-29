# AgendaGlow — Documentação Técnica do Front-end

> **Protótipo (Stitch):** `projects/10665029368600725999`  
> **Tela de referência:** `screens/7728601fe7284acb8764fe39ebf9fb14`  
> **Documentos relacionados:** [PRD.md](./PRD.md) · [PRD_GAP_ANALYSIS.md](./PRD_GAP_ANALYSIS.md) · [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md)

Este documento detalha a arquitetura, o sistema de design e os requisitos funcionais para o desenvolvimento do front-end do **AgendaGlow**, um SaaS de gestão para o mercado de beleza de luxo.

---

## 1. Visão geral tecnológica

### 1.1 Stack alvo (protótipo / greenfield)

| Camada | Tecnologia |
|--------|------------|
| Framework | React (Vite) |
| Linguagem | TypeScript (strict mode) |
| Estilização | Tailwind CSS (mobile-first) |
| Ícones | Lucide React |
| Tipografia títulos | Playfair Display |
| Tipografia corpo | Sans-serif (system / Inter) |
| Estado | Context API ou Zustand |
| HTTP | Axios ou `fetch` com cookie JWT |

### 1.2 Implementação atual no repositório

| Camada | Atual (em migração) |
|--------|---------------------|
| **UI principal** | React + Vite em `frontend-react/` (Aura Beauty) — porta **8082** |
| **Dados (dev)** | `mockDb.ts` (LocalStorage) ou API Go via `VITE_DATA_SOURCE=api` |
| **UI legado** | Go `html/template` + HTMX em `frontend/templates/` (tema escuro — descontinuando) |
| **API** | Binário Go (`backend/cmd/api`, porta **8081**) |

> Ver [MIGRACAO_FRONTEND.md](./MIGRACAO_FRONTEND.md) para o plano mock → produção.

---

## 2. Sistema de design — Aura Beauty

Visual baseado em ouro rosé, superfícies claras e bordas suaves (luxo, legibilidade).

### 2.1 Tokens

| Token | Valor | Uso |
|-------|--------|-----|
| Primária | `#b78472` | Botões, links, destaques |
| Superfície | `#faf9f8` | Background principal |
| Texto | Antracite / cinza escuro | Corpo e títulos secundários |
| Border radius | `8px` | Cards, inputs, botões (“Round Eight”) |

### 2.2 Tipografia

```css
/* Sugestão tailwind.config.ts */
fontFamily: {
  display: ['"Playfair Display"', 'serif'],
  sans: ['Inter', 'system-ui', 'sans-serif'],
}
```

### 2.3 Componentes base (`src/components/ui`)

| Componente | Variantes / notas |
|------------|-------------------|
| `Button` | primary (ouro rosé), secondary, ghost, danger |
| `Input` | label, erro, disabled |
| `Card` | `rounded-lg` (8px), sombra suave |
| `Modal` | reserva rápida, confirmações |
| `Badge` | status assinatura, status agendamento |
| `Alert` | limite plano, conflito agenda, erro API |

### 2.4 Navegação

| Breakpoint | Padrão |
|------------|--------|
| Desktop (`lg+`) | `SideNavBar` persistente |
| Mobile | `BottomNav` (ícones Lucide) |

Itens Dona (sugestão): Painel · Agenda · Equipe · Serviços · Financeiro · Clientes · Config.

---

## 3. Matriz de telas e módulos (Stitch)

### 3.1 Autenticação

| Tela | Stitch | Rota alvo React | Implementado hoje |
|------|--------|-----------------|-------------------|
| Login hub (seleção perfil) | SCREEN_30, SCREEN_31, SCREEN_37 | `/login` | ✅ `GET /login` |
| Login Super Admin | — | `/login/superadmin` | ✅ |
| Login Dona | — | `/login/dona` | ✅ |
| Login Profissional | — | `/login/profissional` | ✅ |

**Comportamento:** redirect pós-login por role; cookie `agendaglow_session`.

---

### 3.2 Super Admin (gestão global)

| Tela | Stitch | Rota alvo | Implementado hoje |
|------|--------|-----------|-------------------|
| Dashboard (MRR, tenants) | SCREEN_3, SCREEN_7 | `/superadmin/dashboard` | 🟡 UI salões/planos; **sem MRR** |
| Gestão de salões | SCREEN_22 | `/superadmin/dashboard` | ✅ tabela + ações HTMX |
| Planos SaaS | SCREEN_15 | `/superadmin/planos` | ✅ CRUD parcial (criar/editar) |

**API JSON:** `GET/POST /api/v1/admin/establishments`, `GET/POST /api/v1/admin/plans`.

---

### 3.3 Dona do salão (administrativo)

| Tela | Stitch | Rota alvo | Implementado hoje |
|------|--------|-----------|-------------------|
| Painel da dona | SCREEN_23 | `/dashboard/gerencial` | ✅ métricas + equipe + lançamento |
| Agenda geral | SCREEN_35 | `/admin/agenda` | ❌ |
| Gestão de equipe | SCREEN_18, SCREEN_12, SCREEN_29 | `/admin/equipe` | ✅ + `/admin/especialidades` |
| Gestão de serviços | SCREEN_25, SCREEN_21, SCREEN_24 | `/admin/servicos` | ✅ criar serviço/adicionais |
| Financeiro / caixa | SCREEN_19, SCREEN_14, SCREEN_20, SCREEN_32 | `/admin/caixa`, `/dashboard/gerencial` | 🟡 caixa + dashboard; sem fechamento diário |
| CRM clientes | SCREEN_33, SCREEN_28, SCREEN_8 | `/admin/clientes` | ❌ |
| Configurações | SCREEN_26, SCREEN_36 | `/admin/config` | 🟡 API `POST /v1/estabelecimentos/config`; sem UI dedicada |

---

### 3.4 Profissional

| Tela | Stitch | Rota alvo | Implementado hoje |
|------|--------|-----------|-------------------|
| Agenda do dia (mobile) | SCREEN_29 (mobile) | `/dashboard/profissional` | ✅ timeline + concluir |

---

### 3.5 Fluxo público (agendamento externo)

| Tela | Stitch | Rota alvo | Implementado hoje |
|------|--------|-----------|-------------------|
| Catálogo / bio | SCREEN_27, SCREEN_11, SCREEN_10 | `/{slug}` | ✅ HTML catálogo |
| Seleção de horários | SCREEN_17 | `/{slug}/agendar` ou wizard | 🟡 API `GET /api/v1/public/{slug}/slots` only |
| Confirmação reserva | — | `POST` público | ❌ |

---

## 4. Convenções de implementação (Cursor / React)

### 4.1 Estrutura de pastas sugerida

```
frontend-react/          # futuro app Vite (não existe ainda)
├── src/
│   ├── components/
│   │   └── ui/          # Button, Input, Card, Modal…
│   ├── layouts/
│   │   ├── SideNavBar.tsx
│   │   └── BottomNav.tsx
│   ├── pages/
│   │   ├── auth/
│   │   ├── superadmin/
│   │   ├── dona/
│   │   ├── profissional/
│   │   └── public/
│   ├── hooks/
│   ├── lib/             # api client, auth context
│   └── styles/          # tailwind + tokens Aura
├── tailwind.config.ts
└── vite.config.ts
```

### 4.2 Autenticação no cliente

```typescript
// Todas as requisições autenticadas
fetch(url, { credentials: 'include' })

// Ou Axios
axios.defaults.withCredentials = true
```

Login: `POST /api/v1/auth/login` → `{ token, role }` + cookie HttpOnly.

### 4.3 Tratamento de erros (regras de negócio na UI)

| Código / erro | UX esperada (protótipo) |
|---------------|-------------------------|
| HTTP **402** | Banner “Assinatura vencida ou suspensa” + CTA upgrade |
| HTTP **403** `limit_reached` | Modal limite de profissionais do plano |
| HTTP **409** conflito agenda | Toast + sugestão de outro horário |
| `EM_APROVACAO` | Badge “Aguardando aprovação” na agenda |

Ver detalhes em [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md).

### 4.4 Integração com API existente

| Módulo | Endpoints principais |
|--------|----------------------|
| Auth | `POST /api/v1/auth/login` |
| Serviços | `GET/POST /api/v1/services`, `POST .../additionals` |
| Equipe | `GET/POST /api/v1/professionals` |
| Financeiro | `GET /api/v1/finance/report`, `POST .../pay` |
| Público | `GET /api/v1/public/{slug}/slots` |
| Super Admin | `GET/POST /api/v1/admin/establishments`, `.../plans` |

Rotas HTML atuais podem coexistir até migração tela a tela.

---

## 5. Matriz Stitch → rotas → status

| SCREEN | Módulo | Rota atual (Go) | Rota React alvo | Status |
|--------|--------|-----------------|-----------------|--------|
| 3, 7 | Super Admin dashboard | `/superadmin/dashboard` | `/superadmin` | 🟡 |
| 15 | Planos SaaS | `/superadmin/planos` | `/superadmin/planos` | ✅ |
| 22 | Salões | `/superadmin/dashboard` | `/superadmin/saloes` | ✅ |
| 30–37 | Login | `/login*` | `/login` | ✅ |
| 23 | Painel dona | `/dashboard/gerencial` | `/dona` | ✅ |
| 35 | Agenda geral | — | `/dona/agenda` | ❌ |
| 18, 12, 29 | Equipe | `/admin/equipe` | `/dona/equipe` | ✅ |
| — | Especialidades | `/admin/especialidades` | `/dona/especialidades` | ✅ |
| 25, 21, 24 | Serviços | `/admin/servicos` | `/dona/servicos` | ✅ |
| 19, 14, 20, 32 | Financeiro | `/admin/caixa`, `/dashboard/gerencial` | `/dona/financeiro` | 🟡 |
| 33, 28, 8 | CRM | — | `/dona/clientes` | ❌ |
| 26, 36 | Config | API only | `/dona/config` | 🟡 |
| 29 | Profissional mobile | `/dashboard/profissional` | `/profissional` | ✅ |
| 27, 11, 10 | Catálogo público | `GET /{slug}` | `/{slug}` | ✅ |
| 17 | Slots / reserva | API slots | `/{slug}/agendar` | 🟡 |

---

## 6. Templates HTML atuais (referência)

| Arquivo | Função |
|---------|--------|
| `frontend/templates/superadmin/dashboard.html` | Salões + ações |
| `frontend/templates/superadmin/planos.html` | Planos SaaS |
| `frontend/templates/dashboard_dona.html` | Painel gerencial |
| `frontend/templates/dashboard_profissional.html` | Agenda profissional |
| `frontend/templates/config_servicos.html` | Procedimentos |
| `frontend/templates/config_equipe.html` | Equipe |
| `frontend/templates/config_especialidades.html` | Especialidades |
| `frontend/templates/caixa_fluxo.html` | Caixa |
| `frontend/templates/admin_common.html` | Nav + head compartilhado |

Login: inline em `backend/internal/handler/login_ui.go`.

---

## 7. Roadmap front-end sugerido

### Fase 1 — Fundação
- [ ] Scaffold Vite + React + TS + Tailwind
- [ ] Tokens Aura Beauty em `tailwind.config`
- [ ] `components/ui` + layouts SideNav / BottomNav
- [ ] Auth context + rotas protegidas por role

### Fase 2 — Paridade com HTML atual
- [ ] Super Admin (dashboard + planos)
- [ ] Dona: gerencial, serviços, equipe, especialidades, caixa
- [ ] Profissional: agenda mobile
- [ ] Login multi-perfil

### Fase 3 — Gaps PRD
- [ ] Agenda geral (SCREEN_35)
- [ ] CRM clientes (SCREEN_33, 28, 8)
- [ ] Fluxo público completo (SCREEN_17 + POST reserva)
- [ ] Configurações (SCREEN_26, 36)
- [ ] MRR no Super Admin

### Fase 4 — Desligar templates Go (opcional)
- [ ] Proxy estático na 8082 ou servir build Vite pelo Go
- [ ] Remover HTMX das telas migradas

---

## 8. Portas e ambiente dev

| Serviço | Porta |
|---------|-------|
| API AgendaGlow | **8081** |
| Postgres (host) | **5435** |
| WhatsApp Gateway | **8080** |
| Front React (futuro) | **8082** (reservada no docker-compose) |

Variáveis: ver `.env` e `internal/config/runtime.go`.

---

*Documento para guiar o desenvolvimento front-end no Cursor IDE.*
