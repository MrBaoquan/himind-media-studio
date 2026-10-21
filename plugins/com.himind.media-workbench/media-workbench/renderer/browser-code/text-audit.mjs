import http from 'node:http';
import path from 'node:path';
import {existsSync, statSync, createReadStream, mkdirSync, rmSync} from 'node:fs';
import {createRequire} from 'node:module';
import {fileURLToPath} from 'node:url';

const args = {};
for (const raw of process.argv.slice(2)) {
  if (!raw.startsWith('--')) continue;
  const [key, ...rest] = raw.slice(2).split('=');
  args[key] = rest.length ? rest.join('=') : true;
}
const source = path.resolve(String(args.source || ''));
if (!source || !existsSync(source)) {
  console.error('source is required');
  process.exit(2);
}
const pagePath = String(args.page || 'reel/index.html').replace(/^\/+/, '');
const times = String(args.times || '1.2,3.5,9,16,23,28').split(',').map(Number).filter(Number.isFinite);
const require = createRequire(import.meta.url);
const puppeteer = require(path.join(source, 'node_modules', 'puppeteer-core'));

function findChrome() {
  if (process.env.CHROME_PATH && existsSync(process.env.CHROME_PATH)) return process.env.CHROME_PATH;
  const roots = [process.env.ProgramFiles, process.env['ProgramFiles(x86)'], process.env.LOCALAPPDATA].filter(Boolean);
  for (const root of roots) {
    for (const rel of ['Google/Chrome/Application/chrome.exe', 'Microsoft/Edge/Application/msedge.exe']) {
      const candidate = path.join(root, rel);
      if (existsSync(candidate)) return candidate;
    }
  }
  return '';
}

function mime(file) {
  const ext = path.extname(file).toLowerCase();
  return {'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.css':'text/css; charset=utf-8','.json':'application/json; charset=utf-8','.png':'image/png','.jpg':'image/jpeg','.jpeg':'image/jpeg','.webp':'image/webp','.svg':'image/svg+xml'}[ext] || 'application/octet-stream';
}

function serve(root) {
  const server = http.createServer((req, res) => {
    const requestPath = decodeURIComponent((req.url || '/').split('?')[0]);
    const target = path.resolve(root, '.' + requestPath);
    if (!target.startsWith(root)) return res.writeHead(403).end('forbidden');
    let file = target;
    if (existsSync(file) && statSync(file).isDirectory()) file = path.join(file, 'index.html');
    if (!existsSync(file)) return res.writeHead(404).end('not found');
    res.writeHead(200, {'content-type': mime(file), 'cache-control': 'no-store'});
    createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(0, '127.0.0.1', () => resolve({server, port: server.address().port})));
}

const chrome = findChrome();
if (!chrome) {
  console.error('Chrome not found');
  process.exit(2);
}
const {server, port} = await serve(source);
const chromeProfile = path.join(source, 'out', '.text-audit-chrome-profile-' + process.pid);
mkdirSync(chromeProfile, {recursive: true});
const browser = await puppeteer.launch({executablePath: chrome, userDataDir: chromeProfile, headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage', '--hide-scrollbars', '--force-device-scale-factor=1']});
try {
  const page = await browser.newPage();
  await page.setViewport({width: 1080, height: 1920, deviceScaleFactor: 1});
  page.on('pageerror', (err) => console.error('PAGEERROR', err?.message || String(err)));
  await page.evaluateOnNewDocument(() => {
    window.__TEXT_LOG__ = [];
    const original = CanvasRenderingContext2D.prototype.fillText;
    CanvasRenderingContext2D.prototype.fillText = function(text, x, y, ...rest) {
      const value = String(text);
      const metrics = this.measureText(value);
      const size = Number((this.font.match(/(\d+(?:\.\d+)?)px/) || [])[1] || 16);
      let rawLeft = x;
      if (this.textAlign === 'center') rawLeft = x - metrics.width / 2;
      if (this.textAlign === 'right') rawLeft = x - metrics.width;
      const rawRight = rawLeft + metrics.width;
      const rawTop = y - size * 0.9;
      const rawBottom = y + size * 0.28;
      const matrix = this.getTransform();
      const corners = [[rawLeft, rawTop], [rawRight, rawTop], [rawRight, rawBottom], [rawLeft, rawBottom]].map(([px, py]) => [matrix.a * px + matrix.c * py + matrix.e, matrix.b * px + matrix.d * py + matrix.f]);
      const left = Math.min(...corners.map(([px]) => px));
      const right = Math.max(...corners.map(([px]) => px));
      const top = Math.min(...corners.map(([, py]) => py));
      const bottom = Math.max(...corners.map(([, py]) => py));
      window.__TEXT_LOG__.push({canvas: this.canvas.id, text: value, left, right, top, bottom, alpha: this.globalAlpha, font: this.font, fontSize: size});
      return original.call(this, text, x, y, ...rest);
    };
  });
  await page.goto(`http://127.0.0.1:${port}/${pagePath}`, {waitUntil: 'networkidle0'});
  await page.waitForFunction('window.__READY__ === true && typeof window.renderAt === "function"', {timeout: 30000});
  const scenes = [];
  for (const time of times) {
    const meta = await page.evaluate((t) => window.__AUDIT_META__ ? window.__AUDIT_META__(t) : null, time);
    const entries = await page.evaluate((t) => {
      window.__TEXT_LOG__ = [];
      window.renderAt(t);
      return window.__TEXT_LOG__;
    }, time);
    const texts = entries.filter((entry) => entry.alpha > 0.45);
    scenes.push({time, title: meta?.title || '', caption: meta?.caption || '', texts});
  }
  console.log(JSON.stringify({schema: 'himind.video.text-audit.v1', width: 1080, height: 1920, scenes}, null, 2));
} finally {
  await browser.close();
  server.close();
  rmSync(chromeProfile, {recursive: true, force: true});
}
