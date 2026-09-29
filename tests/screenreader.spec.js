const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const path = require('node:path');

// Emulated screen reader (@guidepup/virtual-screen-reader) run inside the real page.
// It evidences the spoken names, roles and reading order the accessibility tree yields.
// It is a simulation, not VoiceOver/NVDA output, and never proves keyboard operability.
const BUNDLE = path.join(__dirname, '.build', 'vsr.js');
const PAGES = ['', 'news/', 'news/call-for-stands/', 'full/', 'full/schedule/'];
const MAX_STEPS = 600;

async function readPage(page) {
  await page.addScriptTag({ path: BUNDLE });
  return page.evaluate(async (max) => {
    const { virtual } = window.GuidepupVSR;
    await virtual.start({ container: document.body });
    for (let i = 0; i < max; i++) {
      const before = await virtual.lastSpokenPhrase();
      await virtual.next();
      if ((await virtual.lastSpokenPhrase()) === before && i > 0) break;
    }
    const log = await virtual.spokenPhraseLog();
    await virtual.stop();
    return log;
  }, MAX_STEPS);
}

for (const p of PAGES) {
  test(`virtual screen reader reads /${p} sensibly`, async ({ page }, testInfo) => {
    await page.goto(p, { waitUntil: 'networkidle' });
    const log = await readPage(page);

    fs.mkdirSync('a11y-report', { recursive: true });
    const name = `sr-${testInfo.project.name}-${p.replace(/\W+/g, '_') || 'home'}.txt`;
    fs.writeFileSync(path.join('a11y-report', name), log.join('\n'));

    expect(log.length, 'screen reader read nothing').toBeGreaterThan(5);
    expect(log.some((l) => /main/i.test(l)), 'no main landmark spoken').toBe(true);
    expect(log.some((l) => /heading/i.test(l) && /level 1/i.test(l)), 'no level-1 heading spoken').toBe(true);
    // An image with no accessible name is spoken as a bare "image" (or "graphic").
    const bare = log.filter((l) => /^(image|graphic|img)$/i.test(l.trim()));
    expect(bare, 'images spoken without an accessible name').toEqual([]);
    // Links and buttons must never be spoken without a name.
    const nameless = log.filter((l) => /^(link|button)$/i.test(l.trim()));
    expect(nameless, 'links/buttons spoken without a name').toEqual([]);
  });
}
