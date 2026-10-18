import {spawnSync} from 'node:child_process';

const args = {};
for (const raw of process.argv.slice(2)) {
  if (!raw.startsWith('--')) continue;
  const [key, ...rest] = raw.slice(2).split('=');
  args[key] = rest.length ? rest.join('=') : true;
}

const input = args.image || args.video;
if (!input) {
  console.error('usage: node verify-banding.mjs --image=<png|jpg> | --video=<mp4> [--time=9] [--width=1080] [--height=1920]');
  process.exit(2);
}

const time = Number(args.time || 9);
const probeArgs = ['-v', 'error', '-select_streams', 'v:0', '-show_entries', 'stream=width,height', '-of', 'json', input];
const probe = spawnSync('ffprobe', probeArgs, {encoding: 'utf8'});
if (probe.status !== 0) {
  console.error(probe.stderr || 'ffprobe failed');
  process.exit(2);
}
const stream = JSON.parse(probe.stdout).streams[0];
const width = Number(args.width || stream.width);
const height = Number(args.height || stream.height);
const frameArgs = ['-hide_banner', '-loglevel', 'error'];
if (args.video) frameArgs.push('-ss', String(time));
frameArgs.push('-i', input, '-frames:v', '1', '-pix_fmt', 'rgb24', '-f', 'rawvideo', '-');
const frame = spawnSync('ffmpeg', frameArgs, {encoding: null, maxBuffer: 1024 * 1024 * 24});
if (frame.status !== 0) {
  console.error(frame.stderr?.toString() || 'ffmpeg failed');
  process.exit(2);
}
const buf = frame.stdout;
if (buf.length < width * height * 3) {
  console.error(`unexpected frame size: ${buf.length}, expected ${width * height * 3}`);
  process.exit(2);
}

function luma(x, y) {
  const i = (y * width + x) * 3;
  return buf[i] * 0.2126 + buf[i + 1] * 0.7152 + buf[i + 2] * 0.0722;
}

function analyzeLine(label, points) {
  const values = points.map(([x, y]) => luma(x, y));
  const runs = [];
  let start = 0;
  for (let i = 1; i <= values.length; i += 1) {
    if (i === values.length || Math.abs(values[i] - values[start]) > 0.5) {
      runs.push({value: Number(values[start].toFixed(2)), length: i - start});
      start = i;
    }
  }
  const unique = new Set(values.map((value) => Math.round(value))).size;
  const maxRun = Math.max(...runs.map((run) => run.length));
  return {label, unique, maxRun, runs: runs.filter((run) => run.length >= 20).slice(0, 12)};
}

const lines = [];
const yPositions = [Math.round(height * 0.26), Math.round(height * 0.74)];
for (const y of yPositions) {
  const points = [];
  for (let x = 0; x < width; x += 1) points.push([x, y]);
  lines.push(analyzeLine(`y=${y}`, points));
}
for (const x of [Math.round(width * 0.02), Math.round(width * 0.98)]) {
  const points = [];
  for (let y = Math.round(height * 0.05); y < Math.round(height * 0.95); y += 1) points.push([x, y]);
  lines.push(analyzeLine(`x=${x}`, points));
}

const maxRunLimit = Math.max(24, Math.round(width * Number(args['max-run-ratio'] || 0.08)));
const minLevels = Number(args['min-levels'] || 12);
const results = lines.map((line) => ({
  ...line,
  ok: line.maxRun <= maxRunLimit && line.unique >= minLevels,
}));
const ok = results.every((line) => line.ok);
console.log(JSON.stringify({
  input,
  width,
  height,
  time: args.video ? time : null,
  thresholds: {maxRunLimit, minLevels},
  lines: results,
  ok,
}, null, 2));
process.exit(ok ? 0 : 1);
