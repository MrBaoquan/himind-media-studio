# AI 生成 Remotion 能力的实施计划

> 状态：实施计划  
> 日期：2026-10-17  
> 上游方向：[generator-direction.md](generator-direction.md)  
> 生成主链路：[generator-implementation-plan.md](generator-implementation-plan.md)  
> 拆分政策：[plugin-split-policy.md](plugin-split-policy.md)  
> 当前进展：SankeyFlow 试点已落地（Shot + Technique + Contract + Template + Fixture + MP4）；`video-capability-author` Skill 与 `author-remotion-capability` Workflow 已实现；`capability.list/describe/audit` 已把能力六件套变成可查询、可机械审计的资产。

## 一、目标

让 AI 不仅能选择已有 Shot，还能在现有能力不足时：

1. 提出新能力；
2. 编写新的 Remotion Shot / Technique；
3. 写数据契约；
4. 注册 Template；
5. 生成黄金样例；
6. 预览、渲染、QC；
7. 沉淀为下一次可直接调用的能力。

目标是形成两条链路：

```text
模式 A：使用能力
需求 → 选择已有 Shot → Plan → Recipe → 预览/渲染

模式 B：创造能力
需求 → 能力缺口 → 提案 → 写 Shot/Technique → 注册 → 门禁 → Candidate → 能力库
```

模式 A 解决“现在能出片”；  
模式 B 解决“能力越用越多”。

## 二、非目标

1. 不在渲染时执行任意用户 JSX；
2. 不把新 Shot 拆成独立 plugin；
3. 不复制 anything2explainer 的固定黑底风格；
4. 不把每条视频都做成一次性代码工程；
5. 不在没有数据契约和黄金样例时把新 Shot 标为 `ready`。

## 三、能力的定义

一个正式能力不是“一个 JSX 文件”，而是下面六件东西：

```text
Capability
├─ Shot：React/Remotion 组件
├─ Technique：纯函数布局/几何/调度
├─ Contract：输入数据 JSON Schema
├─ Template：catalog/templates.json 条目
├─ Fixture：recipe + 黄金静帧
└─ Metadata：适用结构、标签、版本、示例
```

只有六项齐全并通过门禁，才能从 `candidate` 升级为 `ready`。

## 四、架构组件

### 1. Skill：`video-capability-author`

负责 AI 判断：

- 现有能力是否足够；
- 不足时应该新增什么结构；
- 视觉方案是什么；
- 数据契约是什么；
- 哪些部分可以复用；
- 何时需要用户确认。

Skill 不负责执行引擎。

### 2. Workflow：`author-remotion-capability`

负责多步骤编排：

```text
需求
→ 能力目录检查
→ 能力缺口
→ 提案
→ 用户确认
→ 代码实现
→ 自动注册
→ 契约/静帧/预览/成片门禁
→ Candidate
→ 提升
```

Workflow 独立安装，不进入 Feature Pack。

### 3. Native Shot / Technique

继续留在自媒体创作台内部：

```text
renderer/remotion/shots/
renderer/remotion/techniques/
catalog/contracts/
catalog/fixtures/
catalog/templates.json
```

符合 `plugin-split-policy.md`：领域视觉能力不拆 plugin。

### 4. Candidate

Candidate 保存：

- 新 Shot 源码；
- Technique；
- Contract；
- Template 注册；
- Fixture；
- QC 报告；
- 适用标签；
- 与基础能力的差异。

## 五、AI 作者流程

参考 anything2explainer 的多阶段方法，但改造成“能力沉淀”流程。

### 阶段 1：能力缺口

AI 先检查 catalog：

- 有没有结构匹配的 ready Shot；
- 数据契约是否覆盖；
- 是否只是样式不同，能否复用现有 Shot；
- 是否真的需要新代码。

输出：

```json
{
  "decision": "reuse | extend | new_capability",
  "reason": "...",
  "candidate_shots": ["composition", "flow"],
  "missing": ["flow-distribution"]
}
```

### 阶段 2：能力提案

