// Accessibility checks for the built site: see tests/a11y/README.md for what is and is not covered.
//
//   just a11y                       build the site, then run everything
//   npm run test:a11y               run against an existing ./public
//   A11Y_SITE_DIR=dist/staging npm run test:a11y
//
// The site is served from a local static server under the path of baseURL in hugo.yaml (/2027/), so
// the root-relative links Hugo writes resolve as they do in production.
import fs from 'node:fs';
import { defineConfig, devices } from '@playwright/test';

const siteDir = process.env.A11Y_SITE_DIR || 'public';
const basePath = new URL(fs.readFileSync('hugo.yaml', 'utf8').match(/^baseURL:\s*(\S+)/m)[1]).pathname;
const port = process.env.A11Y_PORT || '8765';
const origin = `http://127.0.0.1:${port}`;

// Viewport widths are not device emulation: there is no touch input or mobile user agent here.
const sizes = {
  desktop: { width: 1280, height: 800 },
  tablet: { width: 820, height: 1180 },
  mobile: { width: 390, height: 844 },
  reflow: { width: 320, height: 640 },
};

export default defineConfig({
  testDir: 'tests/a11y',
  timeout: 60_000,
  retries: 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: [['list'], ['json', { outputFile: 'a11y-report/results.json' }]],
  use: { baseURL: origin + basePath },
  projects: ['light', 'dark'].flatMap((colorScheme) =>
    Object.entries(sizes).map(([name, viewport]) => ({
      name: `${colorScheme}-${name}`,
      use: { ...devices['Desktop Chrome'], colorScheme, viewport },
    })),
  ),
  webServer: {
    command: 'node tests/a11y/serve.mjs',
    url: origin + basePath,
    reuseExistingServer: !process.env.CI,
    env: { A11Y_SITE_DIR: siteDir, A11Y_PORT: port, A11Y_BASE_PATH: basePath },
  },
});
