// SPDX-License-Identifier: MIT
// Automated checks that show a change is present. Each check runs against a "before" and an "after"
// site in Chromium and returns { pass, summary, details }. By design a check FAILS on upstream
// (so it proves it can detect the problem) and PASSES once the change is applied.
//
// These are narrow, mechanical checks of specific properties. They are not a WCAG conformance
// test: axe only finds what a machine can decide, and passing here does not replace testing with
// a keyboard and assistive technology.
import AxeBuilder from '@axe-core/playwright';

export const SCHEMES = ['light', 'dark'];
export const PAGES = ['', 'news/', 'faq/', 'calendar/', 'about/', 'practical/', 'support/'];

// --- colour helpers (opaque colours only: computed styles on this site have no alpha) ---
const parse = (c) => (c.match(/[\d.]+/g) || []).slice(0, 3).map(Number);
const lin = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; };
const lum = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
export const ratio = (a, b) => {
  const [x, y] = [lum(parse(a)), lum(parse(b))].sort((p, q) => q - p);
  return (x + 0.05) / (y + 0.05);
};

async function open(browser, url, colorScheme) {
  const context = await browser.newContext({ colorScheme, viewport: { width: 1280, height: 800 } });
  const page = await context.newPage();
  const response = await page.goto(url);
  if (!response || !response.ok()) throw new Error(`${url} returned ${response && response.status()}`);
  return { page, close: () => context.close() };
}
const pageBackground = (page) => page.evaluate(() => {
  const own = (e) => getComputedStyle(e).backgroundColor;
  const body = own(document.body);
  return body === 'rgba(0, 0, 0, 0)' ? own(document.documentElement) : body;
});
const done = (failures, passMsg) => ({
  pass: failures.length === 0,
  summary: failures.length === 0 ? passMsg : `${failures.length} problem${failures.length === 1 ? '' : 's'}`,
  details: failures,
});

