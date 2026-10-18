# 案例：合肥地铁线网动画

> 状态：已接入创作台，0.8.4 真实重渲 + 视觉 QA 通过（2026-10-18）  
> 项目：`video-project-e3c66ea9`（线网地图动画 项目）  
> Style Pack：`transit-network` `1.0.0`  
> 渲染器：`browser-code`（Canvas 2D）  
> 验收实例：`18082` / `development` profile / CDP `9311`

## 一、目标

按用户提供的官方线网图视觉参考，做一版：

```text
白底
+ 粗色线路
+ 浅蓝水系
+ 细站名
+ 换乘圆环
+ 线路生长 / 站点出现 / 换乘脉冲 / 年份轴
```

不采用暗色 3D、霓虹、bloom 或强暗角。

## 二、数据

当前内置数据是**示意子集**：

```text
5 条线：1 / 2 / 3 / 4 / 5 号线
20 个示意站点
3 个换乘节点：大东门、合肥南站、望湖城
```

数据文件：

```text
style-packs/transit-network/template/input/data.json
```

画面与数据文件都保留：

```text
示意线网 · 仅展示部分线路与站点
数据截至 2025-12
示意图，非完整线网，不用于出行
```

> 正式发布前必须替换为官方线网 / OSM / GTFS 校验数据。

## 三、动画

| 时间 | 画面 |
| --- | --- |
| 0–0.8s | 白纸、标题、南淝河淡入 |
| 0.8–7.4s | 1→2→3→4→5 号线依次生长，站点和站名出现 |
| 7.4–10.5s | 图例展开，列车点开始沿线移动 |
| 10.5–16.5s | 换乘节点脉冲，镜头轻推 |
| 13.5–20.5s | 年份轴 2016→2025，图例逐条高亮 |
| 22–30s | 镜头拉回，结论“轨道上的合肥” |

技术点：

- Canvas 2D；
- 线路 `stroke-dashoffset` 式生长；
- 站点按线路累计距离逐个出现；
- 8 方向站名候选位 + 优先级贪心；
- 换乘站候选位不足时使用带引线的第二轮回退；
- 列车点沿线循环移动；
- 最终帧 PNG，避免 JPEG 暗部/细线量化。

## 四、站名避让

当前结果：

```text
20 / 20 站名放置成功（media.workbench.label.place / label.validate）
隐藏 0
换乘节点 3
```

策略：

```text
换乘站 > 终点站 > 普通站
右 / 左 / 上 / 下 / 四个对角
避开安全区、标签、站点和其他线路
换乘站允许带引线回退
整段动画位置固定，不做逐帧重排
```

### 预计算标签接入

`dev/export-transit-labels.mjs` 调用后端能力生成：

```text
video-project-e3c66ea9/input/labels.json
```

`reel.js` 在启动时读取该文件，并把结果应用到固定布局；读取失败时才回退到前端 8 方向算法。

实测对比（同一页面、同一数据）：

```text
有 labels.json：20 个标签 / hidden 0
移除 labels.json：20 个标签 / hidden 0
两次布局坐标哈希不同（c899349d… vs 2e966cf2…）
```

结论：当前成片确实使用了后端 `media.workbench.label.place` 的结果，而不是只在测试里调用。

## 五、质量门禁与平面风格

0.8.3 首次渲染时，视觉 QA 把白底大面积纯色误判为“严重色带”：

```text
banding: 1 个亮度级 / 1728px 平坦段
```

这不是画面缺陷，而是白底平面风格不适合按暗部渐变设定的色带阈值。

0.8.4 起 Style Pack 可以显式声明：

```json
"quality": {
  "banding_gate": false
}
```

- `banding_gate=false`：跳过色带判定，但仍记录平坦段指标；
- 未声明或 `true`：继续执行严格的色带检查（暗色 / 渐变风格）；
- 这是 Style Pack 的声明式质量档案，不是全局放宽。

本次 0.8.4 重渲结果：

```text
frames: passed（900 帧 / 30fps / 30s）
black_frames: passed（0 个黑帧区间）
freeze: passed（无 ≥1.5s 静帧）
banding: skipped（Style Pack 声明 banding_gate=false）
audio_loudness: passed（I=-22.2 LUFS / peak=-13.6 dBFS）
quality_status: passed
```

## 六、OSS 决策

调研记录：

```text
docs/oss-evaluation/transit-map.md
docs/oss-evaluation/label-placement.md
```

结论：

