#!/usr/bin/env python3
"""Check that a built comparison is complete. Usage: validate.py [_site]

Fails (exit 1) if the comparison is missing anything a visitor needs, so a broken build is never
published with the rest of the site.
"""
import json
import sys
from pathlib import Path

root = Path(sys.argv[1] if len(sys.argv) > 1 else "_site")
compare = root / "compare"
errors = []


def need(path):
    if not path.exists():
        errors.append(f"missing {path}")
        return False
    return True


if need(compare / "index.html") and need(compare / "variants.json"):
    data = json.loads((compare / "variants.json").read_text())
    if data.get("failed"):
        errors.append(f"variants failed to build: {', '.join(data['failed'])}")
    if not data.get("variants"):
        errors.append("no variants were built")
    for key in ("upstream", "data", "hugo"):
        if not data.get(key):
            errors.append(f"variants.json has no '{key}' provenance")
    for v in data.get("variants", []):
        slug = v["slug"]
        need(compare / "after" / slug / "index.html")
        need(compare / "before" / v["before_sha"] / "index.html")
        need(compare / "diffs" / f"{slug}.txt")
results = compare / "results.json"
if need(results):
    r = json.loads(results.read_text())
    if not r.get("ok"):
        errors.append("results.json: automated checks did not behave as expected")
    for v in data.get("variants", []):
        if v["slug"] not in r.get("results", {}):
            errors.append(f"results.json has no results for {v['slug']}")
if not (root / "index.html").exists():
    errors.append(f"missing {root / 'index.html'}")

if errors:
    print("comparison is incomplete:", *errors, sep="\n  ", file=sys.stderr)
    sys.exit(1)
print(f"ok: {len(data['variants'])} comparisons, upstream {data['upstream']['sha']}")
