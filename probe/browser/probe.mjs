// quail browser-driven probe.
//
// Invoked by the Go binary when QUAIL_BROWSER_PROBE=1 (or --use-browser
// is set). Takes a single URL on argv, performs a bounded BFS crawl using
// Chromium, and writes a JSON document to stdout describing the rendered
// state of each page.
//
// Schema:
//   { origin: string, pages: [{
//       url, finalURL, title, h1, h2s, links, images, meta, hasForm,
//       interactions, errors
//     }, ...], errors: string[] }
//
// Bounds (env-overridable): max 20 pages, max depth 3, single Chromium
// instance, sequential page fetches (rate-friendly).

// v0.89: engine selection via QUAIL_ENGINE (chromium|firefox|webkit,
// default chromium) and stealth wrapping via QUAIL_STEALTH
// (on|off, default on). The Go side cascades chromium→firefox→webkit
// in auto mode; per-engine binaries are lazy-installed by the Go
// runner cache. playwright-extra + StealthPlugin patch JS-layer bot
// detection (navigator.webdriver, plugins, chrome runtime); WAF
// detection at the TLS/HTTP2 layer falls to the engine's native
// fingerprint, which is why Firefox/WebKit tend to slip past Akamai-
// class WAFs where Chromium gets dropped.
import { chromium as chrStock, firefox as ffStock, webkit as wkStock } from '@playwright/test'
import { addExtra } from 'playwright-extra'
import StealthPlugin from 'puppeteer-extra-plugin-stealth'

const ENGINES = { chromium: chrStock, firefox: ffStock, webkit: wkStock }
const ENGINE_NAME = (process.env.QUAIL_ENGINE || 'chromium').toLowerCase()
const STOCK_ENGINE = ENGINES[ENGINE_NAME] || chrStock
const STEALTH_ON = (process.env.QUAIL_STEALTH || 'on').toLowerCase() !== 'off'
const engine = STEALTH_ON ? addExtra(STOCK_ENGINE).use(StealthPlugin()) : STOCK_ENGINE

const TARGET = process.argv[2]
if (!TARGET) {
  console.error('quail-browser-probe: usage: node probe.mjs <url>')
  process.exit(2)
}

const MAX_PAGES = Number(process.env.QUAIL_MAX_PAGES ?? 20)
const MAX_DEPTH = Number(process.env.QUAIL_MAX_DEPTH ?? 3)
const NAV_TIMEOUT_MS = Number(process.env.QUAIL_NAV_TIMEOUT ?? 15_000)
const IDLE_MS = 800
const ACCEPT_BUTTON_RE = /^(accept|agree|got it|continue|i understand|allow|ok|allow all|accept all)/i

async function dismissCookieBanner(page) {
  // Best-effort: click anything that looks like an accept/agree button.
  const triggers = page.locator(
    'button, [role="button"], a:has-text("accept"), a:has-text("agree")'
  )
  const n = Math.min(await triggers.count(), 8)
  for (let i = 0; i < n; i++) {
    const t = triggers.nth(i)
    try {
      const text = ((await t.innerText({ timeout: 500 })) || '').trim()
      if (ACCEPT_BUTTON_RE.test(text) && (await t.isVisible({ timeout: 200 }))) {
        await t.click({ timeout: 1000 })
        await page.waitForTimeout(200)
        return true
      }
    } catch { /* keep going */ }
  }
  return false
}

async function expandPopups(page) {
  // Hover/click each aria-haspopup trigger to surface dropdown links.
  const popups = page.locator('[aria-haspopup]')
  const n = Math.min(await popups.count(), 6)
  for (let i = 0; i < n; i++) {
    const t = popups.nth(i)
    try {
      if (await t.isVisible({ timeout: 200 })) {
        await t.hover({ timeout: 500 })
        await page.waitForTimeout(100)
      }
    } catch { /* ignore — some popups need real click */ }
  }
}

