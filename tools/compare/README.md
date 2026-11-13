# Before/after comparison of the 2027 site

Builds the upstream FOSDEM site and one "after" per proposed change, then serves them side by side in
iframes. Navigation, scrolling and the theme toggle stay in step, so you can click through the real
site and see exactly what a change does. Under each pair is a diff of the **built** output.

- Live: `https://<user>.github.io/<repo>/compare/` (built by `.github/workflows/pages.yml`)
- Each change is listed on that page and has a stable link:
  `.../compare/#variant=focus-visible&page=`. Older `pr03-focus-visible` style links still work
  (`aliases` in `variants.json`).

## How a comparison is built

- Every change is a branch (`2027/NN-name`) made **from `FOSDEM/website` main**, so it contains only
  the change. Its "before" is the commit it forked from (`git merge-base <branch> upstream/main`),
  so the diff is that change and nothing else, even after upstream moves on.
- The `all` entry has `"merge": [...]`: upstream's tip with those branches merged in. If they stop
  fitting together, the build fails, which is the signal to rebase one.
- All trees use the same Pretalx data, fetched once with `scripts/fetch-pretalx.py`. Its hash and
  fetch time are shown in the page footer and in `variants.json`.
- Each tree is built a second time with a neutral base URL; only those builds are diffed, so base
  URLs cannot appear as differences. Pages link to a fingerprinted stylesheet, so a CSS change
  touches every page's `<link>`.
- **Nothing is dropped silently.** A branch that is missing, does not merge or does not build fails
  the run, and `validate.py` re-checks the output before Pages publishes it.

## Run it locally

Needs `git`, `hugo` (extended), `python3` and network access to `git.fosdem.org` and
`pretalx.fosdem.org`. Node is only needed to serve.

```sh
tools/compare/build.sh                 # writes _site/compare
python3 tools/compare/validate.py      # checks it is complete
node tools/compare/serve.mjs           # http://127.0.0.1:8093/, caching off
```

`NO_FETCH=1` skips fetching upstream; `UPSTREAM_REF` and `EVENT` override the upstream ref and the
Pretalx slug (default `fosdem-2027`). For a hosted copy set `PUBLIC_BASE` to the URL prefix (no
trailing slash).

## Add a change

```sh
scripts/start-work.sh my-change        # merges upstream into main, branches 2027/my-change from upstream
```

Commit the change on that branch, then add an entry to `variants.json` (`slug`, `ref`, `label`,
`description`, optionally `pr` with the pull request URL, and add it to the `all` merge list).
Push the branch: CI resolves refs as `origin/<ref>`.

## Keeping up with upstream

`scripts/start-work.sh` fetches `https://git.fosdem.org/FOSDEM/website` as `upstream` and merges it
into `main` before every new change. `.github/workflows/sync-upstream.yml` does the same daily and
opens a pull request (or an issue if the merge conflicts).

## Limits

- Frames are same-origin, because everything is served from one host, so the theme setting
  (`localStorage`) is shared by both sides and all previews.
- *Simulate a dark-mode OS* rewrites the `prefers-color-scheme: dark` media query in both frames. It
  is CSS only; the pages' scripts still read your real setting. It does not change your OS.
- Mirrored scroll and navigation are not keyboard testing: press Tab in each frame yourself.
- It compares built HTML/CSS/JS. It is not a conformance claim and does not replace testing with a
  keyboard and assistive technology.
- The old Playwright/axe/Guidepup suite from the 2026 fork is not ported to this site yet.
