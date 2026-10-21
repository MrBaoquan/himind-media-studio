# 剪辑工作台 UI 成熟度路线与开源技术审查

> 日期：2026-10-20
> 当前版本：`com.himind.media-workbench@0.67`
> 目标：把当前“可用的创作台 Beta”推进到“结构接近成熟剪辑产品、操作具有专业手感、保留 AI-native 差异化”的工作台。
> 结论先行：**继续自研 Timeline 交互、项目模型和 AI 编排；只在边界能力上选择性引入开源库。不要整体替换成某个开源剪辑应用。**

## 一、当前成熟度评估

以 CapCut、DaVinci Resolve、Descript 等成熟产品作为 100 分基准：

| 维度 | 当前分数 | 判断 |
| --- | ---: | --- |
| 信息架构 | 70 | 三模式与工作区边界已成立 |
| 视觉系统 | 60 | 统一但缺内容型视觉与操作状态 |
| 专业时间轴体验 | 45 | 能执行命令，尚未形成直接操作手感 |
| 媒体管理 | 40 | 仍是文本媒体架，不是媒体库 |
| Inspector | 35 | 属性操作藏在命令带，缺选择驱动面板 |
| 简易创作 | 65 | AI 动作齐，探索性和资产浏览不足 |
| AI-native 体验 | 65 | 命令与 QA 很强，前端 diff/解释不足 |
| 交付与 QA | 65 | 有预设、QC，缺对比与版本视图 |
| 宽屏利用 | 55 | 1440px 下仍是单列工作区 |
| 综合产品体验 | 55–60 | 可交付 Beta，距离成熟商业产品一代 |

当前问题不是“缺少按钮”，而是缺少成熟工作台的内容密度、直接操作、状态反馈与专业面板。

## 二、对标策略

### 简易模式：对标 CapCut / Canva / Descript

- 默认只显示素材、预览、简化时间轴和 5–7 个主动作。
- 素材以缩略图、时长、类型呈现，而不是纯文本 ID。
- 智能动作先显示 diff / 预览，再应用。
- 模板、字幕样式、音乐、转场通过 Asset Pack 浏览，不堆在命令带。

### 专业模式：对标 DaVinci Resolve 的结构纪律

- 左侧媒体池、中间监视器与时间轴、右侧 Inspector。
- 选择驱动 Inspector，而不是所有属性堆在顶部。
- 时间轴以鼠标直接完成 move / trim / ripple / roll / slip。
- 多选、框选、右键菜单、工具态、快捷键一致。
- 不自研 Color/Fusion/Fairlight 全套；只做服务短视频精修的子集。

### AI 协作：对标 Descript / Runway 的反馈方式

- 用户输入自然语言后，先出现“将修改哪些片段”的可视化 diff。
- 每条 AI 操作可解释、可拒绝、可局部应用、可撤销。
- 文本、镜头、字幕、音频统一搜索。
- AI 建议以候选卡片进入时间轴，不直接秘密改写工程。

## 三、目标布局

### 宽屏（≥1200px）

```text
┌──────────────────────────────────────────────────────────────┐
│ 工作区 / 模式 / 项目 / 任务 / 保存 / 渲染                     │
├──────────────┬──────────────────────────────┬────────────────┤
│ 媒体库        │ 源监视器 / 节目监视器         │ Inspector      │
│ 搜索/筛选     │                              │ 片段/变换/速度  │
│ 缩略图/素材箱 ├──────────────────────────────┤ 音频/转场/调色  │
│ 代理/元数据   │ 时间轴：标尺/轨道/波形/字幕   │ 历史/版本       │
└──────────────┴──────────────────────────────┴────────────────┘
```

### 窄屏（<1000px）

```text
媒体库 ⇄ 监视器/时间轴 ⇄ Inspector
```

- 通过抽屉或单面板切换，不压缩到不可用。
- 当前 800×600 已做到无溢出；下一阶段要保证切换后仍有完整时间轴。

## 四、可直接推进的任务清单

### P0：专业工作台骨架

| ID | 任务 | 代码落点 | 验收 |
| --- | --- | --- | --- |
| UIP-01 | 可拖拽三栏编辑壳 | `ui/src/app.tsx`、`timeline-editor.tsx`、`styles.css` | 800px 抽屉模式；1280/1440 三栏；宽度持久化 |
| UIP-02 | 媒体库 v1 | `timeline-editor.tsx`、artifact API | 缩略图/类型/时长/尺寸/QA；搜索；100 项不阻塞 |
| UIP-03 | Clip Inspector v1 | `inspector` + `update_clip` | 选中即显示 Transform/Speed/Audio/Transition/Color |
| UIP-04 | 播放头与节目监视器联动 | 新 compositor hook / 现有 artifact 数据 | 拖动播放头当前帧变化；播放/暂停；源/节目分离 |
| UIP-05 | 时间轴直接操作 | timeline UI + command bus | 拖动移动、边缘 trim、转场手柄、吸附参考线 |
| UIP-06 | 工具态与上下文菜单 | timeline UI | 选择/刀片/修剪/滑动工具；右键菜单；光标反馈 |

