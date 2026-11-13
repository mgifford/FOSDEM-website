// Proves the scan can fail. A check that matches nothing passes without testing anything, so these run
// the same axe configuration against a known-bad page and test the baseline logic.
import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';
import { TAGS, fingerprints, newFindings } from './fingerprint.mjs';

test.describe('scanner self-test', () => {
  test.beforeEach(({}, testInfo) => {
    test.skip(testInfo.project.name !== 'light-desktop', 'runs once, not per project');
  });

  test('reports contrast, a missing name and a missing alt text on a known-bad fixture', async ({ page }) => {
    await page.setContent(`<!doctype html><html lang="en"><head><title>Fixture</title></head><body><main>
      <h1>Fixture</h1>
      <p style="color:#bbb;background:#fff">Low contrast text</p>
      <button></button>
      <img src="data:image/gif;base64,R0lGODlhAQABAAAAACw=">
    </main></body></html>`);
    const found = fingerprints(await new AxeBuilder({ page }).withTags(TAGS).analyze()).map((f) => f.split(' | ')[0]);
    expect(found).toEqual(expect.arrayContaining(['color-contrast', 'button-name', 'image-alt']));
  });

  test('the baseline only hides what it lists', () => {
    const known = ['color-contrast | .a'];
    expect(newFindings(['color-contrast | .a'], known)).toEqual([]);
    expect(newFindings(['color-contrast | .b', 'color-contrast | .a'], known)).toEqual(['color-contrast | .b']);
    expect(newFindings(['link-name | .a'], undefined)).toEqual(['link-name | .a']);
  });
});
