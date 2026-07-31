# AgendaGlow — Regras de Negócio

Documento consolidado a partir do código (`internal/service`, `internal/security`, `migrations`, handlers e rotas em `backend/cmd/api/main.go`).

**Última revisão:** junho/2026  
**Ver também:** [PRD.md](./PRD.md) · [FRONTEND.md](./FRONTEND.md) · [PRD_GAP_ANALYSIS.md](./PRD_GAP_ANALYSIS.md)

---

## Índice

1. [Perfis, autenticação e acesso às rotas](#1-perfis-autenticação-e-acesso-às-rotas)
2. [Multi-tenancy (estabelecimentos)](#2-multi-tenancy-estabelecimentos)
3. [Planos SaaS e assinaturas](#3-planos-saas-e-assinaturas)
4. [Usuários e credenciais](#4-usuários-e-credenciais)
5. [Especialidades](#5-especialidades)
6. [Equipe (profissionais)](#6-equipe-profissionais)
7. [Procedimentos e adicionais](#7-procedimentos-e-adicionais)
8. [Agendamentos](#8-agendamentos)
9. [Clientes e agendamento público](#9-clientes-e-agendamento-público)
10. [Financeiro (fluxo de caixa)](#10-financeiro-fluxo-de-caixa)
11. [Clube de assinatura](#11-clube-de-assinatura)
12. [WhatsApp](#12-whatsapp)
13. [Super Admin](#13-super-admin)
14. [Lacunas e exceções conhecidas](#14-lacunas-e-exceções-conhecidas)
15. [Credenciais de desenvolvimento](#15-credenciais-de-desenvolvimento)

---

## 1. Perfis, autenticação e acesso às rotas

### 1.1 Perfis (roles)

| Perfil | Escopo no JWT |
|--------|----------------|
| `SUPER_ADMIN` | Plataforma inteira; sem `estabelecimento_id` nem `profissional_id` |
| `DONA` | Um salão; `estabelecimento_id` obrigatório |
| `PROFISSIONAL` | Um salão + uma profissional; ambos os IDs obrigatórios |

**Onde:** `internal/security/session.go` (`validateClaimsScope`), migração `000011_create_users.up.sql` (CHECK em `users.role`).

### 1.2 Autenticação

- Login por e-mail e senha (bcrypt); e-mail normalizado em minúsculas.
- Token JWT com validade de **24 horas** (cookie `agendaglow_session` ou header `Authorization: Bearer`).
- Usuário inativo → login bloqueado (`ErrUsuarioInativo` → HTTP 403).
- Credenciais inválidas retornam erro genérico (não revela se o e-mail existe).

**Onde:** `internal/service/auth.go`, `internal/security/session.go`.

### 1.3 Super Admin

- E-mail autorizado: **`ferrariwill@gmail.com`** (validado no login e em toda requisição).
- Seed: `migrations/000012_seed_super_admin.up.sql`.

**Onde:** `internal/service/auth.go`, `internal/security/session.go` (`RequireSuperAdmin`).

### 1.4 Matriz de rotas

| Acesso | Rotas |
|--------|--------|
| **Público** | `/health`, `/login*`, `POST /api/v1/auth/login`, slots públicos, approve/reschedule, webhook WhatsApp, `GET /{slug}` |
| **Super Admin** | `/api/v1/admin/*`, `/superadmin/*` |
| **Dona + assinatura SaaS ativa** | Catálogo, finanças, `/admin/*`, `/dashboard/gerencial/*` |
| **Profissional** | `/dashboard/profissional/*` (**sem** validação de assinatura SaaS) |

**Onde:** `backend/cmd/api/main.go`.

### 1.5 Login HTML

- Formulário pode indicar o perfil (`/login/dona`, etc.).
- Se o `role` do formulário não bater com o do usuário, redireciona para o login correto.

**Onde:** `backend/internal/handler/auth.go` (`LoginForm`).

---

## 2. Multi-tenancy (estabelecimentos)

### 2.1 Identidade do tenant

- Cada salão é um tenant identificado por `estabelecimento_id`.
- Dados isolados: profissionais, serviços, clientes, agendamentos, fluxo de caixa, especialidades, planos do clube.
- FKs compostas garantem que agendamento referencia profissional, serviço e cliente **do mesmo salão**.

**Onde:** `migrations/000005_add_slug_to_establishments.up.sql`.

### 2.2 Slug e URL pública

- Formato: `^[a-z0-9]+(-[a-z0-9]+)*$` (sem acentos, minúsculas).
- Único globalmente.
- Cadastro exige **nome comercial**; salão criado **ativo**.
- Slug duplicado → rejeitado (`ErrSlugAlreadyExists`).

**Onde:** `internal/service/estabelecimento.go`, `internal/service/slug.go`.

### 2.3 Visibilidade pública

- Página de agendamento e catálogo só para estabelecimentos **ativos** (`ativo = TRUE`).
- Salão inativo → **404** na slug.

**Onde:** `internal/service/estabelecimento.go` (`BuscarPorSlug`, `BuscarCatalogoAutoatendimento`).

### 2.4 Configuração do salão

- Atualização só em estabelecimento ativo.
- Slug revalidado; não pode estar em uso por outro salão.
- Logo: máximo **2 MB**; extensões `.png`, `.jpg`, `.jpeg`, `.webp`.

**Onde:** `internal/service/estabelecimento.go`, `backend/internal/handler/estabelecimento_config.go`.

---

## 3. Planos SaaS e assinaturas

### 3.1 Plano corporativo (`planos_saas`)

| Campo | Regra |
|-------|--------|
| `nome` | Obrigatório; único case-insensitive (`LOWER(TRIM(nome))`) |
| `preco_mensal` | ≥ 0 |
| `limite_profissionais` | > 0 |
| `ativo` | Boolean |

**Onde:** `internal/service/plano.go`, migrações `000007`, `000014`.

**Seed dev:** Plano Solo Dev — R$ 97/mês, limite **1** profissional (`000013`).

### 3.2 Assinatura do salão (`assinaturas_estabelecimentos`)

- **Uma assinatura por estabelecimento** (`estabelecimento_id UNIQUE`).
- Status no banco: `ATIVO`, `PAGAMENTO_PENDENTE`, `SUSPENSO`.

#### Atribuir ou renovar plano

- Meses de contratação **> 0** (`ErrMesesContratacaoInvalidos`).
- Plano deve existir e estar **ativo**.
- Status definido como `ATIVO`.
- **Data de vencimento:** se a assinatura ainda não venceu, soma a partir do vencimento atual; senão, a partir de hoje (UTC, meia-noite).

**Onde:** `internal/service/plano.go` (`AssignPlanToEstablishment`, `resolverBaseVencimento`).

#### Suspender / ativar (UI Super Admin)

| Ação | Efeito |
|------|--------|
| Suspender | `estabelecimentos.ativo = false` **e** assinatura `SUSPENSO` |
| Ativar | `estabelecimentos.ativo = true` **e** assinatura `ATIVO` |

**UI:** cadastro de salão com plano → **12 meses**; renovar → **+12 meses**.

**Onde:** `internal/service/plano.go`, `backend/internal/handler/superadmin_ui.go`.

### 3.3 Guarda SaaS (middleware da Dona)

Bloqueia acesso quando:

- Não há linha de assinatura, **ou**
- `status = 'SUSPENSO'`, **ou**
- `data_vencimento` anterior a hoje (UTC).

**Não bloqueia** `PAGAMENTO_PENDENTE` se ainda dentro do vencimento.

- Resposta: HTTP **402** (`subscription_expired_or_suspended`).
- Cache em memória: **60 segundos** por estabelecimento (invalidado ao alterar plano/status).

**Onde:** `internal/security/subscription.go`, `internal/security/middleware.go`.

### 3.4 Status exibido no Super Admin

| Condição | Status exibido |
|----------|----------------|
| Salão inativo ou assinatura `SUSPENSO` | `SUSPENSO` |
| Sem vencimento ou vencido | `VENCIDO` |
| Caso contrário | `ATIVO` |

**Onde:** `internal/service/superadmin.go` (`calcularStatusAssinatura`).

---

## 4. Usuários e credenciais

- E-mail único por usuário (`ErrEmailJaCadastrado`).
- Usuário criado como **ativo**.
- Escopo validado no banco (`users_role_escopo`) e no JWT.

### Criação de Dona pelo Super Admin

- E-mail padrão: `{slug}-dona@glow.local` (ou informado no formulário).
- Senha inicial documentada: **`AgendaGlow@2026`**.
- Role `DONA` vinculada ao `estabelecimento_id`.

**Onde:** `internal/service/auth.go`, `backend/internal/handler/superadmin_ui.go`.

---

## 5. Especialidades

Cadastro **por estabelecimento**; usado na equipe via seleção (não texto livre).

| Regra | Detalhe |
|-------|---------|
| Nome | Obrigatório; único case-insensitive no salão |
| Status | `ativo` boolean; criada como ativa |
| Duplicata | `ErrEspecialidadeNomeDuplicado` |
| Uso na equipe | Profissional nova ou reativada exige especialidade **ativa** |
| Inativa | Profissional desativada pode manter especialidade inativa no histórico |

**Fluxo:** `/admin/especialidades` (cadastro) → `/admin/equipe` (seleção no dropdown).

**Migração:** `000015` — textos legados migrados; fallback **"Geral"** se necessário.

**Onde:** `internal/service/especialidade.go`, `internal/service/profissional.go`.

---

## 6. Equipe (profissionais)

### 6.1 Cadastro e edição

| Campo | Regra |
|-------|--------|
| Nome | Obrigatório |
| Especialidade | `especialidade_id` obrigatório |
| Comissão | 0% a 100% (padrão DB: **40%**) |
| Status | Criada como **ativa** |

**Onde:** `internal/service/profissional.go`.

### 6.2 Limite do plano SaaS

- Conta apenas profissionais **ativas**.
- Se `total_ativas >= limite_profissionais` → `ErrPlanLimitExceeded` (HTTP 403).
- Reativar profissional também respeita o limite.
- Sem assinatura/plano válido → erro ao cadastrar (`ErrPlanoSaasNaoEncontrado`).

**Onde:** `internal/service/profissional.go` (`CreateProfessional`, `UpdateProfessional`, `buscarLimitePlano`).

### 6.3 Expediente (horários de trabalho)

| Regra | Valor |
|-------|--------|
| `dia_semana` | 0–6 (domingo = 0); sem duplicatas na mesma requisição |
| Jornada | `horario_saida` > `horario_entrada` |
| Almoço | Início e fim juntos ou nenhum; intervalo dentro da jornada |
| Formato de hora | `HH:MM` ou `HH:MM:SS` |
| Persistência | Substitui todos os expedientes da profissional (transação atômica) |

**Onde:** `internal/service/profissional.go` (`SetProfessionalHours`), migração `000010`.

### 6.4 Painel da profissional

- Dados isolados por `profissional_id` e `estabelecimento_id` do token.
- Concluir atendimento: agendamento deve ser **`CONFIRMADO`** e pertencer à profissional.
- Resumo semanal: segunda a domingo; comissões pendentes = soma `CUSTO_VARIAVEL` `PENDENTE`; serviços = contagem `CONCLUIDO`.

**Onde:** `internal/service/agenda_profissional.go`.

---

## 7. Procedimentos e adicionais

### 7.1 Serviço base (`servicos`)

| Campo | Regra |
|-------|--------|
| Nome | Obrigatório |
| `preco_base` | ≥ 0 |
| `duracao_base_minutos` | > 0 |
| Status | Criado **ativo** |

**Onde:** `internal/service/procedimento.go`, migração `000001`.

### 7.2 Adicionais (`servico_adicionais`)

| Campo | Regra |
|-------|--------|
| Nome | Obrigatório |
| `preco_adicional` | ≥ 0 |
| `duracao_adicional_minutos` | ≥ 0 (zero permitido) |
| Serviço pai | Do estabelecimento e **ativo** |

No agendamento, adicional deve pertencer ao mesmo `servico_id` (trigger no banco).

**Onde:** `internal/service/procedimento.go`, migração `000001`.

### 7.3 Preço e duração no agendamento

- Duração total = base + Σ adicionais.
- Preço total = base + Σ adicionais.
- Valores monetários arredondados em **2 casas decimais**.

**Onde:** `internal/service/agenda.go`, `internal/service/money.go`.

---

## 8. Agendamentos

### 8.1 Status

| Status | Significado |
|--------|-------------|
| `AGENDADO` | Horário confirmado (padrão na criação) |
| `EM_APROVACAO` | Encaixe com sobreposição parcial no fim do slot |
| `CONFIRMADO` | Confirmado (ex.: botão WhatsApp) |
| `CONCLUIDO` | Atendimento finalizado; financeiro lançado |
| `CANCELADO` | Cancelado |

**Onde:** migrações `000001`, `000009`; `internal/service/agenda.go`.

### 8.2 Origem

- Apenas `INTERNO` ou `EXTERNO`.

**Onde:** `internal/service/agenda.go` (`validarOrigemAgendamento`).

### 8.3 Criação (`CriarAgendamento`)

Em transação atômica:

1. Upsert do cliente (telefone).
2. Valida profissional e serviço **ativos**.
3. Verifica colisão de horário.
4. Insere agendamento + adicionais.

- Status inicial: `AGENDADO` (ou `EM_APROVACAO` se encaixe).
- **Não há endpoint HTTP público de criação** — lógica no serviço e testes E2E.

**Onde:** `internal/service/agenda.go`.

### 8.4 Colisão de horários

| Situação | Comportamento |
|----------|----------------|
| Status que bloqueiam sobreposição | Apenas `AGENDADO` e `CONFIRMADO` |
| `EM_APROVACAO` | **Não bloqueia** outros agendamentos |
| Colisão total no início | Rejeita (`ErrColisaoHorario`) |
| Sobreposição parcial no fim | Status `EM_APROVACAO`; `minutos_invadidos` ≥ 1 |

**Onde:** `internal/service/agenda_collision.go`.

### 8.5 Aprovação e reagendamento (público, sem autenticação)

| Ação | Regra |
|------|--------|
| Aprovar | Só de `EM_APROVACAO` → `AGENDADO` |
| Reagendar | Só de `EM_APROVACAO`; novo horário no **mesmo dia** (`HH:MM`); intervalo totalmente livre |

- E-mail assíncrono para profissional com até **3 horários alternativos** (se tiver e-mail cadastrado).

**Onde:** `internal/service/agenda_approval.go`, handlers públicos em `main.go`.

### 8.6 Slots disponíveis (API pública)

| Parâmetro | Regra |
|-----------|--------|
| Grade | Intervalos de **30 minutos** |
| Pré-requisitos | Profissional ativo, serviço ativo, adicionais válidos |
| Sem expediente no dia | Lista vazia (não é erro) |
| Bloqueios | Agendamentos `AGENDADO`/`CONFIRMADO` + intervalo de almoço |
| Slot válido | `início + duração_total ≤ horario_saida` |

**Onde:** `internal/service/agenda_slots.go`.

---

## 9. Clientes e agendamento público

### 9.1 Cliente

- Telefone único **por estabelecimento**.
- Upsert na reserva: se telefone já existe, **mantém o nome antigo** (não sobrescreve com o nome do formulário).

**Onde:** `internal/service/agenda.go` (`resolverOuCadastrarCliente`), migração `000005`.

### 9.2 Endpoints públicos

| Rota | Função |
|------|--------|
| `GET /{slug}` | Página de agendamento (catálogo) |
| `GET /api/v1/public/{slug}/slots` | Horários livres (`data`, `profissional_id`, `procedimento_id`, `adicionais` opcionais) |

Salão inativo → **404**.

---

## 10. Financeiro (fluxo de caixa)

### 10.1 Tipos de lançamento (`fluxo_caixa.tipo`)

| Tipo | Uso |
|------|-----|
| `ENTRADA` | Receita |
| `CUSTO_FIXO` | Despesa fixa (aluguel, contas) |
| `CUSTO_VARIAVEL` | Comissão / custo variável |

**Onde:** migração `000002`.

### 10.2 Status de pagamento de comissão

- `PENDENTE` ou `PAGO`.
- Comissões geradas automaticamente na conclusão do atendimento: **`PENDENTE`**.

**Onde:** migração `000008`.

### 10.3 Conclusão do atendimento (`ConcluirAtendimento`)

| Pré-condição | Erro |
|--------------|------|
| Já `CONCLUIDO` | `ErrAgendamentoJaConcluido` |
| `CANCELADO` | `ErrAgendamentoCancelado` |

#### Sem clube (`via_clube_assinatura = false`)

- `ENTRADA` = valor total do serviço (base + adicionais).
- `CUSTO_VARIAVEL` = `valor × comissao% / 100`, status `PENDENTE`.

#### Com clube (`via_clube_assinatura = true`)

- **Sem** `ENTRADA` (cliente já pagou na assinatura).
- Debita **1 visita** do assinante.
- Se `valor_repasse_profissional > 0` → lançamento de repasse `PENDENTE`.

**Onde:** `internal/service/financeiro.go`.

### 10.4 Quitar comissões (Dona)

- Quita todos os `CUSTO_VARIAVEL` `PENDENTE` da profissional no período.
- Marca `PAGO` e acrescenta `| LIQUIDADO em DD/MM/YYYY HH:MM` na descrição.
- Sem pendências → `ErrNenhumaComissaoPendente`.

**Onde:** `internal/service/financeiro_report.go` (`PayPartnerCommissions`).

### 10.5 Relatório gerencial

```
lucro_liquido_estimado = entradas − custos_fixos − todas_comissões_variáveis
total_comissoes_pendentes = apenas CUSTO_VARIAVEL com status PENDENTE
```

**Onde:** `internal/service/financeiro_dashboard.go`, `internal/service/financeiro_report.go`.

### 10.6 Lançamento manual

- Descrição obrigatória; valor **> 0**.
- Tipos: `ENTRADA`, `CUSTO_FIXO`, `CUSTO_VARIAVEL`.
- Lançamento manual de `CUSTO_VARIAVEL` **não entra** no fluxo automático de quitação de comissões.

**Onde:** `internal/service/financeiro_dashboard.go` (`RegistrarLancamentoCaixa`).

### 10.7 Tela de caixa (admin)

- Exibe até **100** lançamentos do mês corrente.

**Onde:** `internal/service/admin_pages.go`.

---

## 11. Clube de assinatura

### 11.1 Plano do clube (`planos_assinatura`)

| Campo | Regra |
|-------|--------|
| `preco_mensal` | ≥ 0 |
| `total_visitas_mes` | > 0 |
| `valor_repasse_profissional` | ≥ 0 |
| `ativo` | Boolean |

**Onde:** migração `000003`, escopo por tenant em `000005`.

### 11.2 Assinante (`clientes_assinantes`)

| Campo | Regra |
|-------|--------|
| `status` | `ATIVO`, `INADIMPLENTE`, `CANCELADO` |
| `visitas_restantes` | ≥ 0 |
| Telefone | Único por estabelecimento |

### 11.3 Uso no agendamento (somente consumo — sem CRUD na API)

| Momento | Regra |
|---------|--------|
| Reserva | Se assinante `ATIVO`, créditos > 0 e plano ativo → `via_clube_assinatura = true` |
| Conclusão | Debita 1 visita; sem crédito → `ErrAssinaturaSemCreditos` |
| `INADIMPLENTE` / `CANCELADO` | Não qualificam para clube na reserva |

**Nota:** não há API/UI para cadastrar planos do clube ou assinantes — apenas schema e uso no fluxo de agendamento.

**Onde:** `internal/service/financeiro.go`, `internal/service/agenda.go`.

---

## 12. WhatsApp

### 12.0 Papel do Beleza vs Gateway

| Direção | Contrato |
|---------|----------|
| **Beleza → Gateway** | `POST {WHATSAPP_GATEWAY_URL}/send-notification` + `X-API-Key` |
| **Gateway → Beleza** | Webhooks em `8081` (conexão + respostas de botão) |
| **Embedded Signup** | Meta → Gateway (`/meta/embedded-signup/callback`); Gateway notifica Beleza |
| **Porta Gateway** | **8082** (`WHATSAPP_GATEWAY_URL`) |

No Gateway configure:
`BELEZA_SAAS_WEBHOOK_URL=http://localhost:8081/api/v1/webhook/whatsapp-gateway`

### 12.1 Envio (saída)

- Path: **`/send-notification`** (não mais `/v1/messages/send-template`).
- Payload: `sistema_origem=beleza`, `tenant_id` (ID do salão), `phone_number`, `template_name`, `language_code`, `simple_template`, `appointment_id`, `variables`.
- Lembrete pós-agendamento: template `lembrete_agenda` (assíncrono).

**Onde:** `internal/service/notificacao.go`.

### 12.2 Webhook — conexão Embedded Signup

`POST /api/v1/webhook/whatsapp-connected` (ou unificado `/whatsapp-gateway`)

Payload Gateway:
```json
{
  "event": "whatsapp_connection_completed",
  "sistema_origem": "beleza",
  "tenant_id": "<idDoSalao>",
  "salon_id": "<idDoSalao>",
  "waba_id": "...",
  "phone_number_id": "...",
  "whatsapp_phone_number": "+55 ...",
  "status": "connected"
}
```
→ status do salão = `CONECTADO`.

Se `whatsapp_enabled = false`, a conexão é recusada com HTTP 403,
`whatsapp_feature_disabled`. Essa resposta é terminal e não deve ser retentada
pelo Gateway.

### 12.3 Webhook — respostas do cliente (CONFIRM/CANCEL)

`POST /api/v1/webhook/whatsapp-callback` (ou unificado `/whatsapp-gateway`)

```json
{
  "sistema_origem": "beleza",
  "tenant_id": "<idDoSalao>",
  "phone_number": "5515...",
  "text": "APPT_CONFIRM",
  "event_type": "button_reply",
  "action": "CONFIRM"
}
```

- Se vier `appointment_id`, usa direto; senão resolve o agendamento mais recente do telefone no salão.
- Auth opcional: `X-API-Key` = `WHATSAPP_GATEWAY_KEY` (se a env estiver definida).

### 12.4 Embedded Signup (painel dona)

- Tela `/admin/whatsapp` — botão **Conectar WhatsApp Oficial**.
- `state=beleza_{idDoSalao}` concatenado na URL Meta.
- `GET /api/v1/whatsapp/integration` continua retornando 200 e expõe
  `whatsapp_enabled`; quando desativado, `state` e `signup_url` são vazios.
- `POST /api/v1/whatsapp/integration/start` exige assinatura SaaS ativa e a
  liberação do recurso. A assinatura é validada primeiro (402), depois a flag
  administrativa (403).
- Respostas `CONFIRM`/`CANCEL` continuam sendo processadas quando a flag está
  desligada, mas configuração e novos envios ficam bloqueados.

**Onde:** `internal/service/whatsapp_integration.go`, `internal/handler/whatsapp_webhook.go`, `migrations/000018`.

---

## 13. Super Admin

| Operação | Regra |
|----------|--------|
| Criar salão | Slug válido e único; nome comercial obrigatório |
| Atribuir plano | Meses > 0; plano ativo; estende vencimento |
| Renovar | +N meses no plano atual (UI padrão: 12) |
| Suspender / ativar | Salão + assinatura em conjunto |
| Criar plano SaaS | Preço ≥ 0; limite > 0; nome único |
| Editar plano SaaS | Nome, preço, limite, ativo |
| Criar Dona | E-mail único; senha padrão dev |
| Toggle status (API JSON) | Altera só `ativo` do salão (não mexe assinatura automaticamente) |
| Toggle WhatsApp | Só `SUPER_ADMIN`; altera `whatsapp_enabled`, preservando status, WABA e telefone |

**Onde:** `backend/internal/handler/superadmin_ui.go`, `internal/service/plano.go`, `internal/service/superadmin.go`.

---

## 14. Lacunas e exceções conhecidas

1. **Profissional não passa pelo guarda SaaS** — salão suspenso/vencido ainda acessa o dashboard da profissional.
2. **`PAGAMENTO_PENDENTE`** ainda libera a Dona se dentro do vencimento.
3. **`EM_APROVACAO` não reserva** o calendário para colisões e slots.
4. **Approve/reschedule públicos** sem autenticação (segurança por obscuridade do UUID).
5. **Config do estabelecimento** (`POST /v1/estabelecimentos/config`) não valida se `estabelecimento_id` do form corresponde ao JWT.
6. **Sem endpoint público** para criar agendamento — página de bio é catálogo + slots.
7. **Clube de assinatura** sem CRUD na aplicação.
8. **Nome do cliente** não atualiza em re-reserva com o mesmo telefone.
9. **Webhook WhatsApp** aberto se `WHATSAPP_GATEWAY_KEY` não estiver configurada.
10. **Cache SaaS** pode liberar acesso por até **60 s** após suspensão (até expirar TTL ou invalidação explícita).
11. **Cache da flag WhatsApp** usa TTL de 30 s; o toggle administrativo invalida
    a entrada imediatamente, eliminando a janela no fluxo normal.

---

## 15. Credenciais de desenvolvimento

| Perfil | E-mail | Senha |
|--------|--------|-------|
| Super Admin | `ferrariwill@gmail.com` | `AgendaGlow@2026` |
| Dona | `dona@glow.local` | `AgendaGlow@2026` |
| Profissional | `claudia@glow.local` | `AgendaGlow@2026` |

- Slug dev: `estudio-glow-salto`
- URL pública: `http://localhost:8081/estudio-glow-salto`
- API: porta **8081** | Postgres host: **5435**

**Onde:** `migrations/000012`, `migrations/000013`, `.env`, `internal/config/runtime.go`.

---

## Referência rápida de erros de domínio

| Erro | Contexto |
|------|----------|
| `ErrSlugAlreadyExists` | Slug duplicado no cadastro |
| `ErrPlanoSaasNomeDuplicado` | Nome de plano SaaS duplicado |
| `ErrPlanLimitExceeded` | Limite de profissionais do plano |
| `ErrAssinaturaVencidaOuSuspensa` | Middleware SaaS |
| `ErrEspecialidadeNomeDuplicado` | Especialidade duplicada no salão |
| `ErrEspecialidadeInativa` | Especialidade inativa na equipe |
| `ErrColisaoHorario` | Sobreposição total de horário |
| `ErrAgendamentoJaConcluido` | Tentativa de concluir de novo |
| `ErrNenhumaComissaoPendente` | Quitar comissões sem pendências |
| `ErrAssinaturaSemCreditos` | Clube sem visitas restantes |

**Onde:** pacotes `internal/service/*.go`.
