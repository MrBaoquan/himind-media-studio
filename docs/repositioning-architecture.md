# 重新定位与架构方案：自媒体创作台

> 状态：P0 已落地（契约层），P1/P2 待执行
> 目标版本：0.5.0
> 日期：2026-10-17
>
> **方向更新（2026-10-17）**：对外产品定位已收敛为「以可沉淀视觉能力为质量杠杆的短视频生成器」。
> 本文只描述生成器内部的「创作台」边界，不再代表对外产品定位。
> 最新方向与同类开源对比见 [generator-direction.md](generator-direction.md)。

## 0. 一句话

**把「自媒体创作台」重新定位为「自媒体创作台」：它不渲染、不做领域数据、不写技法代码；它管项目状态、资产解析、契约校验与门禁汇总。真正干活的是独立的能力拓展。**

---

## 1. 为什么改

| 现状 | 问题 |
| --- | --- |
| 名字叫"引擎" | 渲染由 Remotion 完成，插件只 shell out 一个子进程 —— **层错位** |
| 一个插件同时是工作台 / 资产库 / 渲染编排器 | 名字只能取最大公约数，`description` 与名称互相打架 |
| `render.*` 与技法代码绑在创作插件里 | 换引擎要换整个插件，**可插拔是口号** |
| 约束散在 Skill 与代码里 | 没有统一清单，无法审计"哪条规则由谁执行" |

---

## 2. 三条边界规则（用测试守住）

### 规则一：插件之间不互相调用

内核没有跨插件调用通道（`CapabilityHandler::PluginCapability` 只能由内核调起）。
因此 `route.validate → recipe.validate → render.start` 的串联**只能由 Skill / Workflow 完成**。

⇒ **Workflow 是跨插件编排的唯一合法载体**，也是 harness 的核心。

### 规则二：状态跟着创作台走，代码跟着渲染器走

- 项目、修订史、素材台账、资产解析、门禁结果 → **创作台**
- 运行时、技法实现、帧 → **渲染适配器**

判据：**换掉 Remotion 换成 HyperFrames，创作台一行都不用改。**

### 规则三：能力前缀 = 职责边界

见 §4。由 `catalog/boundaries.json` + `boundary_test.go` 强制。

---

## 3. 目标形态

```
AI 客户端对话
   │  Skill 教方法 · Workflow 定顺序 · Capability 给动作
   ▼
自媒体创作台（插件）
   项目 / 系列 / 季 · 修订史 · 素材台账 · 资产解析 · 契约 · 门禁汇总 · 三栏 UI
   ✗ 不渲染  ✗ 不含领域数据  ✗ 不含技法代码
   │
   ├── 领域能力拓展（插件）：地理管线 / 素材检索 / 素材下载 / TTS / 字幕
   ├── 执行适配器（插件）：Remotion / HyperFrames（含技法代码）
   └── 声明资产（JSON，零发版）：版式 / 风格 / 数据契约 / 黄金样例
```

### 为什么技法代码必须随渲染器发版

技法要在渲染器的 JS 运行时里执行。若允许从工作区加载技法代码，等于允许任意代码执行。
所以：

| 层 | 能否零发版扩展 |
| --- | --- |
| 版式 / 风格 / 数据契约 / 黄金样例 | **能**（三级作用域：项目 > 工作区 > 内置） |
| 技法（画法 / 求解器 / 管线） | **不能**（随渲染适配器发版） |

这条边界必须提前写进文档，否则会承诺一个做不到的"随时加技法"。

---

## 4. 能力前缀分配

