# OSS Prior Art：地铁路线图动画

> 调研日期：2026-10-18  
> 需求：白底官方线网图风格，线路生长、站点出现、换乘环脉冲、30 秒竖屏动画。  
> 结论：**没有成熟、宽松许可、可直接嵌入的“地铁线网图动画”开源库。**  
> 建议：自建薄层 `transit-network` SVG/Canvas Style Pack；布局生成器只作为可选外部工具评估，不直接 vendor。

## 一、先区分四个问题

“地铁图动画”不是一个库能全部解决的问题，要拆开看：

| 子问题 | 典型方案 |
| --- | --- |
| 线网布局（示意/地理） | LOOM、transit-map、手工示意布局 |
| 站名避让 | 自研 8 方向候选位 + 碰撞检测 |
| 线路/站点/换乘动画 | SVG / Canvas + Remotion / Motion Canvas / 自研帧函数 |
| 视频导出 | 现有 Browser-Code + PNG 帧 + ffmpeg |

很多“开源库”只解决第一层，不解决后三层。

## 二、候选项目

| 项目 | Stars | License | 语言 | 能力 | 适配判断 |
| --- | ---: | --- | --- | --- | --- |
| [ad-freiburg/loom](https://github.com/ad-freiburg/loom) | 308 | GPL-3.0 | C++ | 地理准确 / 八向示意线网自动布局 | **布局最强，但 GPL + C++/求解器依赖**；只适合作为可选外部 CLI，不嵌入插件 |
| [juliuste/transit-map](https://github.com/juliuste/transit-map) | 200 | ISC | JS | MIP 生成示意地铁图，输出 SVG | 许可友好，但**依赖商业 Gurobi 求解器**；暂不作为默认路径 |
| [richc117/legible-cities](https://github.com/richc117/legible-cities) | 6 | GPL-3.0 | Python | 从 GTFS 生成示意线网 + 时刻表动画，覆盖 22 个铁路网络 | 最接近端到端；**GPL + Python/桌面形态**，只作为参考/可选外部工具 |
| [gopeter/semantic-metro-map-generator](https://github.com/gopeter/semantic-metro-map-generator) | 9 | 无 | JS | GTFS → SVG 语义线网 | **无许可证，不可复用** |
| [DallasUrbanists/transit-simulator](https://github.com/DallasUrbanists/transit-simulator) | 6 | 无 | JS | GTFS 时刻表车辆动画，MapTiler/Transitland | 地理实时动画，不是示意线网；**无许可证** |
| [Fangyuan025/triptrail](https://github.com/Fangyuan025/triptrail) | 14 | MIT | JS | MapLibre 球面地图 + 旅行路线视频导出 | 可参考“地图动画 + MP4 导出”，但不是地铁示意线网 |
| [UniStuttgart-VISUS/map-transition-helper](https://github.com/UniStuttgart-VISUS/map-transition-helper) | 2 | MIT | TS | 地图过渡动画组合 | 通用地图过渡，不含线网布局/站名避让 |
| [motion-canvas/motion-canvas](https://github.com/motion-canvas/motion-canvas) | 19k | MIT | TS | 时间轴动画引擎，可做 SVG 线条/文字动画 | 动画能力强，但**不解决线网布局**；当前 Browser-Code + SVG 已足够 |
| [remotion-dev/remotion](https://github.com/remotion-dev/remotion) | 65k | 自定义许可 | TS/React | React 视频渲染 | 项目已在用；本需求用 Browser-Code 更直接，不必新增栈 |
| [d3/d3](https://github.com/d3/d3) | 113k | ISC | JS | SVG/数据驱动过渡 | 通用，不是地铁专用；手工示意布局下收益有限 |
| [maplibre/maplibre-gl-js](https://github.com/maplibre/maplibre-gl-js) | 11k | BSD 系 | TS/WebGL | 地理地图渲染 | 适合地理模式，不适合“白底官方示意线网图” |

> 注：`motiontemplates/894541` 的 Metro Map 模板已不可免费获取，不能作为依赖。

## 三、结论

### 不建议

- 不要把 LOOM / legible-cities 直接 vendor 进插件：
  - GPL-3.0 会污染分发许可；
  - C++/Python/求解器依赖会破坏 Browser-Code 的轻量运行；
  - 它们的布局输出也不等于参考图的官方排版。
- 不要为了“线条生长”引入 Motion Canvas 或 Remotion 新栈：
  - `stroke-dashoffset` + 帧号纯函数已经足够；
  - 当前 PNG 帧 + ffmpeg 导出链路更可控。
- 不要用 MapLibre 做示意线网：
  - 它是地理地图引擎，不是地铁图排版器。

### 建议

第一版采用：

```text
手工示意布局
+ SVG 路径（0/45/90°）
+ 站名 8 方向避让
+ 线色/站点/换乘数据契约
+ Browser-Code 帧函数
+ 现有 PNG 帧 + ffmpeg
```

第二版再评估：

```text
LOOM 作为可选外部布局工具
或 transit-map（如果 CBC 求解器可用）
```

它们只输出布局/几何，不进入插件运行时。

## 四、决策记录

| 决策 | 结果 |
| --- | --- |
| 是否引入专用动画库 | 否，现有 SVG/Canvas + 帧函数足够 |
| 是否引入 LOOM | 暂不；作为可选外部布局工具评估 |
| 是否引入 legible-cities | 暂不；GPL + Python，仅作参考 |
| 是否引入 transit-map | 暂不；Gurobi 依赖不满足 |
| 是否引入 Motion Canvas | 否；动画不是瓶颈，布局/标签才是 |
| 是否引入 MapLibre | 否；示意模式不需要地理引擎 |
| 第一版布局 | 手工八向示意布局 |
| 第一版动画 | 线路生长 + 站点弹出 + 换乘脉冲 + 图例/年份轴 |

## 五、何时重新评估

满足以下任一条件时，重新做 OSS prior art：

- 需要批量生成 10+ 城市线网；
- 需要从 GTFS 自动生成示意布局；
- 需要地理/示意双模式；
- 现有手工布局维护成本超过自动布局；
- 出现 MIT/ISC/Apache 许可的高质量线网布局库。

重新评估时，按 `docs/oss-evaluation/README.md` 的模板记录，不直接改运行时。
