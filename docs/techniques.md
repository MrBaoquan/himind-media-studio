# 技法资产（Technique Assets）

本扩展的复用单位不是"整块模板"，而是**技法**：可参数化、可独立验证、可组合的原语。
一份能力包 = 技法（代码） × 风格令牌（声明） × 数据契约（schema）。

设计约束（全部体现为可执行的断言，不是文档承诺）：

| 约束 | 由什么守住 |
| --- | --- |
| 技法必须是纯函数、帧号决定论 | `route_test.go` 幂等性/确定性用例 |
| 求解器必须保证无重叠、不越界 | `labels.js` 的 `labelQuality()` + `dev/verify-render.py` |
| 数据必须锚定站点、无折返尖刺、无闭环 | `media.workbench.route.validate` 的不变量 |
| 新建项目生成的配方必须直接合格 | `TestDefaultRecipeForRouteGrowthPassesInvariants` |
| 成片必须有内容、有推进 | `dev/verify-render.py` |

---

## 一、管线技法 `pipeline.geo.route`

对应能力：`media.workbench.route.repair` / `media.workbench.route.validate`

### 为什么需要它

地图类视频的质量**一半来自几何**，而不是画法。图搜索（Dijkstra）在无向路网上会产出
`…→X→Y→X→…` 的**死胡同支线**：路线飞出去几公里再原路返回。渲染时它会变成一条
明显的折返线段，并且把该段的动画节奏拉长。

### 处理链（顺序不可调换）

```
① 全局去环 / 去尖刺      ← 必须在切段之前
② 站点投影 + 索引单调化
③ 切分为站间段
④ 段内反走回消除（弦投影进度单调过滤）
⑤ 条件触发强力清理（垂距过大的绕行点）
⑥ 端点锚定到站点坐标
⑦ 合并 + 相邻去重 + 精度舍入
```

**①为什么必须在切段之前**：死胡同尖刺的顶点常常恰好落在站点索引上
（`hefu` 的站点索引是 `[0 2 13 21 27 39 42 52 67 76 79]`，两个尖刺顶点正好是 67 和 76）。
逐段处理时 `prev/cur/next` 跨越了段边界，段内视角看不到整条往返路径 —— 实测确认：
整条线处理能消除，逐段处理不能。

### 实测效果（在随包分发的示例数据集上）

| 指标 | 修复前 | 修复后 |
| --- | --- | --- |
| 无歧义死胡同尖刺（返回点距出发点 < 250 m） | 3 | **0** |
| 受影响的线路 | `hefu` 80→77 点、`shanghehang` 150→149 点、`hening` 45→42 点 | — |
| 站点偏差 | 95/95 站点 0.00 km | 不变 |
| 最大站间段比率 | 1.50 | 不变 |

### 不变量（`route.validate` 会检查）

| code | 严重度 | 含义 |
| --- | --- | --- |
| `TOO_FEW_POINTS` | error | 点数 < 2 |
| `STATION_OFF_ROUTE` | warn >0.5km / error >5km | 站点到轨迹的**点到线段**距离 |
| `SEGMENT_RATIO_EXCESSIVE` | error > 2.0 | 段长 / 直线距过高，动画节奏被拉长 |
| `SEGMENT_RATIO_HIGH` | warn > 1.5 | 同上，偏高 |
| `SPIKE_VISIBLE` | error | 走出再原路返回（转向角 >150° 且返回点距出发点 <250m） |
| `SPIKE_SHARP_TURN` | warn | 急转折返，可能是真实山区走线，需人工复核 |
| `LOOP_DETECTED` | error | 轨迹回到先前位置，地图上会出现"结" |
| `MAX_ADJACENT_GAP` | warn | 相邻点间距过大，疑似数据缺口 |
| `ENDPOINT_NOT_ANCHORED` | warn | 首末点未锚定到首末站 |
| `SEGMENT_NOT_FOUND` | error | 无法按站点顺序切段 |

**注意：不变量里没有"弦投影反走回"这一条。** 用 A→B 弦的投影进度判断反走回会对**弯曲路径
产生假阳性** —— 皖南山区的线路会有大量点在弦上投影回退，但列车一直在前进。
而且渲染时指示器是按**弧长**推进的，本来就永远向前。折返用**转向角**判断才是对的。

---

## 二、画法技法 `renderer/remotion/techniques/`

| 模块 | 导出 | 说明 |
| --- | --- | --- |
| `geo.js` | `mercator`, `createProjection`, `toPathD` | 墨卡托投影 + 自适应铺满。经度按 `cos(中心纬度)` 缩放，否则地图横向拉伸 |
| `geo.js` | `polylineLength`, `pointAtLength`, `tangentAngle` | 折线弧长工具。自带实现，不依赖 `@remotion/paths` |
| `schedule.js` | `buildTimeline`, `lineProgress`, `deriveMetrics`, `yearMilestones` | 按开通年份分组调度。**时长自适应**：把可用帧按年份组数与线路复杂度分配（参考实现用固定 10/24 帧，换时长就崩） |
| `labels.js` | `solveLabels`, `labelQuality` | 标签避让求解器 |

