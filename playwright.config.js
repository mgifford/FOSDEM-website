const { defineConfig, devices } = require('@playwright/test');

// BASE_URL points the crawl at any deployment (e.g. the live GitHub Pages URL).
// Without it, ./_site is served locally under the Pages subpath.
const local = 'http://127.0.0.1:8080/FOSDEM-website/';
const baseURL = process.env.BASE_URL || local;

// A11Y_BROWSERS=chromium,firefox,webkit runs every project in each engine (default: chromium).
// Install the others first: npx playwright install firefox webkit
const ENGINES = {
  chromium: devices['Desktop Chrome'],
  firefox: devices['Desktop Firefox'],
  webkit: devices['Desktop Safari'],
};
const browsers = (process.env.A11Y_BROWSERS || 'chromium').split(',').map((b) => b.trim()).filter(Boolean);

const variants = [
  ['light-desktop', 'light', 1280],
  ['dark-desktop', 'dark', 1280],
  ['light-320', 'light', 320],
  ['dark-320', 'dark', 320],
];

module.exports = defineConfig({
  testDir: 'tests',
  timeout: 300_000,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'a11y-report/html' }]],
  use: { baseURL },
  projects: browsers.flatMap((engine) => variants.map(([name, colorScheme, width]) => ({
    // Chromium keeps the bare names so existing reports and --project flags still work.
    name: engine === 'chromium' ? name : `${engine}-${name}`,
    use: { ...ENGINES[engine], colorScheme, viewport: { width, height: 800 } },
  }))),
  webServer: process.env.BASE_URL ? undefined : {
    command: 'node scripts/serve.mjs',
    url: local,
    reuseExistingServer: true,
  },
});