### P1：成熟剪辑体验

| ID | 任务 | 说明 | 验收 |
| --- | --- | --- | --- |
| UIP-07 | 多选、框选、批量命令 | 批量删/移/调色/变速 | 10+ 片段批量操作可撤销 |
| UIP-08 | 波形与视频缩略图条 | audio waveform + filmstrip | 200 片段滚动稳定 |
| UIP-09 | 撤销历史与版本对比 | command log + timeline.diff | 可查看、比较、恢复 revision |
| UIP-10 | 素材市场/模板浏览器 | Asset Pack 前端化 | 搜索、预览、槽位替换 |
| UIP-11 | 面板拖拽与时间轴高度 | splitter | 宽屏可保存布局 |
| UIP-12 | 键盘与可访问性 | focus、aria、shortcut map | 核心操作无鼠标可完成 |

### P2：AI-native 差异化

| ID | 任务 | 说明 |
| --- | --- | --- |
| UIP-13 | AI diff 预览层 | 高亮将新增/删除/移动的 clip，可局部应用 |
| UIP-14 | 文本-时间轴双向编辑 | 改一句文稿，定位并改对应字幕/镜头 |
| UIP-15 | 语义素材检索 | “找讨论磁场的镜头”直接定位 |
| UIP-16 | 画中画/字幕/音频关键帧 lane | 专业精修可视化 |
| UIP-17 | 时间轴窗口化渲染 | 1000+ 片段仍可交互 |

## 五、开源技术审查

> 以下 star / license 为 2026-10-20 GitHub API 审查结果；star 只代表关注度，不代表适配度。

| 项目 | 类型 | License | Stars | 判断 |
| --- | --- | ---: | ---: | --- |
| OpenTimelineIO | Timeline 交换格式 / API | Apache-2.0 | 2,028 | **推荐做互操作适配** |
| MLT Framework | 原生多媒体编辑框架 | LGPL-2.1 | 1,866 | 可评估，不适合当前直接替换 |
| WebAV | WebCodecs 视频编辑 SDK | MIT | 2,090 | **适合做预览/浏览器导出 POC** |
| Diffusion Studio Core | 浏览器合成引擎 | MPL-2.0 | 1,248 | 与 Remotion 重叠，暂不引入 |
| OpenCut | 开源 CapCut 替代应用 | MIT | 95,387 | 做 UX/代码参考，不是稳定依赖 |
| Remotion | React 程序化视频 | 定制许可 / NOASSERTION | 65,319 | 已有，继续作为程序化合成入口 |
| Revideo | 代码生成视频 | MIT | 4,123 | 与 Remotion 重叠，不双引擎 |
| Clypra | Rust/Tauri 视频编辑器应用 | MIT | 3,329 | 可研究原生媒体层，不直接嵌入 |
| OpenChatCut | Agent-native 视频编辑器 | AGPL-3.0 | 2,535 | 产品参考；AGPL 不适合直接并入 |
| Etro | 浏览器视频编辑库 | GPL-3.0 | 1,152 | 许可证与插件分发冲突，排除 |
| ffmpeg.wasm | 浏览器 FFmpeg | MIT | 17,891 | 性能与体积不适合作为主预览引擎 |
| Mediabunny | 浏览器媒体读写 | MPL-2.0 | 7,334 | 可做后续浏览器媒体适配候选 |

### 5.1 OpenTimelineIO：采用，但只做边界适配

- 优点：行业交换格式，支持 OTIO/EDL/FCPXML adapter；Apache-2.0；适合导入导出和版本迁移。
- 不适合：它不提供 UI、实时预览、剪辑交互。
- 决策：增加 `timeline.interchange.otio.export/import` 适配层，用 OTIO 做 Timeline 交换，不让它成为内部真值。

### 5.2 WebAV：做受控 POC，不替换现有渲染

- 优点：MIT、TypeScript、WebCodecs、WebView2/Edge 兼容；适合低延迟预览与浏览器端导出。
- 风险：与当前 ffmpeg/Remotion 双引擎边界不清；复杂滤镜和字幕能力不足；需要确认 WebView2 媒体能力。
- 决策：POC 只做 `UIP-04` 的播放头预览；最终交付仍走单一 `render.start`。

### 5.3 MLT：暂不引入

- 优点：成熟非线编引擎，Shotcut 等使用；时间轴、音视频处理完整。
- 风险：C/LGPL 构建链、Windows 打包、与现有 Go/Tauri/ffmpeg 架构重叠；会引入第二套工程模型和渲染语义。
- 决策：记录为“若自研预览/渲染成为性能瓶颈”的备选，不在当前阶段接入。

### 5.4 Etro：排除

- GPL-3.0 对插件分发和商业闭源集成不友好。
- 类型与目标与当前 Timeline 内核重叠。
- 结论：不采用。

### 5.5 OpenCut / Clypra / OpenChatCut：参考产品，不整体换底座

