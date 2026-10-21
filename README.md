# himind-media-studio

HiMind Agent 的自媒体创作拓展源。通过 GitHub 分发，不依赖 Dashboard；安装后由 Agent 将本地插件 Capability 和 Skill 暴露给 HiMind AI 及兼容 Agent Skills 的 AI 工具。

**定位**：对外是一个以可沉淀视觉能力为质量杠杆的**自媒体视频生成器**；对内由**创作台 + 可替换的能力拓展**构成。用户通过 AI 对话提出需求，系统生成可校验的叙事计划与 Recipe，再由渲染器确定性出片。

- **自媒体创作台**（插件）—— 项目与资产中枢：项目/系列/修订史、资产解析、配方契约、质量门禁汇总、三栏工作台界面。它**不渲染、不含领域数据、不含技法代码**。
- **能力拓展**（插件）—— 真正干活的部分：轨迹与地理管线、素材检索、素材下载、音视频执行适配器。
- **声明资产**（JSON）—— 版式、风格、数据契约、黄金样例，三级作用域（项目 > 工作区 > 内置），**零发版**可扩展。
- **约束清单**（`catalog/constraints.json`）—— harness 的可审计来源：每条约束都必须指向一个执行点。
- **开源 prior art**（`docs/oss-evaluation/`）—— 新增能力 / Style Pack / Renderer 前先评估可复用开源库、许可证与依赖；GPL / 商业求解器不直接进入核心。

更名记录与预发布硬切换说明见 [docs/rename-to-media-studio.md](docs/rename-to-media-studio.md)；创意开发与 CreativePackage 见 [docs/creative-development.md](docs/creative-development.md)；故事视频（LibTV 类）能力演进见 [docs/story-video-architecture.md](docs/story-video-architecture.md)；镜头级程序化/AI 能力契约见 [docs/shot-capability-architecture.md](docs/shot-capability-architecture.md)；TTS/字幕/BGM/混音底座见 [docs/audio-subtitle-architecture.md](docs/audio-subtitle-architecture.md)；终极媒体架构与自主实施策略见 [docs/ultimate-media-architecture-implementation.md](docs/ultimate-media-architecture-implementation.md)；合肥人造太阳案例见 [docs/cases/hefei-artificial-sun.md](docs/cases/hefei-artificial-sun.md)；产品方向、同类开源对比与架构价值见 [docs/generator-direction.md](docs/generator-direction.md)；实施架构见 [docs/generator-implementation-architecture.md](docs/generator-implementation-architecture.md)；里程碑与任务拆分见 [docs/generator-implementation-plan.md](docs/generator-implementation-plan.md)；AI 生成新 Remotion 能力的实施计划见 [docs/capability-authoring-plan.md](docs/capability-authoring-plan.md)；现有创作台边界见 [docs/repositioning-architecture.md](docs/repositioning-architecture.md)。

## 安装与更新

在 HiMind Agent 的“扩展源”中添加 `https://github.com/MrBaoquan/himind-media-studio`。仓库根目录的 `extensions.json` 是扩展源发现清单。`.himind/catalog.json` 只在正式发布时生成；当前尚未发布，因此仓库不保留旧版占位目录。插件、Skill 与 Workflow 作为独立制品安装和更新，不依赖 Dashboard。

扩展源默认由用户主动添加，安装和更新也由用户决定。发布制品使用不可变版本号和 GitHub Release，目录记录文件大小、SHA-256 与签名；源码仓库不提交本地构建目录、Remotion 运行时或视频项目产物。

## 组成

