# 终极媒体架构进度台账

> 每次自主推进后更新本文件。状态：DONE / IN_PROGRESS / TODO / BLOCKED。

| 里程碑 | 状态 | 证据 | 下一步 |
| --- | --- | --- | --- |
| M0 架构基线与契约 | DONE | `docs/ultimate-media-architecture-implementation.md` | 保持契约更新 |
| M1 Capability Bridge | DONE | 4 个 Rust 单测；18082 隔离实例验证媒体任务穿透、audio.* 放行、system.health 拒绝 | 保持白名单治理 |
| M2 audio-production 抽取 | DONE | `com.himind.audio-production@0.1.0`；9 clips / 17 cues / QA passed；候选 SHA `65ebe5a8…` | 保持插件独立 |
| M3 Workbench 音频轨挂载与 UI 切换 | DONE | `media.workbench.audio.attach`；UI 生成 9 段旁白并挂载字幕；workbench 0.17.3 候选 SHA `ffadf78c…` | 开始 M4 |
| M4 audio-intelligence | DONE | `audio.align` 244 words；`audio.subtitle.from-alignment` 12 cues；`audio.qa.sync` ok / 0 drift；可懂度 1.0 vs 0.3077；UI 显示「已使用逐词对齐字幕」；候选 SHA `ebfa8b97…` | 开始 M5 |
| M5 media-visual | IN_PROGRESS | 0.6.0 候选 SHA `2f343fd6…`；contact-sheet 3×3；upscale 图片 1620×2880 / 视频 384×384；video.qa 9 帧 / 0 blocker；video.scenes 5/21/29/66s | matte / 插帧 / 视频理解 |
| M6 Workflow / Skill / 跨模态 QA | IN_PROGRESS | `media-qa` Workflow 0.1.1：6 能力 preflight ready / contract dry-run passed；安装受 os error 5 阻塞；creator skill 0.9.3；constraints 67 条 / 0 unresolved | story-video |
| M7 市场与治理 | TODO | — | license / commercial profile |

## M5 设计要点

`com.himind.media-visual` 只承担「视觉领域确定性能力」，不重复注册生成模型：

- 生成：Agent Media Kernel 的 `media.image.generate` / `media.video.generate`；
- 参考沉淀：`visual.reference.pack` 输出标准 `himind.asset-pack.v1`，可被 Workbench Asset Market 直接发现；
- 一致性：`visual.reference.validate` + `visual.reference.character.consistency`；
- 画质：`visual.image.qa` 检测色带/色块、8px 块效应、过曝、细节量；
- 插件外消费者：Skill / Workflow 通过宿主 `asset.list/read` 引用参考包，不直接读文件。

色带检测只在「存在低频渐变」的区域统计相邻像素平坦率，避免把纯色背景误判为色带；
当前输出为 advisory 警告，不直接阻断成片。

## 最近一次运行

- 时间：2026-10-19
- 完成：M5 第一阶段（参考包 + 一致性 + 图像 QA）
- 关键证据：
  - 参考包：`asset-packs/hefei-tokamak-reference/1.0.0`，3 张 1080×1920 stills，SHA 校验 `ok=true`；
  - `visual.image.qa`：合肥 still 色带 0.615（medium），合成色带样本 0.8375（high），平滑抖动样本显著低于色带样本；
  - `visual.reference.character.consistency`：两张候选 best_score=1.0，`state=consistent`；
  - Workbench `media.workbench.asset.list` 返回 `hefei-tokamak-reference`，source=workspace；
  - 插件候选：`dist/plugins/com.himind.media-visual-0.2.0.hmpkg`，SHA-256 `22a7901d…`，candidate test/confirm 通过；
  - 5 个 `visual.*` 能力已在 18082 实例通过 Capability Bridge 真实调用。
- 门禁：media-visual `go test ./...`、`go build`、plugin validate/package、candidate test/confirm。
- 阻塞：成片重渲染 job `render-1792405520996612700` 仍在渲染帧（单 worker，约 3 帧/秒）。

## M6 第一阶段：约束沉淀

新增 `catalog/constraints.json`（`himind.media.constraints.v1`），把今天真实踩到的坑变成可复用规则：

