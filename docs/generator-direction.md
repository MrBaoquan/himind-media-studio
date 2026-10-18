# 从创作台到高质量短视频生成器

> 状态：方向文档  
> 日期：2026-10-17  
> 适用范围：`himind-media-studio` 的产品定位、生成主链路、能力资产与同类开源对比  
> 实施架构：[generator-implementation-architecture.md](generator-implementation-architecture.md)  
> 实施计划：[generator-implementation-plan.md](generator-implementation-plan.md)  
> 当前进度：M0 规划层、`video-director` Skill、`generate.run`、Plan 审阅 UI、单镜头系列、`data.ingest` / `data.bind` 与 `generate-video` Workflow 0.1.4 已落地；安徽高铁已真实出 MP4。下一步是 Workflow 真机运行、Candidate 提升与批量渲染。

## 一、结论先行

我们要做的**是短视频生成器**，不是“只做项目管理、不负责生成”的创作台。

更准确的定位是：

> **一个以可沉淀视觉能力为质量杠杆的短视频生成器。**
>
> 对外，用户只需要和 AI 对话、给数据或给 brief，就能生成一条质量稳定的短视频。  
> 对内，创作台负责项目、系列、修订、反馈、质量与能力沉淀；渲染器负责确定性执行。

创作台不是与生成器并列的产品，而是生成器的**项目面和质量面**。

```text
用户看到的产品
  AI 对话 → 生成短视频 → 预览 → 修改 → 批量/发布

系统内部的架构
  创作台（Plan / Recipe / 项目 / 反馈 / 质量 / 候选）
    + Native 视觉资产（Shots / Techniques / Styles / Contracts / Fixtures）
    + 可选能力包（素材 / 配音 / 字幕 / 发布）
    + 渲染适配器（Remotion，后续可扩展）
```

一句话：

> **对外是生成器，对内是能力生态。**

## 二、生成器的两阶段模型

AI 对话天然不确定，但短视频生产可以被拆成两个阶段：

```text
第一阶段：不确定的规划
  对话 / brief / 数据
    → 叙事计划 NarrativePlan
    → 镜头计划 StoryboardPlan

第二阶段：确定的执行
  Recipe
    → 契约校验
    → 固定运行时与资产版本
    → 渲染
    → 质量门禁
```

### 关键设计：规划可以自由，执行必须锁定

AI 可以每次重新“导演”，但导演结果必须固化成结构化的 Recipe。  
一旦 Recipe、运行时、数据资产和风格令牌确定，后续渲染就是确定的。

这带来三层确定性：

| 层级 | 保证什么 | 由什么负责 |
| --- | --- | --- |
| 语义确定性 | 数字、标签、来源、结论不写错 | 数据契约、事实校验、Recipe 校验 |
| 结构确定性 | 同类内容使用同一套镜头结构与节奏 | Shot、Storyboard、模板版本 |
| 渲染确定性 | 同一份 Recipe 两次渲染结果一致 | 固定运行时、资产哈希、黄金基线 |

“确定性”不等于“每次生成一模一样”。  
它的含义是：

- AI 可以提出不同方案；
- 用户确认后，方案变成可重复执行的生产单；
- 同一生产单不被随机素材、随机布局和隐式环境改变；
- 系列内容只替换数据，不重新发明画面结构。

## 三、沉淀闭环

沉淀不是保存一个项目文件，而是把一次成功生产变成下一次可调用的能力。

```text
生成
  → 用户反馈“这条不错”
  → 保存 Candidate
  → 自动跑契约、静帧、成片验收
  → 升级为可复用能力
  → AI 下次可以检索、复用、派生
```

一条可沉淀的能力至少包含：

```text
Candidate
├─ 适用场景与标签
├─ 叙事结构（为什么这样讲）
├─ Storyboard（镜头顺序、时长、转场）
├─ Recipe（数据、布局、风格、动效参数）
├─ 数据契约（字段、类型、范围）
├─ 黄金样例（关键帧基线）
└─ 质量报告（SSIM、内容推进、可读性、确定性）
```

