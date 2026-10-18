import puppeteer from 'puppeteer-core';
import http from 'node:http';
import {spawn} from 'node:child_process';
import {createReadStream, existsSync, statSync, readFileSync} from 'node:fs';
import {mkdir, readFile, rm, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const ROOT = path.dirname(fileURLToPath(import.meta.url));
const FPS = 30;
const WIDTH = 1080;
const HEIGHT = 1920;

const args = {};
for (const raw of process.argv.slice(2)) {
  if (!raw.startsWith('--')) continue;
  const [key, ...rest] = raw.slice(2).split('=');
  args[key] = rest.length ? rest.join('=') : true;
}

let extraChromeArgs = [];
try {
  extraChromeArgs = JSON.parse(process.env.HIMIND_CHROME_ARGS_JSON || '[]');
} catch { extraChromeArgs = []; }

function findChrome() {
  if (args.chrome) return String(args.chrome);
  if (process.env.CHROME_PATH) return process.env.CHROME_PATH;
  const candidates = [];
  if (process.platform === 'win32') {
    for (const root of [process.env.ProgramFiles, process.env['ProgramFiles(x86)'], process.env.LOCALAPPDATA]) {
      if (!root) continue;
      candidates.push(path.join(root, 'Google', 'Chrome', 'Application', 'chrome.exe'));
      candidates.push(path.join(root, 'Microsoft', 'Edge', 'Application', 'msedge.exe'));
    }
  } else if (process.platform === 'darwin') {
    candidates.push('/Applications/Google Chrome.app/Contents/MacOS/Google Chrome');
    candidates.push('/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge');
  } else {
    candidates.push('/usr/bin/google-chrome', '/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/microsoft-edge');
  }
  return candidates.find((candidate) => existsSync(candidate)) || '';
}

function parseTimes(value, fallback) {
  const source = String(value || fallback);
  return source.split(',').map((item) => Number.parseFloat(item.trim())).filter((value) => Number.isFinite(value));
}

function parseRange(value, fallback) {
  const source = String(value || fallback);
  const [start, end] = source.split(':').map((item) => Number.parseFloat(item.trim()));
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    throw new Error(`invalid range: ${source}`);
  }
  return [start, end];
}

function mimeFor(file) {
  const ext = path.extname(file).toLowerCase();
  return {
    '.html': 'text/html; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.css': 'text/css; charset=utf-8',
    '.json': 'application/json; charset=utf-8',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.jpeg': 'image/jpeg',
    '.webp': 'image/webp',
    '.m4a': 'audio/mp4',
    '.mp3': 'audio/mpeg',
    '.wav': 'audio/wav',
  }[ext] || 'application/octet-stream';
}

function startServer() {
  const server = http.createServer((req, res) => {
    const requestPath = decodeURIComponent((req.url || '/').split('?')[0]);
    const resolved = path.resolve(ROOT, '.' + requestPath);
    if (!resolved.startsWith(ROOT)) {
      res.writeHead(403).end('forbidden');
      return;
    }
    let file = resolved;
    if (existsSync(file) && statSync(file).isDirectory()) file = path.join(file, 'index.html');
    if (!existsSync(file)) {
      res.writeHead(404).end('not found');
      return;
    }
    res.writeHead(200, {'content-type': mimeFor(file), 'cache-control': 'no-store'});
    createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      resolve({server, port: address.port});
    });
  });
}

function run(command, commandArgs, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, commandArgs, {stdio: ['ignore', 'pipe', 'pipe'], ...options});
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (chunk) => { stdout += chunk.toString(); });
    child.stderr.on('data', (chunk) => { stderr += chunk.toString(); });
    child.on('error', reject);
    child.on('close', (code) => {
      if (code === 0) resolve({stdout, stderr});
      else reject(new Error(`${command} exited ${code}\n${stderr || stdout}`));
    });
  });
}

