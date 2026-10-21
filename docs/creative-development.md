# 创意开发与 CreativePackage

> 状态：P0 已落地（创作包、校验、审批、materialize、创意台）
> 首个场景：`science-sketch` ·《天空为什么是蓝色的？》

## 一、为什么要多一层创意开发

原来的主链路是：

```text
Plan → Recipe → Project
```

它适合已经明确“做什么”的情况，但不适合从主题开始持续打磨：

```text
主题
→ Brief
→ 研究/证据
→ 脚本
→ 分镜
→ 制作规格
→ 用户批准
→ 项目
```

因此引入 `CreativePackage`：

```text
CreativePackage
  ├─ brief
  ├─ research
  ├─ script
  ├─ storyboard
  ├─ production
  ├─ approval
  └─ links
```

它位于 `Plan` 和 `Recipe` 之上，是创意开发阶段的唯一事实源。

## 二、职责边界

| 层 | 负责 |
| --- | --- |
| CreativePackage | 想清楚、写出来、画出来 |
| Plan / Recipe | 确定性执行契约 |
| Project | 项目状态、预览、反馈、渲染和产物 |
| Skill | AI 判断与创作方法 |
| Workflow | 多步骤、审批与长任务编排 |

创意开发阶段不渲染，不创建可渲染项目，不直接写入 Recipe。

## 三、状态机

```text
draft
  ↓
approved
  ↓
materialized
```

- 修改任意 section 会产生新 revision，并清除批准；
- `creative.approve` 只批准通过校验的 revision；
- `creative.materialize` 只接受已批准且 revision 匹配的创作包；
- 项目记录 `source_creative_id` 与 `source_creative_revision`，之后创意更新不会自动改变已有项目。

## 四、能力

```text
media.workbench.creative.create
media.workbench.creative.list
media.workbench.creative.get
media.workbench.creative.update
media.workbench.creative.validate
media.workbench.creative.approve
media.workbench.creative.materialize
```

`creative.update` 按 section 更新，避免为每个 section 增加一个能力。

## 五、存储

```text
.himind-media/
├─ creatives/
│  └─ <creative_id>/
│     ├─ creative.json
│     ├─ brief.md
│     ├─ script.md
│     ├─ storyboard.json
│     └─ revisions/
│        ├─ r0001.json
│        ├─ r0002.json
│        └─ ...
└─ projects/
```

## 六、首个场景：天空为什么是蓝色的？

```text
主题：瑞利散射
时长：45 秒
画幅：1080×1920
形式：手绘线条 + 淡彩 + 图解箭头
声音：第一版不做 TTS，以字幕和图形为主
```

Script beats：

```text
hook       天空为什么是蓝色的？
sunlight   阳光包含多种颜色
scattering 空气分子更容易散射蓝光
observer   蓝光从天空各处进入眼睛
sunset     日落路径更长，蓝光先被散掉
takeaway   蓝色来自散射，不是海洋反射
```

Storyboard：

```text
title → spectrum → scatter → observer → sunset → takeaway
```

## 七、验收

```text
creative.create
→ research/script/storyboard/production update
→ creative.validate valid=true
→ creative.approve
→ creative.materialize
→ Preview
→ Determinism
→ Render
→ Visual QA
```

项目必须保留来源创意版本，成片必须是真实 MP4。

## 八、首个场景的质量复盘（science-sketch 0.1.1）

首版 45 秒成片通过了帧数、黑帧、文字边界、安全区与确定性检查，但自动 QA 报了：

```text
freeze: warning — 9 个 ≥1.5s 静帧区间
```

原因不是编码，而是入场动画结束后整屏不再变化。0.1.1 用三层低成本动效修复：

- 手绘线条以 7.5fps 做 Rough.js seed boil；
- 纸张颗粒与光晕做低频漂移；
- 太阳、分子、光谱射线和观察者箭头做呼吸式微动。

同时把科学表述补强为：

- 瑞利散射强度近似与 `1/λ⁴` 成正比；
- 紫光散射更强，但人眼对蓝更敏感，且太阳光谱中紫光较少；
- 日落路径更长，蓝光被散掉更多。

来源固定为：

- NASA Space Place, *Why Is the Sky Blue?* — https://spaceplace.nasa.gov/blue-sky/en/
- NOAA JetStream, *The Atmosphere* — https://www.noaa.gov/jetstream/atmosphere

完整踩坑记录见 [pitfalls/static-frames.md](pitfalls/static-frames.md)。这条经验同时进入 `catalog/constraints.json` 的 `MOTION_CONTINUITY`。
