# Transit Opening Timeline Style Pack

This style pack turns transit openings into a date-led timeline: the current line grows across a light schematic map while a bottom card rolls the date, segment and cumulative mileage.

## Quick start

```bash
npm install
node render.mjs --stills=1,6,12,18,24,30,35,37 --out=out/stills
node render.mjs --frames --workers=1 --duration=38
node render.mjs --encode --out=out/video.mp4 --duration=38
```

`--duration` must be passed to both `--frames` and `--encode`; otherwise the 30 s ambient bed will truncate longer videos.

Final frames default to PNG. Use `--frame-format=jpg` only for quick tests.

## Data

Replace `input/data.json` with a verified network + event dataset. The bundled Hefei data is a schematic demo subset, not an official map.

Required fields:

- `lines[].path`: canvas-space polyline;
- `lines[].stations`: station IDs in path order;
- `stations[].x/y`: canvas-space station position;
- `events[].date/year/lineId/lineName/segment/km`;
- `stats.lines/stations/km`;
- `source`, `as_of`, `disclaimer`.

Keep the source/as-of/disclaimer visible in the final video.