async function openPage(browser, port) {
  const page = await browser.newPage();
  page.on('console', (msg) => { if (msg.type() === 'error' || msg.type() === 'warning') console.log('PAGE', msg.type(), msg.text()); });
  page.on('pageerror', (err) => console.log('PAGEERROR', err?.stack || String(err)));
  page.on('requestfailed', (req) => console.log('REQFAIL', req.url(), req.failure()?.errorText || ''));
  page.on('response', (res) => { if (res.status() >= 400) console.log('HTTP', res.status(), res.url()); });
  await page.setViewport({width: WIDTH, height: HEIGHT, deviceScaleFactor: 1});
  await page.goto(`http://127.0.0.1:${port}/reel/index.html`, {waitUntil: 'networkidle0'});
  await page.waitForFunction('window.__READY__ === true', {timeout: 30000});
  return page;
}

async function shot(page, time, type = 'jpeg', quality = 0.94) {
  // renderAt() draws synchronously; do not wait for requestAnimationFrame.
  // In headless Chrome, background pages can throttle RAF and hang workers.
  await page.evaluate((t) => window.renderAt(t), time);
  return page.screenshot({type, ...(type === 'jpeg' ? {quality} : {})});
}

async function tileImages(inputPattern, output, cols, rows) {
  const ffmpeg = process.env.FFMPEG_PATH || 'ffmpeg';
  await mkdir(path.dirname(output), {recursive: true});
  await run(ffmpeg, [
    '-y', '-hide_banner', '-loglevel', 'error',
    '-framerate', '1', '-start_number', '0', '-i', inputPattern,
    '-vf', `tile=${cols}x${rows}:padding=8:margin=8`,
    '-frames:v', '1', output,
  ]);
}


async function synthesizeAmbientWav(file, duration) {
  const sampleRate = 48000;
  const channels = 2;
  const total = Math.max(1, Math.round(duration * sampleRate));
  const left = new Float32Array(total);
  const right = new Float32Array(total);
  let state = 0x9e3779b9;
  const rand = () => {
    state ^= state << 13;
    state ^= state >>> 17;
    state ^= state << 5;
    state >>>= 0;
    return state / 2147483648 - 1;
  };
  let lowL = 0, lowR = 0, airL = 0, airR = 0;
  let peak = 0;
  for (let i = 0; i < total; i += 1) {
    const t = i / sampleRate;
    const swell = 0.80 + 0.20 * Math.sin(2 * Math.PI * 0.045 * t - Math.PI / 2);
    const drift = 0.22 * Math.sin(2 * Math.PI * 0.031 * t);
    const padL =
      0.090 * Math.sin(2 * Math.PI * 110.00 * t + drift) +
      0.050 * Math.sin(2 * Math.PI * 164.81 * t + 0.35 + drift * 0.6) +
      0.034 * Math.sin(2 * Math.PI * 220.00 * t + 1.10) +
      0.019 * Math.sin(2 * Math.PI * 277.18 * t + 2.05) +
      0.011 * Math.sin(2 * Math.PI * 329.63 * t + 0.72);
    const padR =
      0.090 * Math.sin(2 * Math.PI * 110.18 * t + 0.21 + drift) +
      0.050 * Math.sin(2 * Math.PI * 164.62 * t + 0.92 + drift * 0.6) +
      0.034 * Math.sin(2 * Math.PI * 220.35 * t + 1.71) +
      0.019 * Math.sin(2 * Math.PI * 276.96 * t + 2.51) +
      0.011 * Math.sin(2 * Math.PI * 329.21 * t + 1.24);
    const nL = rand();
    const nR = rand();
    lowL = lowL * 0.998 + nL * 0.002;
    lowR = lowR * 0.998 + nR * 0.002;
    airL = airL * 0.982 + (nL - lowL) * 0.018;
    airR = airR * 0.982 + (nR - lowR) * 0.018;
    const air = 0.62 * Math.sin(2 * Math.PI * 0.028 * t) + 0.38;
    const shimmer = 0.006 * Math.sin(2 * Math.PI * 880 * t + drift * 1.4) * (0.45 + 0.55 * Math.sin(2 * Math.PI * 0.037 * t));
    const fadeIn = Math.min(1, t / 2.4);
    const fadeOut = Math.min(1, (duration - t) / 3.4);
    const env = Math.max(0, Math.min(fadeIn, fadeOut)) * swell;
    left[i] = (padL + 0.075 * (0.55 * airL + 0.45 * lowL) * air + shimmer) * env;
    right[i] = (padR + 0.075 * (0.55 * airR + 0.45 * lowR) * air + shimmer) * env;
    peak = Math.max(peak, Math.abs(left[i]), Math.abs(right[i]));
  }
  const gain = peak > 0 ? 0.21 / peak : 1;
  const dataSize = total * channels * 2;
  const out = Buffer.alloc(44 + dataSize);
  out.write('RIFF', 0);
  out.writeUInt32LE(36 + dataSize, 4);
  out.write('WAVE', 8);
  out.write('fmt ', 12);
  out.writeUInt32LE(16, 16);
  out.writeUInt16LE(1, 20);
  out.writeUInt16LE(channels, 22);
  out.writeUInt32LE(sampleRate, 24);
  out.writeUInt32LE(sampleRate * channels * 2, 28);
  out.writeUInt16LE(channels * 2, 32);
  out.writeUInt16LE(16, 34);
  out.write('data', 36);
  out.writeUInt32LE(dataSize, 40);
  let offset = 44;
  for (let i = 0; i < total; i += 1) {
    out.writeInt16LE(Math.round(Math.max(-1, Math.min(1, left[i] * gain)) * 32767), offset);
    out.writeInt16LE(Math.round(Math.max(-1, Math.min(1, right[i] * gain)) * 32767), offset + 2);
    offset += 4;
  }
  await mkdir(path.dirname(file), {recursive: true});
  await writeFile(file, out);
  return file;
}

