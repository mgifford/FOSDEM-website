// Shared helpers: which pages to scan, how a finding is identified, and the baseline of known findings.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const here = path.dirname(fileURLToPath(import.meta.url));
export const baselineFile = path.join(here, 'baseline.json');

// WCAG 2.0 to 2.2 levels A and AA. A clean result means no violations in this scope, not conformance.
export const TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];

// Every page in the built site, as a path relative to the site root ('' is the home page).
// Hugo alias pages only redirect, so they are skipped.
export function listPages(siteDir) {
  const root = path.resolve(siteDir);
  const pages = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) { walk(full); continue; }
      const isPage = entry.name === 'index.html' || (dir === root && entry.name === '404.html');
      if (!isPage) continue;
      if (/http-equiv=["']?refresh/i.test(fs.readFileSync(full, 'utf8').slice(0, 2000))) continue;
      if (entry.name === '404.html') { pages.push('404.html'); continue; }
      const rel = path.relative(root, dir).split(path.sep).join('/');
      pages.push(rel ? `${rel}/` : '');
    }
  };
  walk(root);
  return pages.sort();
}

// A finding is the axe rule plus the element's selector, so a new instance of a known rule is still new.
export function fingerprints(results) {
  return results.violations.flatMap((v) => v.nodes.map((n) => `${v.id} | ${n.target.join(' ')}`));
}

export function loadBaseline() {
  if (!fs.existsSync(baselineFile)) return {};
  return JSON.parse(fs.readFileSync(baselineFile, 'utf8')).pages || {};
}

export function newFindings(found, known = []) {
  const allowed = new Set(known);
  return found.filter((f) => !allowed.has(f));
}
