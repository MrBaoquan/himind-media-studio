# 自媒体创作台：创作流畅性与 Agent 架构审查（2026-10-20）

> 视角一：高级用户，从真实 UI 走完整创作流。
> 视角二：AI Agent，从 131 个插件能力、5 个 Skill、3 个 Workflow 与 Agent Preset 审查自动化调用链。
> 方法：18082 真实页面 + CDP 交互 + 能力清单/工作流 JSON 静态审查。
> 版本：`com.himind.media-workbench@0.74.1`

## 一、结论

创作台已经具备“可交付的生产骨架”：项目、CreativePackage、Plan、Storyboard、Timeline、预览、QA、确定性检查和渲染入口都在。

但**端到端创作流畅性仍低于单点功能成熟度**：

- 剪辑工作台约 84 分；
- 端到端创作流约 69 分；
- AI 自动化架构约 70 分；
- 综合约 **69/100**。

核心问题不是“缺功能”，而是**入口太多、路径不唯一、跨页面连续性不足、错误不可恢复**。

## 二、高级用户视角

### 2.1 评分

| 维度 | 分数 | 判断 |
| --- | --- | --- |
| 信息架构 | 80 | 项目/审阅/计划/分镜/剪辑/创作脉络层级清楚 |
| 创作启动 | 72 | 新建项目、风格库、创作脉络都在，但主路径不唯一 |
| 审阅与反馈 | 76 | 审查图、QA、反馈框完整；反馈与下一步动作割裂 |
| 计划与分镜 | 64 | 信息完整，但 100 镜头缺搜索/虚拟化/逐镜头 AI 操作 |
| 剪辑工作台 | 86 | 轨道操作、Inspector、播放头、转场、音量/电平已成熟 |
| 渲染与交付 | 58 | 多入口、无预检对话框、原始 blocker 直接暴露 |
| 错误恢复 | 50 | `MEDIA_PATH_MISSING: m1` 无重链路径；插件宿主 502 时白屏 |
| 100+ 规模 | 60 | 列表可滚动，但缺搜索、缩略图、语义缩放 |

### 2.2 做得好的地方

1. **CreativePackage 脉络清晰**：Brief → 证据链 → 叙事脚本 → 分镜 → 制作规格 → 校验/批准 → 项目/成片。
2. **AI 对话与创作台分工明确**：创作台不维护第二套聊天记录，只生成带脉络上下文的任务。
3. **审阅页信息完整**：审查图、产物列表、项目元数据、确定性状态、QA 报告同屏。
4. **剪辑工作台成熟**：5 轨、轨道 M/S/L、排序/重命名/颜色、音量/电平、转场手柄、播放头联动。
5. **质量门禁透明**：67/67、确定性 stale/ready、视觉 QA 都可追溯。

### 2.3 关键摩擦点

#### F1 预览/渲染入口过多，命名不统一

同屏出现：

- 生成审查图
- 生成动效预览
- 检查确定性
- 检查确定性并渲染
- 渲染
- 成片渲染
- 重新渲染

用户无法一眼判断“现在该点哪个才能得到最终 MP4”。

**建议**：收敛为三个动作：
- `预览`（快速看画面）
- `检查并出片`（确定性 + 质量门禁 + 渲染）
- `交付`（平台预设/QC）

#### F2 计划/分镜是“只读展示”，缺少逐镜头动作

- 100 镜头计划页只有列表，没有搜索、跳转、筛选、缩略图。
- 分镜编辑器有表单，但没有“让 AI 改这个镜头”“预览这个镜头”“复制为变体”等就地动作。
- 用户要改第 87 个镜头，只能滚动或回 AI 对话重新描述。

**建议**：
- 列表加搜索/序号跳转/按类型筛选；
- 每个镜头增加“AI 修改此镜头”“预览此镜头”；
- 100+ 镜头做虚拟化与缩略图；
- 右侧质量门禁改为摘要 + 可折叠详情。

#### F3 100 镜头剪辑页的空间与语义缩放

- 50% 缩放下 100 个 clip 成为一条灰带，标签不可读。
- 时间线下方大面积黑色空白，没有“添加轨道/拖入素材”的空态引导。
- 媒体池显示“0 个产物”，但时间线已有 100 个 clip，认知冲突。

**建议**：
- 语义缩放：低倍率隐藏文字、保留色块/缩略图；高倍率显示标签。
- 空轨道区域提供拖入提示。
- 媒体池区分“项目产物”和“时间线引用媒体”。

#### F4 渲染失败没有恢复路径

真实点击渲染得到：