| 前缀 | 归属 | 说明 |
| --- | --- | --- |
| `media.workbench.project.*` | 创作台 | 项目 / 系列 / 季 / 修订 / 归档 |
| `media.workbench.catalog.*` | 创作台 | 聚合视图（模板 + 风格 + 版式 + 适配器） |
| `media.workbench.template.*` | 创作台 | 模板/镜头声明、数据契约与默认数据 |
| `media.workbench.recipe.*` | 创作台 | 配方契约校验 |
| `media.workbench.feedback.*` | 创作台 | 反馈记录与读取 |
| `media.workbench.candidate.*` | 创作台 | 沉淀候选 |
| `media.workbench.quality.*` | 创作台 | 门禁汇总（调用各能力的 findings） |
| `media.workbench.preview.storyboard` | 创作台 | 故事板（HTML，不依赖渲染器） |
| `media.workbench.constraint.*` | 创作台 | **约束清单（harness 的 API）** |
| `media.geo.*` | 地理管线 | 轨迹修复 / 校验（P2 迁出） |
| `media.render.*` | 渲染适配器 | 渲染任务（P1 迁出） |
| `media.runtime.*` | 渲染适配器 | 运行时状态（P1 迁出） |
| `media.artifact.*` | 渲染适配器 | 产物（P1 迁出） |

多个插件共享 `media.workbench` 命名空间是允许的（ADR-0119 只要求能力 ID 不重复）。

---

## 5. 迁移阶段与验收

| 阶段 | 动作 | 验收（可执行） |
| --- | --- | --- |
| **P0**（本次） | 改名；`constraints.json`；`boundaries.json`；边界测试 | `go test ./...` 全过；`boundary_test` 断言当前阶段归属一致 |
| **P1** | `render.*` / `artifact.*` / `runtime.*` + 技法代码 → 新插件 `com.himind.video-render-remotion`；创作台只留故事板 | 用**假适配器**跑通 `preview → render → artifact`，创作台零改动 |
| **P2** | `route.*` → 新插件 `com.himind.geo-pipeline`；加 OSM 快照与验收 | 跨插件串联由 Workflow 跑通 |
| **P3** | `media-workbench-production` workflow：loop 反馈 + candidate freeze + 渲染前审批 | `workflow.run.start` 跑到 `video_ready` |
| **P4** | `library.publish`：候选 → 工作区资产（零发版）；UI 三栏工作台 | 一条反馈改善下一个项目 |

**P0 成本极低**：显示名/描述 + 两份 JSON + 两个测试，能力 ID 与命名空间不动。
**P1/P2 是真正的重构**：会动 capability provider，需要一次版本迁移与新制品。

---

## 6. Harness：约束清单

`catalog/constraints.json` 是**可审计**的约束来源。原则：

> **每条约束必须指向一个执行点；找不到执行点的约束不写进清单。**

```jsonc
{
  "schema": "himind.video.constraints.v1",
  "id": "media-workbench.standard",
  "version": "1.0.0",
  "rules": [
    { "code": "SAFE_AREA", "level": "error", "stage": "recipe",
      "enforced_by": { "kind": "capability", "id": "media.workbench.recipe.validate" } }
  ]
}
```

`enforced_by.kind` 只允许：

| kind | 含义 | 校验方式 |
| --- | --- | --- |
| `capability` | 由某个能力强制执行 | 该能力 ID 必须存在于 `plugin.json` |
| `schema` | 由输入/输出 JSON Schema 强制 | 必须是 `input_schema` / `artifact.schema` |
| `workflow` | 由 Workflow 步骤强制 | 步骤需声明 `approval_required` 或 `depends_on` |
| `test` | 由测试/回归强制 | 必须在测试登记表里 |
| `runtime` | 由运行时环境强制 | 例如 `workspace_root` 边界 |

创作台通过 `media.workbench.constraint.list` 暴露这份清单，AI 可以直接查询"我被什么约束着"。

---

## 7. 边界适应度函数

`catalog/boundaries.json` 声明**每个阶段**谁拥有哪些前缀，`boundary_test.go` 断言实际能力与声明一致：

```jsonc
{
  "stage": "p0",
  "owned": {
    "com.himind.media-workbench": ["media.workbench."]
  },
  "planned": {
    "p1": { "com.himind.video-render-remotion": ["media.render.", "media.artifact.", "media.runtime."] },
    "p2": { "com.himind.geo-pipeline": ["media.geo."] }
  }
}
```