async function collectPage(page, requestedURL) {
  const finalURL = page.url()
  const title = await page.title()

  const h1Texts = await page.locator('h1').allInnerTexts()
  const h2Texts = await page.locator('h2').allInnerTexts()

  // Links: include visibility hint so the Go side can prefer visible ones.
  const linkHandles = await page.locator('a[href]').elementHandles()
  const links = []
  for (const h of linkHandles.slice(0, 200)) {
    const href = (await h.getAttribute('href')) || ''
    const text = ((await h.innerText().catch(() => '')) || '').trim()
    const visible = await h.isVisible().catch(() => false)
    links.push({ href, text, visible })
    await h.dispose()
  }

  // Images with non-empty alt.
  const imgHandles = await page.locator('img[alt]').elementHandles()
  const images = []
  for (const h of imgHandles.slice(0, 50)) {
    const alt = ((await h.getAttribute('alt')) || '').trim()
    if (!alt) { await h.dispose(); continue }
    const src = (await h.getAttribute('src')) || ''
    images.push({ src, alt })
    await h.dispose()
  }

  // Meta tags + canonical.
  const meta = await page.evaluate(() => {
    const get = (name, attr = 'name') => {
      const el = document.querySelector(`meta[${attr}="${name}"]`)
      return el ? (el.getAttribute('content') || '').trim() : ''
    }
    return {
      Description: get('description'),
      ViewportContent: get('viewport'),
      OGTitle: get('og:title', 'property'),
      OGType: get('og:type', 'property'),
      OGDescription: get('og:description', 'property'),
      Canonical: (document.querySelector('link[rel="canonical"]')?.getAttribute('href') || '').trim(),
    }
  })

  // v0.93: shadow-DOM piercing. Modern bank/insurer/telco sites render
  // calculators as flex components (custom elements with open shadow
  // roots) — e.g. ING's hypotheek-berekenen widget. Plain
  // document.querySelectorAll and page.locator(css) don't cross shadow
  // boundaries, so inputs/buttons inside the widget are invisible. We
  // walk shadowRoot recursively (open roots only — closed roots are
  // unreachable by spec).
  // ponytail: same-doc only; iframe traversal when a real fixture demands it.
  const shadowScan = await page.evaluate(() => {
    function deepQueryAll(root, selector, depth = 0, out = []) {
      if (depth > 8) return out
      try {
        for (const el of root.querySelectorAll(selector)) out.push(el)
      } catch { /* selector failure on non-Element root */ }
      const all = root.querySelectorAll ? root.querySelectorAll('*') : []
      for (const el of all) {
        if (el.shadowRoot) deepQueryAll(el.shadowRoot, selector, depth + 1, out)
      }
      return out
    }
    const isVisible = (el) => !!(el.offsetParent || (el.getClientRects && el.getClientRects().length))

    const formEls = deepQueryAll(document, 'form')
    const hasForm = formEls.length > 0
    const forms = []
    for (const f of formEls) {
      const inputs = []
      for (const el of deepQueryAll(f, 'input, select, textarea')) {
        const tag = el.tagName.toLowerCase()
        const t = el.getAttribute('type') || ''
        if (tag === 'input' && (t === 'hidden' || t === 'submit' || t === 'button')) continue
        inputs.push({
          tag,
          type: t,
          name: el.getAttribute('name') || '',
          testid: el.getAttribute('data-testid') || '',
          aria: el.getAttribute('aria-label') || '',
          placeholder: el.getAttribute('placeholder') || '',
          required: el.hasAttribute('required'),
        })
      }
      forms.push({
        action: f.getAttribute('action') || '',
        method: (f.getAttribute('method') || '').toLowerCase(),
        enctype: (f.getAttribute('enctype') || '').toLowerCase(),
        inputs,
      })
    }

    const inputs = []
    for (const el of deepQueryAll(document, 'input, select, textarea').slice(0, 40)) {
      inputs.push({
        tag: el.tagName.toLowerCase(),
        type: el.getAttribute('type') || '',
        name: el.getAttribute('name') || '',
        testid: el.getAttribute('data-testid') || '',
        aria: el.getAttribute('aria-label') || '',
        placeholder: el.getAttribute('placeholder') || '',
        required: el.hasAttribute('required'),
        visible: isVisible(el),
      })
    }
    return { hasForm, forms, inputs }
  })
  const hasForm = shadowScan.hasForm
  const forms = shadowScan.forms
  const inputs = shadowScan.inputs

  // Post-JS-render DOM snapshot — capped at 1 MiB so a single long page
  // doesn't blow the JSON pipe. Writes to tests/e2e/_dom/<slug>.html on
  // the Go side so reviewers can see what the browser actually rendered.
  let domHTML = ''
  try {
    domHTML = await page.evaluate(() => document.documentElement.outerHTML)
    if (domHTML.length > 1_048_576) domHTML = domHTML.slice(0, 1_048_576)
  } catch { /* leave empty — caller skips emission */ }

  // Interactive components (the v0.12 exercise journey kinds).
  // v0.93: shadow-piercing — same rationale as the form/input scan above.
  const interactions = await page.evaluate(() => {
    function deepQueryAll(root, selector, depth = 0, out = []) {
      if (depth > 8) return out
      try {
        for (const el of root.querySelectorAll(selector)) out.push(el)
      } catch { /* */ }
      const all = root.querySelectorAll ? root.querySelectorAll('*') : []
      for (const el of all) {
        if (el.shadowRoot) deepQueryAll(el.shadowRoot, selector, depth + 1, out)
      }
      return out
    }
    const out = []
    deepQueryAll(document, 'input[type="search"]').forEach(el => out.push({ kind: 'search', inputType: 'search' }))
    deepQueryAll(document, '[role="searchbox"]').forEach(() => out.push({ kind: 'search', role: 'searchbox' }))
    deepQueryAll(document, 'details').forEach(el => {
      const summary = el.querySelector('summary')
      out.push({ kind: 'details', text: (summary?.innerText || '').trim() })
    })
    deepQueryAll(document, 'dialog').forEach(() => out.push({ kind: 'dialog' }))
    deepQueryAll(document, '[role="tab"]').forEach(el => out.push({ kind: 'tab', text: (el.innerText || '').trim(), role: 'tab' }))
    deepQueryAll(document, 'input[type="date"],input[type="time"],input[type="datetime-local"]').forEach(el => {
      out.push({ kind: 'date', inputType: el.getAttribute('type'), name: el.getAttribute('name') || '' })
    })
    deepQueryAll(document, '[data-toggle],[data-bs-toggle]').forEach(el => {
      const t = el.getAttribute('data-toggle') || el.getAttribute('data-bs-toggle') || ''
      out.push({ kind: 'data-toggle', toggle: t, text: (el.innerText || '').trim() })
    })
    deepQueryAll(document, '[aria-haspopup]').forEach(el => {
      if (el.getAttribute('aria-expanded') !== null && el.getAttribute('aria-controls') !== null) return
      if (el.getAttribute('role') === 'tab') return
      out.push({ kind: 'popup', text: (el.innerText || '').trim() })
    })
    deepQueryAll(document, '[aria-expanded]').forEach(el => {
      if (out.length >= 80) return
      if (el.hasAttribute('data-toggle') || el.hasAttribute('data-bs-toggle')) return
      if (el.getAttribute('role') === 'tab') return
      out.push({ kind: 'collapsible', text: (el.innerText || '').trim().slice(0, 120), role: el.getAttribute('role') || '' })
    })
    deepQueryAll(document, 'button[aria-pressed]').forEach(el => {
      if (out.length >= 80) return
      out.push({ kind: 'toggle', text: (el.innerText || '').trim().slice(0, 120) })
    })
    deepQueryAll(document, 'input[type="range"]').forEach(el => {
      if (out.length >= 80) return
      out.push({ kind: 'slider', name: el.getAttribute('name') || '', inputType: 'range' })
    })
    return out
  })

  // v0.91: explicit role-tagged actionable capture. The Go-side
  // regex over DOMHTML catches these too, but Playwright queries
  // resolve ARIA roles (both explicit role="..." and implicit
  // roles like <button>) more reliably across engines/frameworks.
  // The Go side dedups by tag+testid+aria+role+name so duplicate
  // captures are harmless. Capped at 50/page.
  const roleAnchors = await page.evaluate(() => {
    function deepQueryAll(root, selector, depth = 0, out = []) {
      if (depth > 8) return out
      try {
        for (const el of root.querySelectorAll(selector)) out.push(el)
      } catch { /* */ }
      const all = root.querySelectorAll ? root.querySelectorAll('*') : []
      for (const el of all) {
        if (el.shadowRoot) deepQueryAll(el.shadowRoot, selector, depth + 1, out)
      }
      return out
    }
    const out = []
    const els = deepQueryAll(document, '[role="button"],[role="submit"],[role="link"],[role="menuitem"]')
    for (let i = 0; i < els.length && out.length < 50; i++) {
      const el = els[i]
      out.push({
        role: el.getAttribute('role') || '',
        text: (el.innerText || '').trim().slice(0, 200),
        ariaLabel: el.getAttribute('aria-label') || '',
        testid: el.getAttribute('data-testid') || '',
        visible: !!(el.offsetParent || el.getClientRects().length),
      })
    }
    return out
  })

  // v0.94: primary-component capture. Identifies "the main thing on
  // this page" so the spec generator can fan scenarios around it
  // (calculator inputs, hero form fields, etc.) instead of just
  // probing the first text input. Detection order:
  //   1. light-DOM <main> or [role="main"] containing a form
  //   2. shadow host whose root contains a form with ≥2 inputs
  //      (catches <flex-calc>-style web-component widgets)
  //   3. plain DOM form with the most inputs (≥2)
  // Returns null when nothing actionable is found.
  // ponytail: light-DOM main + shadow-host fallback covers the
  //   real-world cases we've seen (banking flex widgets, retailer
  //   hero forms); add deeper hierarchies when a real site needs it.
  const primaryComponent = await page.evaluate(() => {
    function deepQueryAll(root, selector, depth = 0, out = []) {
      if (depth > 8) return out
      try {
        for (const el of root.querySelectorAll(selector)) out.push(el)
      } catch { /* */ }
      const all = root.querySelectorAll ? root.querySelectorAll('*') : []
      for (const el of all) {
        if (el.shadowRoot) deepQueryAll(el.shadowRoot, selector, depth + 1, out)
      }
      return out
    }
    function selectorOf(el) {
      if (!el) return ''
      const tag = el.tagName ? el.tagName.toLowerCase() : ''
      const id = el.id ? '#' + el.id : ''
      const role = el.getAttribute && el.getAttribute('role')
      return tag + id + (role ? `[role="${role}"]` : '')
    }
    function inputsIn(root) {
      const out = []
      for (const el of deepQueryAll(root, 'input, select, textarea')) {
        const tag = el.tagName.toLowerCase()
        const t = el.getAttribute('type') || ''
        if (tag === 'input' && (t === 'hidden' || t === 'submit' || t === 'button')) continue
        const row = {
          tag,
          type: t,
          name: el.getAttribute('name') || '',
          testid: el.getAttribute('data-testid') || '',
          aria: el.getAttribute('aria-label') || '',
          placeholder: el.getAttribute('placeholder') || '',
          required: el.hasAttribute('required'),
        }
        if (tag === 'select') {
          const opts = []
          for (const o of el.querySelectorAll('option')) {
            const v = (o.getAttribute('value') || o.textContent || '').trim()
            if (v && opts.length < 12) opts.push(v)
          }
          row.optionValues = opts
        }
        out.push(row)
      }
      return out
    }
    // 1. light-DOM <main> or [role="main"]
    const mainEl = document.querySelector('main, [role="main"]')
    if (mainEl) {
      const ins = inputsIn(mainEl)
      if (ins.length >= 2) return { selector: 'main', inputs: ins }
    }
    // 2. shadow host with a form-like cluster
    for (const el of document.querySelectorAll('*')) {
      if (!el.shadowRoot) continue
      const ins = inputsIn(el.shadowRoot)
      if (ins.length >= 2) return { selector: selectorOf(el), inputs: ins }
    }
    // 3. fallback: plain form with the most inputs
    let bestForm = null
    let bestCount = 0
    for (const f of deepQueryAll(document, 'form')) {
      const ins = inputsIn(f)
      if (ins.length > bestCount) { bestForm = f; bestCount = ins.length }
    }
    if (bestForm && bestCount >= 2) {
      return { selector: selectorOf(bestForm), inputs: inputsIn(bestForm) }
    }
    return null
  })

  return {
    url: requestedURL,
    finalURL,
    title,
    h1: h1Texts.map(s => s.trim()).filter(Boolean),
    h2s: h2Texts.map(s => s.trim()).filter(Boolean),
    links,
    images,
    meta,
    hasForm,
    inputs,
    interactions,
    roleAnchors,
    primaryComponent,
    domHTML,
    forms,
  }
}

