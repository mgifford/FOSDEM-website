#!/usr/bin/env bash
# Build the landing site (root) and the full site (/full/) into _site/ for the tests.
# BASE must match where the site is served (scripts/serve.mjs serves _site under
# /FOSDEM-website/ on :8080).
set -euo pipefail
BASE="${BASE:-http://127.0.0.1:8080/FOSDEM-website/}"
BASE="${BASE%/}/"
cd "$(dirname "$0")/.."
go run . testdata/schedule.json
hugo build --config hugo.landing.yaml -b "$BASE" -D -d _site --cleanDestinationDir
hugo build -b "${BASE}full/" -D -d _site/full
