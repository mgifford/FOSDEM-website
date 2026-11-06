"""Fetch the FOSDEM export from Pretalx into Hugo data files.

Usage: fetch-pretalx <event-slug> [-o <directory>]

What this writes is not committed. Every build fetches first and copies the
result into its output, so the deployed site carries the data it was built
from and an archived edition keeps it, years after Pretalx has moved on to
the next conference.
"""

import argparse
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path

BASE_URL = "https://pretalx.fosdem.org"
SECTIONS = ("tracks", "fringe")
TIMEOUT = 30


def fetch(event):
    """Return the parsed fosdem.json payload for one event."""
    url = f"{BASE_URL}/{event}/p/fosdem.json"
    with urllib.request.urlopen(url, timeout=TIMEOUT) as response:
        return json.loads(response.read())


def serialise(section):
    """Normalised JSON, so an unchanged export leaves the file byte identical."""
    return json.dumps(section, indent=2, sort_keys=True, ensure_ascii=False) + "\n"


def entries(path):
    """How many entries the file holds, 0 if it is not there yet."""
    if not path.exists():
        return 0
    return len(json.loads(path.read_text(encoding="utf-8")))


def main(argv):
    parser = argparse.ArgumentParser(
        prog="fetch-pretalx", description=__doc__.splitlines()[0]
    )
    parser.add_argument("event", help="Pretalx event slug, e.g. fosdem-2027")
    parser.add_argument(
        "-o", "--output", type=Path, default=Path("data/pretalx"),
        help="where to write the data files (default: data/pretalx)",
    )
    options = parser.parse_args(argv)

    try:
        payload = fetch(options.event)
    except (urllib.error.URLError, json.JSONDecodeError) as error:
        print(f"fetch-pretalx: {options.event}: {error}", file=sys.stderr)
        return 1

    missing = [name for name in SECTIONS if name not in payload]
    if missing:
        print(f"fetch-pretalx: export has no {', '.join(missing)}", file=sys.stderr)
        return 1

    options.output.mkdir(parents=True, exist_ok=True)
    for name in SECTIONS:
        path = options.output / f"{name}.json"
        section = payload[name]
        before = entries(path)

        # An export that comes back empty is far more often a Pretalx hiccup
        # than a real emptying, and the build has no network to notice with.
        if not section and before:
            print(
                f"fetch-pretalx: {name} came back empty but {path} holds "
                f"{before}, delete it first if that is real",
                file=sys.stderr,
            )
            return 1

        text = serialise(section)
        unchanged = path.exists() and path.read_text(encoding="utf-8") == text
        path.write_text(text, encoding="utf-8")
        print(f"{'unchanged' if unchanged else 'updated':9} {path} ({len(section)})")

    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