### 标签求解：放不下就不放

真实路网里站点标签的总数远超可读容量（示例数据 97 个站）。
求解器按**优先级**（枢纽 > 普通站）贪心放置，**放不下就丢弃** —— 这是 MapLibre
`symbol-sort-key` 的模型，也是真实地图的做法。

实测（1760×1900 地图区，97 个站）：

| 指标 | 值 |
| --- | --- |
| 放置 / 丢弃 | 79 / 18 |
| 标签重叠对 | **0** |
| 越界标签 | **0** |
| 枢纽站显示率 | **23 / 23**（枢纽从不被丢弃） |

保证：`no-overlap` / `in-bounds` / `order-stable` / `frame-deterministic`。

---

## 三、组合技法 `compositions/RouteGrowth.jsx`

组合画法、调度、求解器，产出"路网生长"镜头。消费 `recipe.style_tokens`
（DTCG 风格令牌）—— **新增风格不需要改渲染器代码**。

分区：标题徽章与主标题 → 里程计数器 → 年份进度条 → 地图（线路 + 站点 + 标签）→ 页脚来源。

派生指标（里程、年份、完成线路数）全部来自数据计算，不是写死的文案。

---

## 四、示例数据资产 `catalog/data/geo/anhui-transit-lines.json`

17 条线路 · 97 个站点 · 2008–2025 · 合计 2,972 km（安徽省境内）。

- 几何来源：OpenStreetMap contributors（ODbL-1.0）
- 迁移自用户认可的参考实现，并已用本扩展的 `route.repair` 重新生成
- 由 `HIMIND_REGEN_GEO=1 go test -run TestRegenerateExampleDataset` 可复跑重生成

---

## 五、成片验收 `dev/verify-render.py`

```bash
python dev/verify-render.py <video.mp4>
```

只依赖 ffmpeg/ffprobe + Python 标准库。检查"画面里到底有没有东西、动没动"：
分辨率、时长、帧数、画面复杂度是否随时间变化、内容是否单调递增、
路线颜色像素是否随线路生长上升、标题/地图/页脚三区是否都有内容。

最新一次实测（`route-growth` @ 安徽省高铁网络，2160×3840，18.05s，540 帧，3.00 MB）：

```
PASS  分辨率与模板画布一致        2160x3840
PASS  时长有效                  18.05s
PASS  有足够帧数                540 帧
PASS  画面不是纯色/空白          最少 327 种颜色
PASS  画面内容随时间变化          1.78% -> 8.82%
PASS  内容整体递增              17 段中 17/17 段非递减
PASS  路线随年份生长            路线像素 0.24% -> 1.74%
PASS  标题区有内容              最亮通道 255
PASS  地图区有内容              最亮通道 255
PASS  页脚水印已绘制            最亮通道 59（背景约 26）
```

---

## 六、Shot 库与公共底座

### 两级模板

```
成片模板（Storyboard）= 3–5 个 Shot 的编排 + 转场 + 节奏
镜头模板（Shot）      = 一个画面结构：数字 / 对比 / 引语 / 榜单 / 地图 …
```

加题材的正确做法是**组合 Shot**，不是写新的 composition。判据：新增题材时 `ui/src` 不应产生改动。

### 公共底座（`renderer/remotion/foundation/`）

| 模块 | 作用 | 为什么要它 |
| --- | --- | --- |
| `theme.js` | 调色板解析（DTCG 令牌 → 实际颜色）、画布缩放、安全区 | 风格是声明资产；渲染器不写死配色 |
| `text.js` | CJK 感知的宽度估算、字号自适应、折行 | 不用 DOM 测量，保证跨机器确定性 |
| `format.js` | 千分位、万/亿、百分比、变化量符号 | 中文短视频的数字表达是固定套路 |
| `motion.js` | `fade-up` / `spring-pop` / `slide-left|right` + 交错延迟 | 所有镜头共用一套节奏语言 |
| `background.jsx` | 渐变 + 网格 + 光斑 | 统一底色，避免每个镜头自己画背景 |

### 已落地的 P0 Shot

| Shot | 组成 | 数据契约 | 时长 |
| --- | --- | --- | --- |
| `BigNumber` 数字冲击 | 标签 → 巨型数字 → 单位 → 变化量 → 说明 | `catalog/contracts/big-number.schema.json` | 5s |
| `Compare` 数据对比 | 标题 → 上下两栏（数值 + 条形）→ 结论 | `catalog/contracts/compare.schema.json` | 7s |
| `Quote` 引语卡 | 引号 → 逐行揭示的正文 → 作者 | `catalog/contracts/quote.schema.json` | 6s |