1. `caption-subtitle-safe-band`：1080×1920 字幕安全带与说明卡上移；
2. `banding-gradient-gate`：暗部渐变/3D 的色带门禁；
3. `reference-pack-license`：参考包 SHA/许可证/商用标记；
4. `subtitle-from-alignment`：字幕必须从逐词时间戳生成；
5. `narration-requires-audio-path`：旁白项目 render.start 必须传 audio_path；
6. `determinism-before-final-render`：同帧集重复渲染哈希一致；
7. `reference-consistency-gate`：角色/主体 drift 不得进入主视觉。

Creator Skill 0.9.1 已引用该约束库，并声明 `visual.*` 可选能力与 `com.himind.media-visual` 可选依赖。
Skill 制品：`dist/skills/com.himind.media.creator-0.9.1.hmskill`，SHA-256 `82d41989…`。
candidate 已保存；`candidate.test` 仍受 AI 客户端目录写入限制（os error 5），因此未 confirm。

### 约束清单外部能力解析（Workbench 0.17.6）

- `plugin.json` 的 `contributes.views[].external_capabilities` 增加 `visual.*`；
- `constraints.go` 读取本插件能力 + 外部可选能力模式（精确 ID / 尾部 `.*`）；
- 约束执行点若指向外部可选能力，解析为“未安装时降级为警告”，不再误报 unresolved；
- 新增 5 条规则：`SUBTITLE_SAFE_BAND`、`BANDING_GRADIENT_GATE`、`REFERENCE_PACK_LICENSE`、`REFERENCE_CONSISTENCY`、`SUBTITLE_ALIGNMENT_DRIFT`；
- 真实调用 `media.workbench.constraint.list`：66 rules / 66 resolved / 0 unresolved；
- Workbench 0.17.6 候选 SHA `a5edb269…`，candidate test/confirm 通过。


## 成片验证（2026-10-19）

- 项目：`hefei-artificial-sun`（72s / 1080×1920 / 30fps）
- 修复：`drawCompare` 结论行 y 1580→1500；`drawImageScene` 说明卡 y 1490→1320
- 修复前：`TEXT_OVERLAP` 在 62s / 69s 判定失败
- 修复后：`text-audit` 在 58.5–71s 区间 fail=0 warn=0
- 最终渲染：`render-1792406571405929100`
- 产物：`.himind-media/artifacts/video-artifact-1792406571405929100.mp4`（23,004,701 B）
- 复制件：`dist/verify/hefei-artificial-sun-72s.mp4`
- SHA-256：`08b7f52dd0f8757efc56bdfd9ad3db969a05f110eafdb18eb51ba4285fa14a82`
- 视觉 QA：`quality_status=passed`
  - 2160 帧 / 30.000fps / 72.000s
  - 0 个黑帧区间；无 ≥1.5s 静帧
  - I=-16.1 LUFS / peak=-2.4 dBFS
  - 113 条文字：bounds / safe_area / text_overlap / title_hit 全部 passed
- 跨能力复核：对成片 62s 抽帧调用 `visual.image.qa`，banding 0.6702 → medium（纸张/平面风格，项目自身 banding_gate=false）

## 制品清单（本次）

| 制品 | SHA-256 |
| --- | --- |
| com.himind.media-visual 0.4.0 | `f85e6aa1…`（image.qa / video.qa / video.scenes / reference.*） |
| com.himind.media-workbench 0.17.8 | `02e57519…`（constraints 1.3.0，含 VIDEO_BANDING_GATE） |
| com.himind.media.creator 0.9.3 | `739f25ae…`（candidate saved，test 受 AI 客户端目录写入限制） |

## 成片抽帧与切点（media-visual 0.4.0）

### `visual.video.qa`

- 输入：`video_path` + 可选 `times` / `sample_count` / `keep_frames`；
- 输出：`probe`（时长/分辨率/fps/codec）、逐帧 `metrics`、`worst_banding`、`worst_blockiness`、`max_clip_high_ratio`、`min_sharpness`；
- 真实成片（72s）采样 `3,10,18,26,35,45,52,60,68`：
  - 9 帧全部抽取成功；
  - `worst_banding=0.6909 @ 52s`（medium，待复核）；
  - `worst_blockiness=1.4641`，`max_clip_high=0.0593`，`min_sharpness=1`；
  - 0 blocker / 4 warnings，`state=ready`。
- 结论：抽帧质检可稳定运行；色带 warning 与纸张渐变场景一致，按 Style Pack 的 `banding_gate` 处理。

### `visual.video.scenes`

