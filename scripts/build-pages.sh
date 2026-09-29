#!/usr/bin/env bash
# Build the landing site (root) and the full site (/full/) into _site/,
# using the GitHub Pages baseURL. Override BASE for a fork or local use.
set -euo pipefail
BASE="${BASE:-https://mgifford.github.io/FOSDEM-website/}"
BASE="${BASE%/}/"
cd "$(dirname "$0")/.."
go run . testdata/schedule.json
hugo build --config hugo.landing.yaml -b "$BASE" -D -d _site --cleanDestinationDir
hugo build -b "${BASE}full/" -D -d _site/full
