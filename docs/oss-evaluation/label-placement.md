# OSS Prior Art：地图标签避让 / Label Placement

> 调研日期：2026-10-18  
> 适用范围：地铁站名、城市/区域名、河流/道路名、地图标注、动态地图动画。  
> 结论：有成熟算法和可用库，但**没有一个库同时解决点标注、面标注、线标注和时序稳定**；建议沉淀为通用 `label-placement` technique + capability。

## 一、问题不是“把文字放上去”

地图标签避让至少包含四类问题：

| 类型 | 例子 | 主要难点 |
| --- | --- | --- |
| 点标注 | 地铁站、城市、POI | 8/16 方向候选、优先级、引线 |
| 面标注 | 国家、省、区、湖、公园 | 多边形内最佳位置、形状适配 |
| 线标注 | 河流、道路、地铁线 | 沿路径排布、曲率、重复标签 |
| 动态标注 | 镜头移动、线路生长 | 时序稳定、不能每帧跳位 |

所以不能只写一个“碰撞检测函数”。

## 二、可用算法

### 1. 候选位 + 贪心

点标注最常用：

```text
每个标签生成 8 或 16 个候选矩形
→ 按优先级从高到低
→ 选第一个无碰撞、代价最小的候选
→ 放不下就隐藏或加引线
```

优点：快、确定、易调试。  
缺点：贪心可能错过全局更优解。

### 2. 空间索引 + 碰撞检测

用 R-tree / 网格索引查询标签与障碍物：

- `rbush`：MIT，R-tree；
- `labelgun`：MIT，mapping-agnostic 的标签碰撞/显隐引擎；
- `Leaflet.LabelTextCollision`：MIT，Leaflet 插件，可参考其网格/碰撞策略。

这是所有算法的底座。

### 3. 模拟退火 / 局部搜索 / ILP

适合密集标签的离线优化：

- 模拟退火：允许暂时变差，跳出局部最优；
- 遗传算法：适合批量地图；
- ILP/MIP：可表达硬约束，但求解慢；
- 强化学习：研究项目多，工程稳定性不足。

不建议作为每帧实时算法的默认方案。

### 4. 面标注：Pole of Inaccessibility

面标注不是质心，而是“离边界最远的内部点”：

- `mapbox/polylabel`：ISC，1.5k stars；
- `LabelPoint`：MIT，基于 polylabel 的多边形标签；
- 距离变换 / 中轴变换也常用。

地铁图里的“区域/河流/湖”可以使用；站名不需要。

### 5. 力导向

`d3-force` 可以把标签当作带锚点的节点，用斥力/弹簧避免重叠：

- 适合散点图、气泡图、非严格地图；
- 对“不能压线、不能出安全区”的硬约束不如候选位 + 碰撞稳定；
- 需要固定迭代次数/种子才能确定。

### 6. 动态地图：时序稳定

动态场景的核心不是“每帧最优”，而是“每帧不要跳”：

```text
上一帧位置作为先验
+ 位置变化惩罚
+ 迟滞（hysteresis）
+ 淡入淡出
+ 优先级不变
```

否则标签会闪烁、跳动，比轻微重叠更难看。

## 三、OSS 评估

