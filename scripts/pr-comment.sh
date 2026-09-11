#!/usr/bin/env bash
# Post a comment on a Forgejo pull request, once. The marker is an HTML comment
# that does not render, so a rerun finds it and skips instead of piling on.
#
# Usage: pr-comment <pr-number> <marker> <body>
#
# Needs FORGEJO_TOKEN (scope write:issue), CI_FORGE_URL and CI_REPO.

set -euo pipefail

: "${FORGEJO_TOKEN:?not set}"
: "${CI_FORGE_URL:?not set}"
: "${CI_REPO:?not set}"

pr=${1:?usage: pr-comment <pr-number> <marker> <body>}
marker=${2:?usage: pr-comment <pr-number> <marker> <body>}
body=${3:?usage: pr-comment <pr-number> <marker> <body>}

[[ "$pr" =~ ^[0-9]+$ ]] || { echo "pr must be a number, got '$pr'" >&2; exit 1; }

api="${CI_FORGE_URL%/}/api/v1/repos/${CI_REPO}/issues/${pr}/comments"
auth="Authorization: token ${FORGEJO_TOKEN}"

found=$(curl -sSf -H "$auth" "$api" |
    jq --arg m "<!-- $marker -->" '[.[] | select(.body | contains($m))] | length')

if [[ "$found" -gt 0 ]]; then
    echo "comment '$marker' already on PR $pr, nothing to do"
    exit 0
fi

jq -n --arg b "<!-- $marker -->"$'\n'"$body" '{body: $b}' |
    curl -sSf -X POST -H "$auth" -H "Content-Type: application/json" \
        --data @- "$api" >/dev/null

echo "posted comment '$marker' on PR $pr"