### 字号基准（2160×3840）

参考实现里 `FONT.hero = 140`，但那是"标题"的尺度。实测发现：

- 数字冲击的数值若用 300px，只占画面高度 **8%**，看起来像正文而不像"冲击"；
- 调到 **640px** 后占 **25% 高度 / 83% 宽度**，才成立；
- 中文单位（公里/亿元）若与数字同行，会因宽度估算迫使字号缩到 520，因此**单位独立成行**。

### 契约门禁

`TestDefaultRecipesMatchDeclaredContracts` 会读取每个模板的 `data_contract`，
断言新建项目生成的默认数据包含 schema 里的 `required` 键 ——
否则用户新建项目会看到一个数据不完整的空壳。

`recipe.validate` 现在会**真正执行**契约子集（`data_contract.go`）：
`type / required / properties / items / minItems / maxItems / minimum / maximum /
minProperties / additionalProperties`。错误以 `data_contract_*` 返回，
AI 必须按路径修正，而不是把脏数据交给渲染器静默过滤。

---

## 七、Storyboard：多镜头组合

### 结构

```jsonc
{
  "transition_frames": 9,
  "storyboard": [
    {"shot": "big-number", "duration_seconds": 3, "data": {"value": 2972, "unit": "公里", "label": "安徽高铁总里程", "delta": "+12.4%"}},
    {"shot": "compare",    "duration_seconds": 4, "data": {"unit": "亿元", "left": {...}, "right": {...}, "verdict": "..."}},
    {"shot": "quote",      "duration_seconds": 4, "data": {"text": "...", "author": "..."}},
    {"shot": "big-number", "duration_seconds": 3, "data": {"value": 16, "unit": "市", "label": "市市通高铁"}}
  ]
}
```

`shot` 的值查 `shots/registry.js`；新增镜头只改那个文件，**不动 Storyboard，也不动创作台**。

### 两个必须知道的实现约束

**① `<Sequence>` 里拿不到镜头时长。**
`useVideoConfig().durationInFrames` 在 Sequence 内仍返回**整片时长**。
所以每个 Shot 必须接受显式 `durationFrames`：

```js
const {fps, durationInFrames: compositionDuration} = useVideoConfig();
const durationInFrames = Number(props.durationFrames) || compositionDuration;
```

不这么写，镜头内的数字滚动、时间轴调度会按整片时长计算，行为完全错乱。

**② 整片时长由镜头求和决定。**
`storyboardMetadata` 会读 `props.data.storyboard` 求和，覆盖 `recipe.duration_seconds`。
所以编辑镜头后不必手动改时长；`defaultRecipe` 也会算出同样的值，保证 UI 显示一致。

### 转场

交叉淡入淡出：相邻镜头在 `transition_frames` 内重叠，前一个淡出、后一个淡入。
不用 `@remotion/transitions`，避免给运行时增加依赖。

### 实测（14s / 4 镜头）

| 帧 | 镜头 | 内容覆盖 | 说明 |
| --- | --- | --- | --- |
| 45 | big-number（2972） | spanX 85% / spanY 27% | 数字主导 |
| 150 | compare | spanY 68% | 两栏分布 |
| 270 | quote | spanX 77% | 四行文字 |
| 380 | big-number（16） | spanX 36% | 短数字，宽度自然收缩 |

两两帧差异 **8.06%–13.21%**，确认镜头确实在切换。
`remotion compositions` 报告 Storyboard 时长 **420 帧（14.00s）**，与 3+4+4+3 一致。


### 可视化分镜编辑器

创作台主区在 Storyboard 项目下出现「预览 / 分镜」两个视图。分镜编辑器解决的是**结构编辑**，不是题材编辑：

- 镜头清单来自 `catalog/templates.json` 的 `storyboard_shot` + `storyboard_order`，外壳不认识 `big-number` / `route-growth`；
- 添加镜头时调用 `media.workbench.template.schema`，拿到该镜头的 `data_contract` 和默认数据；
- 有 schema 的字段按类型渲染表单；数组、自由对象或没有 schema 的镜头退化为原始 JSON；
- 支持上移/下移、删除、改时长、改标题/副标题，时长求和后写回 `recipe.duration_seconds`；
- 保存使用 `expected_updated_at` 做并发保护，保存后自动刷新项目修订。

**嵌套契约**：`storyboard.schema` 只约束镜头外壳；保存时 `validateStoryboardShotContracts` 会逐镜头查 `shots/registry.js` 对应的模板，再执行该模板自己的 `data_contract`。错误路径精确到：

