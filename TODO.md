# Accessibility TODO

## Browsers
- [x] WebKit: downloaded and runs (`A11Y_BROWSERS=webkit npm run test:a11y`). Skip-link, focus, screen reader and axe results match Chromium. macOS WebKit needs Option+Tab to reach links (handled in `tests/keys.js`).
- [ ] Firefox: downloaded but will not launch from the assistant's sandbox ("Could not find profile folder"). Run `A11Y_BROWSERS=firefox npm run test:a11y` in a normal terminal, or add Firefox to the Linux CI job.
- [ ] Link hover and visited colours (proposed: `--c-link-visited`, `--c-link-hover`, thicker underline on hover; all pass 4.5:1 in both themes).
- [ ] `layouts/speaker.html` sets `alt="{{ $speaker.name }}"` on the speaker photo; if the name is already the heading next to it, consider `alt=""`.

## Mobile testing (planned)
- [ ] Playwright device projects with touch and mobile UA: iPhone 14 (WebKit), Pixel 7 (Chromium), one landscape run.
- [ ] Reflow test: no horizontal scroll at 320px and at 400% zoom (WCAG 1.4.10); viewport meta must not block pinch zoom (1.4.4).
- [ ] Orientation check, portrait and landscape (1.3.4).
- [ ] Touch target test: links and buttons at least 24x24 CSS px (2.5.8). Known offenders: schedule A-Z anchors, day/room links.
- [ ] SkipTo on touch: it is set to `popup` (shown only on keyboard focus). Check whether VoiceOver and TalkBack users can reach it by swiping; switch to `static` if not.

## Known failing checks (from `npm run test:a11y`)
- [ ] axe `target-size`: 11 elements on the full-site schedule and speaker pages.
- [ ] axe `color-contrast` in dark theme: 13 elements, light text on track-colour cells (`--c2`..`--c10`).
- [ ] Focus `cantTell`: SkipTo button, Leaflet map controls and attribution links, links over image backgrounds.
