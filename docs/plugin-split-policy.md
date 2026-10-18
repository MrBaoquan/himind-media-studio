# HiMind 拓展拆分策略：什么该做 Plugin，什么不该

> 状态：架构决策  
> 日期：2026-10-17  
> 适用范围：自媒体创作台及其 Native Shots、Renderer、External Packs、Skills、Workflows  
> 依据：HiMind-Agent 的 `plugin.json`、Extension Catalog、Capability Gateway、Market、Skill/Workflow 契约

## 一、结论

> **插件是“通用独立能力”的部署边界，不是代码组织工具。**

只有同时满足多数条件的能力，才值得拆成 plugin：

1. 离开自媒体创作台仍然成立；
2. 可被其他 HiMind 应用复用；
3. 有独立维护者或独立发版节奏；
4. 有独立权限、凭据或运行时；
5. 可选安装，缺失时调用方可以降级；
6. 有稳定的 Capability 输入输出契约；
7. 需要独立签名、分发或回滚。

不满足这些条件的能力，先做应用内部模块。

## 二、HiMind 实际支持的拓展形态

从 `extensions.json`、`extension_source.rs`、`market.rs` 和 `plugin.rs` 看，当前体系支持：

| 形态 | 用途 |
| --- | --- |
| `plugin` | 本地能力包，提供 Capability、View、Command |
| `skill` | AI 客户端判断与流程说明，可声明 Capability/Plugin 依赖 |
| `workflow` | 多步骤编排，支持 Capability、Runtime、Manual、Loop、Provider-defined |
| `expert` | 专家型指令资产 |
| `instruction_pack` | 指令包 |
| `agent_preset` | DSH 等客户端的 Agent 预设 |
| `feature_pack` | 组合包，当前只组合 `plugin_ids`、`skill_ids`、`agent_preset_ids` |

### 必须注意的当前限制

1. **没有独立的 `asset-pack` 类型。**  
   风格、契约、黄金样例不能凭空成为第四种安装类型；要么放进 plugin，要么扩展底座。

2. **Feature Pack 没有 `workflow_ids`。**  
   `ExtensionFeaturePack` 当前只有 plugin/skill/agent preset。Workflow 需要独立安装，或先扩展底座。

3. **外部 `plugin.json` 没有可声明的 `kind`。**  
   外部插件默认按 `plugin` 处理；`connector`/`provider` 语义更多用于内置集成或 Workflow connector，不应假设有通用 connector 安装类型。

4. **凭据 S2 尚未实现。**  
   插件可以声明 credentials，Agent 可以统一保管，但插件目前不能直接取得明文。需要 Key 的 TTS/发布 Provider 不能假设“声明即可用”。

5. **插件调用有默认 30 秒超时、16 MiB 响应上限、并发 4。**  
   TTS、渲染、素材下载等长任务必须声明更长 timeout，或使用 Job 模式。

6. **一个插件 ID 只有一个家。**  
   不允许同一 ID 同时存在内置和外置实现；拆插件必须先确定归属和迁移路径。

## 三、当前项目的正确拆分

### 1. 自媒体创作台：应用型 plugin

```text
com.himind.media-workbench
  kind: plugin
  contributes.views: 创作台 UI
  capabilities:
    - project.*
    - plan.*
    - recipe.*
    - quality.*
    - candidate.*
  internal modules:
    - generation/
    - shots/
    - techniques/
    - workbench/
    - quality/
    - candidates/
```

它是 HiMind 的应用层插件，不是“通用能力插件”。  
内部可以有清晰模块边界，但不需要把每个模块都拆出去。

### 2. Native Shots：应用内部模块，不拆 plugin

```text
media-workbench/
  shots/
  techniques/
  styles/
  contracts/
  fixtures/
```

原因：

- Shot 是自媒体创作台的领域视觉语法；
- 离开这个应用单独存在意义有限；
- 版本与渲染器/应用强耦合；
- 拆出去只增加跨插件调用和签名成本。

**当前阶段：留在应用内，模块化即可。**

如果未来要做“第三方 Shot Pack 市场”，再考虑新增 `asset-pack` / `shot-pack` 类型，或做一个提供 `shots.list/describe` 的专用 plugin。不要把当前领域镜头硬塞进通用 plugin。

### 3. Renderer Adapter：通用 plugin

```text
com.himind.video-render-remotion
  capabilities:
    - render.*
    - artifact.*
    - runtime.*
```

渲染器是通用基础设施，应该独立于 Workbench。

### 4. External Packs：通用、可选 plugin

```text
com.himind.video-tts
com.himind.video-subtitle
com.himind.video-media
com.himind.video-publish
```

特点：

- 可被其他应用复用；
- 有独立凭据；
- 有独立 Provider；
- 有独立发版节奏；
- 可选安装；
- 缺失时创作台降级，不阻塞主链路。

### 5. Geo Pipeline：按复用决定

```text
com.himind.geo-pipeline
```

如果地图/路线会被其他应用复用，就拆；  
如果只有自媒体创作台使用，先做内部模块，等出现第二个调用方再拆。

### 6. Skills / Workflows：不属于 plugin

```text
Skills:
  video-director
  video-creator
  video-series
  video-style-curator

Workflows:
  generate-video
  generate-series
  promote-candidate
```

- Skill 保存 AI 判断和流程；
- Workflow 保存多步骤编排、审批、Candidate 和 Artifact；
- 它们不是本地能力包。

## 四、Feature Pack 的正确用法

可以建立：

```text
com.himind.feature.media-workbench
  plugin_ids:
    - com.himind.media-workbench
    - com.himind.video-render-remotion
  skill_ids:
    - video-director
    - video-creator
    - video-series
  agent_preset_ids:
    - himind-media-creator
```

但当前不能把 `generate-video` 放进 feature pack，因为 `ExtensionFeaturePack` 没有 `workflow_ids`。

两条可行路线：

- **短期**：Workflow 独立安装，并在 Skill 中声明所需 Capability；
- **长期**：给 `ExtensionFeaturePack` 增加 `workflow_ids`，让解决方案包真正覆盖完整生成链路。

## 五、迁移策略

### P0：内部模块化

当前 `com.himind.media-workbench` 内：

```text
generation/
shots/
renderer/
workbench/
providers/
```

先通过目录和架构测试建立边界，不物理拆分。

### P1：先拆 External Packs

优先拆：

- TTS；
- 字幕；
- 素材；
- 发布。

原因：它们最通用、最独立，且最容易出现独立凭据和运行时。

### P2：再拆 Renderer / Geo

等生成主链路稳定后，再拆：

- `com.himind.video-render-remotion`；
- `com.himind.geo-pipeline`。

### P3：最后考虑 Shot Pack

只有在出现第三方镜头包需求时，才设计 `asset-pack` 或专用 Shot plugin。

## 六、不要做的事

- 不要把 Native Shots 为了“架构漂亮”拆成 plugin；
- 不要把 TTS/字幕/发布塞进 Workbench；
- 不要把 Workflow 当成 plugin；
- 不要假设 Feature Pack 能组合 Workflow；
- 不要假设插件已经能拿到 credentials 明文；
- 不要给外部包使用未经登记的新 capability namespace；
- 不要让可选 Provider 成为 Workbench 的硬启动依赖。

## 七、最终判断

> **通用独立能力 → plugin；应用领域能力 → 内部模块；AI 判断 → Skill；多步骤编排 → Workflow；组合安装 → Feature Pack。**

这比“所有东西都拆 plugin”更符合 HiMind 的实际拓展体系，也更适合自媒体创作台当前阶段。