只有“画面 + 参数”不够。  
必须同时保存：

- **为什么这样做**：叙事意图与约束；
- **需要什么数据**：契约与示例；
- **怎么判断它没坏**：基线、验收脚本、版本。

这才是能力记忆，而不是项目垃圾场。

## 四、同类型开源库分析

> 以下信息以 2026-10-17 公开 README / GitHub 元数据为准。Star 只代表社区规模，不代表架构质量。

### 4.1 直接生成器 / 自动化剪辑

| 项目 | 核心路径 | 复用单位 | 强项 | 主要边界 |
| --- | --- | --- | --- | --- |
| [MoneyPrinterTurbo](https://github.com/harry0703/MoneyPrinterTurbo) | 主题/关键词 → 脚本 → 素材搜索/文生视频 → TTS → 字幕 → BGM → 拼接 → 发布 | 模型 Provider、素材源、生成参数 | 覆盖完整、出片快、批量与发布成熟 | 画面质量依赖素材贴合度；数据准确性和品牌一致性较弱 |
| [ShortGPT](https://github.com/RayVentura/ShortGPT) | LLM 编辑语言 + JSON → ContentShortEngine / ContentVideoEngine → MoviePy 渲染 | 编辑块、Prompt、Provider | 较早提出“LLM 可理解的视频编辑语言”；框架化程度高 | 本质仍是线性剪辑流水线；视觉组件和数据契约不是一等公民 |
| [NarratoAI](https://github.com/linyqh/NarratoAI) | 长视频理解 → 解说文案 → 自动剪辑 → 配音 → 字幕 | 影视解说流程、模型/语音服务 | 影视解说垂直场景深，画面理解与解说节奏成熟 | 领域专用；不适合排行榜、占比、流程、地图等结构化视觉叙事 |
| [MoneyPrinterPlus](https://github.com/ddean2009/MoneyPrinterPlus) | AI 视频 + 批量混剪 + 多平台发布 | 片段素材、混剪规则、发布账号 | 批量混剪和国内平台发布非常实用 | 以素材拼接为主，缺少可验证的视觉语法与数据契约 |
| [AI-Youtube-Shorts-Generator](https://github.com/Anil-matcha/AI-Youtube-Shorts-Generator) | 长视频 → Whisper 转写 → LLM 找高光 → 自动裁切 9:16 | 高光片段、裁切参数 | 长转短、自动裁切效率高 | 解决的是“切片”而不是“从零生成结构化内容” |

### 4.2 相邻底座，不是同类生成器

| 项目 | 定位 | 与我们的关系 |
| --- | --- | --- |
| [Remotion](https://github.com/remotion-dev/remotion) | React 程序化视频框架 | 我们的主渲染底座；确定性、组件化、类型化 |
| [Revideo](https://github.com/midrender/revideo) | 代码生成视频的框架 | 同类底座备选；说明“代码即视频”是成熟方向 |
| [Motion Canvas](https://github.com/motion-canvas/motion-canvas) | 用代码做动画与可视化 | 视觉表达参考；不是短视频生成器 |
| [OpenCut](https://github.com/OpenCut-app/OpenCut) | 开源 CapCut 替代品 | 时间线编辑器参考；不是 AI 生成器 |

### 4.3 它们共同证明了三件事

1. **“主题 → 成片”是刚需**：MoneyPrinterTurbo、ShortGPT、MoneyPrinterPlus 都验证了这一点。
2. **last-mile 决定可用性**：TTS、字幕、BGM、素材和发布不是附属功能，而是成片体验的一部分。
3. **单纯堆模型不够**：素材越随机，画面越依赖运气；要稳定质量，必须有结构、约束和复用单位。

## 五、我们的架构价值

### 5.1 不是“功能更多”，而是“下一次更省”

MoneyPrinterTurbo 这类工具的质量提升，主要来自更强的模型、更全的素材源和更多参数。  
我们的质量提升，来自**每一次成功生产都能变成下一次的资产**：

```text
第一次做高铁路线视频：设计镜头、调布局、写技法
第二次做另一省份：只换数据
第十次做同类内容：选择已有能力，几分钟出片
```

这就是资产复利。

### 5.2 质量前置，而不是渲染后补救

我们的门禁不是“生成完再看运气”，而是：

- 数据契约在 `recipe.validate` 拦截脏数据；
- 安全区、画布、时长在配方阶段校验；
- 静帧基线防止视觉漂移；
- 成片验收检查内容是否推进、画面是否空白；
- 同一 Recipe 做确定性校验。

同类开源项目大多把重点放在“能不能生成”，我们同时回答“**生成得对不对、稳不稳、能不能复现**”。

### 5.3 视觉语法是一等公民

我们的复用单位不是“一个模板文件”，而是：

- Shot：数字、对比、引语、排行、时间线、地图、占比、流程；
- Technique：投影、轨迹修复、标签避让、竞速插值、布局分配；
- Style：DTCG 风格令牌；
- Contract：每类内容需要什么数据；
- Golden：这类画面的正确样例。

这让我们能回答一个更细的问题：

> “这条视频为什么好？好在哪里？下一次怎么只复用好的部分？”

### 5.4 AI 被约束，而不是被放任

AI 在我们的架构里不是“随便写一段提示词，然后调用文生视频”。  
它需要：

1. 判断叙事结构；
2. 从能力目录选择 Shot；
3. 按数据契约填数据；
4. 生成可验证的 Recipe；
5. 根据门禁和反馈迭代。

这会让 AI 的输出更接近“导演 + 制片”，而不是“抽卡”。

### 5.5 项目、系列、反馈形成长期记忆

`project / revision / feedback / candidate / artifact` 不是 CRUD 装饰，而是生成器的学习闭环：

- 哪个结构适合哪类数据；
- 哪种风格被用户接受；
- 哪次修改有效；
- 哪个候选值得升级为正式能力。

这类记忆比单次对话历史更有价值。

### 5.6 渲染器可替换，但生成契约不变

Remotion 是当前主渲染器，但不是产品契约。  
生成器的稳定接口应该是：

```text
Recipe + 资产版本 + 运行时版本 → 视频
```

未来接 HyperFrames、FFmpeg、素材混剪器或其他渲染器时，项目、契约、质量门禁和反馈仍然成立。

## 六、必须承认的短板

### 6.1 last-mile 还不完整

目前我们强在结构化视觉，不强在：

- 多 Provider TTS；
- 字幕时间轴；
- BGM；
- 库存素材/B-roll；
- 多平台发布；
- 数字人/口播。

这些应该作为独立能力包接入，而不是塞进创作台本体。

### 6.2 冷启动成本高于纯提示词工具

纯提示词工具输入一句主题就能出片。  
我们需要 Shot 库、契约、风格和基线。前期更慢，但同类内容越多，边际成本越低。

### 6.3 模板可能过度收敛

如果所有内容都套同一套镜头，会产生“模板感”。  
解决方式不是放弃模板，而是：

- 参数化 Shot；
- 同一结构提供多种节奏/转场/构图变体；
- 让 AI 在受控范围内组合，而不是完全自由生成。

### 6.4 规划质量决定上限

`NarrativePlan` 和 `StoryboardPlan` 如果选错结构，后面的渲染再稳定也没用。  
因此需要建立规划评估集：

- 同一 brief 是否选出合理结构；
- 数据是否被正确映射；
- 镜头时长是否合理；
- 是否存在事实/口径错误。

## 七、应该向同类项目学什么

| 能力 | 学谁 | 接入方式 |
| --- | --- | --- |
| 多 TTS / 音色试听 | MoneyPrinterTurbo、MoneyPrinterPlus、NarratoAI | `voice.synthesize` 能力包 |
| 字幕时间轴 / Whisper | MoneyPrinterTurbo、NarratoAI | `subtitle.align` 能力包 |
| 库存素材 / B-roll | MoneyPrinterTurbo、ShortGPT | `media.search` 能力包 |
| BGM | MoneyPrinterTurbo、NarratoAI | `music.pick` 能力包 |
| 批量混剪 | MoneyPrinterPlus | `series.generate` / `media.mix` |
| 长转短高光切片 | AI-Youtube-Shorts-Generator | `clip.highlight` 能力包 |
| 多平台发布 | MoneyPrinterTurbo、MoneyPrinterPlus | `publish.*` 能力包 |
| 编辑语言思想 | ShortGPT | 借鉴“LLM 可读的编辑块”，但我们的契约是 Recipe/Shot，而不是自由剪辑脚本 |

不要学的是：

- 把所有 Provider、素材、渲染、发布塞进一个单体；
- 用随机素材和随机参数掩盖质量问题；
- 把“生成成功”当成“生成合格”。

## 八、目标生成链路

```text
AI 客户端对话
  │
  ├─ generate.plan
  │    ├─ NarrativePlan：讲什么、为什么这样讲
  │    ├─ StoryboardPlan：哪些镜头、多长、什么转场
  │    └─ 预估：时长、所需数据、风险
  │
  ├─ generate.recipe
  │    ├─ 选择 Shot / Style / Contract
  │    ├─ 填入数据
  │    └─ 生成可校验 Recipe
  │
  ├─ validate
  │    ├─ 数据契约
  │    ├─ 安全区/画布/时长
  │    └─ 引用与资产边界
  │
  ├─ preview / quality.check
  │
  ├─ render
  │
  └─ feedback / candidate.promote
       └─ 沉淀为新能力
```

生成器应该有两种入口：

1. **一键生成**：给主题或 brief，AI 自动完成 plan → recipe → render；
2. **系列生成**：固定结构和风格，只替换数据，批量出片。

两者共享同一套契约和门禁，不允许“一键模式”绕过质量检查。

## 九、路线图

### P0：生成主链路

1. `generate.plan`：brief/data → NarrativePlan + StoryboardPlan；
2. `generate.series`：同一结构批量换数据；
3. `candidate.promote`：好作品 → 基线 + 契约 + 标签 → 可复用能力；
4. 生成器预检：时长、成本、数据缺口、风险提示。

### P1：last-mile 能力包

1. `voice.synthesize`；
2. `subtitle.align`；
3. `media.search` / `media.mix`；
4. `music.pick`；
5. `publish.*`。

### P2：创作体验

1. Storyboard 可视化编辑器；
2. 镜头级参数表单；
3. 系列模板管理；
4. 候选对比与 Diff；
5. 质量报告可视化。

### P3：工程边界

1. 渲染适配器独立插件；
2. 地理管线独立插件；
3. 素材/配音/发布独立能力包；
4. 跨机器基线校准。

## 十、衡量这个方向是否成立

不要只看“生成了多少条”，要看：

| 指标 | 说明 |
| --- | --- |
| 首次可用时间 | 从对话到第一条可看预览的时间 |
| 平均修订次数 | 同类内容需要改几轮 |
| 数据错误率 | 数字、标签、来源、口径错误比例 |
| 视觉缺陷率 | 越界、重叠、空白、不可读比例 |
| 能力复用率 | 新项目复用已有 Shot/模板的比例 |
| 系列吞吐 | 同一结构一天能稳定产出多少条 |
| 确定性通过率 | 同 Recipe 两次渲染一致的占比 |
| 用户接受率 | 不需要大改即可发布的比例 |

如果这些指标持续改善，说明“沉淀”真的在变成生产力，而不是变成更多文档和模板文件。

## 十一、最终定位

> **HiMind 短视频生成器：面向高质量、可复用、可复现的内容生产。**
>
> 用户通过 AI 对话提出需求，系统把需求转成叙事计划和镜头计划，再用可验证的视觉能力完成渲染；好的结果会被沉淀为下一次可调用的能力。

我们和 MoneyPrinterTurbo 这类项目的区别，不是“谁更自动”，而是：

- 它们优化“从主题快速拼出一条能发的视频”；
- 我们优化“把一类视觉表达沉淀成稳定、可重复、可组合的生产能力”。

短期看，它们更快；  
长期看，**当内容需要数据准确、品牌一致、系列复用和可审计质量时，我们的架构价值会显现出来。**









