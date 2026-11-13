#!/usr/bin/env bash
# Start a new improvement on top of the latest upstream FOSDEM code.
#
#   scripts/start-work.sh <short-name>     # new branch 2027/<short-name> from upstream/main
#   scripts/start-work.sh --sync-only      # only bring main up to date
#
# 1. fetches FOSDEM/website (git.fosdem.org) as the `upstream` remote;
# 2. merges upstream/main into main (the comparison lab), stopping cleanly on conflicts;
# 3. reports whether your open pull requests on the forge still merge (scripts/check-upstream-prs.py);
# 4. creates the change branch from upstream/main itself, not from main, so the branch holds only
#    the change and can be offered upstream without the lab tooling.
# Then add an entry for the branch to tools/compare/variants.json.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
UPSTREAM_URL="${UPSTREAM_URL:-https://git.fosdem.org/FOSDEM/website.git}"
MAIN="${MAIN_BRANCH:-main}"
arg="${1:?usage: scripts/start-work.sh <short-name> | --sync-only}"

[ -z "$(git status --porcelain)" ] || { echo "Working tree is not clean. Commit or stash first." >&2; exit 1; }
git remote get-url upstream >/dev/null 2>&1 || git remote add upstream "$UPSTREAM_URL"
git fetch -q upstream main
git fetch -q origin 2>/dev/null || echo "note: could not fetch origin, continuing" >&2

start="$(git rev-parse --abbrev-ref HEAD)"
git switch -q "$MAIN"
if git merge-base --is-ancestor upstream/main HEAD; then
  echo "$MAIN already contains upstream/main ($(git rev-parse --short upstream/main))."
elif git merge --no-edit upstream/main; then
  echo "Merged upstream/main into $MAIN. Push it when ready."
else
  git merge --abort
  git switch -q "$start"
  echo "Merging upstream/main into $MAIN conflicts. Resolve it on $MAIN first, then run this again." >&2
  exit 1
fi

# Informational: do my open pull requests on the FOSDEM forge still merge into upstream main?
# Never changes anything and never fails this script.
python3 scripts/check-upstream-prs.py || echo "(see above: fix conflicting pull requests before or alongside new work)" >&2

if [ "$arg" = "--sync-only" ]; then git switch -q "$start"; exit 0; fi
case "$arg" in *[!a-zA-Z0-9._-]*|"") echo "Use letters, digits, dot, dash or underscore in the name." >&2; exit 1 ;; esac
# --no-track: a plain `git push` must never point at the upstream FOSDEM repository
git switch --no-track -c "2027/$arg" upstream/main
echo "On 2027/$arg at upstream $(git rev-parse --short upstream/main). Make the change, commit, and add it to tools/compare/variants.json on $MAIN."