- 基于 ffmpeg `select='gt(scene,threshold)',showinfo` 检测视觉切点；
- 真实成片阈值 0.3 / 0.15 均检出 `5,21,29,66s`；
- 未检出 `12/40/48/58s`：同纸张背景的手绘场景切换视觉差异很小；
- **约束**：场景切点只做“视觉硬切”检测，不能替代 Plan/Storyboard 的真值；同风格场景必须以分镜时间轴为准。

### 约束接入

- Workbench 0.17.8 / constraints 1.3.0 新增 `VIDEO_BANDING_GATE`，enforced_by = `visual.video.qa`；
- `constraint.list`：67 rules / 67 resolved / 0 unresolved。
- Creator Skill 0.9.3 增加成片抽帧复检步骤，制品 SHA `739f25ae…`（candidate saved，test 仍受 AI 客户端目录写入限制）。


## 联系表（media-visual 0.5.0）

- `visual.video.contact-sheet`：按 `times` 抽帧并合成单张 PNG，返回 `cells[{index,time,row,col}]`；
- 真实成片（72s）采样 `3,10,18,26,35,45,52,60,68`：
  - 输出 `dist/verify/hefei-contact-sheet.png`；
  - 尺寸 1080×1920，3 列 × 3 行，cell 360×640；
  - SHA-256 `6c5adc90…`，大小 1,780,172 B；
  - 9 个格子的时间映射完整。
- 用途：AI/人先用一张图审全片，再按时间回溯原片；后续并入 `media-qa` Workflow。
- 命名坑：`visual.video.contact_sheet` 因下划线被 ADR 0119 拒绝，已改为 `visual.video.contact-sheet`。

## 确定性放大（media-visual 0.6.0）

- `visual.upscale`：图片/视频 Lanczos 放大，支持 `scale` 或目标宽高；
- 视频输出 H.264 + AAC，默认 CRF 18 / preset medium，可 `mute`；
- 真实样例：
  - 图片 `dist/verify/hefei-62s.png`（1080×1920）→ `dist/verify/hefei-62s@1.5x.png`（1620×2880），SHA `419146c3…`；
  - 视频 `dist/verify/scene-test/mix.mp4`（256×256, 2s）→ `mix@1.5x.mp4`（384×384, 2s, h264），SHA `08153ef1…`。
- 边界：输出 2..8192 px；确定性放大不增加真实细节，生成式超分留给模型 adapter。

## 整片体检 Workflow（0.1.1）

- 包：`workflows/media-qa`，6 步：contact-sheet → video.qa → video.scenes → image.qa → reference.consistency → workbench.quality.visual；
- validate / package / candidate.test / candidate.confirm 全部通过；
- preflight：6 个能力 available，ffmpeg/ffprobe available，0 blockers；
- contract dry-run：execution plans / risk gates / declarative UI 全部 passed；
- 候选 SHA：`2130366f…`；
- 阻塞：`install_local_workflow_archive` 返回 `os error 5`，未进入 `workflow.catalog.list`；
- 解除方式：桌面端手工安装 `.hmwf`，或修复 `agent_home/workflows` 写权限。

## os error 5 根因与修复（2026-10-19）

### 现象

- `install_local_workflow_archive` 返回 `拒绝访问。 (os error 5)`；
- `extension.skill.candidate.test` 同类失败；
- 工作流候选本身 preflight / contract dry-run 全部通过。

### 排查证据

1. agent 进程 37908：`restricted=1`、`appcontainer=0`（Codex 沙箱受限令牌）；
2. 用 `visual.upscale` 把 `workspace_root` 指到 `%TEMP%`：
   `mkdir C:\Users\mrma6\AppData\Local\Temp\himind-agent-temp-probe: Access is denied`；
3. 同一插件写 `<agent_home>/workflows/probe.png` 与工作区输出均成功；
4. `install_local_archive` 源码：`staging = std::env::temp_dir().join(...)`，随后 `fs::create_dir_all(staging)`；
5. 结论：失败发生在 **系统 `%TEMP%` 暂存目录**，不是 agent_home ACL，也不是工作流内容。

### 修复

- 新增 `store::paths::agent_temp_dir()`：优先 `<agent_home>/tmp`，不可写时才回退系统临时目录；
- `workflow_manager.rs` 生产路径的 5 处 `std::env::temp_dir()` 改用 `agent_temp_dir()`：
  - 本地归档解包；
  - 公共目录 / 工作台目录工作流解包；
  - 公共 / 工作台归档下载。
