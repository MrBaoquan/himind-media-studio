# 终极媒体架构实施策略（自主执行版）

> 目标：把 HiMind Media Studio 从“一个视频插件”演进为“媒体内核 + 领域插件 + Asset Pack + 创作台 + Workflow/Skill + Renderer”的长期生态。
> 执行方式：按里程碑自主推进；每次只做一个可验证切片；先补契约与桥，再拆能力，最后扩生态。

## 一、不可动摇的原则

1. **一个 Capability ID 只能有一个 Owner。**
2. Agent 内核负责“调模型”，插件负责“会做媒体”，Workbench 负责“把媒体做成作品”。
3. 插件只拆通用、独立、可复用、可独立发版/授权的能力。
4. 模型、声音、角色、风格、BGM、SFX 优先沉淀为 Asset Pack，不随插件发版。
5. Skill 保存 AI 判断；Workflow 负责多步骤编排；Feature Pack 只组合 plugin/skill/agent preset。
6. 长任务必须 Job 化；凭据不进 manifest、不进 Asset Pack、不进 prompt。
7. 生成资产必须有 Generation Manifest：provider、model、prompt、seed、reference、license、consent、parent、QA。
8. 确定性能力与 AI 能力同等重要；AI 失败必须有 fallback。
9. 不做每个模型一个插件，不做每个能力一个插件。
10. 先打通一条真实链路，再扩能力；不以“接了多少模型”衡量进度。

## 二、目标分层

```text
Control Plane
  AI 对话 → Skill → Workflow → Workbench → 审批 / 策略

Data Plane
  CreativePackage / Shot / Timeline / Asset / Lineage / QA

Execution Plane
  Agent Media Kernel → Capability Plugins → Providers / Runtimes → Renderers
```

目标插件集：

```text
com.himind.media-workbench       应用型插件
com.himind.audio-production      音频生成与组装
com.himind.audio-intelligence    音频理解、变换、智能 QA
com.himind.media-visual          图片/视频控制、后处理、视觉理解
com.himind.renderer-remotion     渲染器
com.himind.renderer-browser-code  渲染器
com.himind.media-3d              可选，3D/空间资产
```

Agent 内核保留：

```text
media.image.generate
media.image.edit
media.video.generate
media.job.get / wait / cancel
未来如增加：media.audio.speech / music / sfx
```

## 三、里程碑

### M0 架构基线与契约

- 本文件、进度台账；
- 统一所有权矩阵；
- 统一 Capability 命名规则；
- 后续 schema 目录规划。

验收：文档可执行、每条规则有 Owner、每个阶段有门禁。

### M1 Capability Bridge

问题：插件视图当前只能调用本插件声明的 Capability，无法让创作台 UI 调用 `audio-production` / `media-visual`。

方案：

- 插件 Manifest 的 `views[]` 增加 `external_capabilities` 白名单；
- 只允许 `media.*` / `audio.*` / `visual.*` / `asset.*` 等受限命名空间；
- 支持精确 ID 与尾部 `*` 前缀；
- 仍然走 Capability Gateway 的权限、审批、风险检查；
- 禁止 `system.*` / `extension.*` / `plugin.*` / `workflow.*` 等管理面能力。

验收：

- 本插件声明能力仍可调用；
- 白名单内的外部能力可调用；
- 未声明/越权/管理面能力被拒绝；
- 单测覆盖 manifest 校验与调用判定；
- Agent 静态门禁通过。

### M2 抽取 `com.himind.audio-production`

- 新建插件；
- 迁移 TTS / BGM / Mix / QA / Runtime / Voice；
- 新 Capability 使用 `audio.*`；
- Workbench 保留 `subtitle.generate` 与 `audio.attach`；
- 旧 `media.workbench.audio.*` 标记 deprecated，保留一版兼容。

验收：

- 真实 RPC 生成旁白、BGM、混音与 QA；
- 不依赖 Workbench 内部音频实现；
- 输出 manifest 使用工作区相对路径；
- license / commercial_use 写入 manifest。

### M3 Workbench 音频轨挂载与 UI 切换

- 新增 `media.workbench.audio.attach`；
- 字幕由 Workbench `subtitle.generate` 生成；
- 创作台 UI 通过 Capability Bridge 调 `audio.*`；
- 音频面板显示 manifest / QA；
- 渲染门禁继续要求 `audio_path`。

验收：

- UI 真实点击生成音频；
- 字幕写回项目；
- preview / determinism / render 全链路通过；
- e2e 截图与文字证据。

### M4 `com.himind.audio-intelligence`

