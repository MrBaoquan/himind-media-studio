# 短视频生成器实施架构

> 状态：实施方案  
> 日期：2026-10-17  
> 落地状态：M0 规划层、video-director Skill、generate.run、Plan 审阅 UI、单镜头系列、data.ingest / data.bind 与 generate-video Workflow 0.1.4 已落地；安徽高铁已真实出 MP4。Workflow 真机运行、Candidate 提升、批量渲染与多镜头系列尚未开始。
> 上游方向：[generator-direction.md](generator-direction.md)  
> 实施计划：[generator-implementation-plan.md](generator-implementation-plan.md)

## 一、目标与非目标

### 目标

1. 用户通过 AI 对话提出需求，系统能生成一条可预览、可修改、可发布的短视频；
2. 同一类内容可以批量、稳定地生产；
3. 好结果可以沉淀为下一次可调用的能力；
4. 同一份 Recipe 可以被确定性重放；
5. 生成过程可审计：数据、模板、风格、运行时、资产版本都能追溯；
6. 能力可以按插件/能力包扩展，而不是不断改主程序。

### 非目标

1. 不做通用非线性剪辑器（NLE）；
2. 不训练视频生成模型；
3. 不追求“任意主题、任意风格、完全开放”的无限创作；
4. 不在第一阶段做实时交互视频；
5. 不为每个需求新增一个专用模板。

## 二、核心设计原则

### 1. AI 负责规划，执行层负责确定性

AI 客户端负责理解需求、选择叙事结构、生成计划。  
插件能力负责校验、编译、渲染和记录。  
**渲染器不调用 LLM，LLM 不直接控制帧。**

### 2. Recipe 是规划与执行之间的唯一契约

```text
NarrativePlan / StoryboardPlan
        ↓ 编译
      Recipe
        ↓
    Renderer
```

AI 可以反复修改 Plan，但只有通过校验的 Recipe 才能进入渲染。

### 3. 先规划，再渲染

生成器必须先输出可审查的计划：

- 讲什么；
- 为什么这样讲；
- 需要哪些数据；
- 使用哪些镜头；
- 每个镜头多长；
- 预计总时长和风险。

用户确认后再进入渲染。

### 4. 资产和版本不可变

模板、风格、数据契约、黄金样例、运行时都必须有版本。  
新版本不能覆盖旧版本；项目引用固定版本。

### 5. 能力包可替换，创作台不认题材

创作台不知道“铁路、排行榜、地图”这些词。  
它只认：

- 项目；
- 计划；
- Recipe；
- 质量报告；
- Candidate。

题材知识属于 Shot、Technique、Contract 和 Capability Pack。

### 6. 多模态从 Segment 开始，不从单镜头开始

未来一条片子可能包含：

```text
DataShot + MediaClip + TalkingHead + Caption + Voice + Music
```

因此需要一个上层 RenderPlan，把不同类型的 Segment 组合起来；单个 Shot 仍然由 Renderer Adapter 执行。

## 三、逻辑架构

```text
┌────────────────────────────────────────────────────────────┐
│                        AI 客户端对话                         │
│  Codex / DSH / 其他支持 Skills 与 Capability 的客户端          │
└─────────────────────────────┬──────────────────────────────┘
                              │
                    Skill / Workflow 编排
          video-director / video-creator / video-series
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                     Capability Gateway                      │
├────────────────────────────────────────────────────────────┤
│  生成层                                                     │
│  plan.normalize / plan.estimate / recipe.compile             │
│  generate.run / series.plan / series.generate                │
├────────────────────────────────────────────────────────────┤
│  创作台（Workbench）                                        │
│  project / revision / feedback / quality / candidate / UI    │
├────────────────────────────────────────────────────────────┤
│  能力包                                                     │
│  data-shot / geo / media / voice / subtitle / publish        │
├────────────────────────────────────────────────────────────┤
│  渲染适配器                                                 │
│  remotion（当前） / hyperframes / ffmpeg / 其他               │
└─────────────────────────────┬──────────────────────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────┐
│                    workspace/.himind-media                  │
│  projects / plans / recipes / candidates / jobs / artifacts  │
└────────────────────────────────────────────────────────────┘
```

## 四、领域模型

