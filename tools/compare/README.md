# Before/after comparison against upstream

Builds the upstream FOSDEM site (`FOSDEM/website` `main`) and one "after" per change, then serves
them side by side in iframes. Navigation, scrolling and the theme toggle stay in step, so you can
click through the real site and see exactly what a change does. Under each pair is a diff of the
**built** output.

- Live: `https://<user>.github.io/<repo>/compare/` (built by `.github/workflows/pages.yml`)
- Deep links: `.../compare/#variant=pr03-focus-visible&page=full%2Fschedule%2F`

## Run it locally

Needs `go`, `hugo` (extended), `git`, `python3`. Node is only needed to serve.

```sh
tools/compare/build.sh                       # writes _site/compare, needs network for the clone
node tools/compare/serve.mjs          # serves _site/compare on :8093 with caching off
# open http://127.0.0.1:8093/
```

Offline, point it at an existing upstream checkout: `UPSTREAM_DIR=/path/to/upstream tools/compare/build.sh`.
For a hosted copy set `PUBLIC_BASE` to the URL prefix it will be served under (no trailing slash).

After a rebuild, reload the page. `serve.mjs` sends `Cache-Control: no-store`; a plain
`python3 -m http.server` does not, and the browser may keep showing the previous diff and CSS
until you hard-refresh.

## Add or change a comparison

Edit `variants.json`: one entry per change, with the git `ref` (a branch or commit in this repo),
a label, a description, and what it depends on. The build resolves the ref locally or as
`origin/<ref>`.

## How the diff stays honest

- Every tree is built from the same test data (`testdata/schedule.json`), generated **once** from
  upstream. The schedule generator's output is not byte-stable (event order on speaker pages
  changes between runs), so generating per tree would show phantom differences. A change to
  `main.go` itself would therefore not show here.
- Each tree is built a second time with a neutral base URL, and only those builds are diffed, so
  base URLs cannot appear as differences.
- The tests-only variant is a control: it must report **0** differing files.

## Limits

- Frames are same-origin only because everything is served from one host.
- The page cannot emulate the OS colour scheme. To see OS-dependent behaviour (for example the
  dark-mode fix in PR 2), switch your operating system to dark.
- It compares built HTML/CSS/JS. It is not a substitute for testing in a browser with assistive
  technology.