```
data.storyboard[4].data.categories[0].value 必须 ≥ 0
```

这保证了“外层故事板合法、内层镜头数据损坏”不会被写进项目。

门禁：

- `TestStoryboardShotCatalogMatchesRendererRegistry`：模板声明与 `SHOT_REGISTRY` 双向一致；
- `TestTemplateSchemaProvidesDefaultsForStoryboardShots`：每个可编辑镜头都能拿到默认数据，默认数据通过契约；
- `TestStoryboardNestedShotContractIsValidated`：嵌套镜头契约生效；
- `TestStoryboardEditorDoesNotHardcodeShotIDs`：编辑器源码不得出现任何具体镜头 id。

### 实测（编辑后的 5 镜头）

在真机编辑器中完成了：增加「占比构成」镜头 → 把第一个镜头下移 → 把 `big-number` 时长改为 2.5s → 保存。
直接把保存后的 `recipe.json` 渲染成 MP4：

- 2160×3840 / 705 帧 / 23.55s / 6.25 MB
- `verify-render.py --profile storyboard` **8/8 PASS**
- 内容面积出现 **15 个档位**，确认镜头切换正常
- 头部/中部/页脚至少两处有内容：`head=55 / mid=255 / foot=62`

多镜头不能用“面积单调递增”验收：引用卡、纯文字卡的覆盖面积会低于前后镜头。
`--profile storyboard` 专门允许这种回落，但仍要求内容有明显变化且画面非空。

---

## 八、静帧回归（Golden Fixtures）

### 为什么不能只用 SSIM

实测：在 540×960 的基准图上画一个 **120×120 的红块**（占画面 2.8%），
SSIM 只从 1.0000 掉到 **0.9757** —— 高于 0.97 的常用阈值，**门禁会放行**。

SSIM 是全局结构相似度，对"小面积但明显"的改动不敏感。所以必须叠加一层差分统计：

| 指标 | 来源 | 抓什么 | 阈值 |
| --- | --- | --- | --- |
| `SSIM` | `-lavfi ssim` | 整体漂移（布局、配色、字号） | ≥ 0.97 |
| `YAVG` | `blend=difference,signalstats` | 平均亮度差（中等面积改动） | ≤ 0.8 |
| `YMAX` | 同上 | 任何一处强突变（小面积明显改动） | ≤ 40 |

同一张红块：`SSIM 0.9757 / YAVG 1.52 / YMAX 57` → **FAIL**。

### 目录

```
catalog/fixtures/
├── manifest.json                 # 模板 ↔ 配方 ↔ 帧号 ↔ 阈值
├── big-number/{recipe.json, frame-0090.png}
├── compare/{recipe.json, frame-0160.png}
├── quote/{recipe.json, frame-0120.png}
├── leaderboard-tech/{recipe.json, frame-0200.png}
├── route-growth/{recipe.json, frame-0300.png}
└── storyboard/{recipe.json, frame-0045.png, frame-0200.png, frame-0420.png}
```

基线尺寸 **540×960**（设计画布的 1/4），6 个模板共 **0.80 MB**。

### 运行

```powershell
# 比对（改完 Shot 后跑）
pwsh -File dev/regress-stills.ps1 -Runtime <workspace>/.himind-media/runtime/remotion-4.0.521

# 确认新画面正确后，重建基线
pwsh -File dev/regress-stills.ps1 -Runtime <...> -Update

# 只重建某个新增镜头的基线（不影响其他基线）
pwsh -File dev/regress-stills.ps1 -Runtime <...> -SyncFrom . -FixtureId composition -Update
```

### 常驻门禁

`TestGoldenFixturesAreComplete`（不渲染，`go test` 里常跑）检查：

- 清单引用的 recipe 与基线 PNG 都存在；
- 基线尺寸与清单声明一致（读 PNG IHDR）；
- **每个 `status: ready` 的模板都必须有基线** —— 否则新镜头会悄悄逃过回归；
- 阈值在合理区间。

> 判据：**新增一个 ready 镜头却没有基线，`go test` 会失败。**

---

## 九、Shot 库（完整）

