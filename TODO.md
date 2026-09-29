# Accessibility TODO

## Mobile testing (planned, needs a WebKit download of a few hundred MB)
- [ ] Playwright device projects with touch and mobile UA: iPhone 14 (WebKit), Pixel 7 (Chromium), one landscape run.
- [ ] Reflow test: no horizontal scroll at 320px and at 400% zoom (WCAG 1.4.10); viewport meta must not block pinch zoom (1.4.4).
- [ ] Orientation check, portrait and landscape (1.3.4).
- [ ] Touch target test: links and buttons at least 24x24 CSS px (2.5.8). Known offenders: schedule A-Z anchors, day/room links.
- [ ] SkipTo on touch: it is set to `popup` (shown only on keyboard focus). Check whether VoiceOver and TalkBack users can reach it by swiping; switch to `static` if not.

## Known failing checks (from `npm run test:a11y`)
- [ ] axe `target-size`: 11 elements on the full-site schedule and speaker pages.
- [ ] axe `color-contrast` in dark theme: 13 elements, light text on track-colour cells (`--c2`..`--c10`).
- [ ] Focus `cantTell`: SkipTo button, Leaflet map controls and attribution links, links over image backgrounds.
