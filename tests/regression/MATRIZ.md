# Matriz de regressão AgendaGlow (DEV-51)

| Módulo | Cobertura | Arquivo |
| --- | --- | --- |
| Auth e multi-tenant | Perfis, credenciais, slug e rotas da Dona | `auth_tenant_test.go` |
| SaaS Stage 2 | 31 casos de planos, assinatura, limite, equipe e serviços | `saas_stage2_test.go` |
| Limite do plano | POST e PUT de reativação retornam 403 `limit_reached` | `saas_stage2_test.go` |
| Guarda 402 | VENCIDO/SUSPENSO bloqueiam e PAGAMENTO_PENDENTE vigente permite | `saas_stage2_test.go` |
| Agenda e financeiro | Colisão, encaixe, charge, comissão e re-charge 409 | `agenda_finance_test.go` |
| WhatsApp | Signup, PENDENTE/CONECTADO, gateway, idempotência e CONCLUIDO 409 | `whatsapp_test.go` |
| Contrato Gateway | `/send-notification`, `sistema_origem`, tenant, template e API key | `internal/service/whatsapp_contract_test.go` |
| Dinheiro | Comissão ao centavo, 0%, 100% e meio centavo | `internal/service/money_test.go` |
| UI estática | Portal do menu e navegação WhatsApp | `ui_static_test.go` |
| Super Admin | Planos, salões, assinatura, status e UI | `superadmin_stage4_test.go` |

## Execução

```bash
go test ./internal/service -run "WhatsApp|BuildWhatsApp|Money|Arredond|Comissao" -count=1
go test -tags=regression ./tests/regression/ -count=1
go test ./... -count=1
```

Os testes com a tag `regression` requerem API em `:8081` e Postgres em `:5435`.