| Shot | 组成 | 数据契约 | 用到的技法 |
| --- | --- | --- | --- |
| `BigNumber` 数字冲击 | 标签 → 巨型数字 → 单位 → 变化量 | `big-number.schema.json` | 数字格式化 + 字号自适应 |
| `Compare` 数据对比 | 标题 → 上下两栏（数值 + 条形）→ 结论 | `compare.schema.json` | 条形比例 + 左右滑入 |
| `Quote` 引语卡 | 引号 → 逐行揭示正文 → 作者 | `quote.schema.json` | 折行 + 交错入场 |
| `BarRace` 时序竞速 | 年份 → 排行行（名次 + 名称 + 条形 + 数值）→ 进度条 | `bar-race.schema.json` | `techniques/race.js` |
| `Timeline` 时间线 | 纵向主轴 → 左右交错事件卡 | `timeline.schema.json` | `foundation/layout1d.js` |
| `Choropleth` 分级填色 | 地图 + 色阶 + 图例 | `choropleth.schema.json` | 投影 + 色阶 + 标签避让 |
| `ShareBreakdown` 占比构成 | 环形图 → 堆叠条 → 图例 → 结论 | `composition.schema.json` | 角度映射 + 系列色 + 交错入场 |
| `Flow` 流程链路 | 标题 → 纵向节点链 → 结论 | `flow.schema.json` | 主轴生长 + 编号节点 + 交错入场 |
| `SankeyFlow` 桑基流向 | 总量 → 流向带 → 分类节点 | `sankey.schema.json` | `techniques/sankey.js` 布局 + 带状路径 |
| `RouteGrowth` 路网生长 | 地图 + 线路生长 + 站点 | `catalog/data/geo/*` | 投影 + 调度 + 标签避让 |

### bar-race 的关键设计

**整数年稳定快照**：只在整数年计算排名，帧间用 smoothstep 插值名次。
直接用浮点年份排序会因微小数值差异导致名次抖动（"第 3 名和第 4 名反复横跳"）。

### timeline 的关键设计

事件在时间轴上的真实位置会重叠。`spread1D` 做一维非重叠布局：

```
前向：pos[i] = max(anchor[i], pos[i-1] + minGap)
后向：越过 max 时整体回压，并重新保证间距
```

纯函数，无迭代收敛问题；同一输入必然得到同一布局。

### 实测

| Shot | 内容覆盖 | 说明 |
| --- | --- | --- |
| BarRace | spanY 84% | 5 行自适应行高，填满画面 |
| Timeline | spanY 85% | 6 个事件左右交错，主轴贯穿 |

---

## 十、Choropleth：分级填色地图

### 结构

```jsonc
{
  "geo": {"type": "FeatureCollection", "features": [...]},
  "values": {"合肥市": 14210, "芜湖市": 5290, ...},
  "unit": "亿元",
  "subtitle": "2025 年各地市 GDP"
}
```

### 复用

- **投影**：`techniques/geo.js` 的 `createProjection`（经度按 `cos(纬度)` 缩放，否则地图横向拉伸）
- **标签避让**：`techniques/labels.js` 的 `solveLabels`（16 个市名按数值优先级放置，密集处自动让位）
- **配色**：`foundation/theme.js` 的 `withAlpha`，从 `color.accent` 派生 5 档透明度，**不写死色阶**

### 与旧占位模板的关系

`gdp-map-story` 之前挂的是 `MapStory`（网格线 + 取模假坐标，没有地图）。
`choropleth` 是真实实现；旧模板保留为 `placeholder` 用于兼容，描述已标注"已被 choropleth 取代"。

### 实测

- 16 个市级行政区（12 MultiPolygon + 4 Polygon）
- 画面覆盖 `spanY 81%`，图例在底部
- 基线帧 `choropleth#200`，SSIM 1.0000

### 一处性能修正

GeoJSON 内联进配方后 `recipe.json` 达到 **544 KB**（美化展开）。
这暴露了 `project.list` 会返回所有项目的完整配方的问题 —— 已改为**只返回元数据**，需要完整配方时用 `project.get`。
fixture 配方也做了压缩：choropleth 544 → **125.6 KB**，storyboard 88.5 → **23.9 KB**。

---

## 十一、数据资产引用（`$ref`）

### 问题

choropleth 的 GeoJSON 有 124 KB，内联进配方后 `recipe.json` 变成 **544 KB**（美化展开）。
一个"安徽 16 市"系列 16 条视频就要复制 16 份，改一次边界要改 16 个文件。

### 写法

```jsonc
{
  "geo":    {"$ref": "plugin:catalog/data/geo/anhui-cities.geojson"},
  "values": {"$ref": "workspace:assets/2025-gdp.json"}
}
```

| scheme | 读取范围 | 用途 |
| --- | --- | --- |
| `plugin:` | 插件包内（`pluginRoot()`） | 随扩展分发的地理、字典等基础数据 |
| `workspace:` | 调用方的 `workspace_root` 内 | 用户自己的数据文件 |

### 关键设计：**物化发生在渲染/预览前**

```
project.json / recipe.json   只存 $ref（1.8 KB）
        ↓
preview.start / preview.player / render.start
        ↓
materializeRecipe()          展开所有 $ref
        ↓
jobs/<job>.recipe.json       完整数据（562 KB）→ 渲染器只需读它
```

这样做有三个好处：

