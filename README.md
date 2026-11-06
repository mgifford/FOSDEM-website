# fosdem.org

Hugo static site for fosdem.org: the edition announcement, news, and the standing pages.

> The conference site (schedule, speakers, tracks, rooms, and the Go program that turns the
> Pretalx export into Hugo data files) lives on the `archive/conference-site` branch. It is
> not built from `main`. Kept for reference, to copy from when the schedule goes up.

## Prerequisites

All tools are provided by the Nix dev shell:

```sh
nix develop
```

This gives you `hugo`, `just`, `woff2`, `validate-ics`, `check-links` and `fetch-pretalx`.

Without Nix, install Hugo 0.155+ manually.

## Build

```sh
hugo build -b https://fosdem.org/2027/
```

## Dev Server

```sh
just serve
```

Always test with a `baseURL`. The site is deployed under a subpath, and all internal links
use `relURL` (e.g. `{{ url | relURL }}`) to resolve paths correctly against it. Without a
`baseURL`, broken links won't surface during development.

`-D` includes drafts. News posts start `draft: true`, so without it a new post is invisible.

## Data

`data/pretalx/` is generated and gitignored. Every build recipe depends on `fetch`, and
copies the result to `<output>/pretalx/`, so the deployed site and its archive carry the
data they were built from.

```sh
just fetch
```

Source is `https://pretalx.fosdem.org/fosdem-{year}/p/fosdem.json`, slug built from `year`
in the `justfile`. Output is sorted and indented.

`tracks.json` holds every track that has tracksettings, whatever its type. `cfp_url` is
empty until a devroom fills it in, so templates must treat that as the normal case.

`data/staff.yaml` is hand-maintained and carries over between editions.

## Deploy

Both targets are served from `www-public0.fosdem.org` and authenticate with Kerberos, so
get a ticket first. It prompts for a password and lasts a few hours:

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

Deploys run from the `justfile`. Both build first, then rsync:

```sh
just deploy-staging          # dry run
just deploy-staging go=1     # writes
just deploy-live go=1
```

Staging sits under `/2027/`, mirroring production, so the `relURL` paths
exercised on staging are the ones that ship. Staging builds with `-D`, production
without, so drafts are reviewable on staging and never reach production.

Each target has its own output directory, `dist/staging` and `dist/live`, built with
`--cleanDestinationDir`. Without it Hugo leaves stale output in place, leaking drafts and
the wrong `baseURL` from the previous build.

`--delete` is scoped to the `2027/` subdirectory, so it cleans up files removed since the
last deploy without touching anything else in the docroot. Never point it at the docroot
root.

Production is `/var/www/fosdem.org/public/2027/` as `www-live`. A dry run does not test
write permission, so a clean `-n` pass is no guarantee the real sync will succeed.

### PR Previews

Every pull request is published at `https://staging.fosdem.org/pr-{number}/` and emptied
when the PR closes. Locally:

```sh
just build-preview 123         # dist/pr-123, baseURL https://staging.fosdem.org/pr-123/
just deploy-preview 123        # dry run
just deploy-preview 123 go=1   # writes
just undeploy-preview 123 go=1 # removes the remote directory
```

Previews build with `-D`. `pr` must be a number.

## Pages

The feed is published at `/rss.xml`, not Hugo's default `/index.xml`. That is the URL
subscribers use (`fosdem.org/rss.xml` redirects to the current edition). News items get
real pages so `<link>`/`<guid>` resolve, and `/news/` lists them, linked from the footer
and the homepage.

### News Posts

Posts are page bundles, created with the directory form. Passing `index.md` on the end
silently falls back to Hugo's built-in archetype and you lose the `slug`:

```sh
nix develop -c hugo new content \
    --kind news news/2026-10-02-call-for-participation
```

The directory keeps the date so `content/news/` stays chronological, while the
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

The draft `2026-08-31-example-post-with-image` exercises every element a post can
contain, callouts included. Preview it with `-D` rather than documenting it here.

`/news/` is an index of title and date, newest first, grouped by year once posts span
more than one. Full text lives on the post page.

### Calendars

The homepage emits `/2027/fosdem.ics` from the `DateStart`/`DateEnd` params. Keep those
in sync with `DateFull`.

It also emits `/2027/fosdem-all.ics`: every `events:` entry on the site plus the weekend,
sorted by date. `/calendar/` lists the same set. Both pick up a new post with no edit.

The homepage has two `.ics` outputs, so each template names its format:
`home.calendar-weekend.ics`, `home.calendar-all.ics`. Named `home.ics`, one template
would serve both and the two files would come out identical. Posts keep `page.ics`,
having one calendar each.

A post ships its own calendar by listing dates and picking a calendar output:

```yaml
outputs:
  - html
  - calendar-cfp
events:
  - date: 2026-10-04
    summary: "FOSDEM 2027: deadline for developer room proposals"
```

The filename comes from the output format, so another name means another format in
`hugo.yaml` (`calendar-cfp` writes `fosdem-cfp.ics`). No template needed:
`page.ics` carries no format name, so it serves them all.

All-day `VEVENT`s, no timezone. Each carries the post link as `URL` and in
`DESCRIPTION`, since Google Calendar drops `URL` on import.

> `UID`s derive from the post directory and the list position. Append to `events:`;
> reordering re-issues them and subscribers see the entries again as new.

Validation runs in CI after the build:

```sh
nix develop -c validate-ics public
```

### Links

`check-links` fetches every external link in a built site. Internal links are covered by
Hugo: an unresolved `relref` fails the build.

```sh
nix develop -c check-links dist/staging --skip payment-links.mollie.com
```

Non-zero exit on failure. Not in CI. Mollie payment links 404 on every path, valid ones
included.

### Metrics

The homepage emits `/2027/metrics` in Prometheus text exposition format, from the
`metrics` output format and `home.metrics`:

```
# HELP website_last_build_time Starting time of the last website build
# TYPE website_last_build_time gauge
website_last_build_time 1792391980
```

`now.Unix` at build time, so a scrape that stops advancing means the build stopped. The
scrape target carries the year prefix: `https://fosdem.org/2027/metrics`.

The extensionless filename comes from the `text/prometheus` media type: empty `suffixes`
and `delimiter`.

### Standalone Pages

Paths come from the file tree. Use `url:` only to pin a page to a path whose section does
not exist here yet:

```yaml
title: "Code of Conduct"
url: /practical/conduct/
```

Dateless, so the layout omits the dateline. Link them from `_partials/footer.html`, or
`_partials/header.html` for the few that belong in the top nav.

Sponsors and FAQ entries are one file each, assembled into a single page with anchors and
never rendered on their own:

```
content/support/sponsors/<id>/index.md   tier: cornerstone|main|thanks
content/support/sponsors/<id>/logo.svg   optional, page resource
content/faq/<category>/_index.md         category title and weight
content/faq/<category>/<question>.md     weight orders within the category
```

The directory name is the anchor. `/support/` lists its own children.

> Feed `<guid>`s are path-independent tag URIs derived from the **directory name**, so
> changing a `slug` is free. Renaming the directory re-issues the guid and subscribers see
> the post again as new.

> The `/rss.xml` -> `/<year>/rss.xml` redirect is server-side, not in this repo. It points
> at the current edition and needs repointing when a new one opens, or subscribers keep
> getting the previous feed.
