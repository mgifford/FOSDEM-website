// Rewrite baseline.json from a full run with A11Y_UPDATE_BASELINE=1 (npm run a11y:baseline).
// Review the diff: every entry is a known barrier, not an accepted one, and should shrink as fixes land.
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { baselineFile } from './fingerprint.mjs';

const dir = path.join('a11y-report', 'found');
if (!fs.existsSync(dir)) {
  console.error('No results in a11y-report/found. Run: npm run a11y:baseline');
  process.exit(1);
}

const pages = {};
for (const file of fs.readdirSync(dir)) {
  const { page, found } = JSON.parse(fs.readFileSync(path.join(dir, file), 'utf8'));
  pages[page] = [...new Set([...(pages[page] || []), ...found])].sort();
}
for (const key of Object.keys(pages)) if (!pages[key].length) delete pages[key];

const previous = fs.existsSync(baselineFile) ? JSON.parse(fs.readFileSync(baselineFile, 'utf8')) : {};
const git = (...args) => { try { return execFileSync('git', args, { encoding: 'utf8' }).trim(); } catch { return 'unknown'; } };
const axeVersion = JSON.parse(fs.readFileSync('node_modules/axe-core/package.json', 'utf8')).version;

const meta = {
  purpose:
    'Findings that already exist on main. The scan fails only on findings that are NOT listed here, so new regressions ' +
    'are blocked while existing ones are fixed. This is a temporary inventory, not an acceptance of these barriers.',
  scope: 'union over light and dark at 1280, 820, 390 and 320 px, plus the site theme toggle on the home page',
  owner: 'unassigned',
  review_by: 'unset',
  remove_entries_when: 'the changes that fix them are merged; run npm run a11y:baseline and review the diff',
  ...previous._meta,
  generated: new Date().toISOString().slice(0, 10),
  commit: git('rev-parse', '--short', 'HEAD'),
  axe_core: axeVersion,
  total_findings: Object.values(pages).reduce((n, list) => n + list.length, 0),
};

fs.writeFileSync(baselineFile, `${JSON.stringify({ _meta: meta, pages }, null, 2)}\n`);
console.log(`baseline.json: ${meta.total_findings} findings on ${Object.keys(pages).length} pages`);