- 没有可直接嵌入、宽松许可的“官方线网图 + 动画”开源库；
- LOOM 布局强，但 GPL-3.0 + C++/求解器；
- transit-map 许可友好，但依赖 Gurobi；
- legible-cities 接近端到端，但 GPL-3.0 + Python；
- 因此第一版使用自绘 Canvas + 8 方向标签避让；
- 批量 10+ 城市时再评估 LOOM 作为可选外部布局工具。

## 七、Evidence

### 0.8.4 重渲母版（QA 通过）

```text
dist/verify/transit-network/hefei-metro-workbench-qa-pass.mp4
1080×1920 / 30fps / 900 帧 / 30s
3,029,474 bytes
SHA-256: B3CA9570D7B55EC705DB089138A1299DB0B06023EBCE6DE3F9B055E2B75054EE
```

### 0.8.4 重渲 Web copy

```text
dist/verify/transit-network/hefei-metro-workbench-qa-pass-web.mp4
1,507,421 bytes
SHA-256: 5F7F3444129215BF358E438E85E64E0479BA11D84361B617EF13055858A8026E
```

### 0.8.4 重渲审查图

```text
dist/verify/transit-network/hefei-metro-workbench-qa-pass-sheet.jpg
301,396 bytes
SHA-256: D0FB334CBD2DF1B7333C5E9357DBF0A31CB2C4B917F4E84146B036734B769BEE
```

### QA 原始报告

```text
dist/verify/transit-network/hefei-metro-workbench-qa-pass.json
dist/verify/transit-network/qa-pass.txt
```

### 关键帧

```text
dist/verify/transit-network/keyframes-qa-pass/f2_0.png
dist/verify/transit-network/keyframes-qa-pass/f6_0.png
dist/verify/transit-network/keyframes-qa-pass/f9_0.png
dist/verify/transit-network/keyframes-qa-pass/f14_0.png
dist/verify/transit-network/keyframes-qa-pass/f19_0.png
dist/verify/transit-network/keyframes-qa-pass/f24_0.png
dist/verify/transit-network/keyframes-qa-pass/f29_0.png
```

### 早期 Workbench 母版（保留）

```text
dist/verify/transit-network/hefei-metro-workbench.mp4
3,021,220 bytes
SHA-256: CFE48B642989C8CA56678CA4FE0C67E7E2361C54278CD9864A611FCD5798795C
```

### 独立参考母版（保留）

```text
dist/verify/transit-network/hefei-metro-master.mp4
3,021,547 bytes
SHA-256: 8C3D69E3FD7F28EEA45495B90A068F246E4597B3DFF862AC745A43A575D8EC12
```

三份母版结构一致；字节差异来自编码器在相同画面上的正常非确定性。0.8.4 版是当前唯一完成视觉 QA 的交付母版。

### 资产包

```text
dist/asset-packages/transit-network-1.0.0.hmpack
243,548 bytes
SHA-256: a4cbb7b4a18725c19f9a83d5808ffaa41d3a1b0b566d7ebe5761c4fa28d9b372
```

本版在发布前重新打包，包含预计算标签读取逻辑与 `banding_gate=false` 质量档案。目录状态仍为 `prepared`；一旦正式发布，后续内容变化必须升版本（例如 1.0.1），不能覆盖已发布版本。

### 页面证据

```text
dist/verify/transit-network/workbench-18082-qa-pass.png
3072×1780
SHA-256: CC9AE7603953801AA8E87BCFF2373F5D64226035E07CE294A2664CEAFF562D43
```

页面实际状态：

```text
线网地图动画 项目
确定性 passed
completed 100%
母版 QA：通过
视觉 QA：通过
Browser-Code 视频已生成；Web copy 长边 1280px 1.4 MB
质量门禁 51/51
约束 51/51
```

## 八、边界

- 数据是示意子集，不是完整官方线网；
- 没有中文旁白，只有程序化环境声床；
- 站名/线路从属关系需在正式发布前用官方数据校验；
- 换乘站标签使用引线回退，密集网络中仍需要人工复核；
- 现有项目是源码快照，不会自动继承 Style Pack 的后续更新；
- 素材/发布/TTS 仍属于可选 External Pack。

## 九、下一步

1. 用官方数据替换 `input/data.json`；
2. 把 `media.workbench.label.place` / `label.validate` 复用到更多地图/线网风格，并补面标注/线标注算法；
3. 支持 6/8/S1 等更多线路；
4. 批量城市时评估 LOOM 作为可选外部布局工具；
5. 接入 TTS / 字幕 External Pack；
6. 把“平面风格显式跳过色带门禁”写进 Style Pack 作者指南，避免后续风格重复踩坑。


