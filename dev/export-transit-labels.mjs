import {readFile, writeFile} from 'node:fs/promises';
import {spawnSync} from 'node:child_process';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '..');
const args = {};
for (const raw of process.argv.slice(2)) {
  if (!raw.startsWith('--')) continue;
  const [key, ...rest] = raw.slice(2).split('=');
  args[key] = rest.length ? rest.join('=') : true;
}
const dataPath = path.resolve(args.data || path.join(root, 'style-packs/transit-network/template/input/data.json'));
const outPath = path.resolve(args.out || path.join(path.dirname(dataPath), 'labels.json'));
const binary = path.resolve(args.binary || path.join(root, 'plugins/com.himind.media-workbench/media-workbench/bin/media-workbench.exe'));
const data = JSON.parse(await readFile(dataPath, 'utf8'));
const stations = new Map((data.stations || []).map((station) => [station.id, station]));
const lineCount = new Map();
const terminals = new Set();
for (const line of data.lines || []) {
  for (const id of line.stations || []) lineCount.set(id, (lineCount.get(id) || 0) + 1);
  if (line.stations?.length) {
    terminals.add(line.stations[0]);
    terminals.add(line.stations[line.stations.length - 1]);
  }
}
const labels = [];
for (const station of data.stations || []) {
  const interchange = (lineCount.get(station.id) || 0) > 1;
  const terminal = terminals.has(station.id);
  labels.push({
    id: station.id,
    text: station.name,
    x: station.x,
    y: station.y,
    priority: interchange ? 100 : terminal ? 60 : 20,
    font_size: interchange ? 26 : terminal ? 24 : 22,
    hideable: false,
    leader: true,
  });
}
const obstacles = [];
for (const line of data.lines || []) obstacles.push({kind: 'line', points: line.path, width: 12});
for (const station of data.stations || []) {
  obstacles.push({kind: 'circle', x: station.x, y: station.y, r: (lineCount.get(station.id) || 0) > 1 ? 13 : 7});
}
const viewport = {width: 1080, height: 1920, safe_area: [60, 330, 960, 1150]};
const request = JSON.stringify({jsonrpc: '2.0', id: 1, method: 'media.workbench.label.place', params: {viewport, labels, obstacles}});
const result = spawnSync(binary, [], {input: request, encoding: 'utf8', maxBuffer: 16 * 1024 * 1024});
if (result.status !== 0) throw new Error(result.stderr || `plugin exited ${result.status}`);
const response = JSON.parse(result.stdout);
if (response.error) throw new Error(response.error.message || 'label.place failed');
const payload = {
  schema: 'himind.label-placement.v1',
  source: 'media.workbench.label.place',
  viewport,
  placements: response.result.placements,
  metrics: response.result.metrics,
};
await writeFile(outPath, JSON.stringify(payload, null, 2) + '\n');
console.log(JSON.stringify({data: dataPath, out: outPath, metrics: payload.metrics}, null, 2));
