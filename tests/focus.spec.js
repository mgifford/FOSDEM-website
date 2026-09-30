const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const { tabKey } = require('./keys');

// Focus-indicator check: real Tab presses, one measurement per focus stop.
//   WCAG 2.4.7 Focus Visible: some indicator must appear on focus.
//   WCAG 1.4.11 Non-text Contrast: an outline/ring must reach 3:1 against the colour next to it.
//   WCAG 1.4.3: if focus changes the text colour, the focused text must still reach 4.5:1.
// Measured from computed styles, not pixels. Where the background is an image or gradient the
// stop is reported as "cantTell" (never counted as pass or fail). WCAG 2.4.13 (AAA) is not gated.
const PAGES = [
  '', 'news/', 'news/call-for-stands/', 'full/', 'full/about/',
  'full/schedule/', 'full/schedule/day/saturday/', 'full/schedule/speakers/',
];
const MAX_STOPS = 200;
const RING_MIN = 3;
const TEXT_MIN = 4.5;

// Runs in the page: measure the currently focused element.
function measureFocused() {
  const ctx = document.createElement('canvas').getContext('2d', { willReadFrequently: true });
  ctx.canvas.width = ctx.canvas.height = 1;
  const rgba = (css) => {
    ctx.clearRect(0, 0, 1, 1);
    ctx.fillStyle = '#000';
    ctx.fillStyle = css;
    ctx.fillRect(0, 0, 1, 1);
    const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data;
    return { r, g, b, a: a / 255 };
  };
  const over = (top, bottom) => {
    const a = top.a + bottom.a * (1 - top.a);
    if (a === 0) return { r: 0, g: 0, b: 0, a: 0 };
    const mix = (t, b2) => (t * top.a + b2 * bottom.a * (1 - top.a)) / a;
    return { r: mix(top.r, bottom.r), g: mix(top.g, bottom.g), b: mix(top.b, bottom.b), a };
  };
  const lum = ({ r, g, b }) => {
    const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; };
    return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
  };
  const ratio = (a, b) => {
    const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
  };
  // Effective opaque background behind `el` (its own layers included when withSelf).
  const backdrop = (el, withSelf) => {
    const layers = [];
    let unknown = false;
    for (let n = withSelf ? el : el.parentElement; n; n = n.parentElement) {
      const cs = getComputedStyle(n);
      if (cs.backgroundImage !== 'none') unknown = true;
      const c = rgba(cs.backgroundColor);
      if (c.a > 0) layers.push(c);
      if (c.a === 1) break;
    }
    let out = { r: 255, g: 255, b: 255, a: 1 };
    for (let i = layers.length - 1; i >= 0; i--) out = over(layers[i], out);
    return { color: out, unknown };
  };
  const snap = (el) => {
    const cs = getComputedStyle(el);
    return {
      outlineStyle: cs.outlineStyle, outlineWidth: parseFloat(cs.outlineWidth) || 0,
      outlineColor: cs.outlineColor, outlineOffset: parseFloat(cs.outlineOffset) || 0,
      boxShadow: cs.boxShadow, color: cs.color, backgroundColor: cs.backgroundColor,
      textDecoration: cs.textDecorationLine, borderColor: cs.borderTopColor,
      borderWidth: parseFloat(cs.borderTopWidth) || 0,
    };
  };

  const el = document.activeElement;
  if (!el || el === document.body || el === document.documentElement) return null;
  const rect = el.getBoundingClientRect();
  const id = el.tagName.toLowerCase()
    + (el.id ? `#${el.id}` : '')
    + (el.getAttribute('href') ? `[href="${el.getAttribute('href')}"]` : '')
    + (el.textContent ? ` "${el.textContent.trim().replace(/\s+/g, ' ').slice(0, 40)}"` : '');
  window.__seen = window.__seen || new WeakSet();
  const repeat = window.__seen.has(el);
  window.__seen.add(el);

  const focused = snap(el);
  const outside = backdrop(el, false);
  const inside = backdrop(el, true);
  const result = { id, repeat, visible: rect.width > 0 && rect.height > 0, indicators: [], notes: [] };

  // Outline: sits outside the box when offset >= 0, otherwise over the element itself.
  if (focused.outlineStyle !== 'none' && focused.outlineWidth > 0) {
    const c = rgba(focused.outlineColor);
    if (c.a > 0) {
      const bg = focused.outlineOffset >= 0 ? outside : inside;
      result.indicators.push({
        kind: 'outline', auto: focused.outlineStyle === 'auto', ratio: ratio(over(c, bg.color), bg.color), unknown: bg.unknown,
        detail: `${focused.outlineWidth}px ${focused.outlineStyle} ${focused.outlineColor} offset ${focused.outlineOffset}`,
      });
    }
  }
  // Box-shadow ring (first non-inset, non-blurred layer with a colour).
  if (focused.boxShadow && focused.boxShadow !== 'none') {
    const m = focused.boxShadow.match(/(rgba?\([^)]*\)|color\([^)]*\))\s+(-?\d+(?:\.\d+)?)px\s+(-?\d+(?:\.\d+)?)px\s+(\d+(?:\.\d+)?)px(?:\s+(\d+(?:\.\d+)?)px)?/);
    if (m && !/inset/.test(focused.boxShadow)) {
      const c = rgba(m[1]);
      const spread = parseFloat(m[5] || '0');
      const blur = parseFloat(m[4]);
      if (c.a > 0 && spread + blur > 0) {
        result.indicators.push({
          kind: 'box-shadow', ratio: ratio(over(c, outside.color), outside.color), unknown: outside.unknown,
          detail: focused.boxShadow,
        });
      }
    }
  }

  // Compare against the unfocused state for colour/background/underline changes.
  el.blur();
  const plain = snap(el);
  el.focus({ focusVisible: true });
  const changed = [];
  if (plain.backgroundColor !== focused.backgroundColor) changed.push('background');
  if (plain.color !== focused.color) changed.push('text colour');
  if (plain.textDecoration !== focused.textDecoration) changed.push('text-decoration');
  if (plain.borderColor !== focused.borderColor || plain.borderWidth !== focused.borderWidth) changed.push('border');
  if (plain.outlineStyle !== focused.outlineStyle || plain.outlineColor !== focused.outlineColor) changed.push('outline');
  result.changed = changed;

  // Focus-state text contrast (only meaningful when focus alters colour or background).
  if (changed.includes('background') || changed.includes('text colour')) {
    const bg = backdrop(el, true);
    result.textRatio = ratio(over(rgba(focused.color), bg.color), bg.color);
    result.textUnknown = bg.unknown;
  }
  return result;
}

