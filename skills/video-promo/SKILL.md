---
name: video-promo
description: 把产品名、网址、仓库或一句话需求做成真实产品宣传片：调研、风格样帧、CreativePackage、promo-video 模板、旁白字幕、质量门禁、渲染与交付。
---

# 产品宣传片导演

## 目标

让没听过产品的人，看完就想去用。画面必须有真实产品素材，叙事必须能让外行一遍看懂，交付必须是一支真实 MP4，不是计划或预览截图。

## 必须先做

1. 调用 `media.workbench.intent.plan`。`route=promo` 时按本 Skill 执行；不是 promo 时交给对应 Skill。
2. 读取 `media.workbench.catalog.list`、`media.workbench.template.describe`（`promo-video`）与 `media.workbench.style.describe`（`promo-cinema`）。
3. 没有产品名/网址/仓库时先问；没有真实产品素材时先问用户要，或使用公开页面/仓库素材。**禁止编造产品事实、数字、截图或 CTA 链接。**

## 人工卡点

只有三类：

- **需求确认**：产品定位、受众、时长、画幅、平台、声音方案、真实素材来源、版权边界、CTA。
- **风格样帧**：给 3–4 个差异明显的风格方向，每个出 1 张 16:9 样帧；用户选一个后才进入生产。
- **试听拍板**：旁白音色与 BGM 试听后确认；用户说“你决定”可跳过。

## 十阶段流程

### 1. 预调研（10 分钟内）
确认产品是否存在、官网/仓库/文档、目标用户、核心卖点、可证明数字、品牌色/logo、已有截图。找不到官方来源就停下来问定位。

### 2. 需求问询
按 `references/需求问询.md`。至少问：产品名/网址、受众、时长、画幅、风格方向、叙事结构、开场钩子、真实素材来源、声音方案、版权边界、CTA、导出规格。

### 3. 建 CreativePackage
调用 `media.workbench.creative.create`，随后 `creative.update` 分区写入 brief/research/script/storyboard/production。不要直接创建可渲染项目。

### 4. 深度调研与真实素材
按 `references/素材获取.md`（或本 Skill 内置方法）：
- 事实逐条带出处；
- 真实产品界面截图优先；
- 图片检查尺寸/清晰度；
- 每张素材登记来源与版权边界；
- 不用二手演示图冒充产品截图。

### 5. 风格样帧
按 `references/风格方向.md` 给 3–4 个方向；每个方向生成样帧；用户确认后把选择写入 `production`。

### 6. 脚本与分镜
`script.beats[].narration` 是旁白与字幕的唯一文本源；`on_screen_text` 只用于画面标题/关键词。每个 beat 给出镜头类型、时长、素材、转场。

### 7. 校验与批准
调用 `media.workbench.creative.validate`；修复 blockers；用户明确批准当前 revision 后调用 `media.workbench.creative.approve`。未批准不得 materialize。

### 8. 落成项目
调用 `media.workbench.creative.materialize`，模板使用 `promo-video`，风格使用 `promo-cinema`。确认项目产物、Plan 与 Recipe 已生成。

### 9. 预览、音频、质检
调用 `preview.start` 并轮询 `preview.status`；生成旁白/字幕/BGM（优先现有 audio 插件；无 TTS 时用大字幕 + 可商用 BGM）；调用 `quality.check`，修复文字安全区、响度、黑帧、静帧、缺失素材等问题。

### 10. 渲染与交付
调用 `render.start`（审批后）并轮询 `render.status` 到 `completed`；用 `artifact.export` 返回真实 MP4。交付至少包含：

```text
宣传片.mp4
需求确认.md
调研笔记.md
项目/工程
可选的网页版
```

用户反馈时记录原话，按“只换音轨/只换某场景/只换风格”决定最小重做范围，生成新 revision，不覆盖旧版。

## 质量底线

- 开场 3 秒内给出痛点或核心亮点，5 秒内第一个高能点。
- 每屏只讲一件事；同一版式最多连续用 2 次。
- 带名字的重点展示镜头至少 1.3 秒；CTA 至少 3 秒。
- 真实产品界面优先于装饰图；AI 图片必须标注“示意图”。
- 不宣称未实现的能力；不编造数字/URL/客户/奖项。
- 旁白与字幕同源，避免 TTS 与字幕漂移。
- 渲染前必须通过素材检查、文字安全区、音频响度、成片规格与确定性检查。

## 参考

- `references/需求问询.md`
- `references/风格方向.md`
- `references/校验清单.md`
- `references/踩坑记录.md`
- `references/案例复盘.md`

本 Skill 的流程参考 `liyupi/ai-promo-video-maker`（MIT），并按 HiMind 创作台能力重写。
