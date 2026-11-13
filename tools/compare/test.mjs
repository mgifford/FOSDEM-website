// SPDX-License-Identifier: MIT
// Run the automated checks for every built comparison and write results.json next to variants.json.
//
//   node tools/compare/test.mjs                 # serves _site/compare itself
//   BASE_URL=https://user.github.io/repo/compare/ node tools/compare/test.mjs    # test a deployment
//   ONLY=skipto node tools/compare/test.mjs
//
// For every check listed on a variant, the check must FAIL on "before" (otherwise it cannot detect
// the problem) and PASS on "after". Anything else is reported and the exit status is 1.
import { chromium } from '@playwright/test';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { checks } from './checks.mjs';

const here = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.resolve(process.env.OUT || path.join(here, '../../_site/compare'));
const only = process.env.ONLY;
let base = process.env.BASE_URL && process.env.BASE_URL.replace(/\/?$/, '/');
let server;

// The pages were built for PUBLIC_BASE (for example /FOSDEM-website/compare/ on Pages) and use
// root-relative URLs, so the local server must mount the build at that same path.
async function startServer(publicBase) {
  const port = 8095 + Math.floor(Math.random() * 500);
  const prefix = new URL(publicBase).pathname.replace(/\/+$/, '');
  server = spawn(process.execPath, [path.join(here, 'serve.mjs'), outDir, String(port)], { stdio: 'ignore', env: { ...process.env, PREFIX: prefix } });
  base = `http://127.0.0.1:${port}${prefix}/`;
  for (let i = 0; i < 50; i++) {
    try { if ((await fetch(base + 'variants.json')).ok) return; } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error('could not start the local server');
}

async function run(browser, id, url) {
  try { return await checks[id].run(browser, url); }
  catch (error) { return { pass: false, errored: true, summary: `error: ${error.message}`, details: [] }; }
}

try {
  const data = JSON.parse(fs.readFileSync(path.join(outDir, 'variants.json'), 'utf8'));
  if (!base) await startServer(data.public_base || 'http://127.0.0.1/');
  const browser = await chromium.launch();
  const beforeCache = new Map();
  const results = {};
  let ok = true;

  for (const v of data.variants) {
    if (only && v.slug !== only) continue;
    const entries = [];
    for (const id of v.checks || []) {
      if (!checks[id]) { entries.push({ id, ok: false, error: 'unknown check' }); continue; }
      const beforeUrl = `${base}before/${v.before_sha}/`;
      const key = `${v.before_sha}:${id}`;
      if (!beforeCache.has(key)) beforeCache.set(key, await run(browser, id, beforeUrl));
      const before = beforeCache.get(key);
      const after = await run(browser, id, `${base}after/${v.slug}/`);
      // An errored run is never a pass or a fail: it is reported as UNKNOWN and fails the build.
      const good = !before.errored && !after.errored && before.pass === false && after.pass === true;
      entries.push({ id, title: checks[id].title, before, after, ok: good });
      const mark = good ? 'ok  ' : 'FAIL';
      console.log(`${mark} ${v.slug} / ${id}: before ${before.pass ? 'passes' : 'fails'} (${before.summary}); after ${after.pass ? 'passes' : 'fails'} (${after.summary})`);
      if (!good) {
        ok = false;
        if (before.pass) console.log('     the check passes on upstream, so it cannot detect the problem');
        if (!after.pass) after.details.slice(0, 6).forEach((d) => console.log(`     after: ${d}`));
      }
    }
    if (!entries.length) { console.log(`FAIL ${v.slug}: no automated checks defined`); ok = false; }
    results[v.slug] = { ok: entries.length > 0 && entries.every((e) => e.ok), checks: entries };
  }

  const output = {
    ok,
    generated: new Date().toISOString(),
    browser: `Chromium ${browser.version()}`,
    note: 'Recorded at build time with narrow mechanical checks. Not a WCAG conformance test.',
    results,
  };
  await browser.close();
  fs.writeFileSync(path.join(outDir, 'results.json'), JSON.stringify(output, null, 2));
  console.log(ok ? `all checks behave as expected (${output.browser})` : 'some checks did not behave as expected');
  process.exitCode = ok ? 0 : 1;
} finally {
  if (server) server.kill();
}
