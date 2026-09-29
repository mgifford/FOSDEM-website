const { defineConfig } = require('@playwright/test');

// BASE_URL points the crawl at any deployment (e.g. the live GitHub Pages URL).
// Without it, ./_site is served locally under the Pages subpath.
const local = 'http://127.0.0.1:8080/FOSDEM-website/';
const baseURL = process.env.BASE_URL || local;

module.exports = defineConfig({
  testDir: 'tests',
  timeout: 300_000,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'a11y-report/html' }]],
  use: { baseURL },
  projects: [
    { name: 'light-desktop', use: { colorScheme: 'light', viewport: { width: 1280, height: 800 } } },
    { name: 'dark-desktop', use: { colorScheme: 'dark', viewport: { width: 1280, height: 800 } } },
    { name: 'light-320', use: { colorScheme: 'light', viewport: { width: 320, height: 800 } } },
    { name: 'dark-320', use: { colorScheme: 'dark', viewport: { width: 320, height: 800 } } },
  ],
  webServer: process.env.BASE_URL ? undefined : {
    command: 'node scripts/serve.mjs',
    url: local,
    reuseExistingServer: true,
  },
});
