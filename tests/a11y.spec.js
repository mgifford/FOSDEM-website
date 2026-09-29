const { test, expect } = require('@playwright/test');
const AxeBuilder = require('@axe-core/playwright').default;
const fs = require('node:fs');

// Detector, not verdict: axe-core covers only the machine-decidable subset of WCAG.
// A clean run means axe found nothing on these pages/viewports/themes, not conformance.
const TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];
const SEEDS = ['', 'full/']; // landing site at the root, full site under /full/

function normalize(href, base) {
  const u = new URL(href, base);
  u.hash = '';
  if (u.pathname.length > 1 && u.pathname.endsWith('/')) u.pathname = u.pathname.slice(0, -1);
  return u;
}

async function crawl(page, baseURL) {
  const root = new URL(baseURL);
  const seen = new Set();
  const queue = SEEDS.map((s) => new URL(s, root).href);
  const pages = [];
  while (queue.length) {
    const url = queue.shift();
    if (seen.has(url)) continue;
    seen.add(url);
    const res = await page.goto(url, { waitUntil: 'networkidle' });
    const type = res && res.headers()['content-type'];
    if (!res || res.status() >= 400 || !type || !type.includes('text/html')) continue;
    pages.push(url);
    const hrefs = await page.$$eval('a[href]', (as) => as.map((a) => a.getAttribute('href')));
    for (const h of hrefs) {
      let u;
      try { u = normalize(h, url); } catch { continue; }
      // stay on this origin and under the site's base path; the landing crawl must not stray into /full/
      if (u.origin !== root.origin || !u.pathname.startsWith(root.pathname)) continue;
      if (/\.(xml|ics|json|png|jpe?g|svg|webp|pdf|css|js)$/i.test(u.pathname)) continue;
      if (!seen.has(u.href)) queue.push(u.href);
    }
  }
  return pages;
}

// Theme via the OS preference (project colorScheme) and via the manual toggle, which stores
// 'theme' in localStorage and overrides the OS preference. The toggle runs the opposite theme.
const MODES = [{ name: 'os-preference', manual: false }, { name: 'manual-toggle', manual: true }];

for (const mode of MODES) test(`axe-core crawl (${mode.name})`, async ({ page, baseURL, colorScheme }, testInfo) => {
  if (mode.manual) {
    await page.addInitScript((t) => { try { localStorage.setItem('theme', t); } catch {} }, colorScheme === 'dark' ? 'light' : 'dark');
  }
  const pages = await crawl(page, baseURL);
  expect(pages.length, 'crawl found no pages').toBeGreaterThan(0);

  const results = [];
  for (const url of pages) {
    await page.goto(url, { waitUntil: 'networkidle' });
    const r = await new AxeBuilder({ page }).withTags(TAGS).analyze();
    results.push({
      url,
      violations: r.violations.map((v) => ({
        id: v.id, impact: v.impact, help: v.help, helpUrl: v.helpUrl,
        nodes: v.nodes.map((n) => ({ target: n.target, html: n.html, summary: n.failureSummary })),
      })),
      incomplete: r.incomplete.map((v) => ({ id: v.id, nodes: v.nodes.length })),
    });
  }

  fs.mkdirSync('a11y-report', { recursive: true });
  const out = `a11y-report/${testInfo.project.name}-${mode.name}.json`;
  fs.writeFileSync(out, JSON.stringify({ axeTags: TAGS, results }, null, 2));

  const byRule = {};
  for (const p of results) for (const v of p.violations) {
    byRule[v.id] = byRule[v.id] || { impact: v.impact, pages: 0, nodes: 0 };
    byRule[v.id].pages++;
    byRule[v.id].nodes += v.nodes.length;
  }
  console.log(`[${testInfo.project.name} ${mode.name}] pages: ${pages.length}, rules failing: ${Object.keys(byRule).length}`);
  console.table(byRule);
  const failing = results.filter((p) => p.violations.length).map((p) => p.url);
  expect(failing, `pages with axe violations (details in ${out})`).toEqual([]);
});
