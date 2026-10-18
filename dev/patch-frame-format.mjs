import {readFile, writeFile} from 'node:fs/promises';
const files = process.argv.slice(2);
let patched = 0, skipped = 0;
for (const file of files) {
  let src = await readFile(file, 'utf8');
  if (src.includes('const frameExt =')) { skipped += 1; continue; }
  src = src.replace(/\r\n/g, '\n');
  const framesDirLine = "    const framesDir = path.resolve(ROOT, args['frames-dir'] || 'out/frames');";
  if (!src.includes(framesDirLine)) throw new Error('framesDir anchor missing: ' + file);
  src = src.replace(framesDirLine, framesDirLine + `\n    const requestedFormat = String(args['frame-format'] || '').toLowerCase();\n    const hasPng = existsSync(path.join(framesDir, 'f00000.png'));\n    const frameExt = (requestedFormat === 'jpg' || requestedFormat === 'jpeg') ? 'jpg' : (requestedFormat === 'png' ? 'png' : (hasPng ? 'png' : 'jpg'));`);
  const filterOld = "? (await import('node:fs')).readdirSync(framesDir).filter((name) => /^f\\d{5}\\.jpg$/.test(name)).sort()";
  if (!src.includes(filterOld)) throw new Error('filter anchor missing: ' + file);
  src = src.replace(filterOld, "? (await import('node:fs')).readdirSync(framesDir).filter((name) => name.startsWith('f') && name.endsWith('.' + frameExt)).sort()");
  const inputOld = "'-i', path.join(framesDir, 'f%05d.jpg'),";
  if (!src.includes(inputOld)) throw new Error('input anchor missing: ' + file);
  src = src.replace(inputOld, "'-i', path.join(framesDir, 'f%05d.' + frameExt),");
  const totalOld = '      const total = Math.round(duration * FPS);';
  if (!src.includes(totalOld)) throw new Error('total anchor missing: ' + file);
  src = src.replace(totalOld, totalOld + `\n      const frameFormat = String(args['frame-format'] || 'png').toLowerCase() === 'jpg' ? 'jpg' : 'png';`);
  const shotOld = "const buffer = await shot(page, index / FPS, 'jpeg', 0.94);";
  if (!src.includes(shotOld)) throw new Error('shot anchor missing: ' + file);
  src = src.replace(shotOld, 'const buffer = await shot(page, index / FPS, frameFormat, 0.94);');
  const writeOld = "await writeFile(path.join(frameDir, `f${String(index).padStart(5, '0')}.jpg`), buffer);";
  if (!src.includes(writeOld)) throw new Error('write anchor missing: ' + file);
  src = src.replace(writeOld, "await writeFile(path.join(frameDir, 'f' + String(index).padStart(5, '0') + '.' + frameFormat), buffer);");
  if (!src.includes('const frameExt =') || !src.includes('const frameFormat =') || !src.includes("'f%05d.' + frameExt")) {
    throw new Error('post-check failed: ' + file);
  }
  await writeFile(file, src);
  patched += 1;
  console.log('patched ' + file);
}
console.log(JSON.stringify({patched, skipped, total: files.length}));
