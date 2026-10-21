# 镜头能力架构：程序化与 AI 生成共用同一套制作契约

> 核心判断：镜头是契约，不是渲染器。一个分镜可以由程序化手绘、Remotion 组件、AI 图片、AI 视频、素材库视频等不同方式生产；创作台负责规划、分配、追溯和组装，具体生成能力由 HiMind Agent 的扩展提供。

## 一、为什么需要这一层

同一条视频里，不同镜头适合不同生产方式：

- 光谱、地图、路线、排行榜：程序化 Canvas / WebGL / Remotion，确定性高、成本低；
- 手绘科普、白板讲解：程序化手绘线条，可控且可复现；
- 角色特写、真实场景、复杂动作：AI 图片 / AI 视频；
- 空镜、环境、转场：素材库视频；
- 占位和草稿：占位图 / 纯色卡。

如果把这些都做成不同的 Style Pack，会产生大量重复项目结构和渲染分支。正确做法是：

```text
分镜（Shot）
  ├─ provider：谁生产
  ├─ technique：怎么生产
  ├─ prompt：生成提示
  ├─ reference_assets：参考资产
  ├─ asset_refs：产出资产
  └─ status：计划 / 就绪 / 已生成 / 已批准 / 失败
```

创作台只认这套契约，不关心底层是 Canvas、Remotion、Stable Diffusion 还是视频模型。

## 二、Provider 能力族

### 内置 Provider

| ID | 类型 | 用途 | 成本 | 延迟 |
| --- | --- | --- | --- | --- |
| `programmatic.browser-code` | programmatic | Canvas/WebGL 手绘、图表、地图、白板 | free | instant |
| `programmatic.remotion` | programmatic | React/Remotion 排版、动效、数据动画 | free | instant |

### Agent 内置媒体能力

`ai.image` / `ai.video` 不需要新插件，底座已经提供：

- `media.image.generate` / `media.image.edit`
- `media.video.generate`（异步）
- `media.job.get` / `media.job.wait` / `media.job.cancel`

它们通过本机 AI 服务的 `image.generate` / `video.generate` 槽位工作。槽位未配置时，provider 仍然可用，但生成前需要先在 AI 服务里配置图片/视频模型。

### External Pack Provider

| ID | 类型 | 所需扩展 |
| --- | --- | --- |
| `stock.video` | stock | `com.himind.media-stock-pack` |

External Pack 只实现“生成一个镜头资产”的契约，不接管项目、剧本、分镜和时间线。

## 三、能力接口

### `media.workbench.shot.providers`

列出所有镜头生产方式、类型、成本、延迟、是否可用、需要哪个扩展。

### `media.workbench.shot.assign`

为某个分镜指定：

```json
{
  "shot_id": "shot3",
  "provider": "stock.video",
  "technique": "stock",
  "prompt": "a cinematic close-up of the character...",
  "fallback_provider": "ai.image",
  "reference_assets": ["asset_char_01_front"]
}
```

写入 CreativePackage 的 `storyboard.shots[]`，产生新 revision 并清除批准。

### `media.workbench.timeline.plan`

把分镜编译成镜头制作计划：

```json
{
  "state": "blocked",
  "shots": [
    {
      "id": "shot3",
      "provider": "ai.video",
      "provider_name": "AI 视频",
      "available": false,
      "prompt": "...",
      "duration_seconds": 4,
      "status": "planned"
    }
  ],
  "blockers": [
    {
      "code": "SHOT_PROVIDER_UNAVAILABLE",
      "message": "shot3 的生成方式 素材库视频 尚未安装",
      "remediation": "安装 com.himind.media-stock-pack 后重新生成计划"
    }
  ]
}
```

检查项：

- provider 是否存在；
- External Pack 是否已安装；
- AI 图片/视频镜头是否有 prompt；
- shot 时长是否有效；
- beat_id 是否存在；
- 镜头总时长与制作规格是否一致。

### 后续接口

- `media.workbench.shot.produce`：调用对应 External Pack 生成单个镜头资产；
- `media.workbench.shot.status`：查询生成任务；
- `media.workbench.timeline.assemble`：把已生成镜头、配音、音乐、字幕组装成 Remotion 时间线；
- `media.workbench.timeline.render`：最终渲染。

## 四、回退策略

每个镜头可以声明 `fallback_provider`：

```text
ai.video 失败
→ ai.image + Ken Burns
→ programmatic.browser-code 占位
```

