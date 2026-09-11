year         := "2027"
host         := "www-public0.fosdem.org"
config       := "hugo.landing.yaml"
staging_root := "/var/www/staging.fosdem.org/public"
live_root    := "/var/www/fosdem.org/public"

default:
    @just --list

# Kerberos ticket for www-public0, lasts a few hours
ticket id:
    kinit {{id}}@FOSDEM.ORG
    klist

# Landing site, staging baseURL, drafts included
build-staging:
    hugo build --config {{config}} -b https://staging.fosdem.org/{{year}}/ -D \
        -d dist/staging --cleanDestinationDir
    validate-ics dist/staging

# Landing site, production baseURL, drafts excluded
build-live:
    hugo build --config {{config}} -b https://fosdem.org/{{year}}/ \
        -d dist/live --cleanDestinationDir
    validate-ics dist/live

# Landing site, per-PR preview baseURL, drafts included
build-preview $pr:
    #!/usr/bin/env bash
    set -euo pipefail
    [[ "$pr" =~ ^[0-9]+$ ]] || { echo "pr must be a number, got '$pr'" >&2; exit 1; }
    hugo build --config {{config}} -b "https://staging.fosdem.org/pr-$pr/" -D \
        -d "dist/pr-$pr" --cleanDestinationDir
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
