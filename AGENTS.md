# AGENTS.md

Guidance for AI coding agents working in this repository. Human docs: [README.md](README.md) (the
FOSDEM site) and [tools/compare/README.md](tools/compare/README.md) (the comparison lab).

## What this is

A fork of the FOSDEM website (`https://git.fosdem.org/FOSDEM/website`, Hugo, the **2027** edition,
built with `hugo build -b https://fosdem.org/2027/`) used to propose accessibility changes and show
them before/after on GitHub Pages (`https://mgifford.github.io/FOSDEM-website/`).

Branch roles:

| Branch | Contents |
|--------|----------|
| `main` | upstream `main` **plus the lab only**: `tools/compare/`, `scripts/start-work.sh`, `.github/workflows/`, `AGENTS.md`. No accessibility fixes. |
| `2027/NN-name` | One change, branched from `upstream/main` itself. Must hold only the change, so it can be offered upstream. |

Never deploy to or push toward upstream FOSDEM infrastructure (`just deploy-*` rsyncs to FOSDEM hosts;
do not run them).

## Before starting any improvement: sync with upstream

FOSDEM changes this site often. **Always run this first, then work on the branch it creates:**

```sh
scripts/start-work.sh <short-name>      # fetch upstream, merge it into main, branch 2027/<short-name> from upstream/main
```

If the merge into `main` conflicts the script stops cleanly. Resolve that before doing anything else.
Never base a change branch on `main` (it would carry the lab files).

## Commands

```sh
hugo build -b http://127.0.0.1:8094/2027/ -D -d _site   # needs data/pretalx, see below
python3 scripts/fetch-pretalx.py fosdem-2027             # writes data/pretalx (gitignored); `just fetch` does the same
tools/compare/build.sh                                   # every comparison into _site/compare
python3 tools/compare/validate.py                        # check _site is complete
node tools/compare/serve.mjs                             # http://127.0.0.1:8093/
```

`data/pretalx/` is gitignored and **every build needs it**, otherwise Hugo fails with
`can't iterate over <nil>`. The comparison build fetches it once and shares it between all trees.
Prefer `nix develop` for `hugo`, `just`, `fetch-pretalx`; without Nix, Hugo extended 0.155+.

## Rules that are easy to break

1. **Always build with a subpath baseURL** (`/2027/`, `/FOSDEM-website/compare/after/<slug>/`). Use
   `relURL`/`absURL`; never hard-code root-relative paths. Works at `/` but not under a subpath is a bug.
2. **Dark mode has two independent inputs**: the OS `prefers-color-scheme` media query and the site's own
   toggle (`localStorage`, `data-theme`). Fixes must work for both; an explicit site choice beats the OS.
   (Upstream already has `:root:not([data-theme='light'])` in `assets/css/010-tokens.css`.)
3. **Do not weaken accessibility**: no loosening axe rules, removing tests or hiding violations. A clean
   automated run is not a conformance claim.
4. **Keep change branches mergeable.** Different branches must not edit the same lines, or the `all`
   variant fails to build. Add new tokens/rules in distinct places (not right after `--c-link`, not right
   after the base `a {}` rule) and check with `tools/compare/build.sh`.
5. CSS is in `assets/css/NNN-*.css`, concatenated and fingerprinted by `layouts/baseof.html`. A change to
   CSS therefore touches every built page's `<link>`; that is expected in the diff.
6. Third-party code: vendor under `static/`, add the licence file next to it and a row in
   `THIRD-PARTY.md`. Do not hand-edit `static/js/skipto.min.js` or `static/js/leaflet.js`.
7. Never commit `_site/`, `dist/`, `public/`, `resources/`, `data/pretalx/`.
8. Test keyboard behaviour with real key presses in each page; mirrored events do not count.

## Comparison lab notes

- A variant's "before" is `git merge-base <branch> upstream/main`, never GitHub's `baseRefOid`, which is
  the base branch tip and shows unrelated changes.
- Published comparisons must resolve to full commit SHAs and record upstream commit, Pretalx data hash and
  Hugo version (`variants.json`). Report missing, conflicting or unbuilt variants as failures; never drop one.
- Old links such as `#variant=pr03-focus-visible` must keep working: keep `aliases` in `variants.json`.
- Not ported yet from the 2026 fork: the Playwright/axe/Guidepup suite, Lighthouse, forced-colours and
  reduced-motion scenarios. See `fosdem_compare_handoff/` if present on your branch for the phased plan.

## Conventions

- One change per branch, small commits; match existing style; comment only where non-obvious.
- State what you ran and what you did not (build, browsers, screen readers).
