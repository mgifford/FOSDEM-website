"""Validate the .ics files Hugo generates against the parts of RFC 5545 that
calendar clients actually reject on.

Usage: validate-ics <path>...   (files, or directories searched recursively)
"""

import sys
from pathlib import Path

from icalendar import Calendar

REQUIRED_CALENDAR = ("VERSION", "PRODID")
REQUIRED_EVENT = ("UID", "DTSTAMP", "DTSTART")


def check(path):
    """Return a list of problems found in one .ics file."""
    problems = []
    raw = path.read_bytes()

    # RFC 5545 3.1: lines end with CRLF and are at most 75 octets before folding.
    for number, line in enumerate(raw.split(b"\r\n"), start=1):
        if b"\n" in line:
            problems.append(f"line {number}: bare LF, expected CRLF")
        if len(line) > 75:
            problems.append(f"line {number}: {len(line)} octets, over the 75 limit")

    try:
        calendar = Calendar.from_ical(raw)
    except ValueError as error:
        return problems + [f"does not parse: {error}"]

    for key in REQUIRED_CALENDAR:
        if key not in calendar:
            problems.append(f"VCALENDAR is missing {key}")

    events = calendar.walk("VEVENT")
    if not events:
        problems.append("no VEVENT")

    seen = set()
    for event in events:
        summary = event.get("SUMMARY", "<no summary>")
        for key in REQUIRED_EVENT:
            if key not in event:
                problems.append(f"{summary}: missing {key}")

        uid = str(event.get("UID", ""))
        if uid in seen:
            problems.append(f"{summary}: duplicate UID {uid}")
        seen.add(uid)

    # Only registered properties, so nothing depends on a single vendor's reading.
    for component in calendar.walk():
        for key in component.keys():
            if key.upper().startswith("X-"):
                problems.append(f"{component.name}: non-standard property {key}")

    return problems


def main(argv):
    paths = []
    for argument in argv:
        path = Path(argument)
        paths.extend(sorted(path.rglob("*.ics")) if path.is_dir() else [path])

    if not paths:
        print("validate-ics: no .ics files found", file=sys.stderr)
        return 1

    failed = 0
    for path in paths:
        problems = check(path)
        if problems:
            failed += 1
            print(f"FAIL {path}")
            for problem in problems:
                print(f"     {problem}")
        else:
            print(f"ok   {path}")

    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