```text
MEDIA_PATH_MISSING: m1
```

只有右上角红色 toast，没有：
- 哪个片段缺媒体；
- 一键重新链接；
- 跳过该片段；
- 查看预检详情。

**建议**：渲染前预检对话框，列出缺失媒体、分辨率、时长、预计耗时，并提供“重新链接/替换/移除”动作。

#### F5 插件 UI 宿主 502 时白屏

真实观察到 `plugin-ui.localhost` 返回 502，窗口只剩空白 `#root`，没有错误态或重试按钮。重启 Agent 并重新打开创作台后恢复。

**建议**：插件窗口加载失败时显示可读错误页 + 重试 + 打开日志；不要让用户面对白屏。

#### F6 保存与未保存状态

- 分镜/时间线都是显式保存；切换项目或 tab 时没有未保存提示。
- 本轮还发现并修复了 `savedRevision` 未更新导致连续保存 `REVISION_CONFLICT` 的问题。

**建议**：脏状态标记 + 离开确认 + 自动保存草稿（可配置）。

#### F7 创作脉络的三栏密度

- 左创意列表、中 Brief 表单、右逻辑/能力/风险三栏都在滚动。
- “交给 AI”在左下角，视觉权重低。
- “校验 / 批准 / 创建”三个动作并列，主次不清。

**建议**：
- 把“下一步”做成 stepper 上的单一 CTA；
- “交给 AI”提升为固定底部操作条；
- 右栏默认只显示摘要，详情折叠。

## 三、AI Agent 视角

### 3.1 评分

| 维度 | 分数 | 判断 |
| --- | --- | --- |
| 能力契约质量 | 82 | schema 严格、Plan/Recipe/Timeline 契约清晰 |
| 能力可发现性 | 62 | 131 个插件能力 + 285 个总能力，选择负担高 |
| 编排合理性 | 58 | Workflow、Skill、CreativePackage 三条路径并存 |
| 长任务/Job | 78 | preview/render status/cancel 存在，workflow 标记 long_running |
| 审批与安全 | 76 | generate/render 审批、风险等级、宿主审批栈 |
| 可观测性 | 68 | 有 job/artifact/QA，但缺统一 trace 与错误恢复建议 |

### 3.2 现有拓扑

```text
4 个插件：media-workbench / media-visual / audio-intelligence / audio-production
5 个 Skill：video-director / video-creator / style-curator / capability-author / code-project
3 个 Workflow：generate-video / media-qa / author-remotion-capability
1 个 Agent Preset：himind-media-workbench
1 个 Feature Pack：com.himind.feature.media-workbench
```

优点：
- 插件按职责拆分，Skill 保存判断，Workflow 负责多步骤编排；
- CreativePackage、Plan、Recipe、Timeline 都有明确数据契约；
- generate/render 在 workflow 中标记 `long_running` + `approval_required`；
- preview/render 都有 status/cancel 能力。

### 3.3 核心架构问题

#### A1 两条创作路径没有统一决策树

路径 A（Workflow）：

```text
request → 单/多镜头 Plan → generate.run → quality.check → render.start
```

路径 B（CreativePackage）：

```text
creative.create → research/script/storyboard/production → approve → materialize → project → timeline/audio → QA → render
```

- `video-director` 要求科普/新闻/数据解释优先走 B；
- `generate-video` workflow 走 A，且不包含 `creative.*`；
- Agent Preset 的 persona 只描述 A 的 Recipe/项目/预览/渲染；
- 三者没有“什么时候走哪条”的强制路由。

**结果**：同一个“做一期科普短片”的请求，不同模型可能走出完全不同的质量和审批路径。

**建议**：增加一个 `media.workbench.intent.plan`（或 Skill）作为入口路由，返回：
- 推荐路径（fast/story）；
- 缺失输入；
- 需要的能力序列；
- 是否需要 CreativePackage 审批。

#### A2 Agent Preset 的 persona 落后于产品

当前 persona 只提到：

```text
catalog → project → Recipe → preview → feedback → quality → render
```

没有提到：
- CreativePackage；
- storyboard/分镜；
- audio.build / TTS / BGM；
- timeline 精修；
- `generate-video` workflow；
- `project.context` 继续创作；
- preview/render 都需轮询。

**建议**：把 persona 升级为“路径选择 + 强制顺序 + 长任务轮询 + 恢复策略”的单一事实源。

#### A3 Feature Pack 不包含 Workflow

Feature Pack 只能组合 plugin/skill/agent preset，不能包含 workflow。
安装“自媒体创作台”功能包的用户，可能没有 `generate-video` 编排能力，只剩 persona + skills。