1. **渲染器不需要知道引用** —— 它拿到的永远是完整数据；
2. **作业快照仍然自包含** —— "同配方同输出"不受影响，回放不依赖外部文件是否还在；
3. **读路径受限** —— `plugin:` 越不出插件包，`workspace:` 越不出工作区，测试里有专门的越界用例。

### 实测

| 项 | 内联 | 引用 |
| --- | --- | --- |
| 存储的 `recipe.json` | 544 KB | **1.8 KB** |
| 预览 HTML（物化后） | — | 562 KB（含 MultiPolygon，无 `$ref`） |

### 相关测试

- `TestRecipeRefsMaterializeFromPluginAndWorkspace`
- `TestRecipeRefsRejectEscapingPaths`（`plugin:../../secret.txt`、`workspace:../outside.json`、`https://…`、裸相对路径全部拒绝）
- `TestDefaultChoroplethRecipeUsesReference`（默认配方必须是引用形式，且物化后 16 个区域）

---

## 十二、占比构成（`composition`）

### 它解决的不是「对比」，而是「整体由什么组成」

`compare` 回答 A 与 B 谁更大；`composition` 回答一件事由哪些部分组成、各占多少。
这两类问题在短视频脚本里经常相邻出现，但不应该共用一个模板：

```
对比：合肥 12,674 亿元  vs  芜湖 5,121 亿元
构成：文旅消费 = 交通 28% + 住宿 24% + 餐饮 22% + 景区 16% + 购物 10%
```

### 画面结构

```
标题 / 副标题
      ↓
环形图（中心显示总计）
      ↓
100% 堆叠条（回答结构比例）
      ↓
自动图例（标签 + 原值 + 百分比）
      ↓
一句话结论（自动找最大项，可被 data.insight 覆盖）
```

### 数据契约

```json
{
  "categories": [
    {"label": "交通", "value": 2860},
    {"label": "住宿", "value": 2450},
    {"label": "餐饮", "value": 2200}
  ],
  "unit": "亿元",
  "total_label": "文旅消费",
  "decimals": 0,
  "insight": "可选，覆盖自动结论"
}
```

约束（由 `catalog/contracts/composition.schema.json` 声明）：

- 2–8 个分类；每项必须有 `label` 与非负 `value`；
- **百分比不由调用方填写**，渲染器从 `value / total` 推导 —— 避免"分类值相加不等于 100%"；
- 类别颜色可以逐项覆盖；没有覆盖时使用风格令牌 `color.series.1..8`。

### 关键设计

1. **系列色是声明资产**
   `styles.json` 新增 `color.series.1..8`，渲染器只读令牌。
   新增风格时不需要改 `ShareBreakdown.jsx`；没有令牌的旧配方回退到内置色板。

2. **不是"把饼图搬到竖屏"**
   竖屏宽度有限，所以采用「环形图 + 100% 堆叠条 + 两列图例」：
   环形图负责整体印象，堆叠条负责结构，图例负责可读数值。

3. **自动结论是模板的一部分**
   短视频需要一个"所以呢"的落点。模板自动取最大类别生成结论，
   调用方可以用 `data.insight` 覆盖，避免把叙事判断硬编码在渲染器里。

4. **确定性**
   SVG 弧线路径、入场动画、百分比文本全部由帧号与输入决定；
   不使用 DOM 测量，也不依赖随机数。

### 实测

- 模板 `composition` / 10 秒 / 5 个分类
- 基线帧 `composition#30/#90/#180/#270`
- `go test ./...` 通过，静帧回归 **18/18 PASS**（SSIM 1.0000）
- 真实成片：2160×3840 / 300 帧 / 10.05s / 2.11 MB
- `dev/verify-render.py` **10/10 PASS**（内容面积 10.44% → 13.44%，+3.00pp）
- 两次渲染 SHA-256 完全一致：`8C117485…`

### 定向重建基线

新增镜头时不必重算全部基线：

```powershell
pwsh -File dev/regress-stills.ps1 -Runtime <...> -SyncFrom . -FixtureId composition -Update
```

---

## 十三、预览 Player 的缓存失效

### 问题

`preview.player` 把 `player.jsx` 用 esbuild 打成 `player.bundle.js` 后缓存在运行时目录。
最初只判断"bundle 是否存在"，于是新增 Shot 时出现了一个很隐蔽的错位：

```
index.jsx        已同步，包含 ShareBreakdown
player.bundle.js 仍是旧缓存，只注册了前 4 个 composition
预览             显示"未知的 composition：ShareBreakdown"
成片渲染         正常（走的是 index.jsx，不走 bundle）
```

这种"预览坏、渲染好"的问题最容易在联调时被误判成模板写错。

### 修复

`ensurePlayerBundle()` 在复用缓存前先计算渲染器源码指纹：

