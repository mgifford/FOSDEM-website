"""Fetch every external link in the built site and report its status code.

Usage: check-links <path>...   (files, or directories searched recursively)

Only http(s) links leaving the site are checked. Internal links are already
verified at build time: Hugo fails the build on an unresolved relref.
"""

import argparse
import concurrent.futures
import sys
import urllib.error
import urllib.request
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import urlsplit

# Some servers answer a bare urllib request with 403 and a real browser with 200.
USER_AGENT = "Mozilla/5.0 (compatible; fosdem-check-links/1.0; +https://fosdem.org/)"

# Servers that mishandle HEAD are retried with GET, so treat these as "ask again".
RETRY_WITH_GET = (403, 404, 405, 501)

LINK_ATTRIBUTES = {
    "a": "href",
    "link": "href",
    "img": "src",
    "script": "src",
    "iframe": "src",
    "source": "src",
}


class LinkParser(HTMLParser):
    """Collect the URLs an HTML document points at."""

    def __init__(self):
        super().__init__()
        self.urls = []

    def handle_starttag(self, tag, attrs):
        wanted = LINK_ATTRIBUTES.get(tag)
        if not wanted:
            return
        for name, value in attrs:
            if name == wanted and value:
                self.urls.append(value)


def external_links(paths):
    """Return {url: [pages that link to it]} for every external link found."""
    found = {}

    for path in paths:
        parser = LinkParser()
        parser.feed(path.read_text(encoding="utf-8", errors="replace"))

        for url in parser.urls:
            scheme = urlsplit(url).scheme
            if scheme in ("http", "https"):
                found.setdefault(url, []).append(path)

    return found


def status(url, timeout):
    """Return (code, note) for one URL. code is None when nothing came back."""
    for method in ("HEAD", "GET"):
        request = urllib.request.Request(
            url, method=method, headers={"User-Agent": USER_AGENT}
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                return response.status, ""
        except urllib.error.HTTPError as error:
            if method == "HEAD" and error.code in RETRY_WITH_GET:
                continue
            return error.code, ""
        except urllib.error.URLError as error:
            return None, str(error.reason)
        except Exception as error:  # socket errors, bad redirects, malformed URLs
            return None, str(error)

    return None, "no response"


def main(argv):
    parser = argparse.ArgumentParser(prog="check-links")
    parser.add_argument("paths", nargs="+", help="built site directory, or .html files")
    parser.add_argument("--timeout", type=float, default=20.0, help="seconds per request")
    parser.add_argument("--workers", type=int, default=16, help="parallel requests")
    parser.add_argument(
        "--skip",
        action="append",
        default=[],
        metavar="SUBSTRING",
        help="skip URLs containing this (repeatable)",
    )
    arguments = parser.parse_args(argv)

    pages = []
    for argument in arguments.paths:
        path = Path(argument)
        pages.extend(sorted(path.rglob("*.html")) if path.is_dir() else [path])

    if not pages:
        print("check-links: no .html files found", file=sys.stderr)
        return 1

    links = external_links(pages)
    for url in list(links):
        if any(fragment in url for fragment in arguments.skip):
            del links[url]

    if not links:
        print("check-links: no external links found")
        return 0

    failed = 0
    with concurrent.futures.ThreadPoolExecutor(max_workers=arguments.workers) as pool:
        results = pool.map(lambda url: status(url, arguments.timeout), sorted(links))

        for url, (code, note) in zip(sorted(links), results):
            if code is not None and 200 <= code < 400:
                print(f"{code}  {url}")
                continue

            failed += 1
            print(f"{code or 'ERR'}  {url}{f'  ({note})' if note else ''}")
            for page in sorted(set(links[url])):
                print(f"      linked from {page}")

    print(f"\nchecked {len(links)} external links, {failed} failed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
