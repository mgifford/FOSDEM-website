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

`main.go` reads this file (via `//go:embed`) and writes the following to `data/`:

| File              | Contents                                    |
|-------------------|---------------------------------------------|
| `schedule.json`   | Full parsed conference data                 |
| `events.json`     | Map of event slug to event(s)               |
| `speakers.json`   | Map of speaker GUID to speaker + event list |
| `tracks.json`     | Map of track name to events                 |
| `devrooms.json`   | Filtered list of devroom tracks             |
| `keynotes.json`   | Keynote events (curated by slug)            |
| `maintracks.json` | Main track entries (Janson, K-building)     |

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
hugo server --baseURL=http://127.0.0.1/2026 -D
```

Always test with a `baseURL` — the site will be deployed under a subpath, and all
internal links use `relURL` (e.g. `{{ url | relURL }}`) to resolve paths correctly
against it. Without a `baseURL`, broken links won't surface during development.

## Deploy (Preview)

Deploys to Quint's private server for shareable previews:

```sh
hugo build -b https://0x51.dev/fosdem -D
pagefind --site "public"
rsync -avz --delete public/ root@0x51.dev:/var/www/0x51.dev/html/fosdem/
```

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