- 门禁：`cargo fmt`、`cargo check --locked` 通过。
- 运行中的 18082 实例仍是旧二进制；需重启后重新执行安装验证。
- 测试代码仍使用系统临时目录，避免污染真实 agent_home。

## 界面主题（Workbench 0.17.9）

- 目标：去掉纯黑压迫感，建立高级感分层配色，并支持浅色主题；
- 深色：`#0d1016` 背景、`#131720/#181d27/#1f2532` 分层、暖白 `#e9e6df` 文本、低饱和 `#8fb4ff` 强调色；
- 浅色：`#f3f0e8` 暖纸背景、`#fbf9f4/#ffffff` 分层、`#1f232b` 文本、`#3f6fd8` 强调色；
- 主题通过 `data-theme` + CSS tokens 切换，选择写入 `localStorage`，首次进入跟随系统 `prefers-color-scheme`；
- 右上角新增主题切换按钮（太阳/月亮），主创作台与创作脉络顶栏均可用。

### e2e 证据

- 实例：18082 / PID 37908 / `HIMIND_AGENT_HOME=…/.extension-state/bridge-e2e-full`；
- 浅色：`theme=light`，body `rgb(243,240,232)` / 文本 `rgb(31,35,43)`；
- 深色：`theme=dark`，body `rgb(13,16,22)` / 文本 `rgb(233,230,223)`；
- 刷新后主题保持；切回浅色后再次生效；
- 截图：`.extension-state/bridge-e2e-full/theme-light.png`、`theme-dark.png`。

### 制品

- `dist/plugins/com.himind.media-workbench-0.17.9.hmpkg` — `c6486f01…`；
- UI 构建：`index-Bi_XZnkO.css` / `index-oWj8bmUJ.js`。

## 体检报告 UI（Workbench 0.18.0）

- Inspector 新增“体检报告”区块：最近产物的 QA checks、findings、严重度与修复建议；
- 无报告时显示生成入口提示；
- 真实 e2e（hefei-artificial-sun）：
  - `体检报告 passed`；
  - 9 项检查全部显示（frames / black_frames / freeze / banding / audio_loudness / text_bounds / safe_area / text_overlap / title_hit）；
  - 质量门禁 67/67；
  - 截图 `.extension-state/bridge-e2e-full/quality-panel.png`。
- 制品：`dist/plugins/com.himind.media-workbench-0.18.0.hmpkg` — `048569f4…`。

## 创意包血缘（Workbench 0.18.1）

- 顶栏“创意 r8”徽章；Inspector“来源创意包”；点击打开创作脉络并定位来源；
- e2e：hasChip=true / hasSource=true / clicked=true / creativeOpened=true；
- 截图：.extension-state/bridge-e2e-full/lineage-creative.png；
- 制品：com.himind.media-workbench-0.18.1.hmpkg — 2b7f06c5…。
## AI 协作可见性（Workbench 0.18.2）

- 创作顶栏状态：AI 处理中 / AI 已更新 rN / 未保存 / AI 协作 · 已同步；
- 新增“AI 对话”深链入口（title=在 AI 客户端继续）；
- e2e：synced=true、aiEntry=true，创作脉络打开并可定位来源创意包；
- 截图：.extension-state/bridge-e2e-full/ai-collab.png；
- 制品：com.himind.media-workbench-0.18.2.hmpkg — d2147e59…。
## 分镜可用性第一阶段（Workbench 0.19.0）

- 代码项目 STORYBOARD.md 解析为 9 个场景列表，右侧显示选中场景与 Sources/Visual rules；
- 支持 ↑/↓/Home/End 键盘导航，列表启用 content-visibility；
- e2e：listbox count=9，ArrowDown 后 selected 0→1；
- 截图：.extension-state/bridge-e2e-full/storyboard-nav.png；
- 制品：com.himind.media-workbench-0.19.0.hmpkg — 0a053874…；
- 待做：模板项目多选批量、镜头锁定、100 镜头压力验收。
## 大分镜可用性（Workbench 0.20.0）

- 代码项目：9 场景列表 + ↑/↓/Home/End + content-visibility；
- 模板项目：100 镜头项目 `100-dc8a0201` 实测；
- 多选 2 个镜头批量改为 5s，第 3 个保持 3s；
- 锁定第 1 个镜头后显示“已锁定”；
- 截图：.extension-state/bridge-e2e-full/storyboard-bulk.png；
- 制品：com.himind.media-workbench-0.20.0.hmpkg — 63a6affe…。
## 反馈闭环（Workbench 0.21.0）