```json
{
  "id": "sankey-flow",
  "name": "桑基流向图",
  "structure": "flow-distribution",
  "reuse_for": ["产业结构", "收入分配", "流量分析"],
  "data_contract": ["links", "unit", "total_label"],
  "visual_direction": "节点分层，连线宽度表示数值，科技蓝",
  "acceptance": ["连线比例正确", "标签不重叠", "内容随帧推进"]
}
```

**确认点 1：用户确认能力方向。**

### 阶段 3：数据契约

先写 schema，再写画面：

```text
catalog/contracts/sankey.schema.json
```

要求：

- 必填字段；
- 数值范围；
- 数组长度；
- 来源/单位；
- 与 `data.ingest` / `data.bind` 的映射关系。

**确认点 2：用户确认数据口径和字段。**

### 阶段 4：视觉方案

写清楚：

- 构图；
- 字号层级；
- 颜色令牌；
- 动效节奏；
- 安全区；
- 数据标签策略；
- 空态/错误态。

### 阶段 5：实现 Shot

一个镜头一个文件：

```text
renderer/remotion/shots/SankeyFlow.jsx
renderer/remotion/techniques/sankey.js
```

约束：

- 动画是帧号纯函数；
- 布局算法放 technique；
- 不测 DOM；
- 不用无种子随机；
- 所有颜色来自 style tokens；
- 文字宽度用 `foundation/text.js` 估算。

### 阶段 6：注册

自动检查或补齐：

```text
shots/registry.js
index.jsx
catalog/templates.json
catalog/fixtures/manifest.json
```

**确认点 3：用户确认预览。**

### 阶段 7：门禁

必须跑：

```text
go test
recipe.validate
template/registry 一致性
data contract
fixture 存在性
静帧 SSIM / diff
verify-render.py
确定性哈希
```

### 阶段 8：Candidate

把新能力保存为 Candidate：

- Shot 源码；
- Technique；
- Contract；
- Template；
- Fixture；
- QC 报告；
- 标签。

### 阶段 9：提升

通过后再标记为 `ready`：

```text
candidate → validated → ready
```

**确认点 4：用户确认沉淀为正式能力。**

## 六、首个试点：桑基流向图

### 用户需求

```text
用桑基图展示 2025 年安徽文旅消费从总收入流向
交通、住宿、餐饮、景区、购物，科技蓝，10 秒。
```

### 为什么选它

- 现有能力没有桑基图；
- 数据结构简单，可复用现有文旅消费数据；
- 有明确布局技法，适合沉淀 `sankey.js`；
- 视觉上与 composition/flow 明显不同；
- 能验证“AI 真的在写新 Remotion 能力”。

### 交付物

```text
renderer/remotion/shots/SankeyFlow.jsx
renderer/remotion/techniques/sankey.js
catalog/contracts/sankey.schema.json
catalog/templates.json（新增 sankey-flow）
catalog/fixtures/sankey-flow/recipe.json
catalog/fixtures/sankey-flow/frame-0180.png
shots/registry.js
index.jsx
```

### 数据契约

```json
{
  "links": [
    {"source": "文旅消费", "target": "交通", "value": 2860},
    {"source": "文旅消费", "target": "住宿", "value": 2450},
    {"source": "文旅消费", "target": "餐饮", "value": 2200}
  ],
  "unit": "亿元",
  "total_label": "文旅消费"
}
```

### 验收标准

- 5 个分类金额与来源一致；
- 节点/连线宽度比例正确；
- 标签不重叠、不越界；
- 内容随帧推进；
- 静帧基线通过；
- 10 秒 MP4 通过 `verify-render.py`；
- 同 Recipe 两次渲染一致；
- 晋升为 `sankey-flow` ready 模板。

## 七、实施阶段

### Phase 0：作者契约与脚手架（已完成）

- `schemas/capability-proposal.schema.json`
- `schemas/capability-descriptor.schema.json`
- `video-capability-author` Skill（已接入 `capability.audit`）
- `author-remotion-capability` Workflow（0.1.1，已通过 validate/build/package/candidate/test）
- 能力缺口的输出格式
- 新能力文件清单检查器

