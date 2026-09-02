# fosdem.org

Hugo static site for fosdem.org. Schedule data is preprocessed with a Go program that
reads the Pretalx JSON export and writes derived data files for Hugo to consume.

## Prerequisites

All tools are provided by the Nix dev shell:

```sh
nix develop
```

This gives you `go`, `hugo` and `pagefind`.

Without Nix, install these manually: Go 1.25+, Hugo 0.155+, and Pagefind 1.4+.

## Data Pipeline

The schedule data flows like this:

```
schedule.json  ──>  go run main.go  ──>  data/*.json  ──>  Hugo templates
(Pretalx export)                         (derived data)
```

`schedule.json` is the raw Pretalx schedule export (~5 MB). It is gitignored — download
it from [Pretalx](https://pretalx.fosdem.org/fosdem-2026/schedule/export/schedule_fosdem.json)
and place it in the project root. Note: you must be logged in, otherwise you will get a 404.

`main.go` reads this file at startup and writes the following to `data/`:

| File              | Contents                                    |
|-------------------|---------------------------------------------|
| `schedule.json`   | Full parsed conference data                 |
| `events.json`     | Map of event slug to event(s)               |
| `speakers.json`   | Map of speaker GUID to speaker + event list |
| `tracks.json`     | Map of track name to events                 |
| `tracklist.json`  | Tracks grouped by type (main/devroom/other) |
| `keynotes.json`   | Keynote events (curated by slug)            |
| `roominfo.json`   | Per-room day/time ranges and track listings |
| `roomtracks.json` | Room × time grid for schedule overview      |

`data/sponsors.json` is maintained **by hand**.

All files under `data/` are gitignored.

## Build

```sh
# 1. Generate data files from schedule.json
go run main.go

# 2. Build the site
hugo build -D

# 3. Index for search
pagefind --site "public"
```

## Dev Server

```sh
go run main.go
hugo server --baseURL=http://127.0.0.1/2026 -D --disableFastRender
```

Always test with a `baseURL` — the site will be deployed under a subpath, and all
internal links use `relURL` (e.g. `{{ url | relURL }}`) to resolve paths correctly
against it. Without a `baseURL`, broken links won't surface during development.

## Deploy (Staging)

Staging is served from `www-public0.fosdem.org` and authenticates with Kerberos, so get a
ticket first. It prompts for a password and lasts a few hours:

```sh
kinit {id}@FOSDEM.ORG
klist                      # confirm a krbtgt/FOSDEM.ORG ticket exists
```

`rsync` runs over `ssh`, which needs GSSAPI auth switched on. Either pass it per command
or put it in `~/.ssh/config` once:

```
Host *.fosdem.org
    GSSAPIAuthentication yes
    PreferredAuthentications gssapi-with-mic,publickey,password
```

Then build with the staging `baseURL` and sync. The landing site sits under `/2027/`,
mirroring production, so the `relURL` paths exercised on staging are the ones that ship.
Drafts are included, since staging is where they get reviewed:

```sh
hugo build --config hugo.landing.yaml -b https://staging.fosdem.org/2027/ -D
rsync -avz --delete public/ \
    www-staging@www-public0.fosdem.org:/var/www/staging.fosdem.org/public/2027/
```

`--delete` is scoped to the `2027/` subdirectory, so it cleans up files removed since the
last deploy without touching anything else in the docroot. Never point it at the docroot
root.

## Landing Site

`hugo.landing.yaml` is a separate site in this repo — one page announcing the next
edition, plus the news feed. Own `contentDir`/`layoutDir`, so `hugo.yaml`, `content/`
and `layouts/` are not involved and no `schedule.json` is needed.

```sh
hugo server --config hugo.landing.yaml --baseURL=http://127.0.0.1/2027 --disableFastRender
hugo build  --config hugo.landing.yaml -b https://fosdem.org/2027/
```

The feed is published at `/rss.xml`, not Hugo's default `/index.xml` — that is the URL
subscribers use (`fosdem.org/rss.xml` redirects to the current edition). News items get
real pages so `<link>`/`<guid>` resolve, and `/news/` lists them, linked from the footer
and the homepage.

### News Posts

Posts are page bundles, created with the directory form. Passing `index.md` on the end
silently falls back to Hugo's built-in archetype and you lose the `slug`:

```sh
nix develop -c hugo new content --config hugo.landing.yaml \
    --kind news news/2026-10-02-call-for-participation
```

The directory keeps the date so `content-landing/news/` stays chronological, while the
generated `slug` drops it from the URL (`/2027/news/call-for-participation/`). Posts start
`draft: true`; preview with `-D`.

Images go beside `index.md` and are referenced by filename. A render hook resolves them as
page resources and generates a `srcset` capped to the article width. A quoted title turns
the image into a `<figure>` with that caption:

```markdown
![Alt text](campus.png "Optional caption")
```

The feed and `og:image` use a featured image, defaulting to the first image in the
directory and overridable with `image:` in front matter.

> Feed `<guid>`s are path-independent tag URIs derived from the **directory name**, so
> changing a `slug` is free. Renaming the directory re-issues the guid and subscribers see
> the post again as new.

> The `/rss.xml` → `/<year>/rss.xml` redirect is server-side, not in this repo. It must be
> repointed to `/2027/` or subscribers keep getting the 2026 feed.

## Project Structure

```
content/
  about.md, practical.md, ...   Static pages (Markdown)
  news/                         News articles
  schedule/
    event/_content.gotmpl       Generates a page per event from data
    speaker/_content.gotmpl     Generates a page per speaker from data
    track/_content.gotmpl       Generates a page per track from data
    room/_content.gotmpl        Generates a page per room from data

layouts/
  baseof.html                   Base template (nav, footer, dark mode)
  home.html                     Homepage
  event.html, speaker.html, ... Layout per page type
  partials/                     Shared template fragments

static/
  css/style.css                 All styles (CSS custom properties, dark mode)
  css/fonts/                    Bundled fonts (Signika, DejaVu Sans)
  js/leaflet.js                 Map library
```

Pages under `content/schedule/` use Hugo content adapters (`_content.gotmpl`) to
dynamically create pages from the data files — no individual Markdown file needed per
event/speaker/track/room.
