# PRD × Implementação — Análise de Gaps

Comparativo entre o [PRD](./PRD.md) e o estado atual do repositório AgendaGlow (junho/2026).  
**Front-end alvo:** [FRONTEND.md](./FRONTEND.md)

**Legenda:** ✅ Implementado · 🟡 Parcial · ❌ Não implementado

---

## Resumo executivo

| Módulo PRD | Status geral |
|------------|--------------|
| Autenticação e multi-tenancy | 🟡 |
| Gestão de agenda | 🟡 |
| CRM e clientes | ❌ |
| Equipe e comissões | 🟡 |
| Catálogo e vendas | 🟡 |
| Financeiro | 🟡 |
| Design Aura Beauty | ❌ |
| Cliente final (agendamento) | 🟡 |
| Super Admin SaaS | ✅ |

---

## 4.1 Autenticação e Multi-Tenancy

| Requisito PRD | Status | Implementação atual |
|---------------|--------|---------------------|
| Login por perfil com redirect | ✅ | `/login/*`, JWT, cookie; redirect por role |
| Slug por salão | ✅ | `/{slug}`; validação regex; único global |
| Bloqueio inadimplência | 🟡 | Middleware SaaS na **Dona** (402); profissional **sem** guarda |
| Limite de plano excedido | ✅ | `ErrPlanLimitExceeded` ao cadastrar/reativar profissional |
| Subdomínios | ❌ | Apenas path slug (`/estudio-glow-salto`) |

---

## 4.2 Gestão de Agenda (Core)

| Requisito PRD | Status | Implementação atual |
|---------------|--------|---------------------|
| Visualização diária/semanal/mensal | 🟡 | Timeline **dia** na profissional; sem calendário admin |
| Conflitos de horário | ✅ | Colisão total rejeita; parcial → `EM_APROVACAO` |
| Status (Em Espera, Atendido, Cancelado) | 🟡 | `AGENDADO`, `CONFIRMADO`, `EM_APROVACAO`, `CONCLUIDO`, `CANCELADO` — nomenclatura diferente |
| Reserva rápida (modal admin) | ❌ | `CriarAgendamento` no serviço; **sem UI admin** |
| Slots públicos 30 min | ✅ | `GET /api/v1/public/{slug}/slots` |
| Aprovar / reagendar encaixe | ✅ | Endpoints públicos + e-mail profissional |
| Criar agendamento público (cliente) | ❌ | Página pública é **catálogo**; sem POST de reserva |

---

## 4.3 CRM e Gestão de Clientes

| Requisito PRD | Status | Implementação atual |
|---------------|--------|---------------------|
| Cadastro de clientes | 🟡 | Tabela `clientes`; upsert por telefone no agendamento |
| Histórico de procedimentos | ❌ | Sem tela/API de histórico por cliente |
| Alergias e preferências | ❌ | Sem campos no schema |
| Ticket médio | ❌ | — |
| Taxa de retorno | ❌ | — |
| Tela CRM / listagem clientes | ❌ | — |

---

## 4.4 Gestão de Equipe e Comissões

| Requisito PRD | Status | Implementação atual |
|---------------|--------|---------------------|
| Profissionais + especialidades | ✅ | `/admin/especialidades`, `/admin/equipe` (select) |
| Editar nome, comissão, status | ✅ | HTMX na equipe |
| Comissão por profissional | ✅ | `%` por profissional (0–100) |
| Comissão por serviço | ❌ | Apenas % global da profissional |
| Expediente / turnos | 🟡 | `SetProfessionalHours` no serviço; **sem UI** |
| Quitar comissões | ✅ | Dashboard gerencial + API financeira |
| Limite do plano SaaS | ✅ | Por `limite_profissionais` |

---

## 4.5 Catálogo de Serviços e Vendas

| Requisito PRD | Status | Implementação atual |
|---------------|--------|---------------------|
| Serviços base | ✅ | `/admin/servicos` |
| Adicionais (upsell) | ✅ | Por serviço; duração/preço dinâmicos |
| Cálculo dinâmico preço/duração | ✅ | No agendamento e slots |
| Categorias (Cabelo, Unhas…) | ❌ | Sem taxonomia; só especialidade da profissional |
| Edição de serviços na UI | 🟡 | Criação; edição/desativação limitada |

---

## 4.6 Financeiro e Fluxo de Caixa

| Requisito PRD | Status | Implementação atual |
|---------------|--------|---------------------|
| Entradas e saídas | ✅ | `fluxo_caixa`; tipos ENTRADA, CUSTO_FIXO, CUSTO_VARIAVEL |
| Comissão automática na conclusão | ✅ | `ConcluirAtendimento` |
| Liquidação em lote | 🟡 | Por profissional no período; não “lote multi-prof” único |
| Relatório mensal | ✅ | Dashboard gerencial (métricas + equipe) |
| Fechamento diário | ❌ | Sem tela dedicada |
| Caixa admin | ✅ | `/admin/caixa` (mês, 100 lançamentos) |
| Clube de assinatura | 🟡 | Schema + consumo no agendamento; **sem CRUD/UI** |

---

## Perfis (seção 3)

| Perfil | PRD | Status |
|--------|-----|--------|
| Super Admin | MRR, tenants, planos | 🟡 UI salões/planos OK; **MRR agregado** não |
| Dona | Gestão total | 🟡 Serviços, equipe, caixa, dashboard; falta CRM e agenda admin |
| Profissional | Agenda + comissões | ✅ Dashboard mobile; concluir atendimento |
| Cliente final | Agendamento online | 🟡 Catálogo + slots; **sem checkout/reserva** |

---

## Design (Aura Beauty)

| Requisito | Status | Atual |
|-----------|--------|-------|
| Ouro Rosé `#b78472` | ❌ | Violeta/neon (super admin) e accent roxo (admin) |
| Playfair Display | ❌ | DM Sans |
| Superfícies `#faf9f8` | ❌ | Dark theme (`surface-900`, `void-950`) |
| Mobile-first premium | 🟡 | Profissional mobile OK; resto desktop-first |

**Próximo passo de UI:** alinhar tokens Tailwind ao Aura Beauty ou portar telas Stitch para React conforme PRD seção 6.

---

## Integrações

| Integração | Status |
|------------|--------|
| WhatsApp lembrete | ✅ Async na criação |
| WhatsApp confirm/cancel webhook | ✅ |
| E-mail encaixe | ✅ SMTP opcional |
| Gateway WhatsApp (8080) | ✅ Configurável |

---

## Priorização sugerida (backlog)

### P0 — Fechar loop do cliente final
1. `POST` público de agendamento (reserva com slots)
2. UI admin para criar agendamento (modal rápido)

### P1 — CRM mínimo
3. Listagem de clientes por salão
4. Histórico de atendimentos por cliente
5. Campos alergias/preferências (migration + form)

### P2 — Agenda admin
6. Calendário semanal/mensal (Dona)
7. UI de expediente por profissional

### P3 — Produto e marca
8. Categorias de serviço
9. Design system Aura Beauty (ou app React)
10. MRR no Super Admin

### P4 — Clube
11. CRUD planos assinatura + assinantes

---

## Referência cruzada

| Documento | Conteúdo |
|-----------|----------|
| [PRD.md](./PRD.md) | Visão de produto e requisitos |
| [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md) | Regras já codificadas |
| `backend/cmd/api/main.go` | Rotas ativas |
| `migrations/` | Schema e constraints |
