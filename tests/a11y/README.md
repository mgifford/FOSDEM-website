# Accessibility checks

axe-core, driven by Playwright, over the built site. This page is the contract for the check: what it covers,
what it does not, how it is run and what to do when it fails.

A passing run means axe-core found no violations in the scope below. It is not a statement that the site
conforms to WCAG. Automated rules find only some barriers, so keyboard and screen reader testing by a person is
still needed.

## Run it

```sh
just a11y                      # fetch data, build, install dependencies and Chromium, run everything
npm run test:a11y              # run against an existing ./public
A11Y_SITE_DIR=dist/staging npm run test:a11y
A11Y_NO_BASELINE=1 npm run test:a11y   # ignore the baseline and show every finding
```

`npm ci` installs the exact versions in `package-lock.json`. `npx playwright install chromium` downloads the
matching browser; on NixOS use a Playwright image or package instead.

## The contract

| | |
|---|---|
| Pages | Every HTML page in the built site (the list is read from the build output, so new pages are included automatically). Hugo redirect pages are skipped. |
| States | Page as loaded. The home page is also scanned with the site's own theme toggle set opposite to the OS preference, because the two can disagree. |
| Conditions | Light and dark colour scheme, each at 1280 px (desktop), 820 px (tablet), 390 px (mobile) and 320 px (reflow). |
| Rules | axe-core `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa`. |
| Tools | `@playwright/test` 1.65.0, `@axe-core/playwright` 4.11.3 (axe-core 4.11.4), Chromium from Playwright. The CI image is pinned by digest to the same Playwright version. |
| Config | `playwright.config.js`, `tests/a11y/*`. |
| Result | Fails on any finding that is not in `tests/a11y/baseline.json`. Whether a failed check blocks merging is set by the forge's branch protection, not by this repository. |
| Evidence | The failing page, configuration and selector in the test output; `a11y-report/results.json`; the full axe violations of each test attached under `test-results/`. CI does not keep these files after the run. |
| Owner | See `owner` in `baseline.json` for the existing findings. New findings belong to the pull request that introduced them. |

## What this does not cover

- Keyboard behaviour: tab order, focus visibility, skip links, menus and dialogs. Axe reads markup and styles
  and does not press keys.
- Forced colours, reduced motion, increased contrast, zoom and text spacing.
- Browsers other than Chromium. Safari and Firefox can differ.
- The narrow widths are viewport sizes, not device emulation: there is no touch input or mobile user agent.
- Third-party content. Requests to other hosts (the map tiles) are blocked so the result does not depend on them.
- Whether text alternatives, headings and link names are meaningful. Axe can only tell that they exist.
- Assistive technology. Nothing here shows what a screen reader says.

## The baseline

`tests/a11y/baseline.json` lists findings that already exist on `main`, as `rule | selector` per page. The check
fails only on findings that are not listed, so new regressions are blocked while the existing ones are fixed.
It is a temporary inventory. Listing a finding does not make it acceptable.

- Fix the problem if you can. Then run `npm run a11y:baseline` and the entry disappears.
- For a reviewed exception, keep the entry and say why in the pull request. Every entry is a
  rule and a selector on a page, so the exception stays narrow.
- `_meta` records who owns the list, when it was generated, the axe-core version, and when to review it
  (`review_by`).

When a fix lands, regenerate the baseline and review the diff: it should only shrink.

## Proving that it can fail

`tests/a11y/self-test.spec.js` runs the same axe configuration against a page that is known to be bad and
checks that low contrast, a button without a name and an image without alt text are reported. A scan that matches
nothing passes without testing anything, so keep these tests. Do the same when changing the rules or selectors.

## Cost

Measured: about one minute locally, and 1.8 minutes in the pinned CI image with two workers (216 test cases, 202 run
and 14 are skipped by design), plus `npm ci`. To reduce it, drop widths or the site theme toggle test in
`playwright.config.js` before dropping pages.