- 覆盖 `player.jsx`、`index.jsx`、`package.json`；
- 覆盖 `foundation/`、`shots/`、`techniques/`、`compositions/` 下的 JS/JSX；
- 忽略 `node_modules/`、`package-lock.json` 和 `player.bundle.js` 本身；
- 指纹写入 `player.bundle.js.sha256`，不一致就重建。

门禁：`TestPlayerBundleFingerprintTracksRendererSources`
（Shot 改动必须改变指纹；`node_modules` 等非输入文件不能改变指纹）。

### 实测

- 旧 bundle：预览报"未知的 composition"
- 移除/失效缓存后重建：预览正常显示 `文旅消费 10,210 亿元 / 5 类构成 / 28%…`
- 同一份 `preview` HTML 内含 5 个分类和 Remotion Player，不是静态占位图

---

## 十四、流程链路（`flow`）

### 与 timeline 的分工

```
timeline：2008 年开通 → 2011 年通车 → 2020 年贯通      （时间）
flow：    洞察 → 脚本 → 分镜 → 成片 → 分发              （步骤）
```

`timeline` 的位置由年份映射决定；`flow` 的位置由**执行顺序**决定，不应把年份硬塞进流程节点。

### 画面结构

```
标题 / 副标题
      ↓
纵向主轴（随帧生长）
      ↓
编号节点 + 卡片（label + description）
      ↓
结论胶囊（data.conclusion，可选）
```

### 数据契约

```json
{
  "steps": [
    {"label": "洞察", "description": "从数据里找出值得讲的改变"},
    {"label": "脚本", "description": "把改变写成三句话的叙事"},
    {"label": "分镜", "description": "为每句话选择画面与节奏"}
  ],
  "conclusion": "一条流水线，从想法到成片"
}
```

约束（`catalog/contracts/flow.schema.json`）：

- 2–6 个步骤；每项必须有 `label`，`description` 可选；
- 步骤编号由渲染器生成，调用方不传；
- 主轴生长、卡片交错入场全部由帧号驱动；
- 颜色复用 `color.series.1..6`，新增风格不改代码。

### 实测

- 模板 `flow` / 10 秒 / 5 个步骤
- 基线帧 `flow#60`、`flow#150`、`flow#270`（分别覆盖主轴生长、卡片入场、结论出现）
- 静帧回归通过（SSIM 1.0000）
- 真实成片：2160×3840 / 300 帧 / 10.05s / 1.21 MB
- `dev/verify-render.py` **10/10 PASS**（内容面积 2.06% → 33.90%，+31.85pp）

---

## 十五、桑基流向图（`sankey-flow`）

### 它解决的不是“占比”，而是“流向”

`composition` 回答“每一类占多少”；  
`sankey-flow` 回答“总量流向了哪些部分，各占多少”。

```text
文旅消费 10,210 亿元
        ├── 交通   2,860 亿元
        ├── 住宿   2,450 亿元
        ├── 餐饮   2,200 亿元
        ├── 景区   1,680 亿元
        └── 购物   1,020 亿元
```

### 结构

- `techniques/sankey.js`：纯函数节点/流向布局；
- `shots/SankeyFlow.jsx`：节点、流向带、标签和动画；
- `catalog/contracts/sankey.schema.json`：links 契约；
- `catalog/fixtures/sankey-flow/`：黄金静帧。

### 布局技法

```text
link.value / total → 连线厚度
node.value / total → 节点高度
source/target 顺序 → 节点列顺序
```

同一侧节点按输入顺序排列，空余高度居中；连线在节点内部按值顺序堆叠。

### 数据契约

```json
{
  "links": [
    {"source": "文旅消费", "target": "交通", "value": 2860},
    {"source": "文旅消费", "target": "住宿", "value": 2450}
  ],
  "unit": "亿元",
  "total_label": "文旅消费"
}
```

### 与 data.bind 的配合

`data.bind` 支持：

- 已有 `links` 直接透传；
- `items` 自动映射为 `source=total_label, target=name, value=value`；
- 映射后执行 `sankey.schema.json` 契约校验。

### 实测

- 模板 `sankey-flow` / 10 秒 / 5 个目标节点
- 真实成片：2160×3840 / 300 帧 / 10.05s / 1.23 MB
- `verify-render.py` **10/10 PASS**
- 内容面积 1.22% → 43.81%（+42.60pp）
- 两次渲染 SHA-256 一致：`A3233E1E…`
- 静帧基线：`sankey-flow#30/#90/#180/#270`

## 十六、线网生长图（`transit-network` / Canvas 2D）

> 适用于地铁、市域铁路、有轨电车等“官方线网图”风格；
> 目标不是 3D 地理地图，而是白底、粗色线路、清晰站名的信息图。

### 何时使用

