---
name: video-creator
description: 通过对话创建模板化短视频项目。用户提出做排行榜、占比构成、流程链路、地图数据、图片叙事、数据可视化短视频，或要求预览、修改、构建视频时使用。
---

# 短视频创作助手

> 复杂需求优先交给 `video-director`：先产出 Plan，再回来执行项目保存、预览与构建。本 Skill 不跳过 Plan 门禁。

## 目标

将自然语言需求转化为可重复的短视频 Recipe。优先复用已有模板和风格，只让用户确认数据、标题、时长和少量定制字段。项目可位于任意已有目录，不要求目录属于扩展源码仓库，也不依赖 Dashboard。

## 先判断叙事结构，再选模板

不要按“画面像什么”选模板，先判断这句话要回答什么问题：

| 用户的问题 | 结构 | 模板 |
| --- | --- | --- |
| 谁排第一 / 谁在增长 | 排序、变化 | `leaderboard-tech`、`bar-race` |
| 这个数字有多大 | 单点冲击 | `big-number` |
| A 和 B 谁更强 | 二元对比 | `compare` |
| 整体由哪些部分组成 | 占比、构成 | `composition` |
| 先做什么、再做什么 | 流程、链路 | `flow` |
| 什么时候发生 | 时间顺序 | `timeline` |
| 事情发生在哪里 | 地理分布、路径 | `choropleth`、`route-growth` |
| 线路什么时候开通 / 城市地铁生日 | 时间顺序 + 网络生长 | `transit-opening-timeline` |
| 一句话值得被记住 | 观点、引语 | `quote` |
| 需要多个镜头连成片子 | 多镜头叙事 | `storyboard` |

硬规则：

- `composition` 的百分比是**推导值**，不要传 `percentage`；传 `categories[].label` 和 `value`。
- `flow` 的步骤顺序就是画面顺序，不要为了“好看”打乱；`timeline` 才按年份/时间定位。
- 一条片子需要多个结构时，用 `storyboard` 把 Shot 串起来，而不是把多个主题硬塞进一个模板。
- 先调用 `template.describe` 读取 `data_contract`，按契约填数据；不要猜字段。
## 标准闭环

1. 调用 `media.workbench.catalog.list`，了解模板族、风格包和渲染引擎；需要筛选时使用 `media.workbench.template.list` 和 `media.workbench.style.list`。
2. 结合需求选择模板和风格。先调用 `media.workbench.template.describe` 或 `media.workbench.style.describe` 获取插槽、兼容性和质量规则，不凭名称猜测字段。
3. 如果用户是在做同类视频或已有成熟项目，优先调用 `media.workbench.project.duplicate`。传入源项目的 `workspace_root`、`project_id` 和新的目标目录/名称；它会继承模板、风格、Recipe、数据结构和 brief，并允许只替换本次标题、数据、模板、风格或 brief。不要重新从零创建同类项目。
4. 没有可复用项目时调用 `media.workbench.project.create`，传入用户明确给出的 `workspace_root`、项目名称、`template_id`、`style_id` 和数据。`workspace_root` 可以是 Unity、Remotion、HyperFrames 或普通文件夹；中文项目名称会保留为展示名，并由插件生成安全的内部项目 ID。后续始终使用返回的 `project.id`，不要使用项目展示名。
5. 使用返回的 Recipe 生成内容。数据放在 `data` 或模板声明的插槽中；`adapter` 只允许 `remotion` 或 `hyperframes`，通常沿用模板声明，不要把 Three.js、地图或 FFmpeg 写成 adapter。
6. 调用 `media.workbench.recipe.validate`。发现 `blockers` 时按 `code` 修复，不能跳过模板兼容性、版本、画布、时长、数据和安全区检查。

6.1. **地图/线路类项目必须先过几何质量门禁。** 若模板的 `data.lines` 携带坐标与站点，
    先调用 `media.workbench.route.validate`。返回 `ok: false` 时按 finding 的 `code` 处理：

    - `SPIKE_VISIBLE` / `LOOP_DETECTED` / `SEGMENT_RATIO_*` → 调用 `media.workbench.route.repair` 修复后重新校验；
    - `STATION_OFF_ROUTE` → 核对站点坐标，或修复后重试；
    - `SPIKE_SHARP_TURN` / `MAX_ADJACENT_GAP` 是 warn，属"需人工复核"，**不要**当作可通过而忽略，
      要在交付说明里写清楚是哪一段、为什么保留。

    这一门禁不能跳过：几何有折返尖刺时，画面会出现一条飞出去又折回的线段，
    并且该段动画节奏会被拉长 —— 这是肉眼可见的缺陷，不是数据洁癖。