async function main() {
  const chrome = findChrome();
  if (!chrome) throw new Error('Chrome not found: pass --chrome=<path> or set CHROME_PATH');
  if (args.encode) {
    const framesDir = path.resolve(ROOT, args['frames-dir'] || 'out/frames');
    const requestedFormat = String(args['frame-format'] || '').toLowerCase();
    const hasPng = existsSync(path.join(framesDir, 'f00000.png'));
    const hasJpg = existsSync(path.join(framesDir, 'f00000.jpg'));
    const frameExt = (requestedFormat === 'jpg' || requestedFormat === 'jpeg') ? 'jpg' : (requestedFormat === 'png' ? 'png' : (hasPng ? 'png' : 'jpg'));
    const frames = existsSync(framesDir)
      ? (await import('node:fs')).readdirSync(framesDir).filter((name) => name.startsWith('f') && name.endsWith('.' + frameExt)).sort()
      : [];
    if (!frames.length) throw new Error(`no frames in ${framesDir}; run --frames first`);
    const output = path.resolve(ROOT, args.out || 'out/video.mp4');
    await mkdir(path.dirname(output), {recursive: true});
    const ffmpeg = process.env.FFMPEG_PATH || 'ffmpeg';
    const commandArgs = [
      '-y', '-hide_banner', '-loglevel', 'error',
      '-framerate', String(FPS), '-start_number', '0',
      '-i', path.join(framesDir, 'f%05d.' + frameExt),
    ];
    let audioPath = args.audio ? path.resolve(ROOT, String(args.audio)) : '';
    if (!audioPath && !args.silent) {
      let duration = Number(args.duration || 0);
      if (!duration) {
        try {
          const pack = JSON.parse(readFileSync(path.resolve(ROOT, 'style-pack.json'), 'utf8'));
          duration = Number(pack.duration_seconds || 0);
        } catch {}
      }
      if (!duration) duration = 30;
      const ambient = path.resolve(ROOT, 'out/audio/ambient.m4a');
      const wav = path.resolve(ROOT, 'out/audio/ambient.wav');
      await synthesizeAmbientWav(wav, duration);
      await run(ffmpeg, ['-y', '-hide_banner', '-loglevel', 'error', '-i', wav, '-c:a', 'aac', '-b:a', '192k', '-movflags', '+faststart', ambient]);
      audioPath = ambient;
    }
    if (audioPath) commandArgs.push('-i', audioPath);
    commandArgs.push('-c:v', 'libx264', '-preset', 'medium', '-crf', '18', '-pix_fmt', 'yuv420p', '-movflags', '+faststart');
    if (audioPath) commandArgs.push('-c:a', 'aac', '-b:a', '192k', '-shortest');
    commandArgs.push(output);
    await run(ffmpeg, commandArgs);
    console.log(output);
    return;
  }

  const {server, port} = await startServer();
  const browser = await puppeteer.launch({
    executablePath: chrome,
    headless: 'new', protocolTimeout: 180000,
    args: ['--no-sandbox', '--disable-dev-shm-usage', '--hide-scrollbars', '--force-device-scale-factor=1', ...extraChromeArgs],
  });
  try {
    if (args.frames) {
      const frameDir = path.resolve(ROOT, args['frames-dir'] || 'out/frames');
      await mkdir(frameDir, {recursive: true});
      const duration = Number(args.duration || 10);
      const workers = Math.max(1, Number(args.workers || 1));
      const total = Math.round(duration * FPS);
      const frameFormat = String(args['frame-format'] || 'png').toLowerCase() === 'jpg' ? 'jpg' : 'png';
      const pages = await Promise.all(Array.from({length: workers}, () => openPage(browser, port)));
      let next = 0;
      let done = 0;
      const started = Date.now();
      await Promise.all(pages.map(async (page) => {
        while (next < total) {
          const index = next++;
          const buffer = await shot(page, index / FPS, frameFormat, 0.94);
          await writeFile(path.join(frameDir, 'f' + String(index).padStart(5, '0') + '.' + frameFormat), buffer);
          done += 1;
          if (done % 30 === 0 || done === total) {
            console.log(`${done}/${total} frames (${Math.round((Date.now() - started) / done)} ms/frame)`);
          }
        }
      }));
      return;
    }

    const page = await openPage(browser, port);
    if (args.sheet) {
      const times = parseTimes(args.sheet, '1,3,5,7,9');
      const cols = Math.max(1, Number(args.cols || 3));
      const rows = Math.ceil(times.length / cols);
      const temp = path.resolve(ROOT, `.tmp-sheet-${process.pid}`);
      await mkdir(temp, {recursive: true});
      for (let i = 0; i < times.length; i += 1) {
        await writeFile(path.join(temp, `f${String(i).padStart(3, '0')}.jpg`), await shot(page, times[i], 'jpeg', 0.95));
      }
      await tileImages(path.join(temp, 'f%03d.jpg'), path.resolve(ROOT, args.out || 'out/check/sheet.jpg'), cols, rows);
      await rm(temp, {recursive: true, force: true});
      console.log(path.resolve(ROOT, args.out || 'out/check/sheet.jpg'));
      return;
    }
    if (args.strip) {
      const [start, end] = parseRange(args.strip, '2:3');
      const cols = Math.max(1, Number(args.cols || 6));
      const times = [];
      for (let t = start; t < end + 1e-6; t += 1 / FPS) times.push(Number(t.toFixed(4)));
      const capped = times.slice(0, 120);
      const rows = Math.ceil(capped.length / cols);
      const temp = path.resolve(ROOT, `.tmp-strip-${process.pid}`);
      await mkdir(temp, {recursive: true});
      for (let i = 0; i < capped.length; i += 1) {
        await writeFile(path.join(temp, `f${String(i).padStart(3, '0')}.jpg`), await shot(page, capped[i], 'jpeg', 0.93));
      }
      await tileImages(path.join(temp, 'f%03d.jpg'), path.resolve(ROOT, args.out || 'out/check/strip.jpg'), cols, rows);
      await rm(temp, {recursive: true, force: true});
      console.log(path.resolve(ROOT, args.out || 'out/check/strip.jpg'));
      return;
    }
    if (args.stills) {
      const times = parseTimes(args.stills, '2,5,8');
      const outputDir = path.resolve(ROOT, args.out || 'out/stills');
      await mkdir(outputDir, {recursive: true});
      for (let i = 0; i < times.length; i += 1) {
        const buffer = await shot(page, times[i], 'png');
        await writeFile(path.join(outputDir, `still-${String(i + 1).padStart(4, '0')}.png`), buffer);
      }
      console.log(outputDir);
      return;
    }
    throw new Error('one of --sheet, --strip, --stills, --frames or --encode is required');
  } finally {
    await browser.close();
    server.close();
  }
}

main().catch((error) => {
  console.error(error?.stack || String(error));
  process.exit(1);
});




