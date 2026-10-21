# 通用剪辑能力缺口与补齐方案

> 日期：2026-10-19
> 结论：**缺少通用片段剪辑能力。** 当前系统强在“从 Plan/分镜生成”，弱在“对已有片段做修剪、拼接、变速、叠加与重排”。这不是分镜能替代的。

## 一、现状

已有：

- `storyboard`：镜头级编排（shots + duration + 顺序 + 批量时长 + 锁定）；
- `media.workbench.timeline.plan`：把创作包编译成“镜头制作计划”，不是剪辑时间轴；
- `audio.*`：TTS、BGM、混音、对齐、字幕；
- `visual.*`：QA、切点、联系表、放大；
- 渲染：browser-code / Remotion 合成。

缺少：

- 片段级 trim / split / concat / reorder；
- 变速与倒放；
- 裁剪、缩放、位移（竖屏重构）；
- 转场、叠加、画中画；
- 音轨提取/替换、自动 ducking；
- 多轨时间轴与撤销/重做；
- 以时间轴为输入的确定性渲染。

## 二、判断：需要补，但不要做“完整 NLE”

目标不是 Premiere 式自由剪辑，而是：

> **AI 可生成、可解释、可回退的程序化剪辑图（Edit Graph / Timeline JSON）。**

创作台负责时间轴模型与交互；剪辑操作是可复用的确定性 Capability；渲染器消费时间轴。

## 三、目标数据模型

```json
{
  "schema": "himind.timeline.v1",
  "duration": 72,
  "fps": 30,
  "canvas": {"width": 1080, "height": 1920},
  "tracks": [
    {"id": "v1", "kind": "video", "clips": [
      {"id": "c1", "src": "artifacts/a.mp4", "in": 2.0, "out": 8.5, "start": 0, "speed": 1.0,
       "transform": {"scale": 1.0, "x": 0, "y": 0, "crop": null},
       "transition_in": {"type": "fade", "duration": 0.3}}
    ]},
    {"id": "a1", "kind": "audio", "clips": []},
    {"id": "s1", "kind": "subtitle", "clips": []}
  ]
}
```

约束：

- `storyboard` 是创作意图；`timeline` 是制作结果，由 storyboard 派生，可人工编辑；
- 不允许两套真值并行：时间轴修改必须可回写为 shot 级覆盖或独立 revision；
- 所有操作幂等、可重放、可 diff。

## 四、能力拆分

### 通用、可复用 → 新插件 `com.himind.media-edit`

| 能力 | 说明 | 优先级 |
| --- | --- | --- |
| `clip.trim` | 入点/出点裁剪 | P0 |
| `clip.split` | 按时间切分 | P0 |
| `clip.concat` | 多片段拼接（含 gap/黑场） | P0 |
| `clip.retime` | 变速/倒放/定格 | P0 |
| `clip.transform` | 裁剪、缩放、位移、旋转（竖屏重构） | P0 |
| `clip.transition` | 淡入淡出、叠化、擦除 | P1 |
| `clip.overlay` | 画中画、贴纸、字幕烧录 | P1 |
| `clip.audio` | 提取/替换/静音/增益 | P1 |
| `clip.color` | 基础校正、LUT | P2 |
| `timeline.render` | 消费 Timeline JSON 输出成片 | P0 |

### 应用层 → Workbench

- Timeline 模型、轨道、片段、吸附、分割、波纹删除；
- 撤销/重做、选择驱动 Inspector；
- 时间轴与分镜双向联动（选中镜头 → 定位；拖动 → 更新 shot 覆盖）。

### Renderer

- 消费 Timeline JSON；
- 简单 cut 走 ffmpeg graph，复杂合成走 Remotion/browser-code；
- 输出母版 + web copy + QA。

## 五、实施阶段

### Phase 1（P0，先打通）

1. `himind.timeline.v1` schema + validate；
2. `clip.trim` / `clip.split` / `clip.concat` / `clip.retime`；
3. `timeline.render`（ffmpeg，先支持单视频轨 + 音频轨）；
4. 创作台时间轴只读预览 + 分割/删除/重排；
5. 真实样例：60s 素材剪成 30s，3 段拼接。

### Phase 2（P1）

6. `clip.transform` 竖屏重构；
7. 转场、画中画、字幕烧录；
8. 音频提取/替换与 ducking；
9. 时间轴交互（吸附、波纹删除、撤销/重做）。

### Phase 3（P2）

10. 调色/LUT；
11. 关键帧与速度曲线；
12. 多机位与跟踪。

## 六、验收场景

- 访谈 60s → 30s 快剪（trim + split + concat）；
- 横屏素材 → 9:16 竖屏（transform + 安全区）；
- 3 段 B-roll 按节拍拼接（retime + concat）；
- 字幕烧录 + 旁白/BGM 混音（overlay + audio）；
- 每次操作可撤销、可 diff、可重渲染。

## 七、风险

- **双真值**：storyboard 与 timeline 必须明确主从关系；
- **音画同步**：变速/裁剪后字幕与旁白需重新对齐；
- **确定性**：同一 timeline 重复渲染哈希一致；
- **性能**：长素材需要代理文件与分段渲染；
- **权限**：clip 操作写工作区，需 workspace 边界与审批。
