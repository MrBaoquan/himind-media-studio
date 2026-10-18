# 短视频生成器实施计划

> 状态：执行计划  
> 日期：2026-10-17  
> 当前进展：M0 规划层、`video-director` Skill、`generate.run`、Plan 审阅 UI、系列生成初版、`data.ingest` / `data.bind`、`generate-video` Workflow 0.1.4 与能力作者链路（`video-capability-author` + `author-remotion-capability` 0.1.1 + `capability.list/describe/audit`）已落地；安徽高铁与 SankeyFlow 已真实出 MP4；Browser-Code 代码项目模式已完成 import/sync/context/review/quality/render 闭环。下一阶段是 Style Pack 生态与真机批量运行。
> 上游方向：[generator-direction.md](generator-direction.md)  
> 架构方案：[generator-implementation-architecture.md](generator-implementation-architecture.md)

## 一、执行原则

1. **先打通主链路，再扩能力**  
   不先做完整 Storyboard 编辑器，不先拆插件，不先接十个 TTS Provider。

2. **每个里程碑交付一个可验收的生成闭环**  
   不是“写完一个模块”，而是“用户能走完一条路径，并且有测试证明没坏”。

3. **保持现有能力兼容**  
   `project.*`、`recipe.validate`、`preview.*`、`render.*` 不破坏；新能力优先新增，不重写。

4. **规划与执行分离**  
   AI 只产出 Plan；插件负责规范化、编译、校验和渲染。

5. **每个能力都要有边界、契约和证据**  
   没有 schema、测试、黄金样例和文档的能力，不算完成。

## 二、总体节奏

按 1 名全职工程师 + AI 辅助估算：

| 阶段 | 内容 | 估算 |
| --- | --- | --- |
| M0 | 规划与契约层 | 1–1.5 周 |
| M1 | 一键生成主链路 | 2–3 周 |
| M2 | 系列生成 | 1.5–2 周 |
| M3 | Candidate 提升 | 1–1.5 周 |
| M4 | last-mile 能力包 | 3–4 周 |
| M5 | 创作体验 | 2–3 周 |
| M6 | 插件拆分与发布 | 2–3 周 |

建议先完成 **M0–M3（约 6–8 周）**，形成一个真正可用的“高质量生成器 MVP”，再决定是否投入 M4–M6。

```text
M0 ──► M1 ──► M2 ──► M3
        │             │
        ├──► M4       └──► M6
        └──► M5
```

## 三、M0：规划与契约层

### 落地状态（2026-10-17）

- [x] `schemas/brief.json`、`schemas/plan.json`、`schemas/storyboard.json`
- [x] `plan.go`：normalize / validate / estimate / from_recipe
- [x] `media.workbench.plan.normalize`、`plan.validate`、`plan.estimate`、`plan.from_recipe`
- [x] `media.workbench.recipe.compile`（单镜头 + Storyboard）
- [x] 3 个 Plan 示例：composition / flow / route-growth
- [x] 6 条 Plan 执行约束
- [x] `video-director` Skill 初版
- [x] Plan → Recipe 静帧一致性：compiled composition 与 fixture SSIM **1.000000**
- [ ] Plan 审阅 UI（M1）
- [x] `media.workbench.generate.run`（Plan → Recipe → 项目落盘 → 可选预览）

### 目标

让系统先具备“计划”这个一等公民，但不改变现有渲染链路。

### 交付物

1. 新增 schema：
   - `schemas/brief.json`
   - `schemas/plan.json`
   - `schemas/storyboard.json`
   - `schemas/render-plan.json`（草案即可）
2. 新增 Go 模块：
   - `generation/plan.go`
   - `generation/plan_test.go`
3. 新增能力：
   - `media.workbench.plan.normalize`
   - `media.workbench.plan.from_recipe`
   - `media.workbench.plan.validate`
   - `media.workbench.plan.estimate`
4. 兼容旧项目：
   - `plan.from_recipe` 能把现有 Recipe 反推成 Plan；
   - 没有 `plan.json` 的项目仍然可以预览和渲染。
5. 新增 fixture：
   - `composition`、`flow`、`route-growth` 三个 Plan 示例。

### 验收标准

