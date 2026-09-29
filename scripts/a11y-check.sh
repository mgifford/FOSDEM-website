#!/usr/bin/env bash
# Serve _site/ under the Pages subpath and run pa11y-ci (axe + HTML_CodeSniffer)
# over every built page. Run scripts/build-pages.sh first.
set -euo pipefail
cd "$(dirname "$0")/.."
PORT="${PORT:-8080}"
PREFIX="/FOSDEM-website"
serve=$(mktemp -d)
ln -s "$PWD/_site" "$serve${PREFIX}"
python3 -m http.server "$PORT" --bind 127.0.0.1 -d "$serve" >/dev/null 2>&1 &
srv=$!
trap 'kill $srv 2>/dev/null; rm -rf "$serve"' EXIT
sleep 1
urls=$(cd _site && find . -name '*.html' | sed 's|^\./||; s|index\.html$||' | sort \
  | sed "s|^|http://127.0.0.1:${PORT}${PREFIX}/|")
mkdir -p a11y-report
node -e '
const urls = process.argv[1].trim().split("\n");
const cfg = {
  defaults: {
    runners: ["axe", "htmlcs"],
    standard: "WCAG2AA",
    timeout: 60000,
    chromeLaunchConfig: { args: ["--no-sandbox"] }
  },
  urls
};
require("fs").writeFileSync("a11y-report/pa11yci.json", JSON.stringify(cfg, null, 2));
' "$urls"
npx --yes pa11y-ci --config a11y-report/pa11yci.json --json > a11y-report/results.json || status=$?
node -e '
const r = require("./a11y-report/results.json");
console.log(`pages: ${r.total}, passed: ${r.passes}, errors: ${r.errors}`);
for (const [u, issues] of Object.entries(r.results))
  for (const i of issues) console.log(`${u}\n  ${i.code}: ${i.message}\n  ${i.selector}`);
'
exit "${status:-0}"