6.2. **线网地图 / 站名避让。** 目标是白底官方线网图时，优先 `transit-network` Style Pack，
    不要改用暗色 3D、霓虹或 bloom。站名不要交给前端每帧贪心：

    - 先调用 `media.workbench.label.place`，传入站点、线路折线、安全区和已有标签；
    - 把返回的 `placements` 写入项目的 `input/labels.json`，渲染器会优先使用；
    - 调用 `media.workbench.label.validate` 检查重叠、越界和不可见标签；
    - 换乘站允许带引线回退，但整段动画应保持固定布局，不做逐帧重排；
    - 密集线网出现 hidden 标签时，先缩短站名或扩大画布安全区，不要静默丢弃。

    示意图必须明确写“示意线网 / 非完整官方线网 / 不用于出行”，正式发布前替换为官方或 OSS 校验数据。

7. 用 `media.workbench.project.update` 保存修订后的 Recipe，并传入上次 `project.get` 返回的 `updated_at` 做并发保护；然后调用 `media.workbench.preview.start` 生成 HTML 故事板，向用户返回 `artifact.path` 供浏览器预览。
8. 根据用户反馈再次更新 Recipe、重新校验并预览。确认有效反馈后调用 `media.workbench.feedback.record`，分类使用 `style`、`layout`、`motion`、`data`、`narrative` 或 `quality`；需要分析历史反馈时调用 `media.workbench.feedback.list`。
9. 调用 `media.workbench.quality.check`。只有 `passed: true` 才进入构建。
10. 调用 `media.workbench.runtime.status` 检查渲染运行时，再调用 `media.workbench.render.start`。默认 `remotion` 由插件内置后台任务生成真实 MP4；返回后轮询 `media.workbench.render.status`，不要因调用已返回就判断构建完成。
11. `hyperframes` 与 Remotion 是同级可插拔引擎；Three.js、地图、图表等专业能力只能在两种引擎的模板内部组合。HyperFrames 适配器读取 `recipe_path` 后调用 `media.workbench.render.complete` 回写结果。任务完成后用 `media.workbench.artifact.export` 获取路径和 SHA-256，需要本机查看时调用 `media.workbench.artifact.open`。

## Storyboard 的两种编辑方式

- **对话编辑**：直接改 `data.storyboard`，每个镜头仍是 `{shot, duration_seconds, data}`；
- **画布编辑**：在「自媒体创作台」切到「分镜」页，调整镜头顺序、时长和字段。
  镜头清单来自 `storyboard_shot` 声明，字段表单来自 `media.workbench.template.schema`；
  没有声明 schema 的镜头会退化为原始 JSON 编辑，而不是猜测字段。

保存时后端会先校验外层 `storyboard` 契约，再逐个镜头校验该镜头自己的 `data_contract`。
错误路径会精确到 `data.storyboard[i].data...`，不要绕过这个门禁直接改项目文件。

## 持续复用

成功项目的 Recipe、反馈和产物元数据都保存在 `<workspace_root>/.himind-media/`。重复出现的反馈交给 `video-style-curator` 评估，形成新的模板变体、Style Pack 版本或质量规则；不要直接覆盖既有版本。

## 约束

- 不执行任意 Shell，不自行拼接 Remotion、HyperFrames 或外部 URL 命令；只调用插件提供的固定参数能力。
- 不把 Dashboard 服务当作前置条件；本 Skill 的本地能力在独立模式可用。
- 大文件只通过 Artifact 元数据传递，不把视频内容塞入 MCP 响应。
- `blockers` 必须向调用方保留 `code`、`stage`、`message`、`remediation` 和 `retryable`。
- 地图类模板的 `data.lines[].coords` 必须是**已经过 `route.repair` 的几何**。
  不要把未校验的原始图搜索结果直接写进 Recipe。
- `route.repair` 是纯函数且几何幂等：重复调用不会累积改变坐标，可以放心重试。

- 质量/色带反馈先检查最终帧格式：暗部渐变、3D、粒子、vignette 风格不得用 JPEG 中间帧；最终帧应为 PNG。
- 需要量化时优先调用 `media.workbench.quality.banding`；本地开发可运行 `dev/verify-banding.mjs --image=<still>`。暗色/渐变风格的最大平坦段超过画面宽度 8% 或亮度级数少于 12 时不得直接交付；白底平面风格应在 Style Pack 声明 `quality.banding_gate=false`，不能把大面积纯白误判为色带。
- 同类质量坑必须沉淀为 `catalog/constraints.json` 规则、Golden Fixture 和 Skill，而不是只修当前项目。
- 成片后调用 `media.workbench.quality.visual`；黑帧、严重色带或响度失败时不得交付。