- Inspector 新增反馈收件箱：全部/待处理/已解决/已忽略筛选；
- 每条反馈显示 category、时间、状态与备注；
- 支持“标记解决 / 忽略”，直接调用 media.workbench.feedback.resolve 并刷新；
- e2e：2 条 open 反馈 → 点击解决后出现“已解决”，待处理计数下降；
- 截图：.extension-state/bridge-e2e-full/feedback-inbox.png；
- 制品：com.himind.media-workbench-0.21.0.hmpkg — 9692d02d…。
## 任务中心与失败恢复（Workbench 0.22.0）

- Inspector 新增任务中心：状态、进度、阶段、消息；
- failed/cancelled 任务可“重新渲染”，直接调用 render.start 并展示 blocker；
- 状态栏补充任务阶段；
- e2e：6 条任务可见；cancelled 任务重试后显示 `code_preview_required: 最终渲染前必须先完成并审查最新预览`；
- 截图：.extension-state/bridge-e2e-full/job-center.png；
- 制品：com.himind.media-workbench-0.22.0.hmpkg — 750a14cc…。
## 字幕轨（Workbench 0.23.0）

- 新增 media.workbench.subtitle.list / subtitle.update；
- Inspector 显示字幕轨道、cue 列表与逐句编辑器（文本/起止时间）；
- 自动检测 CUE_OVERLAP / CUE_GAP / CUE_INVALID_RANGE / CUE_EMPTY_TEXT；
- e2e：hefei 12 cues 轨道可见；编辑首句并保存显示“已保存 12 条字幕”；已恢复原文；
- 截图：.extension-state/bridge-e2e-full/subtitle-track.png；
- 制品：com.himind.media-workbench-0.23.0.hmpkg — 831eb09f…。
## 交接摘要（Workbench 0.24.0）

- Inspector 生成项目、创意包@revision、风格、预览、确定性、产物 QA、待处理反馈摘要；
- 支持“复制摘要”与“打开 AI 对话”；
- e2e：summary/project/creative/qa/copy/ai 全部 true，复制提示出现；
- 截图：.extension-state/bridge-e2e-full/handoff-summary.png；
- 制品：com.himind.media-workbench-0.24.0.hmpkg — 2cd183d8…。
## 新手引导（Workbench 0.25.0）

- 顶栏“引导”打开三步上手：绑定工作区 → 描述主题/模板 → 审查图迭代；
- 提供“绑定工作区 / 让 AI 描述主题 / 从模板开始”三个入口；
- e2e：clicked=true，title/steps/workspace/ai/template 全部 true；
- 截图：.extension-state/bridge-e2e-full/onboarding.png；
- 制品：com.himind.media-workbench-0.25.0.hmpkg — 7ab80fad…。
## M1 剪辑内核（第一阶段，Workbench 0.26.0）

- 新增 timeline_core.go：帧时间 Timeline、Track、Clip、Command、validate；
- 命令：insert / overwrite / blade / lift / ripple_delete / trim / link / unlink；
- revision 冲突检测（expected_revision）；
- 新增能力：media.workbench.timeline.apply / timeline.validate；
- Go 测试 6 项：blade / ripple delete / insert / overwrite / overlap / revision conflict；
- 真实调用：blade at 120 → 两条 clip（0-120、120-300，source_in 120），revision=1；
- 制品：com.himind.media-workbench-0.26.0.hmpkg — d235f06d…；
- 待补：undo/redo、持久化、render.start 接收 timeline、真实素材 60s→30s。
## Timeline 历史（Workbench 0.27.0）

- 命令日志快照 + undo/redo，批内合并为一个撤销步；
- 新增能力：media.workbench.timeline.history.apply / timeline.undo / timeline.redo；
- 测试：10 步 trim 后 undo 10 次恢复原文，redo 10 次恢复终态；新命令清空 redo；
- 真实调用：两次 trim → revision 2 / 298；undo → revision 0 / 300；redo → revision 2 / 298；
- 制品：com.himind.media-workbench-0.27.0.hmpkg — bfa79960…；
- 待补：快捷键表、UI 撤销按钮、持久化。
## Timeline 导出适配（Workbench 0.28.0）

- 新增 media.workbench.timeline.export：单视频轨 ffmpeg 拼接，bitexact 确定性输出；
- 测试：3 段 0.5s 片段导出两次 SHA 完全一致；
- 真实素材：
  - 72s 成片取 2 段 → 6s，SHA 211ce90f…；
  - 72s 成片取 2 段各 15s → 30s，SHA 607d98d0…；
