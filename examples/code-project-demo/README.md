# Browser-Code 项目样例

这个目录遵循 HiMind 的 `render.mjs` 固定 CLI：

```bash
node render.mjs --sheet=1,3,5,7,9 --cols=3 --w=540 --out=out/check/sheet.jpg
node render.mjs --strip=2:3 --cols=6 --w=320 --out=out/check/strip.jpg
node render.mjs --stills=2,5,8 --out=out/stills
node render.mjs --frames --workers=4
node render.mjs --encode --out=out/video.mp4
```

它通过 `style-pack.json` 声明入口与输出路径。AI 客户端把本目录复制到工作区后，
调用 `media.workbench.project.import` 登记项目，再调用 `preview.review` 生成审查图。
