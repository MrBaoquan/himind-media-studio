# Whiteboard Explainer Template

这是 `whiteboard-explainer` 的 Browser-Code 渲染模板，遵循 HiMind 固定 CLI：

```bash
node render.mjs --sheet=1,3,5,7,9 --cols=3 --w=540 --out=out/check/sheet.jpg
node render.mjs --strip=2:3 --cols=6 --w=320 --out=out/check/strip.jpg
node render.mjs --stills=2,5,8 --out=out/stills
node render.mjs --frames --workers=4
node render.mjs --encode --out=out/video.mp4
```

## 输入

- `input/data.json`：标题、步骤、结论、来源；
- `input/brief.json`：项目说明与创作约束。

## 依赖

Rough.js 4.6.6 已本地化到 `reel/vendor/rough.esm.js`，渲染时不访问网络。
`puppeteer-core` 由创作台在首次预览时安装。

## 约束

- 不直接修改 `style-packs/` 内的模板，先复制到工作区；
- 所有图形使用固定 seed；
- 每一帧只依赖时间 `t` 与输入数据；
- 新增图标时优先用 Rough.js 基础图形组合，不引入外部图片。