### 4.1 CreativeBrief

用户意图的结构化表达。

```json
{
  "version": "himind.video.brief.v1",
  "goal": "解释 2025 年安徽文旅消费构成",
  "audience": "短视频平台泛知识用户",
  "platform": "douyin",
  "duration_seconds": 10,
  "tone": "克制、数据感",
  "must_include": ["总消费", "五类构成"],
  "must_avoid": ["未经证实的结论"],
  "data_refs": ["workspace:data/tourism-2025.json"],
  "style_id": "tech-blue"
}
```

### 4.2 NarrativePlan

回答“讲什么、为什么这样讲”。

```json
{
  "version": "himind.video.plan.v1",
  "thesis": "安徽文旅消费以交通和住宿为主",
  "structure": "hook -> breakdown -> insight",
  "key_points": [
    {"claim": "交通是最大构成项", "evidence_ref": "data.categories[0]"},
    {"claim": "前两类合计超过一半", "evidence_ref": "derived:top2_share"}
  ],
  "required_data": ["categories", "unit", "total"],
  "constraints": ["百分比由系统推导", "来源必须标注"],
  "risk_notes": []
}
```

### 4.3 StoryboardPlan

回答“用什么镜头、多长、怎么转场”。

```json
{
  "version": "himind.video.storyboard.v1",
  "shots": [
    {
      "shot": "composition",
      "role": "breakdown",
      "duration_seconds": 7,
      "data_binding": "data",
      "style_overrides": {},
      "transition": "fade",
      "rationale": "占比结构最适合回答整体构成"
    },
    {
      "shot": "big-number",
      "role": "insight",
      "duration_seconds": 3,
      "data_binding": "derived.top2_share",
      "style_overrides": {},
      "transition": "cut",
      "rationale": "用单一数字收束结论"
    }
  ]
}
```

### 4.4 Recipe

渲染器唯一消费的执行契约。  
保持现有 `schemas/recipe.json` 和 `catalog/contracts/*.schema.json` 不变，Plan 编译后仍然产出 Recipe。

### 4.5 RenderPlan（多模态阶段引入）

```json
{
  "version": "himind.video.render-plan.v1",
  "segments": [
    {"kind": "data-shot", "recipe_ref": "recipes/shot-1.json"},
    {"kind": "media-clip", "asset_ref": "assets/broll-1.mp4", "duration_seconds": 3},
    {"kind": "voice", "asset_ref": "assets/voice-1.mp3"},
    {"kind": "captions", "asset_ref": "assets/captions-1.json"}
  ],
  "output": {"width": 2160, "height": 3840, "fps": 30}
}
```

单镜头数据视频的 RenderPlan 可以只有一个 `data-shot` Segment。

### 4.6 QualityReport

```json
{
  "version": "himind.video.quality.v1",
  "passed": true,
  "checks": [
    {"id": "data_contract", "level": "error", "passed": true, "evidence": "..."},
    {"id": "golden_still", "level": "error", "passed": true, "evidence": "ssim=1.0000"},
    {"id": "content_progress", "level": "error", "passed": true, "evidence": "+3.00pp"}
  ]
}
```

### 4.7 Candidate

```json
{
  "version": "himind.video.candidate.v1",
  "candidate_id": "cand-xxx",
  "source_project_id": "project-xxx",
  "base_capability": "composition",
  "plan_ref": "plans/plan.json",
  "recipe_ref": "recipes/recipe.json",
  "quality_ref": "quality/report.json",
  "tags": ["composition", "brand-tech-blue", "data-story"],
  "promotion_state": "validated"
}
```

### 4.8 CapabilityDescriptor

能力目录中的每一项都必须能回答：

- 适合什么叙事问题；
- 输入契约是什么；
- 输出是什么；
- 成本/耗时等级；
- 质量等级；
- 示例项目；
- 版本。

```json
{
  "id": "composition",
  "family": "composition",
  "structures": ["part-to-whole"],
  "input_contract": "catalog/contracts/composition.schema.json",
  "cost_class": "low",
  "latency_class": "fast",
  "quality_tier": "ready",
  "examples": ["catalog/fixtures/composition/recipe.json"]
}
```

## 五、能力边界

### 5.1 当前阶段（P0）

