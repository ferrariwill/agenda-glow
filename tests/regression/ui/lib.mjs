import puppeteer from 'puppeteer-core'

export const UI = 'http://localhost:5173'
export const API = 'http://localhost:8081'
export const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'

export const CREDS = {
  dona: { email: 'dona@glow.local', password: 'AgendaGlow@2026' },
  profissional: { email: 'claudia@glow.local', password: 'AgendaGlow@2026' },
  superadmin: { email: 'ferrariwill@gmail.com', password: 'AgendaGlow@2026' },
}

export async function launch() {
  return puppeteer.launch({
    executablePath: CHROME,
    headless: 'new',
    args: ['--no-sandbox', '--disable-dev-shm-usage'],
    defaultViewport: { width: 1440, height: 900 },
  })
}

export async function newPage(browser, viewport = { width: 1440, height: 900 }) {
  const page = await browser.newPage()
  await page.setViewport(viewport)
  page.on('pageerror', (e) => console.log('   [pageerror]', String(e).slice(0, 200)))
  return page
}

export async function clearSession(page) {
  await page.goto(`${UI}/login`, { waitUntil: 'domcontentloaded' })
  await page.evaluate(() => {
    localStorage.clear()
    sessionStorage.clear()
  })
}

export async function uiLogin(page, who, path = '/login') {
  const { email, password } = CREDS[who]
  await page.goto(`${UI}${path}`, { waitUntil: 'domcontentloaded' })
  await page.evaluate(() => localStorage.clear())
  await page.goto(`${UI}${path}`, { waitUntil: 'domcontentloaded' })
  await page.waitForSelector('#email')
  await page.type('#email', email)
  await page.type('#password', password)
  await Promise.all([
    page.waitForFunction(() => !location.pathname.startsWith('/login'), { timeout: 20000 }),
    page.click('button[type=submit]'),
  ])
  await page.waitForFunction(
    () => !document.body.innerText.includes('Carregando dados do salão'),
    { timeout: 20000 },
  )
  return new URL(page.url()).pathname
}

export async function apiLogin(who) {
  const { email, password } = CREDS[who]
  const res = await fetch(`${API}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) throw new Error(`login ${who} -> ${res.status}`)
  return res.json()
}

/** Inspect the currently open role=menu: portal, positioning, viewport fit, hit-testability. */
export const inspectMenu = () =>
  // eslint-disable-next-line no-undef
  (() => {
    const menu = document.querySelector('div[role=menu]')
    if (!menu) return { found: false }
    const cs = getComputedStyle(menu)
    const r = menu.getBoundingClientRect()
    const items = [...menu.querySelectorAll('button[role=menuitem]')].map((b) => {
      const br = b.getBoundingClientRect()
      const cx = br.left + br.width / 2
      const cy = br.top + br.height / 2
      const hit = document.elementFromPoint(cx, cy)
      return {
        label: b.textContent.trim(),
        rect: { top: br.top, bottom: br.bottom, left: br.left, right: br.right },
        insideViewport:
          br.top >= 0 &&
          br.left >= 0 &&
          br.bottom <= window.innerHeight + 0.5 &&
          br.right <= window.innerWidth + 0.5,
        insideMenuBox: br.top >= r.top - 0.5 && br.bottom <= r.bottom + 0.5,
        hitTestOk: !!hit && (menu.contains(hit) || hit === menu),
      }
    })
    // walk ancestors of the trigger's original container is not needed: portal => body child
    return {
      found: true,
      parentIsBody: menu.parentElement === document.body,
      position: cs.position,
      zIndex: cs.zIndex,
      overflowY: cs.overflowY,
      rect: { top: r.top, bottom: r.bottom, left: r.left, right: r.right, width: r.width, height: r.height },
      insideViewport:
        r.top >= -0.5 &&
        r.left >= -0.5 &&
        r.bottom <= window.innerHeight + 0.5 &&
        r.right <= window.innerWidth + 0.5,
      scrollHeight: menu.scrollHeight,
      clientHeight: menu.clientHeight,
      needsScroll: menu.scrollHeight > menu.clientHeight + 1,
      items,
      viewport: { w: window.innerWidth, h: window.innerHeight },
    }
  })()