| 项目 | Stars | License | 能力 | 判断 |
| --- | ---: | --- | --- | --- |
| [Geovation/labelgun](https://github.com/Geovation/labelgun) | 115 | MIT | 标签碰撞、优先级、显隐 | 可作为碰撞/显隐层参考或可选依赖；不负责候选位生成 |
| [mapbox/polylabel](https://github.com/mapbox/polylabel) | 1.5k | ISC | 多边形内部最佳标签点 | 可直接用于面标注 |
| [Leaflet.LabelTextCollision](https://github.com/yakitoritabetai/Leaflet.LabelTextCollision) | 107 | MIT | Leaflet 标签碰撞 | 只适用于 Leaflet，算法可参考 |
| [GeorgeGardiner/LabelPoint](https://github.com/GeorgeGardiner/LabelPoint) | 16 | MIT | 多边形标签点 | 面标注参考 |
| [mourner/rbush](https://github.com/mourner/rbush) | 2.8k | MIT | R-tree 空间索引 | 碰撞检测底座 |
| [d3/d3-force](https://github.com/d3/d3-force) | 2k | ISC | 力导向布局 | 可选，不适合硬约束主算法 |
| [mapbox/earcut](https://github.com/mapbox/earcut) | 2.6k | ISC | 多边形三角化 | 几何辅助，不是标签算法 |

结论：

- **点标注**：自研 8/16 候选位 + `rbush` 碰撞 + 优先级贪心；
- **面标注**：可用 `polylabel`（ISC）；
- **线标注**：自研沿路径采样 + 曲率/碰撞代价；
- **动态稳定**：自研时序惩罚 + 迟滞；
- **不建议**直接引入一个重型布局库解决全部问题。

## 四、建议沉淀的通用能力

### `label-placement` technique

输入：

```json
{
  "labels": [
    {
      "id": "station.S001",
      "text": "合肥南站",
      "kind": "point",
      "anchor": {"x": 0.62, "y": 0.78},
      "priority": 100,
      "maxDistance": 40,
      "leader": true,
      "hideable": true
    }
  ],
  "obstacles": [
    {"kind": "line", "points": [], "width": 12},
    {"kind": "station", "x": 0.62, "y": 0.78, "radius": 6},
    {"kind": "safeArea", "rect": [70, 100, 940, 1600]}
  ],
  "viewport": {"width": 1080, "height": 1920},
  "previous": []
}
```

输出：

```json
{
  "placements": [
    {
      "id": "station.S001",
      "box": [680, 900, 120, 28],
      "side": "right",
      "visible": true,
      "leader": [[660, 914], [680, 914]],
      "score": 0.92
    }
  ],
  "findings": [],
  "metrics": {
    "collisions": 0,
    "hidden_priority_labels": 0,
    "average_displacement": 2.1
  }
}
```

### 能力拆分

```text
media.workbench.label.place     静态/单帧布局
media.workbench.label.animate   时序稳定布局
media.workbench.label.validate  碰撞/越界/优先级检查
```

### 约束

```text
LABEL_NO_COLLISION
LABEL_SAFE_AREA
LABEL_PRIORITY_PRESERVED
LABEL_TEMPORAL_STABILITY
LABEL_MAX_DISPLACEMENT
```

### 质量指标

- 碰撞数；
- 越界数；
- 高优先级标签被隐藏数；
- 平均位移/最大位移；
- 帧间跳变次数；
- 运行时；
- 人工可读性评分。

## 五、落地顺序

1. 点标注：8 方向候选 + `rbush` + 优先级贪心；
2. 时序稳定：上一帧先验 + 迟滞 + 淡出；
3. 面标注：接入 `polylabel`；
4. 线标注：沿路径采样 + 曲率代价；
5. 密集场景：局部搜索/模拟退火，仅离线使用；
6. 最后才考虑学习型/全局优化。

## 六、决策记录

| 决策 | 结果 |
| --- | --- |
| 是否引入 labelgun | 可参考/可选；默认自研薄层，避免依赖其显隐模型 |
| 是否引入 polylabel | 面标注可用（ISC） |
| 是否引入 rbush | 可用作空间索引（MIT） |
| 是否引入 d3-force | 不作为主算法；仅散点/非严格场景 |
| 是否引入 ILP/RL | 不作为实时默认；离线批量再评估 |
| 第一版 | 点标注 8 方向 + rbush + 优先级 + 时序迟滞 |

## 七、已实现（2026-10-18）

已沉淀为通用能力：

```text
media.workbench.label.place     8 方向候选位 + 优先级 + 线路/站点障碍 + 带引线回退
media.workbench.label.validate  碰撞 / 越界 / 障碍校验
```

实现：

- `label_placement.go`
- `label_placement_test.go`
- `LABEL_PLACEMENT_VALID` 约束
- 合肥 20 站 fixture：20/20 可见，validation 0 findings

带 leader 的换乘标签允许引线跨线；普通标签仍必须避开线路和站点。
