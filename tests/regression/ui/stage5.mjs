/**
 * QA Stage 5 — Regressão de UI React (§7 e §11 do brief DEV-10).
 * Roda contra a UI real em http://localhost:5173 (Vite) + API Go em 8081.
 * Navegador: Chrome instalado, via puppeteer-core.
 */
import { execFileSync } from 'node:child_process'
import { launch, newPage, uiLogin, apiLogin, inspectMenu, UI, API, CREDS } from './lib.mjs'

const psql = (sql) =>
  execFileSync(
    'docker',
    ['exec', 'postgres-glow', 'psql', '-U', 'postgres', '-d', 'agenda_glow_prod', '-t', '-A', '-c', sql],
    { encoding: 'utf8' },
  ).trim()

const results = []
let currentSection = ''

function section(name) {
  currentSection = name
  console.log(`\n=== ${name} ===`)
}
function check(id, desc, ok, detail = '') {
  results.push({ section: currentSection, id, desc, ok: !!ok, detail })
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${id}  ${desc}${detail ? ' :: ' + detail : ''}`)
}
function info(msg) {
  console.log(`      ~ ${msg}`)
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

async function pathOf(page) {
  return page.evaluate(() => location.pathname)
}

// ---------------------------------------------------------------- menu helper
async function openMenuByIndex(page, visibleIndex) {
  const box = await page.evaluate((idx) => {
    const vis = [...document.querySelectorAll('button[aria-haspopup=menu]')].filter(
      (b) => b.offsetParent !== null && b.getBoundingClientRect().width > 0,
    )
    const el = vis[idx]
    if (!el) return null
    el.scrollIntoView({ block: 'center', behavior: 'instant' })
    return { count: vis.length }
  }, visibleIndex)
  if (!box) return null
  await sleep(150)
  const rect = await page.evaluate((idx) => {
    const vis = [...document.querySelectorAll('button[aria-haspopup=menu]')].filter(
      (b) => b.offsetParent !== null && b.getBoundingClientRect().width > 0,
    )
    const r = vis[idx].getBoundingClientRect()
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
  }, visibleIndex)
  await page.mouse.move(rect.x, rect.y)
  await sleep(80)
  await page.mouse.click(rect.x, rect.y)
  await sleep(200)
  return page.evaluate(inspectMenu)
}

async function closeMenu(page) {
  await page.keyboard.press('Escape')
  await sleep(120)
}

async function auditMenu(page, { caseId, label, viewport, expectItems, whichTrigger = 'last' }) {
  const total = await page.evaluate(
    () =>
      [...document.querySelectorAll('button[aria-haspopup=menu]')].filter(
        (b) => b.offsetParent !== null && b.getBoundingClientRect().width > 0,
      ).length,
  )
  if (total === 0) {
    check(caseId, `${label} @ ${viewport} — menu "…" presente`, false, 'nenhum trigger visível')
    return null
  }
  const idx = whichTrigger === 'last' ? total - 1 : 0
  const m = await openMenuByIndex(page, idx)
  if (!m || !m.found) {
    check(caseId, `${label} @ ${viewport} — menu abre`, false, 'role=menu não renderizou')
    return null
  }
  const allItemsVisible = m.items.every((i) => i.insideViewport)
  const allHit = m.items.every((i) => i.hitTestOk)
  const countOk = expectItems == null || m.items.length === expectItems
  const ok =
    m.parentIsBody &&
    m.position === 'fixed' &&
    Number(m.zIndex) >= 200 &&
    m.insideViewport &&
    !m.needsScroll &&
    allItemsVisible &&
    allHit &&
    countOk
  check(
    caseId,
    `${label} @ ${viewport} — lista completa sem corte`,
    ok,
    `itens=${m.items.length}${expectItems != null ? '/' + expectItems : ''} portal=${m.parentIsBody} pos=${m.position} z=${m.zIndex} dentroViewport=${m.insideViewport} precisaScroll=${m.needsScroll} itensVisiveis=${allItemsVisible} clicaveis=${allHit} rect=[${m.rect.top.toFixed(0)},${m.rect.bottom.toFixed(0)}] vh=${m.viewport.h}`,
  )
  if (!ok) info(JSON.stringify(m.items.map((i) => [i.label, i.insideViewport, i.hitTestOk])))
  await closeMenu(page)
  return m
}

async function clickMenuItem(page, triggerIndexFromEnd = 0, itemLabel) {
  const total = await page.evaluate(
    () =>
      [...document.querySelectorAll('button[aria-haspopup=menu]')].filter(
        (b) => b.offsetParent !== null && b.getBoundingClientRect().width > 0,
      ).length,
  )
  await openMenuByIndex(page, total - 1 - triggerIndexFromEnd)
  const pos = await page.evaluate((lbl) => {
    const items = [...document.querySelectorAll('div[role=menu] button[role=menuitem]')]
    const el = items.find((b) => b.textContent.trim() === lbl)
    if (!el) return null
    const r = el.getBoundingClientRect()
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 }
  }, itemLabel)
  if (!pos) return false
  await page.mouse.click(pos.x, pos.y)
  await sleep(600)
  return true
}

// ================================================================== execução
const browser = await launch()
const t0 = Date.now()
try {
  // ---------------------------------------------------------------- §11.3
  section('§11.3 — Rotas protegidas redirecionam para /login')
  {
    const page = await newPage(browser)
    const protectedPaths = [
      '/admin/dashboard',
      '/admin/clientes',
      '/admin/servicos',
      '/admin/equipe',
      '/admin/financeiro',
      '/admin/whatsapp',
      '/admin/bloqueado',
      '/superadmin/dashboard',
      '/superadmin/saloes',
      '/superadmin/planos',
      '/profissional/dashboard',
      '/profissional/agenda',
      '/secretaria/agenda',
    ]
    await page.goto(`${UI}/login`, { waitUntil: 'domcontentloaded' })
    for (const p of protectedPaths) {
      await page.evaluate(() => localStorage.clear())
      await page.goto(`${UI}${p}`, { waitUntil: 'networkidle2' })
      await sleep(200)
      const dest = await pathOf(page)
      check(`UI-01/${p}`, `sem sessão ${p} → /login`, dest === '/login', `destino=${dest}`)
    }
    // cross-role
    await uiLogin(page, 'dona')
    await page.goto(`${UI}/superadmin/saloes`, { waitUntil: 'networkidle2' })
    await sleep(300)
    check('UI-02', 'DONA em /superadmin/saloes → /login', (await pathOf(page)) === '/login', `destino=${await pathOf(page)}`)

    await uiLogin(page, 'profissional')
    await page.goto(`${UI}/admin/dashboard`, { waitUntil: 'networkidle2' })
    await sleep(300)
    check('UI-03', 'PROFISSIONAL em /admin/dashboard → /login', (await pathOf(page)) === '/login', `destino=${await pathOf(page)}`)

    await uiLogin(page, 'superadmin')
    await page.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(300)
    check('UI-04', 'SUPER_ADMIN em /admin/clientes → /login', (await pathOf(page)) === '/login', `destino=${await pathOf(page)}`)

    // rota inexistente
    await page.evaluate(() => localStorage.clear())
    await page.goto(`${UI}/rota-que-nao-existe-qa`, { waitUntil: 'networkidle2' })
    await sleep(400)
    const nf = await pathOf(page)
    check('UI-05', 'rota inexistente → catálogo público 404 ou /login', nf === '/login' || nf === '/rota-que-nao-existe-qa', `destino=${nf}`)
    await page.close()
  }

  // ---------------------------------------------------------------- §11 redirect por role
  section('§11 / §1.11 — Redirect pós-login por perfil')
  {
    const page = await newPage(browser)
    const expected = {
      dona: '/admin/dashboard',
      profissional: '/profissional/dashboard',
      superadmin: '/superadmin/dashboard',
    }
    for (const [who, exp] of Object.entries(expected)) {
      const dest = await uiLogin(page, who)
      check(`UI-06/${who}`, `login ${who} → ${exp}`, dest === exp, `destino=${dest}`)
    }
    // telas de login por perfil também redirecionam corretamente
    const destDonaScreen = await uiLogin(page, 'dona', '/login/dona')
    check('UI-07', '/login/dona → /admin/dashboard', destDonaScreen === '/admin/dashboard', `destino=${destDonaScreen}`)
    const destSuper = await uiLogin(page, 'superadmin', '/login/superadmin')
    check('UI-08', '/login/superadmin → /superadmin/dashboard', destSuper === '/superadmin/dashboard', `destino=${destSuper}`)

    // credencial inválida permanece no login com erro genérico
    await page.goto(`${UI}/login`, { waitUntil: 'domcontentloaded' })
    await page.evaluate(() => localStorage.clear())
    await page.goto(`${UI}/login`, { waitUntil: 'domcontentloaded' })
    await page.waitForSelector('#email')
    await page.type('#email', 'dona@glow.local')
    await page.type('#password', 'senha-errada-qa')
    await page.click('button[type=submit]')
    await sleep(1500)
    const err = await page.evaluate(() => document.body.innerText)
    check(
      'UI-09',
      'senha inválida: continua em /login com erro genérico',
      (await pathOf(page)) === '/login' && /inválid/i.test(err) && !/não existe|not found/i.test(err),
      `msg="${(err.match(/.*inválid.*/i) || [''])[0].trim()}"`,
    )
    await page.close()
  }

  // ---------------------------------------------------------------- §11.2 sidebar WhatsApp
  section('§11.2 — Sidebar da Dona com item WhatsApp → /admin/whatsapp')
  {
    const page = await newPage(browser)
    await uiLogin(page, 'dona')
    const sidebar = await page.evaluate(() => {
      const links = [...document.querySelectorAll('aside a')].map((a) => ({
        label: a.textContent.trim(),
        href: a.getAttribute('href'),
      }))
      return links
    })
    const wa = sidebar.find((l) => /whatsapp/i.test(l.label))
    check('UI-10', 'item WhatsApp existe na sidebar da Dona', !!wa, `href=${wa?.href}`)
    check('UI-11', 'item WhatsApp aponta para /admin/whatsapp', wa?.href === '/admin/whatsapp', `href=${wa?.href}`)

    await Promise.all([
      page.waitForFunction(() => location.pathname === '/admin/whatsapp', { timeout: 10000 }).catch(() => {}),
      page.evaluate(() => {
        const a = [...document.querySelectorAll('aside a')].find((x) => /whatsapp/i.test(x.textContent))
        a.click()
      }),
    ])
    await sleep(1200)
    const waPage = await page.evaluate(() => ({
      path: location.pathname,
      text: document.body.innerText,
      hasConnectBtn: [...document.querySelectorAll('button')].some((b) =>
        /Conectar WhatsApp Oficial/i.test(b.textContent),
      ),
    }))
    check('UI-12', 'clique no item navega para /admin/whatsapp', waPage.path === '/admin/whatsapp', `path=${waPage.path}`)
    check('UI-13', 'tela WhatsApp renderiza botão Conectar', waPage.hasConnectBtn)
    const stateMatch = waPage.text.match(/beleza_[0-9a-f-]{36}/i)
    const tenantId = await page.evaluate(
      () => JSON.parse(localStorage.getItem('agendaglow_session')).user.tenant_id,
    )
    check(
      'UI-14',
      'tela exibe state=beleza_{idDoSalao} do tenant logado',
      !!stateMatch && stateMatch[0] === `beleza_${tenantId}`,
      `state=${stateMatch?.[0]} tenant=${tenantId}`,
    )
    check(
      'UI-15',
      'tela WhatsApp não carrega nada da porta 8082 no browser',
      !waPage.text.includes('8082'),
    )
    await page.close()
  }

  // ---------------------------------------------------------------- §11.1 / §7.3 menus
  section('§11.1 e §7.3 — Menus "…" completos, sem corte (portal/fixed)')
  {
    const viewports = [
      { name: '1440x900', width: 1440, height: 900 },
      { name: '1280x720', width: 1280, height: 720 },
      { name: '1366x600 (baixo)', width: 1366, height: 600 },
      { name: '390x844 (mobile)', width: 390, height: 844 },
    ]
    const donaPages = [
      { path: '/admin/clientes', label: 'Clientes (§7.3)', items: 4 },
      { path: '/admin/servicos', label: 'Serviços', items: 3 },
      { path: '/admin/equipe', label: 'Equipe (§4.7)', items: 2 },
    ]
    for (const vp of viewports) {
      const page = await newPage(browser, { width: vp.width, height: vp.height })
      await uiLogin(page, 'dona')
      for (const p of donaPages) {
        await page.goto(`${UI}${p.path}`, { waitUntil: 'networkidle2' })
        await sleep(900)
        await auditMenu(page, {
          caseId: `UI-20/${p.path}/${vp.name}`,
          label: p.label,
          viewport: vp.name,
          expectItems: p.items,
        })
      }
      await page.close()

      const sp = await newPage(browser, { width: vp.width, height: vp.height })
      await uiLogin(sp, 'superadmin')
      await sp.goto(`${UI}/superadmin/saloes`, { waitUntil: 'networkidle2' })
      await sleep(900)
      await auditMenu(sp, {
        caseId: `UI-20//superadmin/saloes/${vp.name}`,
        label: 'Salões Super Admin (§9.7)',
        viewport: vp.name,
        expectItems: 5,
      })
      await sp.close()
    }

    // pior caso: última linha com a página rolada até o fim, viewport baixo
    const page = await newPage(browser, { width: 1366, height: 620 })
    await uiLogin(page, 'dona')
    await page.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(900)
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
    await sleep(400)
    await auditMenu(page, {
      caseId: 'UI-21',
      label: 'Clientes — última linha, página rolada até o rodapé',
      viewport: '1366x620',
      expectItems: 4,
    })
    await page.close()
  }

  // ---------------------------------------------------------------- §7 CRM
  section('§7 — Clientes / CRM da Dona')
  {
    const page = await newPage(browser, { width: 1440, height: 900 })
    await uiLogin(page, 'dona')
    await page.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(900)

    const tenantId = await page.evaluate(
      () => JSON.parse(localStorage.getItem('agendaglow_session')).user.tenant_id,
    )
    const listInfo = await page.evaluate(() => {
      const db = JSON.parse(localStorage.getItem('agendaglow_db') ?? 'null')
      return {
        rows: document.querySelectorAll('tbody tr').length,
        counter: (document.body.innerText.match(/Mostrando[\s\S]{0,40}clientes/) || [''])[0].replace(/\n/g, ' '),
        dbClients: db?.clientes?.length ?? null,
        tenantsOfClients: db ? [...new Set(db.clientes.map((c) => c.tenant_id))] : null,
      }
    })
    check('UI-30', '§7.1 lista de clientes do tenant renderiza', listInfo.rows > 0, `linhas=${listInfo.rows} ${listInfo.counter}`)
    check(
      'UI-31',
      '§7.1 clientes carregados pertencem só ao tenant logado',
      listInfo.tenantsOfClients == null || (listInfo.tenantsOfClients.length <= 1 && (listInfo.tenantsOfClients[0] ?? tenantId) === tenantId),
      `tenants=${JSON.stringify(listInfo.tenantsOfClients)} sessao=${tenantId}`,
    )

    // filtro por busca
    const firstName = await page.evaluate(
      () => document.querySelector('tbody tr a')?.textContent.trim() ?? '',
    )
    await page.type('input[placeholder*="Buscar"]', firstName.split(' ')[0])
    await sleep(700)
    const filtered = await page.evaluate(() => ({
      rows: [...document.querySelectorAll('tbody tr a[href^="/admin/clientes/"]')].map((a) =>
        a.textContent.trim(),
      ),
    }))
    check(
      'UI-32',
      '§7.1 busca filtra a lista',
      filtered.rows.length > 0 &&
        filtered.rows.length <= listInfo.rows &&
        filtered.rows.every((n) => n.toLowerCase().includes(firstName.split(' ')[0].toLowerCase())),
      `busca="${firstName.split(' ')[0]}" resultados=${filtered.rows.length} (lista cheia=${listInfo.rows})`,
    )
    await page.evaluate(() => {
      const i = document.querySelector('input[placeholder*="Buscar"]')
      const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set
      setter.call(i, '')
      i.dispatchEvent(new Event('input', { bubbles: true }))
    })
    await sleep(500)

    // deep-link WhatsApp
    const waLink = await page.evaluate(() => {
      const a = document.querySelector('tbody a[href^="https://wa.me/"]')
      return a ? { href: a.getAttribute('href'), target: a.getAttribute('target'), rel: a.getAttribute('rel') } : null
    })
    check(
      'UI-33',
      '§7.4 deep-link WhatsApp wa.me com telefone só dígitos, nova aba',
      !!waLink && /^https:\/\/wa\.me\/\d{8,}$/.test(waLink.href) && waLink.target === '_blank',
      JSON.stringify(waLink),
    )

    // detalhe do cliente
    const clienteHref = await page.evaluate(
      () => document.querySelector('tbody tr a[href^="/admin/clientes/"]')?.getAttribute('href'),
    )
    await page.goto(`${UI}${clienteHref}`, { waitUntil: 'networkidle2' })
    await sleep(900)
    const detalhe = await page.evaluate(() => ({
      path: location.pathname,
      text: document.body.innerText.slice(0, 800),
    }))
    check(
      'UI-34',
      '§7.2 detalhe do cliente abre',
      detalhe.path === clienteHref && !/Nenhum cliente|erro/i.test(detalhe.text.slice(0, 200)),
      `path=${detalhe.path}`,
    )

    // histórico via menu "…"
    await page.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(900)
    const okHist = await clickMenuItem(page, 0, 'Ver histórico')
    const modalText = await page.evaluate(() => document.body.innerText)
    check(
      'UI-35',
      '§7.2/§7.3 item "Ver histórico" do menu abre o modal de histórico',
      okHist && /Histórico —/.test(modalText),
      `modalAberto=${/Histórico —/.test(modalText)}`,
    )
    await page.keyboard.press('Escape')
    await page.evaluate(() => {
      const b = [...document.querySelectorAll('button')].find((x) => x.textContent.trim() === 'Fechar')
      b?.click()
    })
    await sleep(400)

    // agendar via menu "…"
    await page.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(900)
    await clickMenuItem(page, 0, 'Agendar')
    await sleep(800)
    const agPath = await pathOf(page)
    check(
      'UI-36',
      '§7.2/§7.3 item "Agendar" navega para a tela de agendamento do cliente',
      /^\/admin\/clientes\/[^/]+\/agendar$/.test(agPath),
      `path=${agPath}`,
    )
    await page.close()
  }

  // ---------------------------------------------------------------- §11.4 bloqueio
  section('§11.4 — Assinatura bloqueada (experiência da Dona)')
  {
    const su = await apiLogin('superadmin')
    const H = { Authorization: `Bearer ${su.token}`, 'Content-Type': 'application/json' }
    const stamp = Date.now().toString().slice(-8)
    const slug = `qa-ui-bloq-${stamp}`
    const created = await (
      await fetch(`${API}/api/v1/admin/establishments`, {
        method: 'POST',
        headers: H,
        body: JSON.stringify({ nome_comercial: `QA UI Bloqueio ${stamp}`, slug }),
      })
    ).json()
    const plans = await (await fetch(`${API}/api/v1/admin/plans`, { headers: H })).json()
    const plano = plans.find((p) => p.ativo) ?? plans[0]
    await fetch(`${API}/api/v1/admin/establishments/${created.id}/assign-plan`, {
      method: 'POST',
      headers: H,
      body: JSON.stringify({ plano_id: plano.id ?? plano.ID, meses: 12 }),
    })
    const donaEmail = `${slug}-dona@glow.local`
    await fetch(`${API}/superadmin/establishments/${created.id}/create-dona`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${su.token}`, 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ email: donaEmail }).toString(),
    })
    info(`fixture salão=${slug} id=${created.id} dona=${donaEmail}`)

    CREDS.donaFixture = { email: donaEmail, password: 'AgendaGlow@2026' }

    // 1) com assinatura ATIVA a dona entra normalmente
    const page = await newPage(browser)
    const destAtivo = await uiLogin(page, 'donaFixture')
    const textoAtivo = await page.evaluate(() => document.body.innerText)
    check(
      'UI-40',
      'baseline: dona do salão ATIVO acessa /admin/dashboard',
      destAtivo === '/admin/dashboard' && !/Assinatura Suspensa/.test(textoAtivo),
      `destino=${destAtivo}`,
    )

    // 2) suspende o salão
    await fetch(`${API}/superadmin/establishments/${created.id}/suspend`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${su.token}` },
    })
    // guarda SaaS tem cache de 60s; o suspend invalida, mas damos folga
    await sleep(1500)
    const apiStatus = await fetch(`${API}/api/v1/bootstrap`, {
      headers: { Authorization: `Bearer ${(await (await fetch(`${API}/api/v1/auth/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(CREDS.donaFixture) })).json()).token}` },
    })
    info(`GET /api/v1/bootstrap com salão suspenso → HTTP ${apiStatus.status}`)

    const page2 = await newPage(browser)
    const destBloq = await uiLogin(page2, 'donaFixture')
    await sleep(1500)
    const bloq = await page2.evaluate(() => ({
      path: location.pathname,
      text: document.body.innerText,
      tenantStatus: (() => {
        const db = JSON.parse(localStorage.getItem('agendaglow_db') ?? 'null')
        const s = JSON.parse(localStorage.getItem('agendaglow_session') ?? 'null')
        return db && s ? db.tenants.find((t) => t.id === s.user.tenant_id)?.status : null
      })(),
    }))
    const bloqueado = /Assinatura Suspensa/.test(bloq.text) && /HTTP 402/.test(bloq.text)
    check(
      'UI-41',
      '§11.4 dona com assinatura suspensa cai na experiência de bloqueio (HTTP 402)',
      bloqueado,
      `path=${bloq.path} statusTenantNoStore=${bloq.tenantStatus} bloqueio=${bloqueado}`,
    )
    if (!bloqueado) info(bloq.text.slice(0, 400).replace(/\n+/g, ' | '))

    // 3) navegar para outra rota protegida da dona continua bloqueado
    await page2.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(1200)
    const bloq2 = await page2.evaluate(() => document.body.innerText)
    check(
      'UI-42',
      '§11.4 bloqueio persiste em outras rotas /admin/*',
      /Assinatura Suspensa/.test(bloq2),
      `bloqueado=${/Assinatura Suspensa/.test(bloq2)}`,
    )

    // 3b) impacto: a dona bloqueada só descobre o bloqueio quando uma ação falha
    await page2.goto(`${UI}/admin/clientes`, { waitUntil: 'networkidle2' })
    await sleep(1000)
    await page2.evaluate(() => {
      const b = [...document.querySelectorAll('button')].find((x) => /Novo Cliente/i.test(x.textContent))
      b?.click()
    })
    await sleep(500)
    const inputs = await page2.$$('input')
    if (inputs.length >= 2) {
      await inputs[inputs.length - 3].type('QA Bloqueio')
      await inputs[inputs.length - 2].type('5515999990000')
      await page2.evaluate(() => {
        const b = [...document.querySelectorAll('button')].find((x) => x.textContent.trim() === 'Salvar')
        b?.click()
      })
      await sleep(1500)
      const msg = await page2.evaluate(() => document.body.innerText)
      info(
        `ação sob bloqueio: ${/vencida ou suspensa/i.test(msg) ? 'erro 402 exibido só ao salvar' : 'sem mensagem clara de bloqueio'}`,
      )
    }

    // 4) rota /admin/bloqueado direta
    await page2.goto(`${UI}/admin/bloqueado`, { waitUntil: 'networkidle2' })
    await sleep(1000)
    const bloq3 = await page2.evaluate(() => ({ path: location.pathname, text: document.body.innerText }))
    check(
      'UI-43',
      '§11.4 /admin/bloqueado exibe a tela de assinatura suspensa',
      bloq3.path === '/admin/bloqueado' && /Assinatura Suspensa/.test(bloq3.text),
      `path=${bloq3.path}`,
    )

    // 5) reativa e confirma que a dona volta a acessar
    await fetch(`${API}/superadmin/establishments/${created.id}/activate`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${su.token}` },
    })
    await sleep(1500)
    const page3 = await newPage(browser)
    const destReativado = await uiLogin(page3, 'donaFixture')
    const textoReativado = await page3.evaluate(() => document.body.innerText)
    check(
      'UI-44',
      '§11.4 após reativar, a dona volta a acessar o painel',
      destReativado === '/admin/dashboard' && !/Assinatura Suspensa/.test(textoReativado),
      `destino=${destReativado}`,
    )
    await page.close()
    await page2.close()
    await page3.close()

    globalThis.__fixture = { id: created.id, slug, donaEmail }
  }

  // ---------------------------------------------------------------- §11.5 portas
  section('§11.5 — Portas: UI na 5173, Gateway 8082 fora do front')
  {
    const page = await newPage(browser)
    const urls = []
    page.on('request', (r) => urls.push(r.url()))
    await uiLogin(page, 'dona')
    for (const p of ['/admin/dashboard', '/admin/clientes', '/admin/whatsapp', '/admin/calendario', '/admin/financeiro']) {
      await page.goto(`${UI}${p}`, { waitUntil: 'networkidle2' })
      await sleep(600)
    }
    const to8082 = urls.filter((u) => u.includes(':8082'))
    const to8081 = urls.filter((u) => u.includes(':8081'))
    const apiViaProxy = urls.filter((u) => u.startsWith(`${UI}/api/`))
    check('UI-50', 'nenhuma requisição do front para a porta 8082 (Gateway)', to8082.length === 0, `total=${to8082.length}`)
    check(
      'UI-51',
      'chamadas de API saem pela mesma origem 5173 (proxy Vite → 8081)',
      apiViaProxy.length > 0 && to8081.length === 0,
      `viaProxy=${apiViaProxy.length} diretoNa8081=${to8081.length}`,
    )
    await page.close()
  }
} finally {
  await browser.close()
  if (globalThis.__fixture) {
    try {
      psql(`DELETE FROM users WHERE email = '${globalThis.__fixture.donaEmail}'`)
      psql(`DELETE FROM estabelecimentos WHERE id = '${globalThis.__fixture.id}'`)
      console.log(`\nfixture removida do banco: ${globalThis.__fixture.slug}`)
    } catch (e) {
      console.log(`\nfalha ao remover fixture: ${e.message}`)
    }
  }
}

// ------------------------------------------------------------------ resumo
const failed = results.filter((r) => !r.ok)
console.log(`\n================ RESUMO ================`)
console.log(`Total: ${results.length} | PASS: ${results.length - failed.length} | FAIL: ${failed.length} | ${((Date.now() - t0) / 1000).toFixed(0)}s`)
if (failed.length) {
  console.log('\nFalhas:')
  for (const f of failed) console.log(` - [${f.section}] ${f.id} ${f.desc} :: ${f.detail}`)
}
if (globalThis.__fixture) console.log(`\nFixture criada (limpar): ${JSON.stringify(globalThis.__fixture)}`)
process.exit(failed.length ? 1 : 0)