**建议**：
- 在 Feature Pack 安装说明中显式要求安装 workflow；
- 或把编排能力收敛进 Agent Preset 的 instructions（persona 里给出路径）；
- 或提供 `media.workbench.intent.plan` 作为不依赖 workflow 的编排入口。

#### A4 Workflow 覆盖不全

`generate-video` 有 9 步、8 个能力，覆盖：
- catalog / data / plan / normalize / estimate / bind / generate / quality / render。

没有覆盖：
- CreativePackage 研究/脚本/分镜/制作规格；
- audio.build / speech / bgm / mix / qa；
- timeline 精修；
- preview.review / feedback；
- media-qa 视觉 QA；
- 人工审阅后再渲染。

README 还写着“当前只支持单镜头计划”，而 prompt 已支持 storyboard，文档与实现不一致。

#### A5 `allow_network: true` 与 README“禁止联网”冲突

`GV-PLAN` 的 runtime 配置是：

```json
"allow_network": true,
"tool_policy": "none"
```

README 写“禁止联网和工具探索”。需要明确这个 flag 的语义，否则安全审查无法判断。

#### A6 131 个能力对 Agent 的选择压力

- 35 个 timeline 能力；
- 10 个 project；
- 9 个 audio；
- 7 个 creative；
- 以及大量 preview/quality/asset 能力。

虽然宿主支持 capability search/activate，但默认全量暴露时，模型容易：
- 选错粒度（直接 timeline.apply 而不是 CreativePackage）；
- 跳过 Plan/审批；
- 重复造 composite。

**建议**：提供一个 6–10 个工具的“创作 Facade”：
- `media.brief`
- `media.plan`
- `media.project`
- `media.preview`
- `media.timeline`
- `media.audio`
- `media.qa`
- `media.render`
- `media.context`

底层 131 个能力保留给高级/调试场景。

#### A7 元数据不一致

抽查发现：
- 多数老能力 `timeout_seconds` 为空；
- `timeline.apply` 是 `read_only`（纯函数，合理但需在文档中说明）；
- 部分能力描述为“创建/写入”，但 timeout 未声明；
- workflow README 与 workflow JSON 不一致。

**建议**：统一 capability 元数据规范：risk、timeout、long_running、idempotency、approval。

## 四、优先级路线

### P0：先让主路径唯一、可恢复

1. **统一 Preview/Render 动作**：预览、检查并出片、交付三按钮。
2. **渲染预检 + 缺失媒体重链**：替代 `MEDIA_PATH_MISSING: m1` 原始错误。
3. **Agent 入口路由**：`intent.plan` 决定 fast path vs story path。
4. **升级 Agent Preset persona**：写入 CreativePackage、storyboard、audio、timeline、job polling。
5. **插件 UI 502 错误页 + 重试**。

### P1：让 100 镜头可用

6. 计划/分镜列表搜索、跳转、筛选、虚拟化、缩略图。
7. 逐镜头“AI 修改此镜头 / 预览此镜头”。
8. 质量门禁摘要 + 折叠详情。
9. 时间线语义缩放 + 空轨道引导。
10. 脏状态提示 + 自动保存草稿。

### P2：让 Agent 编排完整

11. 新增 `generate-story-video` workflow（CreativePackage → storyboard → audio → timeline → QA → review → render）。
12. 把 `media-qa` workflow 接入最终渲染门禁。
13. 统一 capability 元数据（timeout/risk/idempotency）。
14. 提供创作 Facade（6–10 个高层工具）。
15. 统一 trace：request → creative → plan → project → render → artifact。

## 五、证据

- 审阅页：`flow-review-preview.png`
- 代码分镜：`flow-code-storyboard.png`
- 100 镜头计划：`flow-plan-100shots.png`
- 100 镜头分镜：`flow-storyboard-100shots.png`
- 100 镜头剪辑：`flow-edit-100shots.png`
- 创作脉络：`flow-creative-workspace.png`
- 渲染错误：`flow-render-dialog.png`
- 能力清单：`.himind/media-capabilities.txt`（131 条）

## 六、判断

**底座不需要换。** CreativePackage、Plan、Timeline、Asset Pack、Renderer Adapter 的组合是成立的。

真正要补的是：
1. 入口路由；
2. 动作收敛；
3. 规模化编辑；
4. 错误恢复；
5. Agent 编排与 UI 的同一条事实链。

把这五点补齐，创作台才会从“功能齐全的创作工具”变成“AI 可以稳定交付的创作系统”。
