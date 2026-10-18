import {createRequire} from 'node:module';
import http from 'node:http';
import path from 'node:path';
import {existsSync, statSync} from 'node:fs';
import {createReadStream} from 'node:fs';
import {fileURLToPath} from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '..');
const template = process.env.CELESTIAL_TEMPLATE || path.join(root, 'style-packs', 'celestial-seasons', 'template');
const require = createRequire(import.meta.url);
const puppeteerRoot = process.env.PUPPETEER_CORE_DIR || path.join(template, 'node_modules', 'puppeteer-core');
const puppeteer = require(puppeteerRoot);

function findChrome() {
  if (process.env.CHROME_PATH && existsSync(process.env.CHROME_PATH)) return process.env.CHROME_PATH;
  const roots = [process.env.ProgramFiles, process.env['ProgramFiles(x86)'], process.env.LOCALAPPDATA].filter(Boolean);
  for (const base of roots) {
    for (const rel of ['Google/Chrome/Application/chrome.exe', 'Microsoft/Edge/Application/msedge.exe']) {
      const candidate = path.join(base, rel);
      if (existsSync(candidate)) return candidate;
    }
  }
  return '';
}

function startServer() {
  const server = http.createServer((req, res) => {
    const requestPath = decodeURIComponent((req.url || '/').split('?')[0]);
    const resolved = path.resolve(template, '.' + requestPath);
    if (!resolved.startsWith(template)) return res.writeHead(403).end('forbidden');
    let file = resolved;
    if (existsSync(file) && statSync(file).isDirectory()) file = path.join(file, 'index.html');
    if (!existsSync(file)) return res.writeHead(404).end('not found');
    const ext = path.extname(file).toLowerCase();
    const mime = {'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.json':'application/json; charset=utf-8'}[ext] || 'application/octet-stream';
    res.writeHead(200, {'content-type': mime, 'cache-control': 'no-store'});
    createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(0, '127.0.0.1', () => resolve({server, port: server.address().port})));
}

const samples = [
  {t: 1.2, scene: 'intro', title: '为什么地球有四季？', subtitle: '30 秒看懂：不是离太阳远近', marker: '00 / 05'},
  {t: 3.5, scene: 'orbit', title: '地球在公转', marker: '01 / 05'},
  {t: 9, scene: 'tilt', title: '地轴倾斜 23.44°', marker: '02 / 05'},
  {t: 16, scene: 'north', title: '北半球夏至', marker: '03 / 05'},
  {t: 23, scene: 'south', title: '南半球夏至', marker: '04 / 05'},
  {t: 28, scene: 'conclusion', title: '四季 = 地轴倾斜 + 公转', marker: '05 / 05'},
];
const common = ['SCHEMATIC / NOT TO SCALE', 'NASA Earth Fact Sheet · NOAA Seasons', '30s / 1080×1920'];
const facts = ['23.44°', '0.0167', '1.47–1.52 亿 km'];

function overlap(a, b) {
  const left = Math.max(a.left, b.left), right = Math.min(a.right, b.right);
  const top = Math.max(a.top, b.top), bottom = Math.min(a.bottom, b.bottom);
  if (right <= left || bottom <= top) return 0;
  const area = (right - left) * (bottom - top);
  const smaller = Math.min((a.right - a.left) * (a.bottom - a.top), (b.right - b.left) * (b.bottom - b.top));
  return smaller > 0 ? area / smaller : 0;
}

const chrome = findChrome();
if (!chrome) throw new Error('Chrome not found; set CHROME_PATH');
const {server, port} = await startServer();
const browser = await puppeteer.launch({executablePath: chrome, headless: 'new', args: ['--no-sandbox', '--disable-dev-shm-usage']});
const failures = [];
try {
  const page = await browser.newPage();
  await page.setViewport({width: 1080, height: 1920, deviceScaleFactor: 1});
  await page.evaluateOnNewDocument(() => {
    window.__TEXT_LOG__ = [];
    const original = CanvasRenderingContext2D.prototype.fillText;
    CanvasRenderingContext2D.prototype.fillText = function(text, x, y, ...rest) {
      const value = String(text);
      const metrics = this.measureText(value);
      const size = Number((this.font.match(/(\d+(?:\.\d+)?)px/) || [])[1] || 16);
      let left = x;
      if (this.textAlign === 'center') left = x - metrics.width / 2;
      if (this.textAlign === 'right') left = x - metrics.width;
      window.__TEXT_LOG__.push({canvas: this.canvas.id, text: value, x, y, left, right: left + metrics.width, top: y - size * 0.9, bottom: y + size * 0.28, font: this.font, align: this.textAlign, alpha: this.globalAlpha});
      return original.call(this, text, x, y, ...rest);
    };
  });
  await page.goto(`http://127.0.0.1:${port}/reel/index.html`, {waitUntil: 'networkidle0'});
  await page.waitForFunction('window.__READY__ === true', {timeout: 30000});
  const report = [];
  for (const sample of samples) {
    const entries = await page.evaluate((t) => {
      window.__TEXT_LOG__ = [];
      window.renderAt(t);
      return window.__TEXT_LOG__;
    }, sample.t);
    const visible = entries.filter((e) => e.alpha > 0.45 && e.canvas === 'overlay');
    const texts = visible.map((e) => e.text);
    const missing = [sample.title, sample.marker, ...(sample.subtitle ? [sample.subtitle] : []), ...common].filter((text) => !texts.some((value) => value.includes(text)));
    if (sample.scene === 'tilt' || sample.scene === 'north' || sample.scene === 'south' || sample.scene === 'conclusion') {
      for (const fact of facts) if (!texts.some((value) => value.includes(fact))) missing.push(fact);
    }
    if (sample.scene === 'conclusion' && !texts.some((value) => value.includes('NOT DISTANCE'))) missing.push('NOT DISTANCE');

    const outOfBounds = visible.filter((e) => e.left < -1 || e.right > 1081 || e.top < -1 || e.bottom > 1921).map((e) => e.text);
    const overlaps = [];
    for (let i = 0; i < visible.length; i += 1) {
      for (let j = i + 1; j < visible.length; j += 1) {
        const ratio = overlap(visible[i], visible[j]);
        if (ratio > 0.28) overlaps.push({a: visible[i].text, b: visible[j].text, ratio: Number(ratio.toFixed(2))});
      }
    }
    if (missing.length) failures.push({t: sample.t, missing});
    if (outOfBounds.length) failures.push({t: sample.t, outOfBounds});
    if (overlaps.length) failures.push({t: sample.t, overlaps});
    report.push({t: sample.t, scene: sample.scene, title: sample.title, textCount: visible.length, missing, outOfBounds, overlaps});
  }
  console.log(JSON.stringify({ok: failures.length === 0, samples: report, failures}, null, 2));
  if (failures.length) process.exitCode = 1;
} finally {
  await browser.close();
  server.close();
}

