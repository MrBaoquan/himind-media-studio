# 剪辑架构复核：方向正确性与补充设计

> 日期：2026-10-19
> 结论：**大方向正确，但不能直接按现方案开工。** 需要先修正 4 个架构点，并补齐 6 个相邻方向，否则剪辑会与创作、渲染、交付形成两套真值。

## 一、方向正确的部分

1. 以 Resolve Edit 页心智组织：媒体池、源/节目监视器、Inspector、多轨时间轴；
2. 以 Timeline/EDL 作为可解释、可回退的制作结果；
3. 剪辑操作为命令，而不是直接改像素；
4. AI 只产出命令序列，手工与 AI 共用撤销栈；
5. 不复制 Fusion/Color/Fairlight 的全部页面，只做短视频需要的子集。

这些判断成立，应保留。

## 二、必须修正的 4 个点

### 1. 模式不能只有“媒体 → 剪辑 → 交付”

创作台的核心差异化是 AI 创作，不是剪辑器。Brief、研究、脚本、分镜必须仍有位置。

建议模式：

```text
创作（Brief / 研究 / 脚本 / 分镜）
剪辑（媒体池 / 双监视器 / 时间轴）
交付（QA / 预设 / 导出）
```

媒体池是全局面板，在创作与剪辑都可展开，而不是一个顶层模式。

### 2. 时间必须用帧，而不是浮点秒

`source_in/out`、`start/duration`、转场时长全部用整数帧或有理时间；UI 再格式化为 `HH:MM:SS:FF`。

原因：浮点秒在多次 trim/roll/speed 后累积漂移，字幕与音频对不齐。

### 3. 剪辑命令与媒体渲染必须分层

当前方案把 `clip.trim` 与 `timeline.render` 混在一个插件里，容易形成第二套渲染器。

正确分层：

| 层 | 职责 | 例子 |
| --- | --- | --- |
| Timeline 命令 | 纯数据变换，不读写媒体 | insert / overwrite / blade / ripple / roll / slip / slide / link |
| 媒体操作 | 产出新文件 | proxy / transcode / extract-audio / burn-in |
| Renderer | 消费 Timeline，输出成片 | ffmpeg cut render / Remotion composite |

`clip.*` 不应同时承担“改时间轴”和“渲染成片”。

### 4. 分镜 ↔ 时间轴必须有明确契约

- Storyboard 是创作意图；Timeline 是制作结果；
- Storyboard 变更 → 编译出 Timeline diff，而不是静默覆盖；
- Timeline 手工修改 → 记录为 timeline revision，不回写分镜，除非显式“回写”；
- 锁定镜头在 rebase 时保留；
- 任何一次 rebase 都要能回答：哪些镜头保留、哪些被替换、哪些冲突。

## 三、还需要补充的 6 个方向

| 方向 | 为什么需要 | 优先级 |
| --- | --- | --- |
| 媒体池/代理/元数据 | 真实素材会有长视频、不同帧率、不同色彩空间；没有代理和元数据无法精剪 | P0 |
| 字幕/标题烧录 | 短视频交付的刚需；现有字幕轨只编辑，不参与渲染 | P1 |
| 音频总线/ducking/响度 | 旁白、BGM、SFX 需要总线与自动压低；现有 audio 能力未进入时间轴 | P1 |
| 交付预设/QC | 抖音/B站/YouTube 的画幅、码率、响度、字幕规范不同 | P1 |
| 审阅批注/版本 | 时间码批注、版本对比、批准记录 | P1 |
| AI 粗剪/转写编辑/语义检索 | 区别于传统 NLE 的核心：按文稿剪、按语义找片段、自动粗剪 | P1 |
| 色彩管理/LUT | 多来源素材一致性 | P2 |

## 四、修正后的分层

```text
创作层：Brief / Research / Script / Storyboard
        ↓ compile
Timeline 命令层：纯数据变换 + 撤销/重做 + diff
        ↓
媒体操作层：proxy / transcode / extract-audio / burn-in
        ↓
Renderer：ffmpeg cut render / Remotion composite
        ↓
交付层：QA / 预设 / 导出 / 版本
```

## 五、建议的首个里程碑（M0）

不做完整剪辑 UI，先做“可验证的剪辑内核”：

1. `himind.timeline.v1`：帧为单位、轨道、片段、链接、revision；
2. 命令集：insert / overwrite / blade / lift / ripple_delete；
3. 校验器：重叠、越界、空轨、链接不一致；
4. 一个最小 Renderer adapter：单视频轨 + 音频轨；
5. 真实素材：60s 剪成 30s，3 段拼接，重复渲染 SHA 一致。

M0 通过后再做时间轴 UI（E-11/E-12），避免 UI 先于语义。

## 六、对原计划的调整

- 原 E-10~E-13 保留，但 E-10 增加“帧时间模型”；
- E-03 的 `timeline.render` 移动到 Renderer，不放在 media-edit；
- 新增媒体池/代理、字幕烧录、交付预设、审阅批注、AI 粗剪等任务；
- 先内核、后 UI、再专业修剪。

## 七、最终判断

方向正确，但必须按“创作层 → 命令层 → 媒体层 → 渲染层 → 交付层”分层落地；
否则会出现：分镜与时间轴两套真值、剪辑插件与渲染器重复、浮点时间漂移、交付规范缺失。
