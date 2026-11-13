#!/usr/bin/env python3
"""Compare your pull requests on FOSDEM's forge with their mirrors on GitHub. Read-only.

    scripts/check-upstream-prs.py

The forge (https://git.fosdem.org/FOSDEM/website/pulls) is the source of truth. For every pair in
tools/compare/upstream-prs.json this reports:

  * whether the forge PR still merges into upstream main, and which files conflict;
  * whether the GitHub PR is in step with it. A GitHub PR is closed only AFTER the forge one is
    closed or merged, so this tells you when it is safe, and warns if GitHub is ahead of the forge;
  * forge PRs by you that have no GitHub mirror yet.

It only fetches and reads. It never closes, comments on or pushes to anything.
Exit status: 1 if a forge PR conflicts or the two sides are out of step, otherwise 0.
Network trouble is reported and is not an error.
"""
import json
import subprocess
import sys
import urllib.request
from pathlib import Path

ROOT = Path(subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip())
CONFIG = json.loads((ROOT / "tools/compare/upstream-prs.json").read_text())
FORGE = CONFIG["forgeRepo"]
API = FORGE.replace("https://git.fosdem.org/", "https://git.fosdem.org/api/v1/repos/")


def git(*args, check=False):
    return subprocess.run(["git", "-C", str(ROOT), *args], text=True, capture_output=True, check=check)


def forge_json(path):
    with urllib.request.urlopen(f"{API}/{path}", timeout=20) as response:
        return json.load(response)


def github_state(number):
    out = subprocess.run(
        ["gh", "pr", "view", str(number), "--repo", CONFIG["githubRepo"], "--json", "state", "--jq", ".state"],
        text=True, capture_output=True,
    )
    return out.stdout.strip() if out.returncode == 0 else None


def conflicts(number):
    """Files that conflict when the forge PR head is merged into upstream/main (nothing is modified)."""
    if git("fetch", "-q", "upstream", f"refs/pull/{number}/head").returncode != 0:
        return None, None
    head = git("rev-parse", "FETCH_HEAD").stdout.strip()
    behind = git("rev-list", "--count", f"{head}..upstream/main").stdout.strip()
    merge = git("merge-tree", "--write-tree", "--name-only", "upstream/main", head)
    if merge.returncode == 0:
        return [], behind
    lines = merge.stdout.splitlines()[1:]
    files = []
    for line in lines:
        if not line.strip():
            break
        files.append(line.strip())
    return files, behind


def main():
    if git("remote", "get-url", "upstream").returncode != 0:
        git("remote", "add", "upstream", FORGE + ".git")
    if git("fetch", "-q", "upstream", "main").returncode != 0:
        print("check-upstream-prs: cannot fetch upstream, skipping", file=sys.stderr)
        return 0
    print(f"upstream/main is {git('rev-parse', '--short', 'upstream/main').stdout.strip()}\n")
    problems = 0
    mapped = set()
    for pair in CONFIG["pairs"]:
        mapped.add(pair["forge"])
        forge_no = pair["forge"]
        gh_nos = pair.get("githubPrs") or [pair["github"]]
        print(f"forge #{forge_no}  <->  GitHub {', '.join('#' + str(n) for n in gh_nos)}: {pair['summary']}")
        try:
            pr = forge_json(f"pulls/{forge_no}")
        except Exception as error:
            print(f"    cannot read the forge PR ({error}); skipped\n")
            continue
        forge_open = pr["state"] == "open" and not pr.get("merged")
        forge_state = "merged" if pr.get("merged") else pr["state"]
        gh_states = {n: github_state(n) for n in gh_nos}
        shown = ", ".join(f"#{n} {(st or 'unknown').lower()}" for n, st in gh_states.items())
        print(f"    forge: {forge_state}   GitHub: {shown}")
        if forge_open:
            files, behind = conflicts(forge_no)
            if files is None:
                print("    cannot fetch the forge PR head")
            elif files:
                problems += 1
                print(f"    CONFLICTS with upstream main ({behind} commits behind): {', '.join(files)}")
                if pair.get("currentBranch"):
                    print(f"    updated version: branch {pair['currentBranch']} (merges cleanly); push it to the PR branch and update the PR")
                else:
                    print("    no updated version: check whether upstream main already has the change, then close it on the forge")
            else:
                print(f"    merges cleanly ({behind} commits behind upstream)")
            closed_early = [n for n, st in gh_states.items() if st and st != "OPEN"]
            if closed_early:
                problems += 1
                print(f"    OUT OF STEP: GitHub {', '.join('#' + str(n) for n in closed_early)} closed but forge #{forge_no} is still open; reopen")
            else:
                print(f"    keep GitHub {', '.join('#' + str(n) for n in gh_nos)} open until forge #{forge_no} is closed")
        else:
            still_open = [n for n, st in gh_states.items() if st == "OPEN"]
            if still_open:
                print(f"    forge #{forge_no} is {forge_state}: GitHub {', '.join('#' + str(n) for n in still_open)} can now be closed (not done by this script)")
            else:
                print("    all closed: in step")
        print()

    for item in CONFIG.get("notYetOnForge", []):
        st = github_state(item["github"])
        print(f"GitHub #{item['github']} ({item['branch']}) is not on the forge yet: {item['summary']} [{(st or 'unknown').lower()}]")
    print()
    try:
        mine = [p for p in forge_json("pulls?state=open&limit=50") if p["user"]["login"].lower() == CONFIG["forgeUser"].lower()]
    except Exception:
        mine = []
    for p in mine:
        if p["number"] not in mapped:
            print(f"forge #{p['number']} has no GitHub mirror yet: {p['title']}\n")
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