export const checks = {
  // axe colour-contrast and link-in-text-block (links told apart from text by more than colour).
  'contrast-links': {
    title: 'axe: no colour-contrast or link-in-text-block violations',
    async run(browser, base) {
      const failures = [];
      for (const scheme of SCHEMES) {
        const { page, close } = await open(browser, base, scheme);
        for (const path of PAGES) {
          await page.goto(base + path);
          const result = await new AxeBuilder({ page }).withRules(['color-contrast', 'link-in-text-block']).analyze();
          for (const v of result.violations) {
            failures.push(`${scheme} /${path}: ${v.id} on ${v.nodes.length} element${v.nodes.length === 1 ? '' : 's'}`);
          }
        }
        await close();
      }
      return done(failures, `0 violations on ${PAGES.length} pages in light and dark`);
    },
  },

  // Real Tab key presses: every focus indicator must be at least 2px wide and 3:1 against the page.
  // SkipTo draws its own focus styling (outline: 0 plus a background change), so its widget is
  // skipped here and covered by the "skipto" check. It is reported in the details, not hidden.
  'focus-visible': {
    title: "Keyboard focus indicator on the site's own controls is at least 2px wide and 3:1",
    async run(browser, base) {
      const failures = [];
      for (const scheme of SCHEMES) {
        for (const path of ['', 'faq/']) {
          const { page, close } = await open(browser, base + path, scheme);
          const bg = await pageBackground(page);
          let counted = 0;
          for (let press = 1; press <= 14 && counted < 8; press++) {
            await page.keyboard.press('Tab');
            const f = await page.evaluate(() => {
              const e = document.activeElement, s = getComputedStyle(e);
              return { skipto: !!e.closest('.skip-to'), name: (e.textContent || e.tagName).trim().slice(0, 24), style: s.outlineStyle, width: parseFloat(s.outlineWidth), color: s.outlineColor };
            });
            if (f.skipto) continue;
            counted++;
            const c = ratio(f.color, bg);
            if (f.style === 'none' || f.width < 2 || c < 3) {
              failures.push(`${scheme} /${path} stop ${counted} "${f.name}": outline ${f.style} ${f.width}px, ${c.toFixed(2)}:1`);
            }
          }
          if (counted < 8) failures.push(`${scheme} /${path}: only ${counted} focusable elements reached`);
          await close();
        }
      }
      return done(failures, "8 Tab stops on 2 pages in light and dark all have a 2px+, 3:1 indicator (SkipTo's own widget excluded)");
    },
  },

  // The first Tab stop is a SkipTo button; Enter opens a menu of landmarks and headings; Escape closes it.
  skipto: {
    title: 'First Tab stop is a skip-link menu that lists landmarks and headings',
    async run(browser, base) {
      const failures = [];
      const { page, close } = await open(browser, base + 'faq/', 'light');
      await page.keyboard.press('Tab');
      const first = await page.evaluate(() => {
        const e = document.activeElement;
        return { tag: e.tagName, name: (e.textContent || '').trim() };
      });
      if (first.tag !== 'BUTTON' || !/skip to/i.test(first.name)) {
        failures.push(`first Tab stop is ${first.tag} "${first.name.slice(0, 30)}", not a "Skip to" button`);
      } else {
        await page.keyboard.press('Enter');
        const menu = page.locator('[role="menu"]');
        const items = await menu.locator('[role="menuitem"]').count().catch(() => 0);
        if (!(await menu.isVisible().catch(() => false))) failures.push('Enter did not open a menu');
        else if (items < 3) failures.push(`menu has only ${items} items`);
        await page.keyboard.press('Escape');
        if (await menu.isVisible().catch(() => false)) failures.push('Escape did not close the menu');
      }
      await close();
      return done(failures, 'Tab reaches the skip-link button, Enter opens its menu, Escape closes it');
    },
  },

  // Hover must change the colour and thicken the underline; a :visited colour must exist and be readable.
  // getComputedStyle hides :visited by design, so that part reads the declared custom property instead.
  'link-states': {
    title: 'Text links have a hover state and a readable visited colour (4.5:1)',
    async run(browser, base) {
      const failures = [];
      for (const scheme of SCHEMES) {
        const { page, close } = await open(browser, base + 'faq/', scheme);
        const bg = await pageBackground(page);
        const link = page.locator('footer .license a').first();
        const style = () => link.evaluate((e) => { const s = getComputedStyle(e); return { color: s.color, thickness: parseFloat(s.textDecorationThickness) || 0, line: s.textDecorationLine }; });
        const rest = await style();
        await link.hover();
        const hover = await style();
        if (hover.color === rest.color) failures.push(`${scheme}: hover does not change the link colour (${rest.color})`);
        else if (ratio(hover.color, bg) < 4.5) failures.push(`${scheme}: hover colour ${hover.color} is ${ratio(hover.color, bg).toFixed(2)}:1`);
        if (!hover.line.includes('underline') || hover.thickness < 2) failures.push(`${scheme}: hover underline is ${hover.line} ${hover.thickness}px, expected underline 2px`);
        const visited = await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--c-link-visited').trim());
        if (!visited) failures.push(`${scheme}: no visited link colour is defined`);
        else {
          const probe = await page.evaluate((v) => { const d = document.createElement('i'); d.style.color = v; document.body.append(d); const c = getComputedStyle(d).color; d.remove(); return c; }, visited);
          if (ratio(probe, bg) < 4.5) failures.push(`${scheme}: visited colour ${probe} is ${ratio(probe, bg).toFixed(2)}:1`);
        }
        await close();
      }
      return done(failures, 'hover changes colour and underline; visited colour is 4.5:1 or better, light and dark');
    },
  },

  // OS in dark mode, but the visitor chose Light with the site's own toggle (localStorage): the page must be light.
  'light-choice': {
    title: 'An explicit Light choice wins over a dark OS preference',
    async run(browser, base) {
      const failures = [];
      for (const path of ['', 'news/']) {
        const context = await browser.newContext({ colorScheme: 'dark', viewport: { width: 1280, height: 800 } });
        await context.addInitScript(() => { try { localStorage.setItem('theme', 'light'); } catch {} });
        const page = await context.newPage();
        const response = await page.goto(base + path);
        if (!response || !response.ok()) throw new Error(`${base + path} returned ${response && response.status()}`);
        const bg = await pageBackground(page);
        const text = await page.evaluate(() => getComputedStyle(document.body).color);
        if (lum(parse(bg)) < 0.5) failures.push(`/${path}: background ${bg} and text ${text} are dark although Light was chosen`);
        await context.close();
      }
      return done(failures, 'with a dark OS and Light chosen, the background is light');
    },
  },
};