所有能力仍在 `com.himind.media-workbench` 内，但代码按模块分目录：

```text
plugins/com.himind.media-workbench/media-workbench/
├── generation/        # plan / compile / series
├── workbench/         # project / feedback / candidate / quality
├── renderer/          # remotion + adapter
├── network/           # HTTP / 外部服务
└── internal/
```

先通过架构测试保证模块边界，再物理拆分。

### 5.2 目标边界

| 能力域 | 归属 | 说明 |
| --- | --- | --- |
| `media.workbench.project.*` | 创作台 | 项目、修订、反馈、资产索引 |
| `media.workbench.plan.*` | 创作台（应用层） | 计划规范化、估算、编译 |
| `media.workbench.generate.*` | 创作台（应用层） | 一键生成、系列生成、编排 |
| `media.render.*` | 渲染适配器 | 渲染任务、运行时、产物 |
| `media.artifact.*` | 渲染适配器 | 产物读取/导出 |
| `media.geo.*` | 地理管线 | 轨迹修复/校验 |
| `media.source.*` | 素材能力包 | 库存素材、AI 素材、本地素材 |
| `media.voice.*` | 声音能力包 | TTS、BGM |
| `media.subtitle.*` | 字幕能力包 | 时间轴、样式 |
| `media.publish.*` | 发布能力包 | 平台发布 |
| `media.workbench.candidate.*` | 创作台（应用层） | Candidate 生命周期 |

## 六、生成 API

### 6.1 Plan 阶段

| 能力 | 风险 | 作用 |
| --- | --- | --- |
| `media.workbench.plan.normalize` | read_only | 把 AI 产出的计划规范化并校验 schema |
| `media.workbench.plan.from_recipe` | read_only | 把已有 Recipe 反推成 Plan，用于兼容旧项目 |
| `media.workbench.plan.estimate` | read_only | 估算时长、成本、数据缺口和风险 |
| `media.workbench.plan.validate` | read_only | 检查叙事结构、数据绑定和镜头可用性 |

### 6.2 Compile 阶段

| 能力 | 风险 | 作用 |
| --- | --- | --- |
| `media.workbench.recipe.compile` | read_only | Plan + Data + Style → Recipe |
| `media.workbench.recipe.validate` | read_only | 现有 Recipe 校验，继续保留 |

### 6.3 Generate 阶段

| 能力 | 风险 | 作用 |
| --- | --- | --- |
| `media.workbench.generate.run` | local_write | 一键生成：plan → recipe → preview/render job |
| `media.workbench.series.plan` | read_only | 把一个结构映射到多条数据 |
| `media.workbench.series.generate` | local_write | 批量生成并汇总质量 |
| `media.workbench.candidate.promote` | local_write | 好结果升级为可复用能力 |

### 6.4 关键约束

- `generate.run` 不能绕过 `plan.normalize` 和 `recipe.validate`；
- 任何写入 workspace 的能力都必须遵守 `workspace_root` 边界；
- 渲染、发布、能力提升都需要人类审批；
- AI 产出的 Plan 只能作为候选，不能直接成为系统级能力。

## 七、状态机

```text
draft
  → planned
  → compiled
  → validated
  → previewed
  → approved
  → rendering
  → rendered
  → reviewed
  → promoted
```

失败状态：

```text
blocked_plan
blocked_recipe
blocked_quality
render_failed
promotion_rejected
```

状态转换必须记录：

- 谁触发；
- 使用的 Plan/Recipe/资产版本；
- 质量报告；
- 用户反馈。

## 八、存储布局

```text
<workspace_root>/.himind-media/
├── projects/<project_id>/
│   ├── project.json
│   ├── brief.md
│   ├── plan.json
│   ├── storyboard.json
│   ├── recipe.json
│   ├── feedback.jsonl
│   └── quality.json
├── candidates/<candidate_id>/
│   ├── candidate.json
│   ├── plan.json
│   ├── recipe.json
│   ├── quality.json
│   └── stills/frame-*.png
├── capabilities/
│   └── manifest.json
├── assets/
│   ├── plugin/
│   └── workspace/
├── jobs/
└── artifacts/
```

约束：