- ASR、forced alignment、diarization、audio tagging；
- 复用 `media-ai` 的 runtime/job 模式，但不重复其 capability ID；
- `audio.align` 用于字幕逐词时间戳；
- `audio.qa.intelligibility` / `audio.qa.sync`。

验收：

- 用最终混音回听脚本，检查漏读/错读；
- 字幕与旁白漂移 ≤ 0.2s；
- Job 可取消、可超时、可重试。

### M5 `com.himind.media-visual`

- 不重复 `media.image.generate` / `media.video.generate`；
- 只做底座没有的域能力：control、reference、upscale、matte、segment、video extend、interpolate、lipsync、understand；
- 图片/视频 provider 只做 adapter，不按模型拆插件；
- 角色、LoRA、ControlNet、风格进入 Asset Pack。

已落地（`com.himind.media-visual@0.2.0`）：

- `visual.reference.pack`：参考图 → 标准 `himind.asset-pack.v1`（SHA/尺寸/许可证/商用标记）；
- `visual.reference.validate`：资源完整性、SHA、license 检查；
- `visual.reference.character.consistency`：候选图 × 参考包最佳匹配，输出 consistent / drift；
- `visual.image.compare`：确定性结构+颜色相似度；
- `visual.image.qa`：色带/色块、8px 块效应、过曝、细节量；
- `visual.video.qa`：按时间抽帧复用 image.qa 阈值，输出 worst_banding 与逐帧报告；
- `visual.video.scenes`：ffmpeg 场景分数切点检测（只做视觉硬切，不替代分镜真值）。

参考包写入 `<workspace>/asset-packs/<pack_id>/<version>/`，可被 `media.workbench.asset.list/read` 直接发现。
下一批：upscale / matte / segment / interpolate / video.understand。

验收：

- 角色参考图 → 关键帧 → 图生视频链路；
- 角色一致性检查有结构化 findings；
- 失败可回退到 `ai.image + Ken Burns`。

### M6 Workflow / Skill / 跨模态 QA

- `story-video`、`image-shot`、`video-shot`、`audio-narration`、`audio-dub`、`audio-enhance`、`sound-design`；
- `media-director`、`character-continuity`、`audio-director`、`media-qa`、`rights-license`；
- 跨模态 QA：文图一致、角色一致、音画同步、字幕同步、混音平衡。\n\n已落地：Creator Skill 0.9.1 引用 `catalog/constraints.json`；Workbench 约束清单 1.2.0 支持通过 `external_capabilities` 引用 `visual.*` / `audio.*` 可选能力，未安装时降级为警告。

验收：一条 60–90 秒故事短片可从主题走到成片；每一步可追溯、可回退。

### M7 市场与治理

- Plugin / Asset / Skill / Workflow 市场；
- 版本锁定、SHA、签名、license、consent；
- demo / commercial profile；
- 发布、回滚、审计。

## 四、自主执行协议

每次自主运行：

1. 读取本文件与 `ultimate-media-architecture-progress.md`；
2. 选择第一个 `TODO` 且未被阻塞的里程碑；
3. 只做一个可验证切片；
4. 改代码 + 加测试 + 跑门禁；
5. 更新进度台账（状态、证据、下一步）；
6. 若发现阻塞，记录阻塞原因，转向下一个非阻塞任务；
7. 只有里程碑完成、失败或需要用户决策时才通知用户；
8. 全部里程碑完成后，将本自动化暂停。

## 五、全局验收门禁

- `gofmt -l` 无输出；
- `go vet ./...`；
- `go test ./...`；
- `cargo fmt --check`；
- `cargo check --locked`；
- `npm run typecheck`；
- `npm run build`；
- `git diff --check`；
- 涉及页面改动：18082 development profile 真实 e2e；
- 涉及能力：Capability 必须可真实调用，不能只写文档。

## 六、风险与回滚

| 风险 | 处理 |
| --- | --- |
| Capability 重复注册 | 一个 ID 一个 Owner；冲突直接拒绝 |
| 跨插件调用越权 | external_capabilities 白名单 + Gateway 审批 |
| 插件爆炸 | 只按领域拆；provider 做 adapter |
| 模型/声音许可证不清 | Asset Pack 元数据 + commercial profile |
| 长任务超时 | Job 模式 + 取消 + 重试 |
| 生成不可复现 | Generation Manifest + seed/model/prompt/reference |
| 旧能力破坏 | deprecated 一版兼容 + 明确迁移 |
| 绝对路径污染项目 | manifest/项目数据使用工作区相对路径 |

## 七、当前状态

- M0：进行中
- M1：待开始
- M2–M7：待开始

## 八、下一步

先完成 M1 Capability Bridge，再用它把 `audio-production` 从创作台抽出去。
