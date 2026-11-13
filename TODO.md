# TODO

## Propose to FOSDEM: scan pages with axe and Playwright in CI

FOSDEM's pipeline (`.woodpecker.yml`) builds every pull request, deploys a staging preview and
comments the link, but nothing checks the result for accessibility errors. Contrast and link
problems only get found by hand, after they ship. Proposal for `FOSDEM/website`:

- [ ] Add a `a11y` step to the `pull_request` pipeline, after `build`: serve the built site under
      `/2027/` and run axe-core through Playwright over every reachable page.
      Alternative: run it against the staging preview (`https://staging.fosdem.org/pr-N/`) once
      `deploy-preview` has finished.
- [ ] Crawl internal links from the home page rather than a fixed URL list, so new news posts and
      pages are covered automatically. Fail the step on any violation; do not whitelist silently.
- [ ] Run each page in light and dark theme and at 1280px and 320px. The theme has two inputs
      (OS preference and the site's own toggle), and 320px catches reflow problems.
- [ ] Use the WCAG 2.2 A and AA axe tags and keep the report as a CI artifact (JSON, with rule,
      page, selector and colours for contrast failures).
- [ ] Provide Node and the Playwright browser through `flake.nix` (`nix develop`), so the step runs
      the same way locally and in CI. Pin versions.
- [ ] Add a `just a11y` recipe so contributors can run the same check before pushing.
- [ ] Document in `README.md` that a clean run is not a conformance claim: axe only finds what a
      machine can decide, so keyboard and screen reader checks stay manual.

Why it is worth doing: a scan of upstream `de2d05a` found `color-contrast` on all 24 pages (294
nodes in light theme, 65 in dark) and `link-in-text-block` on 17 pages in dark theme. With the four
changes in `tools/compare/variants.json` applied, the same scan reports zero violations in all four
configurations (light/dark by 1280px/320px). So the check passes on a fixed site and can be a hard
gate.

Not done here: the scan script was run ad hoc and is not in the repository yet, and the comparison
lab (`tools/compare/`) does not run it. Porting the old Playwright suite (keyboard, SkipTo,
Guidepup screen reader checks) is a separate item.

## Comparison lab

- [ ] Run axe on every variant's before and after in `tools/compare/build.sh` and show the rule IDs
      and affected elements per change in the viewer (a lower score does not prove the intended fix).
- [ ] Record forced-colours, reduced-motion and 320px screenshots per change.
- [ ] Show the pull request link for each change once the changes are offered upstream (`pr` in
      `variants.json`).
- [ ] Promote `lab-2027` to `main` and set it as the default branch (see the notes in
      `tools/compare/README.md`).