- 制品：com.himind.media-workbench-0.28.0.hmpkg — c90149d4…；
- 待补：render.start 的 timeline 输入模式、音频轨拼接、字幕烧录。
## render.start 接入 Timeline（Workbench 0.29.0）

- render.start 识别 timeline 输入，走 timeline-ffmpeg 适配，返回 completed job + artifact；
- 真实调用：72s 成片取 2 段 → 6s，artifact SHA 211ce90f…，duration=6；
- 与直接 timeline.export 的 SHA 完全一致，证明单一入口与确定性；
- 制品：com.himind.media-workbench-0.29.0.hmpkg — 63a0d781…；
- 待补：音频轨拼接、字幕烧录、timeline 持久化。
## Timeline 音频轨（Workbench 0.30.0）

- timeline.export / render.start 支持 V1 + A1 双轨拼接；
- 真实调用：2 段视频 + 2 段音频 → 6s，artifacts 输出 h264 + aac；
- has_audio=true，audio_clip_count=2，duration=6；
- 制品：com.himind.media-workbench-0.30.0.hmpkg — e8eeb20c…；
- 待补：字幕烧录、Timeline 持久化、UI。
## Timeline 持久化（Workbench 0.31.0）

- 新增 media.workbench.timeline.get / timeline.save；
- 写入 <project>/timeline.json，支持 expected_revision 并发校验；
- 测试：save→get 往返、revision 冲突；
- 真实调用：hefei save revision=1，get state=ready / clips=1；
- 制品：com.himind.media-workbench-0.31.0.hmpkg — 262f7c01…。
## 时间轴 UI 第一版（Workbench 0.32.0）

- 新增剪辑视图：加载 timeline.get，显示 V1/A1 轨道与 clip；
- 支持选中、切开、删除、波纹删除、保存；
- e2e：tabFound=true，editor=true，四个按钮可见，clip c1 可见；切开后 c1 出现 3 次（原片 + 右半段 + 选中徽章）；
- 保存提示在本次自动化中未稳定捕获，待补一次明确 e2e；
- 截图：.extension-state/bridge-e2e-full/timeline-ui.png；
- 制品：com.himind.media-workbench-0.32.0.hmpkg — 708ac99b…；
- 待补：媒体池、双监视器、播放头/吸附/缩放、Insert/Overwrite、保存 e2e。
## 时间轴 UI 增强（Workbench 0.33.0）

- 播放头、1x-4x 缩放、Insert/Overwrite 接入；
- 修复保存的 expected_revision：使用已保存 revision，而不是内存新 revision；
- e2e：editor/insert/overwrite/zoom 全部可见；切开后保存显示“已保存”；
- 截图：.extension-state/bridge-e2e-full/timeline-ui2.png；
- 制品：com.himind.media-workbench-0.33.0.hmpkg — ef61caee…。
## 时间轴媒体池入口（Workbench 0.34.0）

- 时间轴新增媒体池条：列出项目产物，可选择并“加入轨道”；
- e2e：媒体池/产物数/select 可见；加入轨道动作未稳定复现（inserted=0），未标记完成；
- 制品：com.himind.media-workbench-0.34.0.hmpkg — 1775438b…；
- 待补：定位 select 状态与 insert 点击时序，补明确 e2e。
## 媒体池插入修复（Workbench 0.35.0）

- 产物改为“kind · id +”徽章，一键插入轨道；
- 消除下拉选择状态时序问题；
- e2e：媒体池可见，产物徽章 found=true，插入后 inserted=1，保存“已保存”；
- 截图：.extension-state/bridge-e2e-full/timeline-media-pool3.png；
- 制品：com.himind.media-workbench-0.35.0.hmpkg — 9e887ecd…；
- 待补：源/节目双监视器。
## 双监视器第一版（Workbench 0.36.0）

- 源监视器：显示选中 clip 的 media_id、source in/out、duration；
- 节目监视器：显示时长/fps/clip 数/画布，并提供“渲染 Timeline”；
- e2e：source/program/render 全部 true；sourceInfo=true；
- 渲染动作返回 `media_paths missing` blocker（当前 timeline 的 c1 使用 m1，不在 artifacts 映射中），说明错误可见、未静默；
- 截图：.extension-state/bridge-e2e-full/dual-viewer.png；
- 制品：com.himind.media-workbench-0.36.0.hmpkg — 7d5c3b61…；
- 待补：实际视频画面预览、成功渲染路径、100 clip 性能。
## 源监视器视频预览（Workbench 0.37.0）