- `extensions.json`：独立扩展源目录清单，供 Agent 绑定和发现。
- `plugins/com.himind.media-workbench/media-workbench`：本地 JSON-RPC stdio 插件；包含项目/创意/镜头/音频底座（Piper TTS、字幕 cues、BGM 预设、ffmpeg 混音与音频 QA）。
- `skills/video-director`：先把需求转成叙事计划与分镜计划，再编译和校验 Recipe。
- `workflows/generate-video`：对话→Plan→项目→预览→渲染提交的独立 Workflow（不由 Feature Pack 组合）。
- `workflows/author-remotion-capability`：能力缺口→提案→六件套实现→`capability.audit` 的作者 Workflow。
- `skills/video-creator`：从需求到预览、反馈和构建的创作编排。
- `skills/video-style-curator`：把反馈沉淀为模板变体、Style Pack 和质量规则。
- `skills/video-capability-author`：当现有 Shot 不足时，设计并实现新的 Remotion 能力。
- `skills/video-code-project`：把 AI 写代码的风格型视频接入创作台的审查、反馈、同步与渲染闭环。
- `examples/code-project-demo/`：Browser-Code 项目模式的最小黄金样例（1080×1920 / 10s / Canvas 2D）。
- `style-packs/kinetic-reel/`：来自 opus-video-skills 的 MIT 动态字体 Reel Style Pack（17s / 1920×1080）。
- `style-packs/painted-animation/`：来自同一来源的 MIT 手绘动画 Style Pack（11s / 1920×1080）。
- `style-packs/data-kinetic/`：数据驱动动态字卡，读取 `input/data.json`。
- `style-packs/paper-collage/`：程序化纸张拼贴、照片卡、胶带与印章风格。
- `style-packs/whiteboard-explainer/`：Rough.js 手绘白板讲解，适合知识、流程与方法论拆解。
- `style-packs/science-sketch/`：手绘线条 + 淡彩 + 图解箭头的科学解释风格，首个样例是《天空为什么是蓝色的？》；0.1.1 增加逐帧手绘 boil 与纸张漂移，避免入场动画结束后整屏长静帧。
- `style-packs/science-fusion/`：手绘科学图解 + AI 生成示意图的聚变科普风格，首个样例是《合肥，有一轮“人造太阳”》（EAST / 1 亿°C / 1066 秒）。
- `examples/creative-packages/science-sky/`：从主题、研究、脚本、分镜到 ProductionSpec 的完整创作需求包样例。
- 风格库：显示来源与安装状态，支持搜索、分类筛选、收藏、最近使用、2 秒循环 WebP 动态预览，以及「安装到本机」和「卸载本机副本」。
- 当前 21 个风格：动态字体、手绘动画、数据字卡、纸张拼贴、白板讲解、科学手绘、复古 VHS、漫画分镜、蓝图技术、歌词卡拉 OK、数据图表、极简编辑、等距信息图、霓虹街机、品牌手册、电影感地图路线、3D 产品展示、3D 数据故事、天体四季科普、线网地图动画、线网开通时间轴。
- `style-packs/retro-vhs/`：VHS 扫描线、色差、霓虹日落与故障字效。
- `style-packs/comic-panel/`：漫画分镜、网点、对话气泡与动作线。
- `style-packs/blueprint-tech/`：蓝图网格、尺寸标注与工程图签。
- `style-packs/lyric-karaoke/`：歌词时间轴、逐字填充与波形。
- `style-packs/data-chart-reel/`：动画柱状图与数值滚动。
- `style-packs/editorial-minimal/`：暖白纸张、衬线标题与编辑式数字排版。
- `style-packs/transit-network/`：白底线网地图动画，线路生长、站点出现、换乘脉冲与站名避让。
- `style-packs/celestial-seasons/`：three.js **PBR 天体四季科普**（30s / 1080×1920 / 30fps），使用 NASA 地球昼夜/云层/法线/高光贴图、日间-夜间过渡、海洋高光、大气边缘光与电影感太阳光晕；地轴倾角 23.44°、轨道偏心率 0.0167，明确标注“非真实比例”，附 NASA/NOAA 来源；默认混入确定性的程序化环境声床（无旁白，可用 `--silent` 关闭）。
- `style-packs/isometric-infographic/`：等距模块、立体卡片与流程箭头。
- `style-packs/neon-arcade/`：街机 HUD、像素星光、霓虹分数与连击。
- `style-packs/brand-lookbook/`：品牌字体、色板、网格规范与识别原则。
- `style-packs/cinematic-map-route/`：暗色地形、电影感路线生长与节点标注。
- `style-packs/3d-product-reel/`：three.js 产品工作室、金属材质、环形光效与卖点标注。
- `style-packs/data-story-3d/`：three.js 立体柱状数据空间与冠军高亮。
- `dist/asset-packages/`：确定性 `.hmpack` 资产包；`media.workbench.asset.pack/install/uninstall` 提供 SHA-256 校验、版本目录安装与卸载。
- 资产市场：`catalog/asset-packs.json` + `asset.market.list/install`；当前条目为 `prepared`，发布 GitHub Release 后可切换为 `published`。
- `catalog/`：版本化模板、风格和 Renderer Adapter 声明。
- `catalog/constraints.json`：约束清单（每条规则绑定执行点，由 `media.workbench.constraint.list` 暴露）。
- `catalog/constraint-evidence.json`：安装包用的测试证据清单；Go 源码不随插件分发，证据文件保证 `constraint.list` 在真实安装后仍是 52/52 resolved。
- `catalog/capabilities/`：可复用视觉能力描述（Shot/Technique/Contract/Template/Fixture/证据），由 `capability.list/describe/audit` 暴露。
- `docs/code-project-mode.md`：Browser-Code 项目模式、Style Pack 契约与 AI 审查闭环。
- `catalog/boundaries.json`：能力边界声明与迁移阶段，由架构适应度测试强制。
- `catalog/data/geo/`：随包分发的示例地理数据资产（17 条线路 / 97 站点 / 2,972 km）。
- `schemas/recipe.json`：Recipe 结构约束。
- `renderer/remotion/techniques/`：**技法资产**（投影 / 调度 / 标签避让），纯函数、帧号决定论。
- `dev/verify-render.py`：成片内容级验收脚本（只依赖 ffmpeg + Python 标准库）。
- `dsh/agent-presets/himind-media-workbench/`：可选的 DSH“自媒体创作台”预设。只有安装本扩展源中的创作台插件或功能包后，Agent 才会把它同步到 DSH；未使用扩展源时不会出现在 DSH 预设列表。