创作台在制作计划里展示回退链，避免“AI 视频失败就整条片子卡住”。

## 五、渲染与组装

- 程序化镜头：直接由 Browser-Code 或 Remotion 渲染为片段；
- AI 图片镜头：生成图片 → Ken Burns / 轻微推拉 → 片段；
- AI 视频镜头：生成视频 → 裁剪 / 转场 → 片段；
- 素材库镜头：检索 → 裁剪 → 片段；
- 所有片段统一进入 Remotion 时间线，叠加配音、音乐、字幕、转场。

这样 Remotion 仍然是**时间线组装器**，而不是被迫承担所有镜头的生成方式。

## 六、UI 承载

### 分镜编辑器

每个 shot 卡片增加：

- 生成方式：程序化 / AI 图片 / AI 视频 / 素材库；
- 技法：hand-drawn / cinematic / whiteboard / 3D；
- 生成提示词；
- 回退生成方式；
- 状态：planned / ready / generated / approved / failed；
- 需要安装的 External Pack 提示。

### 逻辑检查

新增“镜头能力”卡：

```text
程序化 4
AI 图片 1
AI 视频 1
缺 prompt 0
需安装扩展 1
```

### 制作计划

右侧新增“计划”页签：

- 镜头列表；
- provider / 技法 / 时长 / 状态；
- blockers 与 warnings；
- 回退链；
- 后续“生成镜头”与“组装时间线”入口。

## 七、分阶段落地

### P0：契约与规划（本轮已开始）

- shot.providers；
- shot.assign；
- timeline.plan；
- 分镜生成方式与 prompt 编辑；
- 镜头能力覆盖与 blockers。

### P1：程序化镜头生产

- shot.produce 对 `programmatic.browser-code` / `programmatic.remotion` 生效；
- 生成片段资产并写入 asset_refs；
- 多镜头拼接为 Remotion 时间线。

### P2：AI 图片 Provider（绑定底座 media.image.generate）

- Image Gen External Pack；
- 角色参考图与 prompt；
- 图片 → Ken Burns / 图生视频；
- 角色一致性检查。

### P3：AI 视频 Provider（绑定底座 media.video.generate）

- Video Gen External Pack；
- 图生视频、镜头延长、首尾帧；
- 回退到 AI 图片；
- 成本与超时控制。

### P4：音频与字幕

- TTS / 音乐 / 音效 External Pack；
- 字幕时间戳；
- 多轨混音；
- 唇形同步。

## 八、边界与风险

- 创作台不直接调用模型，只做规划、分配、状态与组装；
- External Pack 必须有稳定输入/输出契约，不能用 UI 私有协议；
- 生成任务必须异步 Job，支持取消、超时、重试、成本上限；
- 资产必须记录来源、许可证、哈希、模型版本、prompt、seed；
- AI 生成结果不确定，必须保留 fallback 和人工替换入口；
- 程序化镜头仍然是一等公民，不因为 AI 能力强就放弃确定性。

## 九、判断标准

一个镜头能力是否应该进入底座，看三点：

1. 是否是多个视频类型共用的能力；
2. 是否能通过稳定输入/输出契约表达；
3. 是否能被独立发版、授权、替换。

满足这三点，就做成 External Pack / Asset Pack；不满足，就留在创作台内部。


## 十、列表规模化

数百个镜头不能继续用“每段一张大卡片”的方式维护。当前分镜区已改为表格式管理：

- 搜索：按镜头 ID、画面、屏幕文字、prompt、技法搜索；
- 筛选：按生成方式、状态筛选；
- 分页：25 / 50 / 100 条每页，避免一次渲染数百张卡片；
- 批量选择：当前页全选、跨页保留选择；
- 批量操作：批量设置生成方式、批量设置状态、批量删除；
- 行内编辑：直接改生成方式、状态和时长；
- 单镜头编辑：点击“编辑”打开镜头详情弹窗，支持上一个/下一个连续审片；
- 详情弹窗包含：镜头 ID、类型、beat、时长、生成方式、技法、prompt、负面 prompt、回退方式、参考资产、画面、屏幕文字、删除。

这套结构先保证“数百段可维护”，后续再补：

- 按 scene / beat 分组折叠；
- 虚拟滚动（超过 1000 段时）；
- 批量生成与批量状态刷新；
- 时间线视图与镜头缩略图。