- 选中产物 clip 后调用 artifact.read，生成 Blob URL，内嵌 video/img；
- e2e：插入产物 clip → 选中 → 预览元素为 VIDEO，src 以 blob: 开头；
- 截图：.extension-state/bridge-e2e-full/source-preview.png；
- 制品：com.himind.media-workbench-0.37.0.hmpkg — 08a419a3…；
- 待补：source in/out 三点编辑、节目监视器实际画面、100 clip 性能。
## 三点编辑（Workbench 0.38.0）

- 源监视器可编辑源入点/出点；
- “三点插入”按 source 区间插入到播放头；
- e2e：in/out 输入与按钮可见；插入后 clip 计数 5；保存“已保存”；
- 截图：.extension-state/bridge-e2e-full/three-point.png；
- 制品：com.himind.media-workbench-0.38.0.hmpkg — 5bb0581e…；
- 待补：节目监视器实际画面、100 clip 性能、撤销 UI。
## 节目监视器真实画面（Workbench 0.39.0）

- 节目监视器加载最近成片 video（artifact.read + Blob）；
- e2e：节目监视器可见，video src 以 blob: 开头；
- 源监视器 + 节目监视器 + 三点插入形成可用闭环（节目为最近成片，不是播放头当前帧）；
- 截图：.extension-state/bridge-e2e-full/program-preview.png；
- 制品：com.himind.media-workbench-0.39.0.hmpkg — cb9ac678…；
- 待补：播放头当前帧预览、吸附、100 clip 性能、撤销 UI。
## 撤销/重做 UI 与吸附（Workbench 0.40.0）

- 时间轴新增撤销/重做按钮（本地快照栈）与吸附开关；
- 播放头点击可吸附到 clip 边界（15 帧内）；
- e2e：吸附按钮可见；本次自动化中撤销按钮仍为 disabled，说明前一步删除未生效（clip 选择/命令状态待定位）；
- 制品：com.himind.media-workbench-0.40.0.hmpkg — 025fba37…；
- 待补：撤销/重做 UI 动作 e2e、100 clip 性能。

## 剪辑 M1-M4 闭环（Workbench 0.67.0，2026-10-20）

- 完成：一键成片、模板槽位、智能字幕、去静音、节拍 markers、文本粗剪、时间码批注、PIP、retime、调色/LUT、颜色关键帧、速度曲线、timeline.diff、自然语言命令计划、媒体池拖拽；
- 专业命令：ripple_trim / roll / slip / slide / retime / update_clip / multi-clip link；
- 渲染：xfade、PIP overlay、eq/LUT、多音轨 ducking、loudnorm、SRT 烧录、6 个平台预设、render 前 QA；
- UI：简易/专业/交付三模式共用同一 Timeline；快捷键、撤销/重做、100 clips 虚拟化验收；
- 制品：0.67.0 候选 SHA `4dc3940792d57894e3304315adb65708ca02e01fb1898cc28acaaab7ac52b204`；
- 资产包：editing-kit@1.0.0 SHA `7311cf7085c50f5c420b270e38c393c9ad567192aed8b522be97fd12c14b7c2e`，validate 通过；
- 产物：timeline-m3 / audio-ducking / xfade / silence-trim / pip / retime / color / lut；
- 详细证据：`docs/editing-m1-m4-implementation-report.md`；
- UI 审查：`docs/ui-editing-review-2026-10-20.md`；800×600 编辑态无横向/纵向首屏溢出；
- 边界：内容感知重构为非黑边中心对齐，不是逐帧语义跟踪；其余 M1-M4 / E/C backlog 已收口。


## 剪辑工作台成熟度与开源技术审查（2026-10-20）

- 文档：`docs/editing-ui-maturity-and-oss-review.md`；
- 当前 UI 综合成熟度：55–60/100；信息架构约 70，专业 timeline 交互约 45；
- 决策：继续自研 Timeline 交互、Inspector、媒体库和 AI 编排；
- 采用/评估：OpenTimelineIO（Apache-2.0）做交换；WebAV（MIT）做预览 POC；FFmpeg + Remotion 保持主渲染链；
- 排除：Etro（GPL-3.0）、OpenCut/OpenChatCut/Clypra 整体替换、MLT 直接双内核；
- 新增 backlog：UIP-01…UIP-17、OSS-01…OSS-03。

