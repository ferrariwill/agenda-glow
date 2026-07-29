/** Investigação do defeito de bloqueio (§11.4): variantes VENCIDO, suspensão em sessão viva e resíduo de store. */
import { execFileSync } from 'node:child_process'
import { launch, newPage, uiLogin, apiLogin, UI, API, CREDS } from './lib.mjs'

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const psql = (sql) =>
  execFileSync('docker', ['exec', 'postgres-glow', 'psql', '-U', 'postgres', '-d', 'agenda_glow_prod', '-t', '-A', '-c', sql], {
    encoding: 'utf8',
  }).trim()

const su = await apiLogin('superadmin')
const H = { Authorization: `Bearer ${su.token}`, 'Content-Type': 'application/json' }
const stamp = Date.now().toString().slice(-8)
const slug = `qa-ui-venc-${stamp}`

const est = await (
  await fetch(`${API}/api/v1/admin/establishments`, {
    method: 'POST',
    headers: H,
    body: JSON.stringify({ nome_comercial: `QA UI Vencido ${stamp}`, slug }),
  })
).json()
const plans = await (await fetch(`${API}/api/v1/admin/plans`, { headers: H })).json()
const plano = plans.find((p) => p.ativo) ?? plans[0]
await fetch(`${API}/api/v1/admin/establishments/${est.id}/assign-plan`, {
  method: 'POST',
  headers: H,
  body: JSON.stringify({ plano_id: plano.id, meses: 12 }),
})
const donaEmail = `${slug}-dona@glow.local`
await fetch(`${API}/superadmin/establishments/${est.id}/create-dona`, {
  method: 'POST',
  headers: { Authorization: `Bearer ${su.token}`, 'Content-Type': 'application/x-www-form-urlencoded' },
  body: new URLSearchParams({ email: donaEmail }).toString(),
})
CREDS.donaVenc = { email: donaEmail, password: 'AgendaGlow@2026' }
console.log(`fixture: ${slug} id=${est.id} dona=${donaEmail}`)

// vencimento no passado, assinatura permanece ATIVO e salão ativo=true
psql(`UPDATE assinaturas_estabelecimentos SET data_vencimento = CURRENT_DATE - 30 WHERE estabelecimento_id = '${est.id}'`)
console.log(
  'estado no banco:',
  psql(
    `SELECT e.ativo, a.status, a.data_vencimento FROM estabelecimentos e JOIN assinaturas_estabelecimentos a ON a.estabelecimento_id = e.id WHERE e.id = '${est.id}'`,
  ),
)
await sleep(1200)

const tok = (await (await fetch(`${API}/api/v1/auth/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(CREDS.donaVenc) })).json()).token
const bs = await fetch(`${API}/api/v1/bootstrap`, { headers: { Authorization: `Bearer ${tok}` } })
console.log(`API GET /api/v1/bootstrap (dona vencida) -> HTTP ${bs.status} ${(await bs.text()).slice(0, 120)}`)

const browser = await launch()
try {
  // A) assinatura VENCIDA (não suspensa) — mesma classe de falha?
  const p = await newPage(browser)
  const dest = await uiLogin(p, 'donaVenc')
  await sleep(1500)
  const t = await p.evaluate(() => document.body.innerText)
  console.log(`\n[A] dona com assinatura VENCIDA -> path=${dest} bloqueio=${/Assinatura Suspensa/.test(t)}`)
  console.log(`    topo da tela: ${t.slice(0, 160).replace(/\n+/g, ' | ')}`)
  await p.screenshot({ path: 'evidencia_dona_vencida_dashboard.png', fullPage: false })

  // rota financeira sensível
  await p.goto(`${UI}/admin/financeiro`, { waitUntil: 'networkidle2' })
  await sleep(1500)
  const fin = await p.evaluate(() => ({ path: location.pathname, txt: document.body.innerText.slice(0, 200) }))
  console.log(`[A2] /admin/financeiro com assinatura vencida -> path=${fin.path} | ${fin.txt.replace(/\n+/g, ' | ')}`)

  // B) suspensão durante sessão viva + reload
  const p2 = await newPage(browser)
  await fetch(`${API}/superadmin/establishments/${est.id}/activate`, { method: 'POST', headers: { Authorization: `Bearer ${su.token}` } })
  await sleep(800)
  await uiLogin(p2, 'donaVenc')
  await fetch(`${API}/superadmin/establishments/${est.id}/suspend`, { method: 'POST', headers: { Authorization: `Bearer ${su.token}` } })
  await sleep(1500)
  await p2.reload({ waitUntil: 'networkidle2' })
  await sleep(2000)
  const t2 = await p2.evaluate(() => ({ path: location.pathname, blocked: /Assinatura Suspensa/.test(document.body.innerText) }))
  console.log(`[B] suspenso durante a sessão + reload -> path=${t2.path} bloqueio=${t2.blocked}`)

  // C) resíduo de store entre tenants na mesma aba (dona seed -> logout -> dona bloqueada)
  const p3 = await newPage(browser)
  await uiLogin(p3, 'dona')
  await p3.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
  await sleep(1200)
  const seedClientes = await p3.evaluate(() => document.querySelectorAll('tbody tr').length)
  await p3.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find((x) => x.textContent.trim() === 'Sair')
    b?.click()
  })
  await sleep(800)
  await p3.waitForSelector('#email', { timeout: 10000 })
  await p3.type('#email', CREDS.donaVenc.email)
  await p3.type('#password', CREDS.donaVenc.password)
  await p3.click('button[type=submit]')
  await sleep(3000)
  const res = await p3.evaluate(() => ({
    path: location.pathname,
    blocked: /Assinatura Suspensa/.test(document.body.innerText),
    head: document.body.innerText.slice(0, 200).replace(/\n+/g, ' | '),
  }))
  console.log(`[C] seed(${seedClientes} clientes) -> logout -> dona bloqueada na MESMA aba: path=${res.path} bloqueio=${res.blocked}`)
  await p3.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
  await sleep(1500)
  const leak = await p3.evaluate(() => ({
    blocked: /Assinatura Suspensa/.test(document.body.innerText),
    rows: document.querySelectorAll('tbody tr').length,
    nomes: [...document.querySelectorAll('tbody tr a[href^="/admin/clientes/"]')].slice(0, 3).map((a) => a.textContent.trim()),
    contador: (document.body.innerText.match(/Mostrando[\s\S]{0,40}clientes/) || [''])[0].replace(/\n/g, ' '),
  }))
  console.log(`[C2] /admin/clientes na mesma aba: bloqueio=${leak.blocked} linhas=${leak.rows} ${leak.contador} amostra=${JSON.stringify(leak.nomes)}`)
  await p3.screenshot({ path: 'evidencia_residuo_mesma_aba.png' })
} finally {
  await browser.close()
}

// limpeza
psql(`DELETE FROM users WHERE email = '${donaEmail}'`)
psql(`DELETE FROM estabelecimentos WHERE id = '${est.id}'`)
console.log(`\nfixture removida: ${slug}`)
