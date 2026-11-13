#!/usr/bin/env bash
# Build a "before" and an "after" of the 2027 site for every entry in variants.json, side by side,
# plus a diff report per variant and the interactive viewer (tools/compare/index.html).
#
#   tools/compare/build.sh                       # serve OUT at http://127.0.0.1:8093
#   PUBLIC_BASE=https://user.github.io/repo/compare tools/compare/build.sh
#
# Each variant is a git ref (a fix branch) based on FOSDEM/website. Its "before" is the commit it
# forked from (merge-base with upstream), so the diff is that change and nothing else. A variant
# with "merge": [refs] is built from the upstream tip with those refs merged in.
#
# Env: UPSTREAM_URL (default git.fosdem.org/FOSDEM/website), UPSTREAM_REF (default upstream/main),
# NO_FETCH=1 (do not fetch upstream), EVENT (Pretalx slug, default fosdem-2027), PUBLIC_BASE (URL
# prefix OUT is served under, no trailing slash), OUT (default _site/compare), VARIANTS.
#
# Every tree is built with the same Pretalx data, fetched once. A second, neutral-base build of
# each tree is used only for the diff, so base URLs cannot show up as differences.
# Any variant that cannot be built fails the whole run: nothing is silently dropped.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
UPSTREAM_URL="${UPSTREAM_URL:-https://git.fosdem.org/FOSDEM/website.git}"
UPSTREAM_REF="${UPSTREAM_REF:-upstream/main}"
EVENT="${EVENT:-fosdem-2027}"
PUBLIC_BASE="${PUBLIC_BASE:-http://127.0.0.1:8093}"
PUBLIC_BASE="${PUBLIC_BASE%/}"
OUT="${OUT:-$repo/_site/compare}"
VARIANTS="${VARIANTS:-$here/variants.json}"
NEUTRAL="https://review.invalid/"
git_id=(-c user.name=compare -c user.email=compare@invalid)

work="$(mktemp -d)"
cleanup() { git -C "$repo" worktree prune; rm -rf "${work:?}"; }
trap cleanup EXIT

if [ -z "${NO_FETCH:-}" ]; then
  git -C "$repo" remote get-url upstream >/dev/null 2>&1 || git -C "$repo" remote add upstream "$UPSTREAM_URL"
  git -C "$repo" fetch -q upstream main
fi
up_full="$(git -C "$repo" rev-parse --verify "$UPSTREAM_REF^{commit}")"
up_sha="${up_full:0:7}"
up_date="$(git -C "$repo" log -1 --format=%cs "$up_full")"
echo "upstream: $UPSTREAM_REF = $up_sha ($up_date)"

# Pretalx data is gitignored upstream and every build fetches it, so it is fetched once here and
# copied into each tree. Its hash is recorded so a published comparison names the data it used.
python3 "$repo/scripts/fetch-pretalx.py" "$EVENT" -o "$work/pretalx"
data_hash="$(cat "$work/pretalx"/*.json | shasum -a 256 | cut -d' ' -f1)"
data_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
hugo_version="$(hugo version | cut -d' ' -f1-2)"
# Public URL of this repository, used for branch links in the viewer.
if [ -n "${GITHUB_REPOSITORY:-}" ]; then repo_url="https://github.com/$GITHUB_REPOSITORY"
else repo_url="$(git -C "$repo" remote get-url origin 2>/dev/null | sed -E 's#^git@([^:]+):#https://\1/#; s#\.git$##' || true)"; fi

resolve() { git -C "$repo" rev-parse --verify -q "$1^{commit}" || git -C "$repo" rev-parse --verify -q "origin/$1^{commit}" || true; }

build_site() { # srcdir base outdir
  mkdir -p "$1/data"; rm -rf "$1/data/pretalx"; cp -r "$work/pretalx" "$1/data/pretalx"
  if [ -f "$1/hugo.landing.yaml" ]; then
    # Commits from before upstream's restructure (2026-10-26) build the landing site from its own
    # config and test schedule data. This is only for pull requests written against that code.
    ( cd "$1" && go run . testdata/schedule.json >"$work/hugo.log" 2>&1 \
        && hugo build --config hugo.landing.yaml -b "$2" -D -d "$3" --cleanDestinationDir >>"$work/hugo.log" 2>&1 ) \
      || { echo "legacy build failed in $1:" >&2; tail -20 "$work/hugo.log" >&2; return 1; }
  else
    ( cd "$1" && hugo build -b "$2" -D -d "$3" --cleanDestinationDir >"$work/hugo.log" 2>&1 ) \
      || { echo "hugo failed in $1:" >&2; tail -20 "$work/hugo.log" >&2; return 1; }
  fi
}

# OUT is rebuilt from scratch: a leftover tree from an earlier run must never end up in a published site.
rm -rf "$OUT"
mkdir -p "$OUT/before" "$OUT/after" "$OUT/diffs" "$work/neutral" "$work/stats" "$work/built"