## 剪辑工作台 UI 成熟度闭环（Workbench 0.69.8，2026-10-20）

- 范围：UIP-01…UIP-06 与 OSS-01 的真实 18082 页面复验；UIP-07…17 保持已实现、待专项 e2e；
- 三栏：1440×900 下媒体 360 / 中栏 700 / Inspector 380，标题栏恢复全宽 1440×52；
- 窄屏：800×600 无溢出，抽屉互斥，标题栏横向可滚动；
- 媒体库：100 个真实产物 1ms 渲染；搜索/筛选/时长探测通过；
- Inspector：选中 4ms 出现；scale 修改提交 revision 并写入 `timeline.json`；
- 播放头：拖动 23ms 更新节目监视器；播放/暂停稳定；
- 鼠标剪辑：move / trim / blade / ripple delete / 新增转场手柄全部通过；
- 交换：OTIO 与 EDL 真实 import/export，3 clips、transform.scale=1.2、transition=23f 完整保留；
- 确定性：`hefei-artificial-sun` 三个采样帧双渲染 SHA 一致（9s，valid=true）；
- 修复的真实缺陷：根 padding 压扁标题栏、命令带 shrink 竖排、媒体库无时长、缺转场手柄、OTIO `format` 字段错配、窄屏双抽屉；
- 门禁：`npm run typecheck`、`npm run build`、`go vet ./...`、`go test ./...`、`gofmt -l` 全通过；
- 制品：`com.himind.media-workbench-0.69.8.hmpkg`，SHA-256 `f0445128a229350338451046a7c162b7296b9a9446474b11a640c1877af68b31`；
- 截图：`.himind/long-tasks/20261020-media-workbench-ui-maturity/evidence/wide-1440x900.png`、`narrow-800x600.png`；
- 详细：`docs/ui-editing-review-2026-10-20.md` 第七节、`docs/editing-ui-maturity-and-oss-review.md` 第十节。

## 时间线 / 轨道成熟度（Workbench 0.70.4，2026-10-20）

- 5 轨真实实验项目：V1 3 clips + V2 PIP + A1 旁白 + A2 BGM（muted）+ S1 2 字幕；
- 轨道头 128px：色条、名称、类型/角色、Eye/Volume 与 Lock 按钮；
- 新增 `add_track` / `remove_track` / `set_track_state` 命令与 Go 测试；
- 视频轨 Muted 参与 ProgramMonitor 与渲染主轨选择；
- 跨项目播放头复位，修复 9030 帧项目切 600 帧项目仍显示 f5000 的串帧；
- 行高 54/48/42，1440×900 五轨首屏完整；800×600 低高度进入紧凑模式；
- 标尺改 mm:ss 标签，字幕 clip 以字幕文本为主标签；
- 综合评分从约 57 提升到 75，仍缺逐轨高度、Solo、排序、音频表、marker lane；
- 制品：`com.himind.media-workbench-0.70.4.hmpkg`，SHA-256 `91935e7e04bcee4b1a75f14cc8fd4768f987a80eb10192dde309253d9e931f1d`；
- 详细：`docs/timeline-track-maturity-review-2026-10-20.md`；
- 截图：`timeline-tracks-final-1440.png`、`timeline-tracks-final-800.png`。

## 轨道操作深度闭环（Workbench 0.74.1，2026-10-20）

- 逐轨高度：拖拽 + 紧凑/标准/高预设，持久化；
- 音频 Solo：solo 优先于 mute，TimelineAudioMixer 试听与导出共用语义；
- 轨道排序/重命名/颜色：同类型拖拽排序、双击重命名、原生色盘、持久化；
- 音频表与轨道音量：WebAudio Analyser 实时电平，track.volume_db 叠加 clip 音量进入导出；
- 轨道头 180px，容纳 grip/色条/名称/音量/电平/M/S/L；
- 1440×900 五轨 scroll 312/312；800×600 紧凑模式 host 176；
- 门禁：gofmt clean、go vet、go test、npm typecheck/build、determinism 3/3 通过；
- 制品：`com.himind.media-workbench-0.74.1.hmpkg`，SHA-256 `5d40aef15d601c7e5f6627d2bf3df47470d3d26b6cde0fbe57272f541c9eef26`；
- 详细：`docs/timeline-track-maturity-review-2026-10-20.md` 第九节。
