---
name: video-director
description: 把短视频需求转成可验证的 NarrativePlan 与 StoryboardPlan，再编译成 Recipe；用于“先出导演方案、再出片”的生成流程。
---

# 短视频导演

## 目标

把用户的一句话、brief 或数据，转成一份**可审查、可校验、可编译**的计划，而不是直接跳到渲染。

```text
对话需求
  → CreativeBrief
  → NarrativePlan
  → StoryboardPlan
  → plan.normalize
  → plan.validate
  → recipe.compile
  → recipe.validate
```

导演负责“讲什么、为什么这样讲、用哪些镜头”；实际项目保存、预览和构建交给 `video-creator`。

## 0. 创意开发阶段（推荐起点）

当用户从“我想做一个关于……的视频”开始，而不是直接给出数据时，先建立 CreativePackage：

```text
主题
→ creative.create
→ research.update
→ script.update
→ storyboard.update
→ production.update
→ creative.validate
→ 用户批准
→ creative.materialize
```

职责：

- Director 负责 Brief、Research、Script、Storyboard、ProductionSpec；
- Research 只在事实型内容中要求 evidence；
- Script 说明“说什么、按什么顺序说”；
- Storyboard 说明“怎么画、怎么动、持续多久”；
- 用户批准后，再交给 `video-creator` materialize 成项目；
- 不要在导演阶段直接创建可渲染项目。

科普、新闻、数据解释类内容一律优先走 CreativePackage，不要跳过研究、脚本和分镜直接生成 Recipe。

### Script 与旁白/字幕

- `script.beats[].narration` 是 TTS 与字幕的唯一文本源；
- `on_screen_text` 是画面标题/关键词，不要和旁白混用；
- 每个 beat 的 `duration_seconds` 是镜头窗口；旁白超出时 `audio.build` 会在 1.5x 内压缩并记录 `fit_warning`；
- 分镜阶段不要另写一份字幕文本，避免 TTS 与字幕漂移；
- 需要影视级旁白时，在 Brief 中声明 voice 风格，但不要假设某个声音可商用；许可证由 voice pack 决定。

## 必须遵守的顺序

1. 调用 `media.workbench.catalog.list`，查看当前 ready 模板与风格。
2. 调用 `media.workbench.template.describe` 和 `media.workbench.template.schema`，确认镜头字段与数据契约。
3. 判断叙事结构，不要先看“哪个模板好看”：
   - 排行/变化 → `leaderboard-tech`、`bar-race`
   - 单点数字 → `big-number`
   - A/B 对比 → `compare`
   - 整体构成 → `composition`
   - 流程/步骤 → `flow`
   - 时间顺序 → `timeline`
   - 地理分布/路径 → `choropleth`、`route-growth`
   - 多镜头 → `storyboard`
4. 生成 Plan JSON，至少包含：
   - `brief`：目标、受众、平台、时长、语气；
   - `narrative`：中心论点、结构、关键点、所需数据、约束；
   - `storyboard.shots`：镜头、角色、时长、数据绑定、选择理由；
   - `presentation`：标题、副标题、来源、风格。
5. 调用 `media.workbench.plan.normalize`。
6. 调用 `media.workbench.plan.validate`。
7. 调用 `media.workbench.plan.estimate`，把时长、镜头数、成本等级告诉用户。
8. 调用 `media.workbench.recipe.compile`，传入 Plan 和真实数据。
9. 调用 `media.workbench.recipe.validate`，确认数据契约、安全区和画布。
10. 用户确认计划后，调用 `media.workbench.generate.run`，传入 `workspace_root`、项目名称、Plan、真实数据和 `preview: true`；它负责落盘项目并生成预览。
11. 把 `project.id`、Plan 摘要、预计时长、数据缺口和预览产物展示给用户；用户确认预览后，再交给 `video-creator` 完成质量检查、审批和构建。

**任何一步出现 blocker，都不要继续渲染。**

## Plan JSON 示例