切到 P1 时改 `stage` 为 `p1`，测试立刻要求把渲染能力搬走 —— **重构变成机械步骤，而不是靠自觉**。

---

## 8. 创作台 UI（P4）

三栏工作台，预览优先：

```
┌──────────────────────────────────────────────────────────────┐
│ 自媒体创作台   工作区 ▾  [新建] [从模板新建] [批量出片] [交给 AI] │
├──────────────┬──────────────────────────┬────────────────────┤
│ 项目树        │        预览画布           │ 属性 / 质量 / 反馈   │
│ 系列·状态     │      (Player / 故事板)     │ 模板 风格 时长       │
│              │                          │ 几何 标签 决定性      │
│              │  ┌────────────────────┐   │                    │
│              │  │ 分镜条 2008 ── 2025 │   │                    │
│              │  └────────────────────┘   │                    │
├──────────────┴──────────────────────────┴────────────────────┤
│ 任务: 渲染 62% │ 产物: v3.mp4 3.0MB │ 候选: 2 个待复核          │
└──────────────────────────────────────────────────────────────┘
```

**约束**：插件窗口 1100×760；单资源 64 MiB；`.hmpkg` **排除 `dist/`** ——
若用 Vite/React，构建产物必须落 `ui/` 而不是 `dist/`。

---

## 9. 取舍与风险

| 取舍 | 决定 | 理由 |
| --- | --- | --- |
| 技法能否独立安装 | **不能** | 需在渲染器进程内执行，独立安装 = 任意代码加载 |
| 现在就把渲染拆出去 | **P1 再做** | 只有一个引擎时物理拆分只增加安装摩擦；先做内部适配器边界 |
| 质量门禁放哪 | **能力自带不变量** | geo 管几何、renderer 管标签、workbench 管契约；Workflow 按序汇总 |
| 项目状态放哪 | 工作区 `.himind-media/` | 作品随工作区走；偏好放 `HIMIND_PLUGIN_DATA_ROOT` |
| 仓库名是否改 | **暂不改** | GitHub 改名会断扩展源地址；先把显示名与能力边界做对 |

---

## 10. 本次落地清单（P0）

1. `plugin.json` 显示名 → **自媒体创作台**；描述改为"项目与资产中枢"
2. `catalog/constraints.json` — 约束清单
3. `catalog/boundaries.json` — 边界声明
4. `media.workbench.constraint.list` — 暴露约束清单
5. `manifest_test.go` — 约束执行点断言 + 边界适应度函数
6. `docs/repositioning-architecture.md` — 本文件

---

## 11. UI 技术选型

### 结论

**用框架。推荐：Vite + React + TypeScript + Tailwind v4 + shadcn/ui（Radix 基元）。**

不用 Ant Design / Element Plus：它们是后台管理式设计语言，体量大、覆盖成本高；创作台需要的是"工具型"观感（面板、时间轴、画布）。

### 为什么是 React

- Remotion 本身就是 React —— 整个视频生态（技法、模板、示例）都是 React 方言，团队心智一致；
- shadcn/ui 是「复制进仓库」的组件，**不是运行时依赖**：没有版本漂移、没有 vendor lock、可以直接改；
- Radix 提供可访问的 Dialog / Popover / Tabs / Tooltip —— 专业工作台的基础件。

### 六条硬约束（不满足会「本地能跑、装完就废」）

