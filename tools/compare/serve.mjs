// Serve the comparison output with caching off, so a rebuild always shows up on reload.
//   node tools/compare/serve.mjs [dir] [port]     (defaults: _site/compare, 8093)
//   PREFIX=/FOSDEM-website/compare node tools/compare/serve.mjs   (serve under a Pages-style subpath)
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(process.argv[2] || '_site/compare');
const port = Number(process.argv[3] || process.env.PORT || 8093);
const prefix = (process.env.PREFIX || '').replace(/\/+$/, '');
const types = {
  '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.js': 'text/javascript', '.json': 'application/json',
  '.svg': 'image/svg+xml', '.xml': 'application/xml', '.ics': 'text/calendar', '.png': 'image/png',
  '.jpg': 'image/jpeg', '.webp': 'image/webp', '.woff': 'font/woff', '.woff2': 'font/woff2', '.ttf': 'font/ttf',
  '.txt': 'text/plain; charset=utf-8',
};

http.createServer((req, res) => {
  let pathname = decodeURIComponent(new URL(req.url, 'http://x').pathname);
  if (prefix) {
    if (pathname !== prefix && !pathname.startsWith(prefix + '/')) { res.writeHead(404).end('not found'); return; }
    pathname = pathname.slice(prefix.length) || '/';
  }
  let file = path.join(root, pathname);
  if (file !== root && !file.startsWith(root + path.sep)) { res.writeHead(403).end(); return; }
  if (fs.existsSync(file) && fs.statSync(file).isDirectory()) file = path.join(file, 'index.html');
  if (!fs.existsSync(file)) { res.writeHead(404).end('not found'); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream', 'cache-control': 'no-store' });
  fs.createReadStream(file).pipe(res);
}).listen(port, '127.0.0.1', () => console.log(`compare: http://127.0.0.1:${port}${prefix}/  (${root})`));
