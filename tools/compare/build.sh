#!/usr/bin/env bash
# Build upstream "before" and one "after" per entry in variants.json, side by side, plus a
# diff report per variant and the interactive page (tools/compare/index.html).
#
#   tools/compare/build.sh                       # serve OUT at http://127.0.0.1:8093
#   PUBLIC_BASE=https://user.github.io/repo/compare tools/compare/build.sh
#
# Env: UPSTREAM_URL, UPSTREAM_REF (default FOSDEM/website main), UPSTREAM_DIR (use an existing
# checkout instead of cloning), PUBLIC_BASE (URL prefix OUT is served under, no trailing slash),
# OUT (default _site/compare), VARIANTS (default tools/compare/variants.json).
#
# Every site is built with test data (testdata/schedule.json). A second, neutral-base build of
# each is used only for the diff, so base URLs cannot show up as differences.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
UPSTREAM_URL="${UPSTREAM_URL:-https://git.fosdem.org/FOSDEM/website.git}"
UPSTREAM_REF="${UPSTREAM_REF:-main}"
PUBLIC_BASE="${PUBLIC_BASE:-http://127.0.0.1:8093}"
PUBLIC_BASE="${PUBLIC_BASE%/}"
OUT="${OUT:-$repo/_site/compare}"
VARIANTS="${VARIANTS:-$here/variants.json}"
NEUTRAL="https://review.invalid/"

work="$(mktemp -d)"
cleanup() { git -C "$repo" worktree prune; rm -rf "${work:?}"; }
trap cleanup EXIT

# The schedule generator's output is not byte-stable (event order on speaker pages varies between
# runs), so it is generated once from upstream and copied into every tree. A variant that changed
# main.go itself would not show that here.
gen_data() { ( cd "$1" && go run . testdata/schedule.json >/dev/null ); }
build_site() { # srcdir base outdir
  ( cd "$1"
    [ "$1" = "$up" ] || cp "$up"/data/*.json "$1/data/"
    hugo build --config hugo.landing.yaml -b "$2" -D -d "$3" --cleanDestinationDir >/dev/null 2>&1
    hugo build -b "${2}full/" -D -d "$3/full" >/dev/null 2>&1 )
}

mkdir -p "$OUT/before" "$OUT/after" "$OUT/diffs" "$work/neutral" "$work/stats"

# Upstream ("before").
if [ -n "${UPSTREAM_DIR:-}" ]; then up="$UPSTREAM_DIR"; else
  up="$work/upstream"; git clone -q --depth 1 --branch "$UPSTREAM_REF" "$UPSTREAM_URL" "$up"
fi
up_sha="$(git -C "$up" rev-parse --short HEAD)"
up_date="$(git -C "$up" log -1 --format=%cs)"
echo "before: upstream $up_sha ($up_date)"
gen_data "$up"
build_site "$up" "$PUBLIC_BASE/before/" "$OUT/before"
build_site "$up" "$NEUTRAL" "$work/neutral/before"

# Variants ("after").
python3 - "$VARIANTS" > "$work/variants.tsv" <<'PY'
import json,sys
for v in json.load(open(sys.argv[1])): print(v["slug"], v["ref"], sep="\t")
PY
while IFS=$'\t' read -r slug ref; do
  sha="$(git -C "$repo" rev-parse --verify -q "$ref^{commit}" || git -C "$repo" rev-parse --verify -q "origin/$ref^{commit}" || true)"
  if [ -z "$sha" ]; then echo "skip $slug: ref $ref not found"; echo "missing" > "$work/stats/$slug"; continue; fi
  src="$work/src-$slug"
  git -C "$repo" worktree add -q --detach "$src" "$sha"
  echo "after:  $slug = $ref @ ${sha:0:7}"
  build_site "$src" "$PUBLIC_BASE/after/$slug/" "$OUT/after/$slug"
  build_site "$src" "$NEUTRAL" "$work/neutral/$slug"
  ( cd "$work/neutral"
    n="$( { diff -rq before "$slug" || true; } | wc -l | tr -d ' ')"
    { echo "Compared: upstream $UPSTREAM_REF @ $up_sha  vs  $ref @ ${sha:0:7}"
      echo "Files that differ, ignoring base URLs: $n"
      echo
      diff -rq before "$slug" || true
      echo
      { diff -ru before "$slug" || true; } | head -c 300000
    } > "$OUT/diffs/$slug.txt"
    echo "$n $sha" > "$work/stats/$slug" )
  git -C "$repo" worktree remove --force "$src"
done < "$work/variants.tsv"

python3 - "$VARIANTS" "$work/stats" "$OUT/variants.json" "$up_sha" "$up_date" "$UPSTREAM_URL" <<'PY'
import json,sys,os
variants=json.load(open(sys.argv[1])); stats=sys.argv[2]
out={"upstream":{"url":sys.argv[6],"sha":sys.argv[4],"date":sys.argv[5]},"variants":[]}
for v in variants:
    s=open(os.path.join(stats,v["slug"])).read().split()
    if s[0]=="missing": continue
    v["files_changed"]=int(s[0]); v["sha"]=s[1][:7]
    out["variants"].append(v)
json.dump(out,open(sys.argv[3],"w"),indent=2)
PY
cp "$here/index.html" "$OUT/index.html"
echo "done: $OUT  (serve it at $PUBLIC_BASE/)"