- 现有 11 个 ready 模板都能从默认 Recipe 生成合法 Plan；
- Plan 缺失镜头、缺失数据、时长越界时，`plan.validate` 返回明确 blocker；
- Plan → Recipe → Plan 的 round-trip 在核心字段上稳定；
- `go test ./...` 全绿；
- 不改 UI 也能通过能力调用完成上述流程。

### 不做

- 不做 AI 自动规划（M1）；
- 不做系列生成（M2）；
- 不做插件拆分。

## 四、M1：一键生成主链路

### 落地状态（2026-10-17）

- [x] `video-director` Skill 初版：先把需求转成 Plan，再调用 normalize/validate/estimate/compile
- [x] `media.workbench.generate.run`：Plan → Recipe → 项目落盘（plan/storyboard/recipe）→ 可选预览
- [x] 真实 UI 闭环：生成项目可在创作台打开，页头预览显示 composition 画面
- [x] `workflows/generate-video` 0.1.4：CSV/JSON 数据入口 + `data.bind` 契约映射 + catalog 驱动的去高铁化模板选择；Workflow build/package 通过，Candidate 已保存
- [x] Workflow 上下文桥：`plan_step` / `project_step` / `data_step` / `workflow_context` 已通过真实能力链测试\n- [x] `data.ingest` + `data.bind`：items 可映射到 leaderboard / composition / flow / timeline / choropleth\n- [x] 安徽高铁真实成片：18.05s / 540 帧 / 3.00 MB，内容级验收 10/10
- [ ] 在安装了开发插件的 Agent home 中完成 Workflow candidate.test 与真实 run
- [ ] Plan 审阅 UI
- [ ] 三个不同 brief 的完整评估集
- [ ] Workflow 封装与一键确认体验

### 目标

用户给一个 brief 或数据，系统能产出一条可预览的视频。

### 交付物

1. 新 Skill：`skills/video-director`
   - 负责把用户需求变成 `NarrativePlan + StoryboardPlan`；
   - 必须调用 `catalog.list` 和 `template.describe`；
   - 必须输出结构化 JSON，而不是自由文本。
2. 新能力：
   - `media.workbench.recipe.compile`
   - `media.workbench.generate.run`
3. 新 Workflow：
   - `workflows/generate-video`
   - brief → plan → 用户确认 → compile → validate → preview → 审批 → render
4. Plan 审阅的最小 UI：
   - 展示叙事结构；
   - 展示镜头列表、时长、数据绑定；
   - 允许“交给 AI 修改”和“生成预览”。
5. 试点模板：
   - `composition`
   - `flow`
   - `route-growth`

### 验收标准

- 三个不同 brief 能分别选出正确结构，并生成可预览视频；
- 用户能在渲染前看到 Plan，而不是直接看到一条不可解释的视频；
- `generate.run` 不能绕过 `plan.normalize` 和 `recipe.validate`；
- 同一 Plan 两次预览的画面一致；
- 选择错误模板时有明确 blocker，而不是静默降级。

### 关键风险

- AI 规划不稳定：需要建立 20 条 brief 的评估集；
- Plan 字段过多：先只保留叙事、镜头、时长、数据绑定四类核心字段；
- UI 过早复杂化：Plan 页先只读，不做拖拽。

## 五、M2：系列生成

### 落地状态（2026-10-17）

- [x] `media.workbench.series.plan`：校验 Plan 与数据项，返回批量预览
- [x] `media.workbench.series.generate`：按同一 Plan 批量创建项目
- [x] 单镜头系列（当前限制）
- [x] 项目名唯一性、中文名稳定 ID 哈希
- [ ] 批量渲染
- [ ] 系列级抽样验收
- [ ] 多镜头系列

### 目标

同一结构换数据，批量生成多条视频。

### 交付物

1. 新能力：
   - `media.workbench.series.plan`
   - `media.workbench.series.generate`
2. 数据绑定规范：
   - `items_path`
   - `bindings`
   - `naming`
   - `batch_size`
3. 批量质量：
   - 每条跑数据契约；
   - 首条、末条、随机中间条跑静帧/成片验收；
   - 失败条目单独记录，不影响成功条目。
4. 系列项目结构：
   - 一个 Series 下挂多个 Project/Artifact；
   - 记录共同的 Plan、Style、Template 版本。

### 验收标准

- 同一 Plan 换 5 份数据，稳定生成 5 条视频；
- 每条都有独立的 Recipe 和产物；
- 至少 3 条通过静帧或成片验收；
- 失败条目的错误可定位到具体数据字段；
- 批量过程可取消、可重试。

