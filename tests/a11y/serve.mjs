// Serve the built site under its production path, on the loopback interface only.
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';

const root = path.resolve(process.env.A11Y_SITE_DIR || 'public');
const prefix = (process.env.A11Y_BASE_PATH || '/').replace(/\/+$/, '');
const port = Number(process.env.A11Y_PORT || 8765);

if (!fs.existsSync(root)) {
  console.error(`a11y: ${root} does not exist. Build the site first (hugo build), or run: just a11y`);
  process.exit(1);
}

const types = {
  '.html': 'text/html; charset=utf-8', '.css': 'text/css; charset=utf-8', '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json', '.svg': 'image/svg+xml', '.xml': 'application/xml', '.ics': 'text/calendar',
  '.png': 'image/png', '.jpg': 'image/jpeg', '.webp': 'image/webp', '.ico': 'image/x-icon',
  '.woff': 'font/woff', '.woff2': 'font/woff2', '.ttf': 'font/ttf', '.txt': 'text/plain; charset=utf-8',
};

http.createServer((req, res) => {
  let pathname;
  try { pathname = decodeURIComponent(new URL(req.url, 'http://x').pathname); } catch { res.writeHead(400).end(); return; }
  if (prefix && pathname !== prefix && !pathname.startsWith(`${prefix}/`)) { res.writeHead(404).end('not found'); return; }
  let file = path.join(root, pathname.slice(prefix.length));
  if (file !== root && !file.startsWith(root + path.sep)) { res.writeHead(403).end(); return; }
  if (fs.existsSync(file) && fs.statSync(file).isDirectory()) file = path.join(file, 'index.html');
  if (!fs.existsSync(file)) { res.writeHead(404).end('not found'); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
}).listen(port, '127.0.0.1', () => console.log(`a11y: ${root} at http://127.0.0.1:${port}${prefix}/`));
