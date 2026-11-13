// axe-core over every page of the built site, in light and dark, at four viewport widths (see the config).
// It fails on findings that are not in baseline.json, so new regressions are blocked while the ones that
// already exist are fixed. A clean run means axe found no violations in this scope; it is not conformance.
import fs from 'node:fs';
import path from 'node:path';
import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';
import { TAGS, fingerprints, listPages, loadBaseline, newFindings } from './fingerprint.mjs';

const siteDir = process.env.A11Y_SITE_DIR || 'public';
const pages = listPages(siteDir);
const baseline = loadBaseline();
const updating = Boolean(process.env.A11Y_UPDATE_BASELINE);
const ignoreBaseline = Boolean(process.env.A11Y_NO_BASELINE);

// Third-party requests (map tiles) are blocked so the result does not depend on other servers.
test.beforeEach(async ({ context, baseURL }) => {
  const host = new URL(baseURL).host;
  await context.route('**/*', (route) => (new URL(route.request().url()).host === host ? route.continue() : route.abort()));
});

async function scan(page, key, label, testInfo) {
  await page.waitForLoadState('networkidle');
  const results = await new AxeBuilder({ page }).withTags(TAGS).analyze();
  const found = fingerprints(results);

  if (updating) {
    const dir = path.join('a11y-report', 'found');
    fs.mkdirSync(dir, { recursive: true });
    const name = `${testInfo.project.name}-${label}-${key.replace(/[^a-z0-9]+/gi, '_')}.json`;
    fs.writeFileSync(path.join(dir, name), JSON.stringify({ page: key, found }));
    return;
  }

  await testInfo.attach('axe-violations.json', { body: JSON.stringify(results.violations, null, 2), contentType: 'application/json' });
  const fresh = ignoreBaseline ? found : newFindings(found, baseline[key]);
  expect(
    fresh,
    `${fresh.length} new accessibility finding(s) on ${key} (${testInfo.project.name}). Fix them, or if one is a ` +
      'reviewed exception add it to tests/a11y/baseline.json with a reason (see tests/a11y/README.md).',
  ).toEqual([]);
}

for (const rel of pages) {
  test(`axe: /${rel}`, async ({ page }, testInfo) => {
    await page.goto(rel);
    await scan(page, `/${rel}`, 'default', testInfo);
  });
}

// The site has its own theme toggle (localStorage) as well as the OS preference, and they can disagree.
test('axe: / with the site theme set opposite to the OS preference', async ({ page }, testInfo) => {
  const opposite = testInfo.project.use.colorScheme === 'dark' ? 'light' : 'dark';
  await page.addInitScript((theme) => localStorage.setItem('theme', theme), opposite);
  await page.goto('');
  await scan(page, '/', `site-theme-${opposite}`, testInfo);
});