### 不做

- 不做跨题材自动选模板；
- 不做自动发布；
- 不做多语言字幕。

## 六、M3：Candidate 提升

### 目标

把“这条不错”变成“下一次可调用”。

### 交付物

1. 新能力：
   - `media.workbench.candidate.promote`
2. Candidate 内容：
   - Plan、Storyboard、Recipe；
   - 质量报告；
   - 关键帧基线；
   - 标签和适用场景；
   - 与基础能力的 Diff。
3. 自动门禁：
   - 契约通过；
   - 静帧基线通过；
   - 成片内容推进通过；
   - 版本和资产哈希可追溯。
4. 能力目录：
   - `capabilities/manifest.json`；
   - 支持按结构、标签、数据契约检索。
5. AI 检索：
   - 新对话可以找到已提升的能力；
   - 能引用它作为 Plan 的候选。

### 验收标准

- 一个成功项目能升级为 Candidate；
- Candidate 通过验收后进入能力目录；
- 下一次 AI 对话可以检索到它；
- 旧版本能力不被覆盖；
- 失败 Candidate 有明确拒绝原因。

## 七、M4：last-mile 能力包

### 目标

从“数据动效生成器”扩展为“混合短视频生成器”。

### 交付物

1. `voice.synthesize`：至少接一个 TTS Provider；
2. `subtitle.align`：时间戳字幕；
3. `media.search`：本地素材 + 一个库存素材源；
4. `music.pick`：BGM 选择与音量；
5. `RenderPlan`：DataShot + MediaClip + Voice + Caption 的合成；
6. 一个混合成片示例：
   - 数据镜头；
   - 口播；
   - 字幕；
   - BGM。

### 验收标准

- 一条视频同时包含数据镜头、配音、字幕；
- 字幕时间轴与音频对齐；
- 视频内容不空白、不越界、不重叠；
- 外部素材和 TTS 凭据不写入项目文件；
- 失败 Provider 有回退路径。

### 不做

- 不做数字人；
- 不做多平台发布；
- 不做文生视频主画面。

## 八、M5：创作体验

### 目标

让创作台真正可用，而不是只靠 API。

### 交付物

1. Plan 审阅与 Diff；
2. Storyboard 只读展示；
3. 镜头级参数编辑（先支持 title/duration/data binding）；
4. 质量报告面板；
5. Candidate 对比；
6. 系列状态视图。

### 验收标准

- 用户可从项目树进入 Plan、Storyboard、Quality；
- 修改镜头时长/数据绑定后能重新校验和预览；
- 错误态、空态、加载态完整；
- 真实界面 e2e 通过。

### 不做

- 不做完整多轨时间线；
- 不做复杂拖拽；
- 不做专业 NLE 功能。

## 九、M6：插件拆分与发布

### 目标

把已经验证的生成能力物理拆分，形成可分发的能力生态。

### 交付物

1. **P1：先拆 External Packs**：`com.himind.video-media`、`com.himind.video-voice`、`com.himind.video-subtitle`、`com.himind.video-publish`；
2. **P2：再拆基础设施**：`com.himind.video-render-remotion`（render/artifact/runtime）、`com.himind.geo-pipeline`（route.*）；
3. Plan / compile / series 仍留在自媒体创作台应用型 plugin 内部，作为 `generation/` 模块，不拆独立生成器 plugin；
4. Native Shots / Techniques / Styles / Contracts / Fixtures 继续留在创作台内部，不拆 asset-pack；
5. 版本迁移与向后兼容；
6. 0.6.0 发布制品与签名；
7. `.himind/catalog.json` 更新。

### 验收标准

- 旧项目仍能打开、预览、渲染；
- 新能力通过独立插件安装；
- 插件边界测试通过；
- 安装、升级、卸载可回滚；
- 发布制品有 SHA-256 与签名。

### 不做

- 不在同一个版本里同时大改 UI 和插件边界；
- 不删除旧能力，先提供兼容层。

## 十、任务优先级

### Must（MVP）

1. Plan schema 与 normalize；
2. `plan.from_recipe`；
3. `recipe.compile`；
4. `generate.run`；
5. `video-director` Skill；
6. 三个试点模板；
7. Plan 审阅最小 UI；
8. 系列生成；
9. Candidate 提升。

### Should