function canonical(u) {
  try {
    const url = new URL(u)
    url.hash = ''
    url.search = ''
    if (url.pathname !== '/' && url.pathname.endsWith('/')) {
      url.pathname = url.pathname.replace(/\/+$/, '')
    }
    if (url.pathname === '/') url.pathname = ''
    return url.toString()
  } catch {
    return ''
  }
}

function sameOrigin(originURL, candidateURL) {
  try {
    const a = new URL(originURL)
    const b = new URL(candidateURL)
    return a.origin === b.origin
  } catch {
    return false
  }
}

const isAvoided = (path) => /privacy|terms|cookie|legal|sitemap|rss|feed/i.test(path)

async function main() {
  if (process.env.QUAIL_PROBE_SELFTEST === '1') {
    // ponytail: open shadow roots only; closed roots unreachable by spec.
    const browser = await engine.launch({ headless: true })
    const page = await browser.newPage()
    await page.setContent(`<!doctype html><html><body>
      <flex-calc></flex-calc>
      <script>
        class FlexCalc extends HTMLElement {
          constructor() {
            super()
            const sr = this.attachShadow({mode:'open'})
            sr.innerHTML = '<form><input type="number" name="bruto"><input type="number" name="partner"><select name="energielabel"><option>A</option><option>B</option><option>C</option></select><button role="button">Bereken</button></form>'
          }
        }
        customElements.define('flex-calc', FlexCalc)
      </script>
    </body></html>`)
    const p = await collectPage(page, 'about:blank')
    const pc = p.primaryComponent
    const ok =
      p.hasForm &&
      p.inputs.length >= 3 &&
      p.roleAnchors.length >= 1 &&
      pc && pc.inputs.length === 3 &&
      pc.selector.includes('flex-calc') &&
      pc.inputs.some(i => i.tag === 'select' && Array.isArray(i.optionValues) && i.optionValues.length === 3)
    await browser.close()
    if (!ok) {
      console.error('selftest FAILED', JSON.stringify({
        hasForm: p.hasForm, inputs: p.inputs.length, roleAnchors: p.roleAnchors.length,
        primaryComponent: pc,
      }))
      process.exit(1)
    }
    console.log('selftest OK: shadow-DOM pierce found', p.inputs.length, 'inputs,', p.roleAnchors.length, 'role-anchors; primary=', pc.selector, 'with', pc.inputs.length, 'scoped inputs')
    process.exit(0)
  }
  const errors = []
  const seen = new Set()
  const order = []
  const pagesOut = {}
  const queue = [{ url: canonical(TARGET), depth: 0 }]
  const browser = await engine.launch({ headless: true })
  const ctx = await browser.newContext({
    userAgent: 'quail-browser-probe/1 (+https://github.com/spriteCloud/quail)',
  })
  const page = await ctx.newPage()
  page.on('pageerror', (e) => errors.push(`pageerror: ${e}`))
  page.setDefaultNavigationTimeout(NAV_TIMEOUT_MS)

  let originResolved = null

  while (queue.length && Object.keys(pagesOut).length < MAX_PAGES) {
    const { url, depth } = queue.shift()
    if (seen.has(url)) continue
    seen.add(url)
    try {
      await page.goto(url, { waitUntil: 'domcontentloaded', timeout: NAV_TIMEOUT_MS })
      await page.waitForTimeout(IDLE_MS)
      if (!originResolved) {
        originResolved = new URL(page.url()).origin
        await dismissCookieBanner(page)
      }
      await expandPopups(page)
      const p = await collectPage(page, url)
      const key = canonical(p.finalURL) || url
      if (pagesOut[key]) continue
      pagesOut[key] = p
      order.push(key)
      if (depth >= MAX_DEPTH) continue
      for (const l of p.links) {
        if (!l.href) continue
        let abs
        try {
          abs = new URL(l.href, p.finalURL).toString()
        } catch { continue }
        if (!sameOrigin(originResolved, abs)) continue
        const c = canonical(abs)
        if (!c || isAvoided(new URL(c).pathname.toLowerCase())) continue
        if (!seen.has(c)) queue.push({ url: c, depth: depth + 1 })
      }
    } catch (e) {
      errors.push(`fetch ${url}: ${String(e?.message ?? e)}`)
    }
  }

  await ctx.close()
  await browser.close()

  const pages = order.map(k => pagesOut[k])
  process.stdout.write(JSON.stringify({ origin: originResolved ?? canonical(TARGET), pages, errors }))
}

main().catch(e => {
  console.error('quail-browser-probe: fatal:', e)
  process.exit(1)
})