- **OpenCut**：MIT、社区巨大，但当前处于重写阶段，未形成稳定 Editor API。
- **Clypra**：MIT、Rust/Tauri、波形与原生性能值得研究，但它是应用，不是可直接嵌入的稳定库。
- **OpenChatCut**：AGPL-3.0，产品概念与 HiMind 最接近（agent + timeline + Remotion），但许可证、产品边界和数据模型都不适合直接复用；只做 UX 拆解。
- 结论：可以借鉴布局、交互和快捷键，不能把应用仓库当“成熟库”塞进插件。

## 六、最终技术决策

### 继续自研

1. Timeline 命令模型、revision、append-only log；
2. 项目/创意包/Asset Pack/QA/MCP/AI 编排；
3. 剪辑工作台的布局、Inspector、媒体库和直接操作；
4. 单一 `render.start` 与交付语义。

### 引入/保留

1. FFmpeg：最终处理与交付；
2. Remotion：程序化视频合成；
3. OpenTimelineIO：后续交换格式适配；
4. WebAV：受控预览 POC，若 POC 通过再进入正式能力。

### 不做

1. 不整体替换成 OpenCut / OpenChatCut / Clypra；
2. 不引入 Etro（GPL-3.0）；
3. 不把 MLT 作为第二套时间轴内核；
4. 不把 ffmpeg.wasm 作为主预览引擎；
5. 不同时引入 Remotion + Revideo 两套程序化渲染器。

## 七、实施顺序

```text
UIP-01 三栏编辑壳
→ UIP-02 媒体库
→ UIP-03 Inspector
→ UIP-04 播放头联动
→ UIP-05 时间轴直接操作
→ UIP-06 工具态/菜单
→ OSS-01 OpenTimelineIO 交换适配
→ OSS-02 WebAV 预览 POC
→ UIP-07…UIP-17
```

## 八、验收门槛

- 800×600 与 1440×900 均无横向溢出；
- 选中片段 100ms 内显示 Inspector；
- 100 个素材可搜索、筛选、拖动；
- 播放头拖动 100ms 内更新节目画面；
- 时间轴仅用鼠标完成移动/切开/ripple delete/trim；
- OTIO 导入导出 round-trip 不改变 clip 时间；
- 所有 AI 操作有 diff、撤销和局部应用路径；
- 同一 Timeline 两次渲染 SHA 一致。

## 九、结论

**当前不应该换底座，而应该升级工作台产品层。**

OpenCut/OpenChatCut/Clypra 证明“agent + timeline”方向成立，但它们不是可直接嵌入的稳定基础设施；Etro 有许可证风险；MLT 过重；WebAV 和 OTIO 值得在边界接入。
最合理的路线是：

> **自研 Timeline 交互与 AI 编排 + Remotion/FFmpeg 渲染 + OpenTimelineIO 交换 + WebAV 受控预览 POC + Asset Pack 生态。**

## 十、实施进展（0.69.8，2026-10-20）

本轮把 UIP-01…UIP-06 与 OSS-01 从“计划”推进到真实页面可验收状态，并修正了复验中发现的六个真实问题。

| 项 | 状态 | 证据 |
| --- | --- | --- |
| UIP-01 三栏编辑壳 | DONE | 1440 宽媒体 360 / 中栏 700 / Inspector 380；800 宽抽屉互斥；宽度持久化 |
| UIP-02 媒体库 | DONE | 100 项 1ms 渲染；搜索 1/100；筛选 25/100；视频时长 1.0s |
| UIP-03 Inspector | DONE | 选中 4ms 出现；scale 1.2 提交 revision 2 并持久化 |
| UIP-04 播放头联动 | DONE | f225 更新 23ms；播放 36 帧/1.2s；暂停稳定 |
| UIP-05 直接操作 | DONE | move / trim / blade / ripple delete / 转场手柄全部鼠标完成 |
| UIP-06 工具态/菜单 | DONE | 选择/刀片/修剪/波纹工具、右键菜单、快捷键提示 |
| OSS-01 OTIO/EDL | DONE | `format=both` 真实 capability 调用；两种格式 round-trip 保留 transform/transition |
| UIP-07…UIP-17 | IMPLEMENTED | 代码与入口已存在，仍需专项 e2e（多选、1000+、AI diff、文稿联动等） |
| OSS-02 WebAV POC | TODO | 保持受控预览评估，不替换 render.start |
| OSS-03 MLT 复评 | TODO | 仅在预览/渲染成为硬瓶颈时启动 |

结论不变：**继续自研 Timeline 交互与 AI 编排，Remotion + FFmpeg 单渲染主链，OTIO 只做交换，WebAV 只做受控 POC。**

本轮新增的关键实现边界：

- 转场手柄属于 Timeline 交互层，不新增第二套转场数据模型，仍写 `transition_in/out`；
- 媒体时长在前端按 Blob 探测，不把 duration 复制进 artifact 真值；
- OTIO/EDL 导入导出继续以 `himind.timeline.v1` 为唯一真值，扩展字段放在 metadata/comment；
- 窄屏面板互斥只在 <1200px 生效，宽屏仍允许媒体 + Inspector 并存。

详细真实页面数据与截图见 `docs/ui-editing-review-2026-10-20.md` 第七节。