## 技法与质量

复用单位是**技法**而不是整块模板。详见 [docs/techniques.md](docs/techniques.md)。

**轨迹质量**由两个能力守住（纯函数，可独立测试）：

```
media.workbench.route.repair     修复折返尖刺与闭环，把首末点锚定到站点坐标
media.workbench.route.validate   校验站点锚定 / 段比率 / 折返尖刺 / 闭环
```

**版式**：`route-growth`（路网生长式地图），画法 × 调度 × 标签避让的组合，
消费 DTCG 风格令牌 —— 新增风格不需要改渲染器代码。

**结构类 Shot**：`composition`（占比/构成）与 `flow`（流程/链路）各自有独立数据契约；
它们回答的是"整体由什么组成"和"先做什么、再做什么"，不与 `compare` / `timeline` 混用。

**Storyboard 编辑器**：创作台在 Storyboard 项目下提供「预览 / 分镜」双视图。
镜头目录来自 `storyboard_shot` 声明，字段表单来自 `template.schema`；
保存时执行外层故事板契约 + 每个镜头自己的嵌套契约，外壳不硬编码任何题材字段。

**成片验收**：

```bash
python dev/verify-render.py <video.mp4>
```

检查画面内容是否随时间推进、路线是否随年份生长、各分区是否都有内容，
而不只是"文件存在"。

## 创作闭环

```text
对话需求 -> catalog/template/style -> project.create -> recipe.validate
  -> preview.start -> 用户反馈 -> feedback.record -> 再次预览
  -> quality.check -> render.start -> Remotion 后台任务 -> render.status -> artifact.export
```

每个项目只在调用方传入的 `workspace_root` 下写入：

```text
<workspace_root>/.himind-media/
├── projects/<project_id>/project.json
├── projects/<project_id>/recipe.json
├── projects/<project_id>/feedback.jsonl
├── candidates/<candidate_id>.json
├── jobs/
└── artifacts/
```

## 双生产模式

创作台现在有两条平行业务链路：

```text
模板模式（mode=template）
  数据型视频：排行榜 / 地图 / 路线 / 桑基图 / 时间线
  Remotion + Recipe + Shot + Contract + Fixture

代码项目模式（mode=code）
  风格型视频：动态字体 / 手绘 / p5 / three.js
  Style Pack + render.mjs + Sheet / Strip / Stills + MP4
```

最终帧默认使用 PNG（`--frame-format=png`），避免 JPEG 中间帧把暗部渐变压成色块；`--frame-format=jpg` 仅用于快速测试。
色带量化：`media.workbench.quality.banding`；旧项目 renderer 升级：`media.workbench.project.upgrade-style`。成片视觉 QA：`media.workbench.quality.visual`。渲染完成后自动执行，结果写入作业与母版 artifact 元数据。Style Pack 声明 `quality.text_audit` 后自动检查文字越界、安全区、重叠与标题命中。平面/白底风格可声明 `quality.banding_gate=false`，只记录平坦段指标而不阻断。