```json
{
  "version": "himind.video.plan.v1",
  "brief": {
    "goal": "解释 2025 年安徽文旅消费构成",
    "audience": "泛知识用户",
    "platform": "douyin",
    "duration_seconds": 10,
    "tone": "克制、数据感"
  },
  "narrative": {
    "thesis": "安徽文旅消费以交通和住宿为主",
    "structure": "hook -> breakdown -> insight",
    "key_points": [
      {"claim": "交通是最大构成项", "evidence_ref": "data.categories[0]"}
    ],
    "required_data": ["categories", "unit"],
    "constraints": ["百分比由系统推导", "来源必须标注"]
  },
  "storyboard": {
    "shots": [
      {
        "shot": "composition",
        "role": "breakdown",
        "duration_seconds": 10,
        "data_binding": "data",
        "transition": "fade",
        "rationale": "占比结构最适合回答整体构成"
      }
    ]
  },
  "presentation": {
    "title": "2025 年安徽文旅消费构成",
    "subtitle": "按消费类型拆分",
    "source": "数据来源：示例数据",
    "style_id": "tech-blue"
  }
}
```

## 硬规则

- 只能使用 `status: ready` 的模板；placeholder/planned 不能进入 Plan。
- 不要编造数据、来源和统计口径；数据缺失时明确写出缺口。
- `composition` 的百分比由渲染器推导，不要传 `percentage` 字段。
- `flow` 的步骤顺序就是画面顺序；不要把时间顺序塞进 flow，用 `timeline`。
- 多镜头 Plan 中，每个 shot 必须携带自己的 `data`，因为不同镜头的数据契约不同。
- 单镜头 Plan 可以直接使用顶层 `data`。
- 地图/线路镜头必须先通过 `media.workbench.route.validate`；失败时用 `media.workbench.route.repair` 修复后重试。
- 用户说“直接生成”时，也不能跳过 `plan.normalize`、`plan.validate` 和 `recipe.validate`；可以缩短计划摘要，但不能省略执行点。
- 不要让 AI 直接调用 `render.start`；渲染前必须有人确认。

## 数据入口

当用户提供 CSV / TSV / JSON 数据时：

1. 先调用 `media.workbench.data.ingest`；
2. 检查 `valid`、`count`、`ranked_items` 和 `issues`；
3. 数据有错误时先修正，不要继续写 Plan；
4. 选定模板后调用 `media.workbench.data.bind`，传 `template_id`、`data`（或 `data_step`）；
5. 检查映射后的 `data` 和模板契约校验结果，再传给 `generate.run`；
6. 保留 `unit`、`period` 和 `source`，不要丢失统计口径。

没有数据文件时，不要编造真实数字；可以生成示例数据，但必须明确标注“示例数据”。

## 系列模式

当用户要“同一结构换一批数据”时：

1. 先按正常流程生成并校验一份单镜头 Plan；
2. 调用 `media.workbench.series.plan`，传入 `name_prefix`、Plan 和 `items`；
3. 每个 item 必须有 `name` 和 `data`，可以覆盖 `title` / `subtitle`；
4. 预检通过后再调用 `media.workbench.series.generate`；
5. 默认 `preview: false`，批量项目创建后再按需预览；
6. 当前系列生成只支持单镜头 Plan；多镜头系列要拆成多个系列，或等待后续版本。

## 旧项目兼容

如果用户是在已有项目上继续创作，先调用 `media.workbench.project.get` 读取 Recipe，再调用 `media.workbench.plan.from_recipe` 反推 Plan。
这样可以先解释“当前视频的结构”，再做增量修改，而不是从零重做。

## 输出给用户的内容

每次生成计划后，至少告诉用户：

1. 选了什么叙事结构，为什么；
2. 用了哪些镜头，各多长；
3. 需要哪些数据，是否齐全；
4. 预计总时长和成本等级；
5. 有哪些风险或需要确认的事实。

## 失败处理

- `plan_shot_not_ready`：改用 ready 模板，或告知用户该能力尚未完成；
- `plan_style_incompatible`：换风格或换镜头；
- `plan_data_binding_missing`：补 data_binding；
- `data_contract_*`：按错误路径修正数据；
- `compile_shot_data_missing`：为每个镜头补 data；
- 其他 blocker：保留 `code`、`message`、`remediation`，不要吞掉。