| # | 约束 | 为什么 | 做法 |
| --- | --- | --- | --- |
| 1 | **base 必须是相对路径** | 资源以 `plugin-ui://localhost/<plugin>/<view>/<entry>` 提供；绝对路径 `/assets/x.js` 会被解析成 plugin_id=assets | Vite `base: './'` |
| 2 | **outDir 不能是 dist/** | 打包器排除 `dist`、`node_modules`，会被**静默丢掉** | `build.outDir: '../ui/app'` |
| 3 | **Hash 路由** | 刷新时路径会变成资源路径 → 404 | `#/projects/xxx`，不用 History 路由 |
| 4 | **运行时不用 CDN** | 离线可用 + 供应链安全 + 导航白名单限制 | 所有依赖打进包 |
| 5 | **不用 Service Worker；不依赖 Web Worker** | 自定义协议下不可靠 | 重计算走能力，不在前端做 |
| 6 | **字体自托管或系统字体** | 中文字体动辄数 MB；单资源上限 64 MiB | 默认 Microsoft YaHei / PingFang SC，特殊字体子集化 |

### 两个容易被忽略的产品约束

| 约束 | 影响 |
| --- | --- |
| **插件窗口固定 1100×760**（内核硬编码 `inner_size(1100, 760)`） | 创作台要按 1100×760 设计成"能用"，而不是按 1440×900 设计再压缩。左/右栏必须可折叠 |
| **每次写能力都过宿主审批**（`risk_level != read_only` 自动需要确认） | UI **不能按每个按键自动保存**，否则会刷屏审批。要显式保存 + 脏状态提示 |

### 体积预算

| 项 | 预算 |
| --- | --- |
| 主 JS（gzip） | ≤ 400 KB |
| CSS（gzip） | ≤ 60 KB |
| 图标 | 按需引入（lucide-react），不用整包字体图标 |
| 字体 | 系统字体优先；确需自托管时按字集子集化 |

### 目录

```
ui/                                   # UI 源码（Web 项目，不随插件包分发）
├── package.json
├── vite.config.ts                    # base:'./' + outDir 指向插件包
├── index.html
└── src/
    ├── main.tsx
    ├── api/capability.ts             # invoke_plugin_view_capability 封装 + 浏览器 mock
    ├── components/ui.tsx             # Badge / Kv / Empty 等原语
    └── features/                     # 项目树 / 预览 / 分镜编辑器 / 时间轴 / 属性+质量 / 状态栏

plugins/com.himind.media-workbench/media-workbench/ui/
├── index.html                        # 旧版单文件（迁移期保留，可删）
└── app/                              # ★ 构建产物：入库、随包分发
    ├── index.html
    └── assets/
```

> 源码与包体分开：插件包只带**构建产物**，不带 node_modules 与源工程。

> 注：`ui/app/` 是**入库的构建产物**。这是刻意取舍 —— 打包器排除 `dist/`，
> 而插件安装后不跑构建，所以产物必须随包分发。

---

## 12. 规则四：创作台不认识题材

这是原型阶段踩过的坑，必须写成硬约束。

### 反例（已修复）

```tsx
// 外壳曾经这样写 —— 错
const lines = recipe?.data?.lines ?? [];
<Kv label="线路" value={`${lines.length} 条`} />
<Kv label="里程" value={`${recipe.data.totalMileage} km`} />
function buildProjection(lines: TransitLine[], ...) { ... }   // 还抄了一份渲染器的投影
```

后果：
- 每加一个题材（排行榜 / 图片叙事 / 口播）都要改外壳；
- 外壳与渲染器各有一份投影实现，几何一旦调整就会漂移；
- 「铁路专用创作台」的观感——而它本该是通用工作台。

### 正确边界

| 层 | 认识什么 | 不认识什么 |
| --- | --- | --- |
| 创作台 | 配方的**通用外壳**（模板/风格/版本/画布/安全区/时长/文案）+ 项目/修订/反馈/任务/产物 | `lines` / `items` / `media` / `openYear` / 任何题材字段 |
| 模板 | 自己的 `data` 结构、composition、插槽 | 其他模板的题材 |
| 渲染器 | 如何把该模板的 data 画成帧 | 项目与资产 |

### 落地方式

1. `Recipe.data` 的类型是 `Record<string, unknown>`，**不写任何题材字段**；
2. 数据面板只做**结构概览**：`lines → 17 项`、`region → 1 个字段`、`totalMileage → 2972`；
3. **预览由渲染器产出**（`preview.player` → artifact → iframe），外壳不绘制画面；
4. 时间轴在外壳里只是**时长标尺**；分镜/年份/事件属于模板能力。

> 判据：**新增一个题材时，`ui/src` 里不应该产生任何改动。**



