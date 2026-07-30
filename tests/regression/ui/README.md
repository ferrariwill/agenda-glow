# Regressão de UI React — Stage 5 (§7 e §11 do brief de QA)

Suíte E2E em navegador real (Chrome instalado + `puppeteer-core`) contra a UI Vite
em `http://localhost:5173` e a API Go em `http://localhost:8081`.

## Pré-requisitos

- API `backend/cmd/api` rodando na 8081 e Postgres na 5435 (container `postgres-glow`).
- `frontend-react` com `VITE_DATA_SOURCE=api` e `npm run dev` na 5173.
- `npm install puppeteer-core` no diretório onde a suíte for executada.
- Chrome em `C:\Program Files\Google\Chrome\Application\chrome.exe` (ajustar `CHROME` em `lib.mjs`).

## Execução

```bash
node stage5.mjs           # suíte completa (60 casos): rotas, redirect por role, menus "…", CRM, bloqueio, portas
node probe_bloqueio.mjs   # investigação do bloqueio SaaS: vencido, suspensão em sessão viva, resíduo de store
```

`stage5.mjs` cria o salão/dona de fixture pela API do Super Admin e remove ambos do banco no fim
(`docker exec postgres-glow psql`). Sai com código 1 se qualquer caso falhar.

## Cobertura

| Casos | Regra |
| --- | --- |
| UI-01…UI-05 | §11.3 rotas protegidas e cross-role → `/login` |
| UI-06…UI-09 | §11 / §1.11 redirect pós-login por perfil e erro genérico de credencial |
| UI-10…UI-15 | §11.2 sidebar da Dona → `/admin/whatsapp` e `state=beleza_{idDoSalao}` |
| UI-20…UI-21 | §11.1 / §7.3 / §4.7 / §9.7 menus "…" em 4 telas × 4 viewports (portal, `fixed`, viewport fit, hit-test) |
| UI-30…UI-36 | §7 CRM: lista, busca, detalhe, histórico, agendar, deep-link WhatsApp |
| UI-40…UI-44 | §11.4 experiência de bloqueio da assinatura |
| UI-50…UI-51 | §11.5 UI na 5173, sem tráfego para a 8082 |
