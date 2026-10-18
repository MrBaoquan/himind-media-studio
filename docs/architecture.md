# 自媒体创作生态架构

## 分层

HiMind Agent 只负责通用 Capability Registry、MCP 暴露、插件生命周期和本地文件边界。短视频业务规则位于扩展源：插件提供确定性文件/Recipe/Job 能力，Skill 提供对话编排和沉淀方法。Dashboard 连接后只增加组织调度、审核和分发，不改变本地创作协议。

## 可替换适配器

模板描述 `adapter` 和 `composition`，Recipe 只携带数据与视觉参数。`adapter` 严格限定为 `remotion | hyperframes`：Remotion 是内置主渲染引擎，以固定版本、后台 Job 和工作区内运行时生成 MP4；HyperFrames 是同级可插拔实现。Three.js、地图等技术只能作为模板实现或引擎组合能力接入。HyperFrames Adapter 读取 `render.start` 返回的 `recipe_path`，把最终视频写回工作区，再调用 `render.complete` 登记 Artifact，因此所有实现共享同一个项目、任务和产物模型。契约细节见 [renderer-adapter.md](renderer-adapter.md)。

插件 UI 只调用项目、预览、渲染和 Artifact Capability，`.himind-media` 是唯一状态源。UI 关闭或 Agent 重启后，后台任务和产物仍可由 `project.get`、`render.status` 恢复查看。

## 反馈到资产

每个预览反馈按类别写入项目目录，项目 Recipe 通过 `project.update` 保存并递增 `revision`。风格 Skill 只在反馈被接受且可复现时提出候选，候选先保存到工作区，不直接覆盖扩展源：

```text
feedback.list -> candidate.save -> evidence -> template variant / style pack / quality rule
  -> golden storyboard -> quality.check -> 新版本资产
```

资产以 Git 版本控制，项目 Recipe 固定引用 `id + version`，支持回滚和复现实验。`expected_updated_at` 用于并发编辑保护，避免多个 AI 会话互相覆盖。

## 约束证据与交付门禁

`catalog/constraints.json` 是 harness 的唯一规则源，每条规则都必须指向一个真实执行点。源码模式可以从 `*_test.go` 扫描测试函数；插件安装包按分发规则排除 Go 源码，因此另存 `catalog/constraint-evidence.json` 记录已执行的测试证据。

`media.workbench.constraint.list` 在源码与安装包中都应返回 52/52 resolved：

```text
catalog/constraints.json      规则与执行点
catalog/constraint-evidence.json  安装包可用的测试证据
TestConstraintEvidenceCoversPackagedTests  防止证据遗漏
```

成片交付也有独立门禁：`media.workbench.artifact.export` 读取母版 Artifact 的 `quality_status`，`failed|error` 时返回 `blocked` 与 `ARTIFACT_QUALITY_FAILED`；`artifact.open` 仍允许调试审片。重型 Browser-Code Style Pack 通过 `runtime.max_workers` 收敛并发，避免单浏览器多页面截图停滞。

## 独立分发

GitHub 仓库是扩展源，插件 `.hmpkg` 和 Skill `.hmskill` 是不可变制品。独立模式安装默认注册本地能力；Dashboard 模式可在相同制品上增加组织审核和分发状态。目录工作区与扩展源码工作区分离，视频项目允许任意用户目录。

外部 AI 客户端通过 Agent 的 `extension.workspace.bind` 绑定本仓库根目录。Agent 读取 `extensions.json` 后发现插件和 Skill；`extension.workspace.current` 返回绑定结果，`extension.workspace.clear` 可恢复到会话目录。绑定只约束扩展创作与候选包，短视频项目的 `workspace_root` 不受此限制。

## 失败诊断

所有可恢复失败返回 `state: blocked` 与 `blockers[]`，每项包含 `code`、`stage`、`message`、`remediation`、`retryable`。外部 AI 按稳定 code 处理，例如 `renderer_runtime_missing` 表示缺少 Remotion 运行时，`adapter_not_found` 表示 Recipe 未使用 `remotion` 或 `hyperframes`。

## 技法层

插件不再只有"整块模板"。`renderer/remotion/techniques/` 下的模块是**独立可测的技法资产**：

| 文件 | 技法 | 契约 |
| --- | --- | --- |
| `techniques/geo.js` | 墨卡托投影 / 自适应铺满 / 折线弧长 | 纯函数，无随机 |
| `techniques/schedule.js` | 按年份分组的时间轴调度 + 派生指标 | 时长自适应，帧号决定论 |
| `techniques/labels.js` | 标签避让求解 | `no-overlap` / `in-bounds` / `order-stable` / `frame-deterministic` |
| `compositions/RouteGrowth.jsx` | 组合上述技法产出镜头 | 消费 `style_tokens`，不硬编码配色 |

对应能力：

- `media.workbench.route.repair` — 折返尖刺与闭环修复 + 站点锚定
- `media.workbench.route.validate` — 不变量校验（返回结构化 findings 与 `ok`）

数据侧的技法由 `catalog/data/geo/` 的示例资产演示，并可用
`HIMIND_REGEN_GEO=1 go test -run TestRegenerateExampleDataset` 复跑重生成。

详细契约、实测数字与"为什么这样设计"见 [techniques.md](techniques.md)。
## 能力层

模板回答"有哪些镜头"，能力回答"沉淀了什么可复用的视觉语法"。

```text
catalog/capabilities/<id>.json
  ├─ Shot          renderer/remotion/shots/<Shot>.jsx
  ├─ Technique     renderer/remotion/techniques/<technique>.js
  ├─ Contract      catalog/contracts/<capability>.schema.json
  ├─ Template      catalog/templates.json
  ├─ Fixture       catalog/fixtures/<capability>/
  └─ Metadata      适用结构、标签、验收标准、成片证据
```

对外由三个只读能力暴露：

- `media.workbench.capability.list` — 按结构/状态筛选；
- `media.workbench.capability.describe` — 读取完整六件套声明；
- `media.workbench.capability.audit` — 机械检查源码、契约、模板、注册、Fixture 与基线帧是否一致。

`capability.audit` 不通过时，能力不得标记为 `ready`。这把"能力沉淀"从文档约定变成可执行门禁，也避免 AI 把一次性镜头当成可复用能力。

作者流程由 Skill `video-capability-author` 负责判断，由 Workflow `author-remotion-capability` 负责多步骤编排与确认点。
## 代码项目模式

模板模式与代码项目模式共享项目、修订、预览、反馈、Artifact 与质量门禁，但渲染契约不同：

```text
template:
  Recipe -> Remotion -> Shot

code:
  Style Pack -> render.mjs -> Browser-Code -> MP4
```

代码项目通过固定 CLI 约束执行：

```text
node render.mjs --sheet / --strip / --stills / --frames / --encode
```

插件只执行白名单子命令，所有路径必须在 `workspace_root` 内。
`project.sync` 用源码 hash 标记 `preview_status=stale`；`quality.check` 与 `render.start` 共同保证旧预览不能进入最终渲染。

详细实现见 [code-project-mode.md](code-project-mode.md)。
