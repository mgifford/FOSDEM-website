year         := "2027"
host         := "www-public0.fosdem.org"
staging_root := "/var/www/staging.fosdem.org/public"
live_root    := "/var/www/fosdem.org/public"

# Reaches hugo as site.Params.commit and site.Params.dirty, for home.metrics
export HUGO_PARAMS_COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
export HUGO_PARAMS_DIRTY := `test -z "$(git status --porcelain 2>/dev/null)" && echo false || echo true`

default:
    @just --list

# Refresh data/pretalx from Pretalx. Not committed, every build fetches first.
fetch:
    fetch-pretalx fosdem-{{year}}

# Kerberos ticket for www-public0, lasts a few hours
ticket id:
    kinit {{id}}@FOSDEM.ORG
    klist

# Local dev server, drafts included
serve: fetch
    hugo server --baseURL=http://127.0.0.1/{{year}} -D --disableFastRender

# Build the site and scan it with axe in Chromium (light and dark, four widths). See tests/a11y/README.md
a11y: fetch
    hugo build
    npm ci
    npx playwright install chromium
    npm run test:a11y

# staging baseURL, drafts included
build-staging: fetch
    hugo build -b https://staging.fosdem.org/{{year}}/ -D \
        -d dist/staging --cleanDestinationDir
    cp -r data/pretalx dist/staging/pretalx
    validate-ics dist/staging

# production baseURL, drafts excluded
build-live: fetch
    hugo build -b https://fosdem.org/{{year}}/ \
        -d dist/live --cleanDestinationDir
    cp -r data/pretalx dist/live/pretalx
    validate-ics dist/live

# per-PR preview baseURL, drafts included
build-preview $pr: fetch
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$pr" =~ ^[0-9]+$ ]] || { echo "pr must be a number, got '$pr'" >&2; exit 1; }
    hugo build -b "https://staging.fosdem.org/pr-$pr/" -D \
        -d "dist/pr-$pr" --cleanDestinationDir
    cp -r data/pretalx "dist/pr-$pr/pretalx"
    validate-ics "dist/pr-$pr"

# Sync to staging. Dry run unless go=1
deploy-staging go="": build-staging
    rsync -avz --delete {{ if go == "" { "--dry-run" } else { "" } }} dist/staging/ \
        www-staging@{{host}}:{{staging_root}}/{{year}}/

# Sync to production. Dry run unless go=1
deploy-live go="": build-live
    rsync -avz --delete {{ if go == "" { "--dry-run" } else { "" } }} dist/live/ \
        www-live@{{host}}:{{live_root}}/{{year}}/

# Sync a PR preview to staging. Dry run unless go=1
deploy-preview $pr go="": (build-preview pr)
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$pr" =~ ^[0-9]+$ ]] || { echo "pr must be a number, got '$pr'" >&2; exit 1; }
    rsync -avz --delete {{ if go == "" { "--dry-run" } else { "" } }} \
        "dist/pr-$pr/" "www-staging@{{host}}:{{staging_root}}/pr-$pr/"

# Remove a PR preview from staging. Dry run unless go=1
undeploy-preview $pr go="":
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$pr" =~ ^[0-9]+$ ]] || { echo "pr must be a number, got '$pr'" >&2; exit 1; }
    target="{{staging_root}}/pr-$pr"
    if [[ -z "{{go}}" ]]; then
        echo "would remove $target on {{host}}"
    else
        ssh "www-staging@{{host}}" rm -rf "$target"
    fi
