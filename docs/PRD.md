# PRD — AgendaGlow: Sistema de Gestão para o Mercado de Beleza de Luxo

> **Protótipo (Stitch):** `projects/10665029368600725999`  
> **Design system:** Aura Beauty  
> **Documentos relacionados:** [FRONTEND.md](./FRONTEND.md) · [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md) · [PRD_GAP_ANALYSIS.md](./PRD_GAP_ANALYSIS.md)

---

## 1. Visão Geral do Produto

O **AgendaGlow** é um SaaS B2B e B2C focado na gestão de alto nível para salões de beleza, clínicas de estética e estúdios de luxo. O objetivo é combinar eficiência operacional (agendamentos, financeiro, equipe) com uma experiência de marca sofisticada para o cliente final.

---

## 2. Objetivos Estratégicos

| Objetivo | Descrição |
|----------|-----------|
| Simplificar a gestão | Centralizar agenda, financeiro e equipe em uma interface intuitiva |
| Experiência premium | Catálogo e fluxo de agendamento público alinhados à marca de luxo |
| Escalabilidade SaaS | Multi-tenant com planos de assinatura por salão |

---

## 3. Perfis de Usuário

| Perfil | Responsabilidades |
|--------|-------------------|
| **Super Admin** | Gestão global (MRR, tenants, planos SaaS) |
| **Dona do Salão** | Gestão da unidade: equipe, financeiro, clientes, configurações |
| **Profissional** | Agenda pessoal e comissões |
| **Cliente Final** | Consulta de serviços e agendamento online |

---

## 4. Requisitos Funcionais

### 4.1. Autenticação e Multi-Tenancy

- Login dinâmico com redirecionamento por perfil
- Slugs personalizados por salão (ex.: `agendaglow.com/salao-luxo`)
- Middleware de bloqueio por inadimplência ou limite de plano excedido

### 4.2. Gestão de Agenda (Core)

- Visualização diária / semanal / mensal
- Detecção automática de conflitos de horário
- Status de atendimento (Em Espera, Atendido, Cancelado)
- Reserva rápida via modal administrativo

### 4.3. CRM e Gestão de Clientes

- Cadastro com histórico de procedimentos
- Alergias e preferências personalizadas
- Métricas: ticket médio e taxa de retorno

### 4.4. Gestão de Equipe e Comissões

- Profissionais vinculados a especialidades
- Comissões diferenciadas por serviço ou profissional
- Turnos e expedientes

### 4.5. Catálogo de Serviços e Vendas

- Serviços base e adicionais (upselling)
- Cálculo dinâmico de duração e preço
- Categorização taxonômica (Cabelo, Unhas, Estética, etc.)

### 4.6. Financeiro e Fluxo de Caixa

- Entradas e saídas
- Liquidação de comissões em lote
- Relatórios de fechamento diário e mensal

---

## 5. Especificações de Design (Aura Beauty)

| Token | Valor |
|-------|--------|
| Estética | Minimalista, sofisticada, luxuosa |
| Cor principal | Ouro Rosé `#b78472` |
| Superfícies | `#faf9f8` |
| Texto | Antracite |
| Tipografia títulos | Playfair Display |
| Tipografia corpo | Sans-serif |
| Abordagem | Mobile-first, responsivo |

---

## 6. Pilha Tecnológica (PRD / protótipo)

| Camada | PRD recomenda | Implementação atual |
|--------|---------------|---------------------|
| Frontend | React + TypeScript + Tailwind | Go + HTML templates + Tailwind CDN |
| Ícones | Lucide React | Emojis / utilitários inline |
| Estado | Context API / Zustand | HTMX + server-render |
| API | Axios + mocks | Go `net/http` + PostgreSQL |

> A migração para React é **roadmap de UI**, não bloqueante para regras de negócio já no backend.

---

## 7. Matriz de Telas (protótipo Stitch)

Mais de 30 telas mapeadas (Desktop + Mobile): cadastro, painéis admin e agendamento público.

| Plataforma | Referências |
|------------|-------------|
| Desktop | SCREEN_34, SCREEN_27, SCREEN_26, SCREEN_9, SCREEN_15 |
| Mobile | SCREEN_24, SCREEN_12, SCREEN_33, SCREEN_29 |

*Tela de referência principal:* `9e431ec675cd4b8a9ba50fb3667ed072`

---

## 8. Documentos complementares

- [FRONTEND.md](./FRONTEND.md) — arquitetura React, Aura Beauty, matriz Stitch × rotas
- [REGRAS_NEGOCIO.md](./REGRAS_NEGOCIO.md) — regras implementadas no código
- [PRD_GAP_ANALYSIS.md](./PRD_GAP_ANALYSIS.md) — o que falta vs. PRD