- 用户说“高铁路线图 / 地铁路线图 / 线网示意图 / 线路生长动画”；
- 重点是拓扑关系和换乘，不是地形起伏；
- 需要在 15–40 秒内讲清“线路增加 / 网络成形 / 换乘节点”。

### 数据最小结构

```json
{
  "lines": [
    {
      "id": "line-1",
      "name": "1 号线",
      "color": "#E60012",
      "points": [[120, 300], [480, 300], [480, 900]],
      "stations": [
        {"id": "station-a", "name": "示例站", "x": 120, "y": 300}
      ]
    }
  ],
  "interchanges": ["station-a"],
  "river": [[0, 120], [300, 220], [620, 180]]
}
```

### 核心算法

1. **折线累计距离**：`points` 先展开为 `[x, y, distance]`，站点用最近投影点落到线上；
2. **线路生长**：用 Canvas 的 `setLineDash([total, total])` + `lineDashOffset` 制造“从起点画到终点”；
3. **站点出现**：按站点在线上的累计距离排序，与线路生长进度同步；
4. **换乘脉冲**：换乘站单独画外环，按时间轴做 1–2 次脉冲，不要全站一起闪；
5. **标签布局**：优先调用 `media.workbench.label.place`，8 方向候选位 + 障碍检测 + 引线回退；
6. **年份轴**：年份变化只改变图例高亮和文案，不重排线路，避免观众迷失。

### 站名避让的可复用部分

后端能力：

```text
media.workbench.label.place
media.workbench.label.validate
```

输入包含：

```text
viewport / safe_area
stations[]（id、x、y、priority、interchange）
lines[]（折线障碍）
existing_labels[]（已放置标签）
```

输出：

```text
placements[]（box、anchor、visible、leader）
findings[]（越界、重叠、不可见）
```

关键约束：

- 同一站在整段动画中位置固定；
- 换乘站优先级高于普通站；
- 允许带引线回退，但不允许标签压住线路或站点；
- `hidden > 0` 必须进入交付说明，不能静默丢站名。

### 质量门禁

- 最终帧必须 PNG；
- 白底平面风格在 Style Pack 声明：

```json
{"quality": {"banding_gate": false}}
```

  这样色带检查返回 `skipped`，但仍记录平坦段指标；
- 暗色/渐变风格不要关闭该门禁；
- 必须检查黑帧、静帧、音频响度和成片时长；
- 示意图必须标注“非完整官方线网 / 不用于出行”。

### 参考实现

```text
style-packs/transit-network/
docs/transit-network-hefei-demo.md
docs/oss-evaluation/transit-map.md
docs/oss-evaluation/label-placement.md
```

## 十七、开通时间轴（`transit-opening-timeline`）

> 适用于“某城市地铁每年 12/26 开通新线”“线路开通史”“多年网络生长”这类叙事。  
> 参考拆解：`docs/pattern/douyin-hefei-1226.md`

### 与 `transit-network` 的区别

| | `transit-network` | `transit-opening-timeline` |
| --- | --- | --- |
| 叙事 | 一次线网生长 | 按开通事件逐条生长 |
| 主角 | 网络 | 日期 + 区段 + 累计里程 |
| 底部信息 | 图例 + 年份轴 | 日期卡片 + 里程表 + 12/26 徽标 |
| 时长 | 30s | 38s（可扩展） |

### 数据契约

```json
{
  "events": [
    {
      "id": "e2016",
      "date": "2016.12.26",
      "year": 2016,
      "lineId": "L1",
      "lineName": "1号线",
      "segment": "合肥火车站 → 望湖城",
      "km": 24.6,
      "note": "首条线路"
    }
  ],
  "stats": {"lines": 5, "stations": 20, "km": 171.2}
}
```

### 运动语法

```text
卡片进入 → 新线描边生长 → 站点弹出
→ 里程滚动 → 镜头推近 → 停留 → 卡片退出
```

### 关键实现

- 当前线路：彩色 + 白色描边 + 轻 glow；
- 历史线路：34% 不透明度；
- 未来线路：隐藏；
- 当前线路站名全显示，历史线路只显示换乘/终点；
- 里程用 `easeOutExpo` 从上一事件滚到当前值；
- 日期卡片是主叙事，地图是证据；
- 所有事件共享同一条 `12/26` 记忆点。

### 常见坑

- 视频时长不是 30s 时，编码阶段必须把 `--duration` 传给 `render.mjs --encode`，否则 30s 环境声床会把成片截断；
- 里程滚动值不是最终值，截图 OCR 只能用于理解节奏，不能直接当官方数据；
- 全量线网站名不能一次性全部标注，否则 9:16 画面会拥挤；
- 参考视频的版式/素材有版权，只能沉淀通用技法，不能复制画面。