### Phase 1：Sankey 试点（1–2 天）

- 完成 `sankey.js` + `SankeyFlow.jsx`
- 完成约定、模板、fixture、registry
- 预览、MP4、QC
- 记录 Candidate

### Phase 2：作者流程 MVP（2–3 天）

- Skill 完整化：9 阶段、4 确认点
- Workflow 完整化：提案 → 实现 → 注册 → 门禁 → Candidate
- 自动检查缺失文件/注册项/contract/fixture
- 输出能力提案和差异摘要

### Phase 3：第二个能力验证复用（2–3 天）

建议选：

- 雷达图；
- 流向地图；
- 日历热力图。

如果第二个能力不需要修改作者流程，说明“能力沉淀”成立。

### Phase 4：last-mile（1–2 周）

按拆分政策做 External Pack：

- TTS；
- 字幕时间轴；
- BGM；
- 素材；
- 发布。

不把 Provider 塞进 Workbench。

## 八、第一周 Backlog

1. 新增 `capability-proposal` / `capability-descriptor` schema；
2. 新建 `skills/video-capability-author`；
3. 新建 `workflows/author-remotion-capability`；
4. 实现 Sankey 的 technique + Shot；
5. 写 contract + template + fixture；
6. 跑 static gate、静帧、预览、MP4；
7. 保存 Candidate；
8. 用 Radar 做第二能力预研。

## 九、验收标准

一条“AI 生成 Remotion 能力”的链路只有满足下面条件才算完成：

```text
需求
→ AI 明确判断现有能力不足
→ 输出可审查的能力提案
→ 写 Shot + Technique
→ 写 Contract
→ 自动注册 Template
→ 生成黄金样例
→ 预览
→ 渲染
→ QC
→ Candidate
```

缺任何一项，都只能算“一次性写了个视频”，不能算能力沉淀。

## 十、风险与边界

### 风险

1. AI 写的 Shot 视觉质量不稳定；
2. 布局算法没有边界测试；
3. 模板注册和 source 不一致；
4. 新 Shot 只对单一数据有效；
5. 把一次性镜头误判为可复用能力；
6. 复制 anything2explainer 的固定风格或受许可限制的代码。

### 边界

- 新 Shot 必须先通过门禁再标记 ready；
- 不允许运行时执行任意 JSX；
- 不把 Native Shot 拆 plugin；
- 不把作者 Workflow 塞进 Feature Pack；
- 新能力必须有 contract、fixture、template 和标签。

## 十一、与 anything2explainer 的关系

借鉴：

- 一镜头一文件的构建方式；
- 调研 → 脚本 → 分镜 → 镜头 → QC 的多阶段流程；
- 多 agent 并行构建；
- 定量 QC 和黄金样例；
- 风格/动效/构图方法文档。

不照搬：

- 固定黑底 MG 风格；
- 只做 16:9 长讲解；
- 每条视频重新写整套镜头代码；
- 非商业许可代码。

我们的目标不是成为 anything2explainer 的替代品，而是：

> **吸收它的作者方法，把 AI 写出的 Remotion 镜头沉淀成长期可复用的能力库。**
## 十二、下一步：从试点到能力库

SankeyFlow 证明"AI 写 Shot → 自动注册 → 审计 → 出片"可行。下一步不是继续堆模板，而是回填能力声明：

1. `route-growth`：补数据契约与 descriptor，把高铁路线经验从"一个满意视频"变成可复用能力；
2. `leaderboard-tech` / `bar-race`：补排行榜布局与竞速调度的 descriptor；
3. `choropleth` / `timeline` / `flow` / `composition`：按同一六件套回填；
4. 第二个新能力建议做雷达图或日历热力图，用来验证作者流程不依赖 Sankey 的特例。

回填顺序按"数据契约是否清楚"决定；没有契约的模板先留在 `template` 层，不强行标记为 `capability`。
