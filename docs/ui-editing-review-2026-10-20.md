# 剪辑工作台 UI / 信息架构审查（2026-10-20）

> 审查对象：`com.himind.media-workbench@0.57` 的真实 18082 页面
> 重构版本：`com.himind.media-workbench@0.67`
> 审查方式：CDP 真实 DOM 尺寸、控件数量、滚动尺寸、模式切换、深/浅主题与关键交互 e2e。

## 一、结论

当前页面已经不是“把所有按钮堆在一条工具栏里”的粗糙形态。重构后，编辑态使用全宽工作区，信息层级收敛为：

```text
工作区标题 / 模式 / 项目状态 / 保存与渲染
→ 模式化命令带
→ 源监视器 + 节目监视器
→ 可拖拽媒体架
→ 时间码标尺 + 轨道
```

## 二、审查前发现的问题

在真实 800×600 WebView 窗口中：

| 问题 | 审查前指标 | 影响 |
| --- | --- | --- |
| 页面横向溢出 | body `1159×748`，viewport `800×600` | 右侧内容被挤出屏幕，必须横向滚动 |
| 编辑态仍保留外部项目栏/Inspector | 左 232px + 右 296px | 真正的时间轴只剩 631px |
| 工具无层级 | 109 个控件，32 个 ≤11px 控件 | 找不到主操作，视觉噪声大 |
| 编辑工作区与底部时长尺重复占高 | 工作区 376px，内容 `scrollHeight=878` | 时间轴落在首屏以下 |
| 命令无分组 | Blade、Trim、Color、PIP、QA 混排 | 不符合剪辑软件心智 |
| 监视器/媒体池/时间轴都是同权区块 | 无主次 | 专业感弱，时间轴被压缩 |
| 时间轴缺少标尺 | 只有一段进度条 | 无法按时间定位 |
| 状态消息占据布局高度 | error/notice 直接插入 flex 流 | 出现消息时把时间轴再次顶出首屏 |

## 三、重构措施

### 3.1 编辑态全宽

- 进入“剪辑”时自动隐藏外部项目栏和项目级 Inspector。
- 中栏改为 `minmax(0,1fr)`，修复 `min-content` 导致的 1679px 假宽度。
- body 最终严格保持 `800×600`，无横向溢出。

### 3.2 标题与动作层级

- 顶栏改为“工作区标识 + 模式 + 项目元数据 + 保存/渲染”。
- 撤销/重做/批注/快捷键降为次级动作。
- 编辑态隐藏“刷新、风格库、创作脉络、生成审查图、引导”等非当前任务动作。

### 3.3 专业命令分组

专业模式命令带按现实剪辑职责分组：

- 编辑
- 修剪
- 速度
- 构图
- 调色
- 转场
- 联动

命令带采用单行横向滚动，不再把 20 多个按钮压成多行。右侧被隐藏的组可通过横向滚动发现。

### 3.4 监视器与媒体架

- 源/节目监视器保持双栏，移动端才降为单栏。
- 800px 窗口下监视器视频高度限制为 96px，为时间轴释放 ~64px。
- 媒体架固定 44px，高度稳定；媒体卡可拖入轨道，也可点击插入。
- 交付模式只保留节目监视器，并隐藏媒体架，形成专注交付视图。

### 3.5 时间轴

- 增加 sticky 时间码标尺，按画布时长显示 6 个刻度。
- 轨道头与轨道 lane 分离，轨道 lane 加入网格纹理。
- clip 字号提升到 10.5px，选中态使用 ring + 阴影。
- 时间轴成为唯一获得剩余高度的区域，滚动由轨道区内部承担。

### 3.6 状态消息

- error / notice 改为右上角浮层 toast，不再参与 flex 高度分配。
- 交付模式即使出现“已调色 revision 7”等消息，也不会裁切时间轴。

## 四、审查后指标

| 指标 | 审查前 | 重构后 |
| --- | --- | --- |
| body | `1159×748`（横向溢出） | `800×600`（无溢出） |
| 编辑中栏 | 631px | 800px 全宽 |
| 顶层控件 | 109 | 54 |
| 专业命令组 | 无 | 7 组 |
| 编辑工作区 | 376px / scrollH 878 | 488px / scrollH 488 |
| 时间轴可用高度 | 首屏不可见 | 131px，首屏可见 |
| 交付工作区 | 消息/媒体架挤压 | 488px / scrollH 488 |

## 五、真实 e2e

- 简易 → 专业 → 交付模式切换：通过
- 专业模式出现 7 个命令组：通过
- 暖调按钮产生 revision：通过；撤销可恢复
- 交付模式：预设、QA、渲染按钮可见；工作区 `488/488` 无裁切
- 深/浅主题切换后：body 仍为 `800×600`，工作区 `488/488`
- 原有时间轴功能：拖拽插入、PIP、重构、颜色关键帧、速度曲线、命令日志未被破坏

截图：

- `.extension-state/bridge-e2e-full/ui-review-easy-067.png`
- `.extension-state/bridge-e2e-full/ui-review-pro-067.png`
- `.extension-state/bridge-e2e-full/ui-review-deliver-067.png`
- `.extension-state/bridge-e2e-full/ui-review-light-067.png`

## 六、仍可继续提升

1. ≥1200px 宽屏下增加可停靠的片段 Inspector，而不是只靠命令带。
2. 媒体架增加真实缩略图与代理状态。
3. 轨道区增加波形/字幕 cue 可视化。
4. 增加面板宽度拖拽与时间轴高度拖拽。
5. 100+ 片段时对 clip 渲染做窗口化。

本轮完成的是从“功能堆叠”到“可用的专业剪辑工作台”的结构升级。