- 所有文件引用不能越出 `workspace_root`；
- `plugin:` 资产只读；
- Candidate 不直接修改正式能力目录；
- 能力提升必须经过 `candidate.promote`。

## 九、质量架构

### 9.1 阶段门禁

| 阶段 | 检查 |
| --- | --- |
| Plan | 叙事结构、数据缺口、时长、风险 |
| Recipe | schema、契约、安全区、画布、引用 |
| Preview | 空画面、内容推进、文字可读性 |
| Render | 固定运行时、产物哈希、确定性 |
| Candidate | 静帧基线、成片验收、回归 |

### 9.2 质量报告

所有门禁输出统一 `QualityReport`：

- `id`；
- `level`：info/warn/error/gate；
- `passed`；
- `evidence`；
- `remediation`；
- `retryable`。

### 9.3 系列质量

批量生成不要求每条都人工看，但必须：

- 首条、末条和随机中间条做人工抽检；
- 每条跑数据契约；
- 抽样跑静帧/成片验收；
- 失败条目单独重试，不污染成功条目。

## 十、UI 架构

### 10.1 创作台视图

```text
项目树
  └─ 项目
      ├─ Brief
      ├─ Plan
      ├─ Storyboard
      ├─ Recipe
      ├─ Preview
      ├─ Quality
      └─ Artifacts
```

### 10.2 生成入口

- 页头「交给 AI」：把当前项目交给 AI 客户端对话；
- 页头「生成预览」：使用当前 Plan/Recipe 生成；
- 页头「构建成片」：进入审批和渲染；
- 系列页：「批量生成」。

### 10.3 Plan 审阅

Plan 审阅页至少展示：

- 叙事结构；
- 镜头列表与时长；
- 数据绑定；
- 风险提示；
- 与上一版的 Diff。

### 10.4 Storyboard 编辑器

第一阶段只做只读展示和少量参数编辑，暂不做完整拖拽时间线。  
完整编辑器放在 P2。

## 十一、测试策略

1. **模型测试**：Plan/Storyboard/RenderPlan 的 schema 和规范化；
2. **编译测试**：Plan → Recipe 的黄金结果；
3. **契约测试**：所有 ready 模板的默认配方和 fixture；
4. **静帧回归**：现有 19 个基线继续扩展；
5. **成片验收**：内容推进、非空、确定性；
6. **端到端测试**：fake AI planner + fake renderer，覆盖 plan → preview → render；
7. **架构适应度函数**：生成层不能依赖 UI，创作台不能依赖具体题材；
8. **系列测试**：同一 Plan 换 5 份数据，逐条跑契约和抽样验收。

## 十二、安全与审批

- `workspace_root` 是写边界；
- `plugin:` 资产只读；
- 外部素材下载、TTS、发布都需要明确审批；
- 不执行任意 Shell；
- 不把 API Key 写入项目文件；
- 发布前必须经过用户确认；
- Candidate 提升为能力需要审批。

## 十三、与当前 0.5.0 的兼容

1. 保持现有 `project.*`、`recipe.validate`、`preview.*`、`render.*` 能力不变；
2. 新增 Plan 是可选层：
   - 新项目：Brief → Plan → Recipe；
   - 旧项目：`plan.from_recipe` 生成兼容 Plan；
3. `Project` 增加 `plan_ref` 和 `storyboard_ref`，但不强制旧项目迁移；
4. Candidate 继续使用现有 `candidate.save`，新增 `candidate.promote`；
5. Renderer/Geo 的物理拆分放到 P3，不在生成主链路落地前进行。

## 十四、关键决策记录

| 决策 | 结论 | 原因 |
| --- | --- | --- |
| AI 是否直接渲染 | 否 | 不可审计、不可复现 |
| Plan 是否替代 Recipe | 否 | Recipe 是渲染契约，Plan 是规划契约 |
| 是否先做 Storyboard 编辑器 | 否 | 先生成主链路，编辑器服务生成 |
| 是否先拆插件 | 否 | 先验证生成价值，再物理拆分 |
| 是否全量支持文生视频 | 否 | 作为 media 能力包接入，不做主画面 |
| 是否要求逐字节确定 | 数据视频要求；外部素材/模型视频按可追溯降级 | 外部模型不可控 |