1. Plan 估算；
2. 质量报告面板；
3. 系列抽样验收；
4. 只读 Storyboard；
5. 失败重试与恢复。

### Later

1. TTS / 字幕 / 素材；
2. Storyboard 拖拽；
3. 发布；
4. 插件物理拆分；
5. 跨机器基线。

## 十一、第一个 Sprint 的具体 Backlog

目标：**让 `composition` 从 brief 走到 Plan，再编译回现有 Recipe，并保持预览一致。**

1. 新建 `schemas/brief.json`、`schemas/plan.json`、`schemas/storyboard.json`；
2. 新建 `generation/plan.go`，实现 Plan 结构和 normalize/validate；
3. 实现 `plan.from_recipe`，从现有 `composition` Recipe 反推 Plan；
4. 实现 `plan.validate`，检查镜头、时长、数据绑定；
5. 实现 `recipe.compile` 的 composition 分支；
6. 添加 round-trip 测试：
   - Recipe → Plan → Recipe；
   - 两次 compile 结果一致；
7. 添加 3 条 brief 示例和期望 Plan；
8. 更新 `plugin.json` 能力声明；
9. 更新 `docs/generator-implementation-architecture.md` 的接口表；
10. 运行 `go test`、静帧回归和一次真实 preview。

### Sprint 验收

```text
brief
  → plan.normalize
  → plan.validate
  → recipe.compile
  → recipe.validate
  → preview.player
```

要求：

- Plan 可读；
- Recipe 与手工版一致；
- 预览通过；
- 没有新增 UI 也完成闭环。

## 十二、Definition of Done

每个新能力必须满足：

- [ ] 有 `plugin.json` 声明和 input schema；
- [ ] 有版本化输出 schema；
- [ ] 有单元测试；
- [ ] 有至少一个真实样例；
- [ ] 有错误码与 remediation；
- [ ] 不破坏旧项目；
- [ ] 通过 `go test ./...`；
- [ ] 涉及页面的能力通过 e2e；
- [ ] 文档与 README 已更新。

## 十三、风险登记

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| AI 规划不稳定 | 选错结构、镜头不合理 | 20 条 brief 评估集；Plan 审阅；模板约束 |
| Schema 频繁变化 | 旧项目失效 | 版本化 schema；迁移函数；兼容层 |
| 渲染成本高 | 迭代慢 | 预览用低分辨率/抽帧；渲染前审批 |
| 模板同质化 | 内容看起来一样 | Shot 参数化；风格令牌；多节奏变体 |
| 能力目录膨胀 | AI 选择困难 | 标签、适用结构、成本/质量等级 |
| Last-mile Provider 差异 | 字幕/配音不稳定 | Adapter 接口；失败回退；人工确认 |
| 数据版权/来源 | 合规风险 | provenance 字段；来源必须可追溯 |
| UI 范围失控 | 迟迟不能上线 | M5 只做 Plan/质量/只读 Storyboard |

## 十四、成功指标

| 指标 | 目标方向 |
| --- | --- |
| 首次可用预览时间 | 结构化内容 < 5 分钟 |
| 同类内容平均修订次数 | 逐步降到 ≤ 2 |
| 数据错误率 | 接近 0 |
| 系列生成成功率 | ≥ 95% |
| 确定性通过率 | 数据镜头 100% |
| 能力复用率 | 第 3 个月起持续上升 |
| 用户接受率 | 不需要大改即可发布的比例上升 |

## 十五、开工前需要确认的决策

1. **Plan 生成位置**：确认由 AI 客户端 Skill 生成，插件只做规范化和编译；这能避免插件内嵌模型、保留可审计性。
2. **Plan 审阅入口**：第一版放在创作台只读展示，还是只在 AI 对话里确认。
3. **首个系列场景**：建议从 `composition` 或 `choropleth` 选一个，不要同时做多个。
4. **首个 last-mile Provider**：TTS 先选 Edge TTS 还是本地模型；这会影响安装体积和网络依赖。
5. **发布节奏**：M0–M3 先内部使用，还是一开始就准备 0.6.0 公开版本。

## 十六、建议立即执行

如果按性价比排序，下一步只做三件事：

1. **Plan schema + normalize/validate/from_recipe**；
2. **composition 的 recipe.compile 试点**；
3. **video-director Skill 的最小闭环**。

完成这三件事后，再决定是否进入系列生成和 Candidate 提升。













