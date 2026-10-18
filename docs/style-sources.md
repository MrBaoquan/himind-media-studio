# 风格来源与转化

> 来源登记：[catalog/style-sources.json](../catalog/style-sources.json)  
> 转换目标：一个来源只抽取一个可复现视觉能力，形成一个 Asset Pack。

## 一、来源优先级

### P0：浏览器型，可直接复用现有 Runtime

- three.js
- p5.js
- Rough.js
- D3
- ECharts

### P1：需要少量适配

- Remotion
- Motion Canvas
- GSAP
- Poly Haven / Fontsource

### P2：需要新 Runtime

- Manim（Python）
- Lottie / Rive（资产授权复杂）

## 二、转化流程

```text
来源登记
→ 许可证与资产授权检查
→ 选一个最小可复现示例
→ 隔离为 style-packs/<id>/template
→ 统一 render.mjs CLI
→ 本地化素材
→ 确定性改造
→ stylepack.validate / smoke / determinism
→ 生成 preview/sheet
→ 标记 ready
```

## 三、首批目标

1. `paper-collage`（已完成，Canvas / 纸张纹理）
2. `whiteboard-explainer`（已完成，Rough.js 4.6.6 本地化）
3. `data-chart-reel`（已完成，Canvas 2D 图表；D3 / ECharts 作为后续增强）
4. `3d-product-reel`（three.js）
5. `lyric-karaoke`（已完成，Canvas 文本时间轴）

## 四、许可证规则

- 代码许可证与资产许可证分开检查
- 优先 MIT / Apache-2.0 / BSD / ISC / CC0 / OFL
- LGPL、CC-BY、平台素材需要逐项确认
- GPL / AGPL、商业字体、商业音乐默认不进入可分发 Asset Pack
- 每个 Pack 必须包含 `LICENSE` 与 `ATTRIBUTION.md`

## 五、新建风格脚手架

```powershell
pwsh -File dev/new-style-pack.ps1 -Id my-style -Name "我的风格"
```

脚本会从 `data-kinetic` 复制基础模板，生成 `asset-pack.json`、`template/style-pack.json` 和 `ATTRIBUTION.md`。


## 六、已完成转化：whiteboard-explainer

```text
来源：rough-stuff/rough
版本：Rough.js 4.6.6
许可证：MIT
本地文件：style-packs/whiteboard-explainer/template/reel/vendor/rough.esm.js
完整许可证：style-packs/whiteboard-explainer/template/reel/vendor/LICENSE.roughjs
```

抽取的能力不是 Rough.js 示例，而是“手绘白板图形驱动”：

- 白板纸面、网格与轻微纸纹；
- 固定 seed 的 hand-drawn 线条、圆圈、箭头与图标；
- 四步流程卡片与竖向连接；
- 黄色重点圈注；
- `input/data.json` 数据契约。

验证结果：

```text
asset.validate     valid=true
stylepack.validate valid=true
preview.review     ready
determinism        1s / 5s / 9s 三次哈希一致
render.start       1080×1920 / 10s / 300 帧
web copy           1280 目标宽度配置，实际 1080×1920
```

## 七、批量打包现有风格

```powershell
pwsh -File dev/pack-asset-packs.ps1
```

脚本读取 `style-packs/*/asset-pack.json`，调用 `media.workbench.asset.pack`，把全部风格输出到 `dist/asset-packages/*.hmpack`，并打印文件数、大小与 SHA-256。

## 八、transit-network（合肥地铁示意）

- 视觉参考：用户提供的 MetroLineHub 太原地铁图，仅参考“白底 + 粗色线 + 浅蓝水系 + 清晰站名”的官方线网图语言，不复制图片或数据。
- 数据：内置合肥 1/2/3/4/5 号线示意子集，不是完整线网，不用于出行；正式发布前需替换为官方线网 / OSM / GTFS 校验数据。
- 技术：Canvas 2D 白底线网，线路生长、站点弹出、换乘脉冲、8 方向站名避让、年份轴、PNG 最终帧。
- OSS 结论：没有可直接嵌入的宽松许可线网动画库；LOOM / transit-map / legible-cities 仅作为可选外部布局工具评估，见 `docs/oss-evaluation/transit-map.md`。
- 验证：`asset.validate` 与 `stylepack.validate` 通过；30 秒母版 1080×1920 / 900 帧；无黑帧、无静帧；20 个站名全部放置；0.8.4 真实重渲视觉 QA 通过（`banding_gate=false`，见 `docs/transit-network-hefei-demo.md`）。

