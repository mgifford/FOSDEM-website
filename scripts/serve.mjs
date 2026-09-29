// Serve ./_site under the GitHub Pages subpath so relURL links resolve as they do live.
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';

const port = Number(process.env.PORT || 8080);
const prefix = process.env.PREFIX || '/FOSDEM-website';
const root = path.resolve('_site');
const types = {
  '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.js': 'text/javascript',
  '.json': 'application/json', '.svg': 'image/svg+xml', '.xml': 'application/xml',
  '.ics': 'text/calendar', '.png': 'image/png', '.jpg': 'image/jpeg', '.webp': 'image/webp',
  '.woff': 'font/woff', '.woff2': 'font/woff2', '.ttf': 'font/ttf',
};

http.createServer((req, res) => {
  const url = new URL(req.url, 'http://x');
  let rel = decodeURIComponent(url.pathname);
  if (rel !== prefix && !rel.startsWith(prefix + '/')) { res.writeHead(404).end('not found'); return; }
  let file = path.join(root, rel.slice(prefix.length));
  if (!file.startsWith(root)) { res.writeHead(403).end(); return; }
  if (fs.existsSync(file) && fs.statSync(file).isDirectory()) file = path.join(file, 'index.html');
  if (!fs.existsSync(file)) { res.writeHead(404).end('not found'); return; }
  res.writeHead(200, { 'content-type': types[path.extname(file)] || 'application/octet-stream' });
  fs.createReadStream(file).pipe(res);
}).listen(port, '127.0.0.1', () => console.log(`serving ${root} at http://127.0.0.1:${port}${prefix}/`));