// Both routes to a theme are covered: the OS preference (project colorScheme) and the manual
// toggle, which stores 'theme' in localStorage and overrides the OS preference.
const MODES = [{ name: 'os-preference', manual: false }, { name: 'manual-toggle', manual: true }];

for (const mode of MODES) test(`focus indicators are visible and have sufficient contrast (${mode.name})`, async ({ page, colorScheme, browserName }, testInfo) => {
  const scheme = colorScheme;
  if (mode.manual) {
    // Opposite of the OS scheme, so a pass proves the toggle path, not the OS path.
    await page.addInitScript((t) => { try { localStorage.setItem('theme', t); } catch {} }, scheme === 'dark' ? 'light' : 'dark');
  }
  const report = [];
  const failures = [];
  const cantTell = [];

  for (const p of PAGES) {
    await page.goto(p, { waitUntil: 'networkidle' });
    // The site sets data-theme from localStorage, else the OS scheme. Assert the theme under test really applied.
    const expected = mode.manual ? (scheme === 'dark' ? 'light' : 'dark') : scheme;
    const theme = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
    expect(theme, `/${p} rendered the wrong theme`).toBe(expected);
    await page.evaluate(() => { document.activeElement && document.activeElement.blur(); window.__seen = new WeakSet(); });
    let stops = 0;
    for (let i = 0; i < MAX_STOPS; i++) {
      await page.keyboard.press(tabKey(browserName));
      const m = await page.evaluate(measureFocused);
      if (!m || m.repeat) break;
      if (!m.visible) continue; // off-screen / zero-size stops are a separate 2.4.7 concern
      stops++;

      const ringOk = m.indicators.filter((x) => !x.unknown && x.ratio >= RING_MIN);
      const ringUnknown = m.indicators.some((x) => x.unknown);
      // Style changes other than the outline itself appearing (which is the ring, measured above).
      const otherChange = m.changed.filter((c) => c !== 'outline');
      const autoRing = m.indicators.some((x) => x.auto);
      let outcome = 'pass';
      let why = '';

      if (!ringOk.length) {
        if (ringUnknown) {
          outcome = 'cantTell';
          why = 'indicator sits over a background image/gradient';
        } else if (autoRing) {
          // outline-style:auto is drawn by the browser (two-tone in Chromium); colour alone under-reports it.
          outcome = 'cantTell';
          why = `browser-default focus ring (${m.indicators.map((x) => `${x.ratio.toFixed(2)}:1`).join(', ')} by colour); rendering is UA-specific, define an explicit :focus-visible style`;
        } else if (m.indicators.length) {
          outcome = otherChange.length ? 'cantTell' : 'fail';
          why = `focus ring below ${RING_MIN}:1 (${m.indicators.map((x) => `${x.kind} ${x.ratio.toFixed(2)}:1`).join(', ')})`
            + (otherChange.length ? `; also changes ${otherChange.join(', ')} (not contrast-measured)` : '');
        } else if (otherChange.length) {
          outcome = 'cantTell';
          why = `only a style change signals focus (${otherChange.join(', ')}); needs a visual check`;
        } else {
          outcome = 'fail';
          why = 'no visible focus indicator (no outline, ring, colour, background, border or underline change)';
        }
      }
      if (m.textRatio !== undefined && !m.textUnknown && m.textRatio < TEXT_MIN) {
        outcome = 'fail';
        why += `${why ? '; ' : ''}focused text contrast ${m.textRatio.toFixed(2)}:1 (< ${TEXT_MIN}:1)`;
      }

      const row = { page: `/${p}`, element: m.id, outcome, why, indicators: m.indicators, changed: m.changed, textRatio: m.textRatio };
      report.push(row);
      if (outcome === 'fail') failures.push(`/${p}  ${m.id}: ${why}`);
      if (outcome === 'cantTell') cantTell.push(`/${p}  ${m.id}: ${why}`);
    }
    expect(stops, `no focus stops found on /${p}`).toBeGreaterThan(0);
  }

  fs.mkdirSync('a11y-report', { recursive: true });
  fs.writeFileSync(`a11y-report/focus-${testInfo.project.name}-${mode.name}.json`, JSON.stringify(report, null, 2));
  console.log(`[${testInfo.project.name} ${mode.name}] focus stops: ${report.length}, fail: ${failures.length}, cantTell: ${cantTell.length}`);
  expect(failures, 'focus indicator failures (cantTell stops are listed in the JSON report, not failed)').toEqual([]);
});
