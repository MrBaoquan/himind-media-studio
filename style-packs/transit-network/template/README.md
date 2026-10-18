# Transit Network Style Pack

This style pack renders a clean, white-background schematic transit map with animated line growth, station reveal, interchange pulses, a legend and a year axis.

## Quick start

```bash
npm install
node render.mjs --stills=1,5,9,13,18,24,28 --out=out/stills
node render.mjs --frames --workers=1 --duration=30
node render.mjs --encode --out=out/video.mp4
```

Final frames default to PNG. Use `--frame-format=jpg` only for quick tests.

## Data

Replace `input/data.json` with a verified network dataset. The bundled Hefei data is a schematic demo subset, not an official map.

Required fields:

- `lines[].path`: canvas-space polyline;
- `lines[].stations`: station IDs in path order;
- `stations[].x/y`: canvas-space station position;
- `source`, `as_of`, `disclaimer`.

Keep the source/as-of/disclaimer visible in the final video.