# Before trees are keyed by commit: variants forked from the same upstream commit share one.
ensure_before() { # full sha -> builds before/<sha7> (and its neutral twin) once
  local sha="$1" b7="${1:0:7}"
  [ -e "$work/built/$b7" ] && return 0
  local src="$work/src-before-$b7"
  git -C "$repo" worktree add -q --detach "$src" "$sha"
  echo "before: $b7"
  build_site "$src" "$PUBLIC_BASE/before/$b7/" "$OUT/before/$b7"
  build_site "$src" "$NEUTRAL" "$work/neutral/before-$b7"
  git -C "$repo" worktree remove --force "$src"
  touch "$work/built/$b7"
}

python3 - "$VARIANTS" > "$work/variants.tsv" <<'PY'
import json,sys
for v in json.load(open(sys.argv[1])):
    print(v["slug"], v.get("ref","-"), ",".join(v.get("merge",[])) or "-", v.get("forgePr","-"), "yes" if v.get("alsoOnCurrentUpstream") else "-", sep="\t")
PY
failures=""
while IFS=$'\t' read -r slug ref merge forge current; do
  src="$work/src-$slug"
  if [ "$merge" != "-" ]; then
    git -C "$repo" worktree add -q --detach "$src" "$up_full"
    base="$up_full"; ok=1; label="upstream + $merge"
    IFS=',' read -ra parts <<<"$merge"
    for m in "${parts[@]}"; do
      msha="$(resolve "$m")"
      if [ -z "$msha" ] || ! git -C "$src" "${git_id[@]}" merge -q --no-edit "$msha" >/dev/null 2>&1; then
        echo "FAIL $slug: cannot merge $m (missing or conflicts)" >&2; ok=0; break
      fi
    done
    [ "$ok" = 1 ] || { git -C "$src" merge --abort 2>/dev/null || true; git -C "$repo" worktree remove --force "$src"; failures="$failures $slug"; echo "failed" > "$work/stats/$slug"; continue; }
    sha="$(git -C "$src" rev-parse HEAD)"
  else
    if [ "$forge" != "-" ]; then
      # The head commit of a pull request on the FOSDEM forge, tested as submitted.
      ref="forge PR #$forge"
      if git -C "$repo" fetch -q upstream "refs/pull/$forge/head" 2>/dev/null; then sha="$(git -C "$repo" rev-parse FETCH_HEAD)"; else sha=""; fi
    else
      sha="$(resolve "$ref")"
    fi
    if [ -z "$sha" ]; then echo "FAIL $slug: ref $ref not found" >&2; failures="$failures $slug"; echo "failed" > "$work/stats/$slug"; continue; fi
    base="$(git -C "$repo" merge-base "$sha" "$up_full")"
    git -C "$repo" worktree add -q --detach "$src" "$sha"
    label="$ref"
  fi
  echo "after:  $slug = $label @ ${sha:0:7} (before ${base:0:7})"
  if { [ "$current" != "yes" ] || ensure_before "$up_full"; } && ensure_before "$base" && build_site "$src" "$PUBLIC_BASE/after/$slug/" "$OUT/after/$slug" \
       && build_site "$src" "$NEUTRAL" "$work/neutral/$slug"; then
    b7="${base:0:7}"
    ( cd "$work/neutral"
      n="$( { diff -rq "before-$b7" "$slug" || true; } | wc -l | tr -d ' ')"
      { echo "Compared: upstream @ $b7  vs  $label @ ${sha:0:7}"
        echo "Files that differ, ignoring base URLs: $n"
        echo
        diff -rq "before-$b7" "$slug" || true
        echo
        { diff -ru "before-$b7" "$slug" || true; } | head -c 300000
      } > "$OUT/diffs/$slug.txt"
      echo "$n $sha $base" > "$work/stats/$slug" )
  else
    failures="$failures $slug"; echo "failed" > "$work/stats/$slug"
  fi
  git -C "$repo" worktree remove --force "$src"
done < "$work/variants.tsv"

python3 - "$VARIANTS" "$work/stats" "$OUT/variants.json" "$UPSTREAM_URL" "$up_full" "$up_date" "$EVENT" "$data_hash" "$data_time" "$hugo_version" "$repo_url" "$PUBLIC_BASE" <<'PY'
import json,sys,os
variants=json.load(open(sys.argv[1])); stats=sys.argv[2]
out={"upstream":{"url":sys.argv[4],"sha":sys.argv[5][:7],"full_sha":sys.argv[5],"date":sys.argv[6]},
     "data":{"event":sys.argv[7],"sha256":sys.argv[8],"fetched":sys.argv[9]},
     "hugo":sys.argv[10],"repo":sys.argv[11],"public_base":sys.argv[12],"variants":[],"failed":[]}
for v in variants:
    s=open(os.path.join(stats,v["slug"])).read().split()
    if s[0]=="failed": out["failed"].append(v["slug"]); continue
    v["files_changed"]=int(s[0]); v["sha"]=s[1][:7]; v["full_sha"]=s[1]
    v["before_sha"]=s[2][:7]; v["before_full_sha"]=s[2]
    out["variants"].append(v)
json.dump(out,open(sys.argv[3],"w"),indent=2)
PY
cp "$here/index.html" "$OUT/index.html"
if [ -n "$failures" ]; then echo "FAILED variants:$failures" >&2; exit 1; fi
echo "done: $OUT  (serve it at $PUBLIC_BASE/)"
