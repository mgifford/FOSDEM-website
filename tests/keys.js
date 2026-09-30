// The key that moves focus to the next focusable element, links included.
// In WebKit on macOS, plain Tab skips links unless Safari's "Press Tab to highlight each item on a
// webpage" is on, so use Option+Tab there (as a Safari user without that setting must).
exports.tabKey = (browserName) => (browserName === 'webkit' && process.platform === 'darwin' ? 'Alt+Tab' : 'Tab');
