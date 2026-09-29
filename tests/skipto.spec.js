const { test, expect } = require('@playwright/test');

// Real keyboard events only: Tab to reveal the SkipTo button, Enter to open the menu,
// arrow to a target and Enter to activate it, then check where focus landed.
const PAGES = ['', 'news/', 'full/', 'full/schedule/'];

for (const path of PAGES) {
  test(`SkipTo is the first Tab stop and moves focus: /${path}`, async ({ page }) => {
    await page.goto(path, { waitUntil: 'networkidle' });

    await page.keyboard.press('Tab');
    const button = page.locator('button[aria-haspopup="true"]').first();
    await expect(button, 'first Tab stop should be the SkipTo button').toBeFocused();
    await expect(button).toBeVisible();

    await page.keyboard.press('Enter');
    const items = page.locator('[role="menuitem"]');
    await expect(items.first()).toBeVisible();

    // Choose the item whose label mentions "main" (landmark or its heading), never a pre-counted key sequence.
    const target = items.filter({ hasText: /main/i }).first();
    const count = await items.count();
    let found = false;
    for (let i = 0; i < count; i++) {
      if (await target.evaluate((el) => el === document.activeElement)) { found = true; break; }
      await page.keyboard.press('ArrowDown');
    }
    expect(found, 'could not reach the "main" menu item by keyboard').toBe(true);
    await page.keyboard.press('Enter');

    const landed = await page.evaluate(() => {
      const a = document.activeElement;
      return a && a.closest('main') ? 'main' : (a ? a.tagName : 'none');
    });
    expect(landed, 'focus should move into <main>').toBe('main');
  });
}