下一阶段完整任务与开源技术决策见：`docs/editing-ui-maturity-and-oss-review.md`。

## 七、0.69.8 真实页面复验与修复（2026-10-20 晚）

复验对象为 `com.himind.media-workbench@0.69.8`，18082 真实 WebView2 页面，项目使用
`media-library-100`（100 个真实可读产物）与 `hefei-artificial-sun`（代码项目确定性样本）。

### 7.1 真实页面暴露的问题

| 问题 | 现象 | 根因 |
| --- | --- | --- |
| 宽屏标题栏被压扁 | 1440 宽下标题栏只有 700px，`EDIT WORKSPACE` 与模式切换竖排 | 面板 padding 施加在 `.timeline-editor` 根节点，连标题栏一起缩进 |
| 命令带文字被压成竖排 | “目标时长”“段素材”等被挤成单字换行 | 命令带子项允许 shrink，700px 内容宽度下被压到最小 |
| 媒体库无时长 | 条目只显示“视频 · 6.8 MB” | 只有缩略图内部探测 duration，元信息未输出 |
| 缺少转场手柄 | 转场只能进 Inspector 点按钮 | Timeline clip 只有左右 trim handle |
| OTIO/EDL `format` 失效 | schema 声明 `format`，调用返回 unknown property 或忽略 both | 后端读取 `in.TypeFilter`（json:type），与 schema 不一致 |
| 窄屏双抽屉挤压 | 800px 下媒体库与 Inspector 同时打开，只剩 60px 画面 | 面板开关彼此独立 |

### 7.2 修复内容

- 标题栏改为全宽：`--editor-pad-left/right` 只作用于内容区，标题栏用负 margin 抵消面板 padding；
- 标题栏/命令带禁止 shrink，单行横向滚动，不再换行压字；
- 媒体库条目新增时长探测（视频/音频），显示为 `视频 · 1.0s · 2.2 KB`；
- clip 左右上角新增转场手柄，可鼠标拖拽设置 `transition_in/out` 时长，并带斜纹时长带反馈；
- 交换能力新增 `Format string` 字段并改用 `in.Format`，`otio/edl/both` 全部走真实 schema；
- <1200px 时侧栏互斥：打开媒体/属性/文稿/历史/市场/AI 找中的任一个会关闭其余；
- 标题栏在窄屏可横向滚动（800px 下 clientWidth 800 / scrollWidth 1176），不再静默裁切。

### 7.3 复验指标

| 验收 | 实测 |
| --- | --- |
| 1440×900 三栏 | 标题栏 1440×52；媒体 360；中栏 700；Inspector 380；`scrollWidth/Height = 1440/900` |
| 800×600 窄屏 | `scrollWidth/Height = 800/600`；抽屉互斥；标题栏可横向滚动 |
| 面板拖拽 | 媒体 360→320→360，刷新后仍为 360 |
| 媒体库 100 项 | 打开并渲染 100 条耗时 1ms；搜索 `e2e-media-099` 得 1/100；图片筛选 25/100 |
| 选中片段 | Inspector 4ms 内出现，6 个 tab 完整 |
| Inspector 编辑 | scale=1.2 提交后 revision 2，保存后 `timeline.json` 持久化 transform.scale=1.2 |
| 播放头联动 | 拖动到 f225，节目监视器 23ms 内更新为 `00:00:07:15 f225` |
| 播放/暂停 | 1.2s 前进 36 帧，暂停后时间码稳定 |
| 鼠标剪辑 | move→r4；trim 150f→123f→r5；blade 3→4 clips→r6；ripple delete 4→3 且后段左移→r7；转场手柄 6f→23f→r3 |
| OTIO/EDL | 真实能力调用 `format=both`，两种格式 import 后 3 clips、revision 7、scale 1.2、transition 23f 全部保留 |
| 确定性 | `hefei-artificial-sun` 3 个采样帧双渲染 SHA 全部一致，耗时 9s，valid=true |

### 7.4 证据

- 宽屏：`.himind/long-tasks/20261020-media-workbench-ui-maturity/evidence/wide-1440x900.png`
- 窄屏：`.himind/long-tasks/20261020-media-workbench-ui-maturity/evidence/narrow-800x600.png`
- 制品：`dist/plugins/com.himind.media-workbench-0.69.8.hmpkg`
- SHA-256：`f0445128a229350338451046a7c162b7296b9a9446474b11a640c1877af68b31`

### 7.5 仍待专项验证

1. 1000+ clip 的滚动/窗口化压力；
2. 多选批量、AI diff 局部应用的本轮 e2e；
3. WebAV 预览 POC（OSS-02）与 MLT no-go 复评（OSS-03）；
4. 命令带横向滚动的可发现性（滚动渐隐/更多菜单）。

## 八、时间线 / 轨道专项审查（0.70.4）

- 用真实 5 轨项目审查：V1/V2/A1/A2/S1，8 clips；
- 轨道头改为 128px 可操作结构：色条、名称、类型/角色、M/L 按钮；
- 支持添加视频/音频/字幕轨、删除空轨道、右键轨道操作；
- 视频轨 Muted 同时影响节目监视器与渲染主轨选择；
- 修复跨项目播放头串帧（9030 帧 → 600 帧仍显示 f5000）；
- 1440×900 五轨首屏完整（scroll 312/312）；
- 800×600 低高度进入紧凑模式，时间线保底 176px；
- 综合评分由约 57 提升到 75；
- 详细报告：`docs/timeline-track-maturity-review-2026-10-20.md`；
- 截图：`timeline-tracks-final-1440.png`、`timeline-tracks-final-800.png`。