视觉 QA 为 `failed` 或无法完成（`error`）的母版会被 `media.workbench.artifact.export` 阻断，不能进入正式交付；`allow_failed=true` 只用于调试。重型 Canvas Style Pack 可用 `runtime.max_workers` 限制并发，默认渲染器只用 1 个 worker；并发截图导致的停滞与修复记录见 [docs/pitfalls/browser-code-workers.md](docs/pitfalls/browser-code-workers.md)；手绘/白板风格的长静帧问题与修复见 [docs/pitfalls/static-frames.md](docs/pitfalls/static-frames.md)。

代码项目仍由创作台管理项目、修订、审查图、反馈、质量门禁与产物；支持 `project.data.write` 注入 `input/data.json`，默认网络策略为 `none`，最终渲染前还要通过 `preview.determinism`：

```text
AI 写代码 -> project.import -> preview.review -> 创作台审查
  -> 确定性检查 -> render.start（创作台内可操作） -> 进度 -> 成片
  -> feedback.record -> AI 读取 project.context -> 修改代码
  -> project.sync -> preview.review -> quality.check -> render.start
```

详见 [docs/code-project-mode.md](docs/code-project-mode.md)、[docs/asset-pack-architecture.md](docs/asset-pack-architecture.md) 与 [docs/asset-pack-implementation-plan.md](docs/asset-pack-implementation-plan.md)。

## 渲染边界

插件以 Remotion 作为内置主渲染引擎：首次构建会在当前视频工作区准备固定版本运行时，随后由后台 Job 生成真实 MP4，不新增常驻端口。HTML 故事板承担快速结构预览；HyperFrames 与 Remotion 是同级可插拔引擎。Recipe 的渲染引擎只允许 `remotion` 或 `hyperframes`，Three.js、地图、图表等能力必须作为两种引擎内部的模板实现，不形成第三套项目或渲染协议。HyperFrames 适配器仍使用 `render.start -> render.complete` 契约。

插件详情中的“自媒体视频项目”窗口用于加载任意视频工作区、创建和切换项目、生成预览、构建视频、查看任务进度以及打开产物。界面只消费插件 Capability 和 `.himind-media` 状态，不维护第二份业务数据。

## 模板沉淀

`video-flow` 只是认可模板和视觉调性的参考来源，不是运行时依赖。迁移时将 `DataRanking`、`GDPRanking`、`ChinaGDPMap`、`AnhuiMap`、`AnhuiHSR`、`AnhuiHSRTime` 的结构、颜色、字号和动效整理为版本化模板/风格资产。新项目尽量通过替换 `data.items`、标题和少量插槽完成复用；重复反馈由风格 Skill 生成候选，不直接覆盖旧版本。

## 本地验证

在插件目录执行：

```text
go test ./...
go build -o bin/media-workbench.exe .
```

随后使用 HiMind Agent 的扩展开发 Capability 完成 `plugin.validate -> plugin.build -> plugin.package -> candidate.save -> extension.test`，Skill 使用对应的 validate/package/candidate/test 链。独立模式保留候选包，不把 Dashboard 提审当成本地创作前置条件。

外部 AI 工具首次使用本仓库时，先调用 `extension.workspace.bind`，传入仓库根目录；再调用 `extension.workspace.current` 确认绑定结果。清除绑定使用 `extension.workspace.clear`。视频项目的 `workspace_root` 仍可指向任意项目目录，不要求放在扩展源内。GitHub 安装默认进入独立模式；连接 Dashboard 只增加组织审核、分发和调度，不改变本地协议。

安装自媒体创作台插件后，Agent 会按目录中的 SHA-256 从同一 GitHub 版本读取 DSH 预设，写入当前 HiMind DSH 交互 Profile 的用户预设目录。预设复用 DSH 标准模式的完整工具集，仅增加模板选择、预览、反馈、质量检查和真实渲染的创作约束；卸载扩展不会删除用户自行创建的同名预设。

发布新版本时必须先提升对应 Manifest 的语义版本并更新 `release_notes`，再完成 `validate -> build/package -> candidate.save -> extension.test`。通过后为每个制品创建独立 GitHub Release，并机械更新 `.himind/catalog.json`；同版本制品不得覆盖。
