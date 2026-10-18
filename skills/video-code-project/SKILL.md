---
name: video-code-project
description: 用 AI 写代码制作风格型短视频（动态字体、手绘动画、p5/three.js 等），并登记到 HiMind 自媒体创作台，生成审查图、记录反馈、同步源码、最终渲染。
---

# 代码型视频项目

## 什么时候用

适合：

- 动态字体 / 作品集 Reel
- 手绘水彩 / 水墨动画
- p5.js / three.js / Canvas / WebGL 风格片
- 需要 AI 直接写镜头代码、逐镜头审查的作品

不适合：

- 排行榜、地图、路线、桑基图等数据型视频
- 需要严格数据契约与批量复现的内容

数据型视频优先使用 `video-director` + `generate-video` 的模板链路。

## 核心原则

代码型项目不是“绕过创作台”，而是创作台的第二种生产模式：

```text
AI 写代码
→ project.import
→ preview.review
→ 创作台审查
→ feedback.record
→ AI 读取 project.context
→ 修改代码
→ project.sync
→ preview.review
→ quality.check
→ render.start
```

创作台负责项目、审查图、反馈、产物与质量门禁；AI 客户端负责写代码和改代码。

## 固定渲染契约

Style Pack 必须包含：

```text
style-pack.json
render.mjs
STORYBOARD.md
```

`render.mjs` 只接受固定 CLI：

```bash
node render.mjs --sheet=1,3,5,7,9 --cols=3 --w=540 --out=out/check/sheet.jpg
node render.mjs --strip=2:3 --cols=6 --w=320 --out=out/check/strip.jpg
node render.mjs --stills=2,5,8 --out=out/stills
node render.mjs --frames --workers=1 --duration=10
node render.mjs --encode --out=out/video.mp4
```

不得让创作台执行任意 shell；只能执行上述固定子命令。

`style-pack.json` 示例：

```json
{
  "schema": "himind.video.style-pack.v1",
  "id": "kinetic-mini",
  "version": "0.1.0",
  "renderer": "browser-code",
  "entry": "render.mjs",
  "video_out": "out/video.mp4",
  "duration_seconds": 10,
  "preview": {
    "sheet_out": "out/check/sheet.jpg",
    "strip_out": "out/check/strip.jpg",
    "stills_out": "out/stills"
  }
}
```

## 步骤

### 1. 选择或创建 Style Pack

优先从 Asset Pack 风格库选择：

```text
media.workbench.asset.list
  workspace_root
```

使用 `stylepack.scaffold` 可直接复制模板并登记项目：

```text
media.workbench.stylepack.scaffold
  pack_id
  name
```

已有 Style Pack：

- `examples/code-project-demo/`：最小 Canvas 2D 样例
- `style-packs/kinetic-reel/`：真实动态字体 Reel，复制其 `template/` 作为项目源码
- `style-packs/painted-animation/`：p5.brush 手绘动画，复制其 `template/` 作为项目源码


优先复用已有 Style Pack。  
如果没有合适的，参考 `examples/code-project-demo/` 创建：

```text
style-pack.json
render.mjs
package.json
STORYBOARD.md
reel/index.html
reel/reel.js
```

### 2. 复制到工作区

把 Style Pack 复制到用户工作区，例如：

```text
<workspace_root>/my-reel/
```

不要直接修改扩展源里的 Style Pack。

### 3. 先写分镜

`STORYBOARD.md` 必须写清楚：

- 时间预算
- 每个镜头观众要读到什么
- 真实数据来源
- 哪些内容是 `SCHEMATIC` 示意

### 3.5 校验 Style Pack

```text
media.workbench.stylepack.validate
  source_dir
```

只有 valid=true 才继续。

### 3.8 注入数据（数据驱动 Style Pack）

如果 Style Pack 读取 `input/data.json`，先把 `data.ingest` / `data.bind` 的结果写入：

```text
media.workbench.project.data.write
  project_id
  data
  brief
```

数据写入后项目会变成 `preview_status=stale`，必须重新预览。

### 4. 登记项目

```text
media.workbench.project.import
  workspace_root
  name
  source_dir
```

### 5. 生成审查图

```text
media.workbench.preview.review
  project_id
  kind=sheet
```

必须看渲染结果，不要只读代码。  
需要检查动作时用 `kind=strip`，检查细节时用 `kind=stills`。

### 6. 反馈与继续调优

用户在创作台记录反馈后，AI 调用：

```text
media.workbench.project.context
```

读取：

- 最新反馈
- 最新审查产物
- 源码是否与预览同步
- 下一步建议

然后修改代码。处理完成后调用：

```text
media.workbench.feedback.resolve
  project_id
  feedback_id
  status=resolved
```

`project.context.pending_feedback` 为空才表示反馈闭环完成。

### 7. 同步源码

修改代码后调用：

```text
media.workbench.project.sync
```

如果源码 hash 变化，创作台会把预览标记为 `stale`。  
必须重新 `preview.review`，旧审查图不能当作最新结果。

### 7.5 确定性检查

```text
media.workbench.preview.determinism
  project_id
  times=1,5,9
```

两次静帧哈希必须一致；不一致时不要最终渲染。

### 8. 质量门禁

```text
media.workbench.quality.check
```

代码项目至少检查：

- `source_dir` 存在
- `entry` 存在
- `STORYBOARD.md` 存在
- 源码与预览同步
- 至少一张审查图
- 成片可选

### 9. 最终渲染

```text
media.workbench.render.start
```

渲染需要审批。  
完成后用 `artifact.open` 打开 MP4，或在创作台查看产物。

渲染完成后插件会自动执行 `media.workbench.quality.visual` 并把结果写入作业的 `quality_status` / `quality_report`；仍可手动调用复核（`project_id` 或 `video_path`）。黑帧/静帧/严重色带/音频响度失败时不得交付。`artifact.export` 会拒绝 `quality_status=failed|error` 的母版；只有调试时才可显式传 `allow_failed=true`。

## 渲染坑：暗部色带 / 连片色块

- 最终帧默认使用 PNG；不要在暗部渐变、vignette、星空、粒子、3D 材质风格中使用 JPEG 中间帧。
- JPEG 即使 `quality=0.94`，也会把 20+ 级暗部渐变压成 2 级大色块；H.264 只会原样保留这个错误。
- 快速测试可显式传 `--frame-format=jpg`，但最终渲染必须回到 PNG。
- 有 vignette / CSS contrast / 大面积暗部渐变时：
  1. 先渲染一张 PNG still；
  2. 优先调用 `media.workbench.quality.banding`（`image_path` 或 `video_path` + `time`）；本地开发可运行 `dev/verify-banding.mjs --image=<still>`；
  3. 最大平坦段超过画面宽度 8%，或亮度级数少于 12，判为不通过；
  4. 修复顺序：移除不必要的 CSS contrast/vignette → 加 dither → 改用 PNG 帧 → 再提高 Web copy 码率。
- Web copy 是二次编码，仍可能出现轻微色带；审片以母版为准。
- 地图/线网标签优先调用 `media.workbench.label.place` / `media.workbench.label.validate`；用 8 方向候选位与障碍检查，并保持整段动画位置稳定。
- 已有项目是源码快照，不会自动继承 Style Pack 的 renderer 更新；调用 `media.workbench.project.upgrade-style` 原地升级 renderer，保留项目代码与数据。
- Style Pack 声明 `quality.text_audit` 后，`quality.visual` 会自动检查 `text_bounds`、`safe_area`、`text_overlap`、`title_hit`；未声明则明确返回 `skipped`。
- 文字审计示例：

```json
{
  "quality": {
    "text_audit": {
      "enabled": true,
      "page": "reel/index.html",
      "times": "1.5,5,6.5,8.5"
    }
  }
}
```

- 重型 Canvas（渐变、扫描线、噪点、多图层）先设 `runtime.max_workers=1`。不要把多个 1080×1920 页面塞进同一个 Chrome 硬提并发；VHS 的多页面截图曾出现“进程活着但帧不增长”。
- 平面/白底风格的大块纯色不是色带：Style Pack 设置 `quality.banding_gate=false`，`banding` 只记录指标、不阻断视觉 QA。


## 硬规则

- 每一帧必须是时间 `t` 的纯函数；
- 不使用无种子的 `Math.random()`；
- 不跨帧保存状态；
- 不在渲染时访问网络；
- 不执行任意 shell；
- 真实数据必须标来源；
- 示意内容必须标 `SCHEMATIC`；
- 没有审查图不得最终渲染；
- 质量 QA 失败的母版不得进入正式交付；
- 重型 Canvas Style Pack 必须声明并遵守 `runtime.max_workers`。

## 适配 opus-video-skills

`kinetic-reel` / `painted-animation` 可以转为 Style Pack：

- 保留其 `SKILL.md` 的方法论；
- 保留 `render.mjs` 的固定 CLI；
- 增加 `style-pack.json`；
- 把 `STORYBOARD.md`、`references/`、`examples/` 一起放入 Style Pack；
- 使用 MIT 许可并保留原作者版权声明。
## 网络策略

默认：

```json
{"mode": "none", "allow": []}
```

只允许 `127.0.0.1`（本地静态服务器）。

需要外部资源时：

```json
{"mode": "declared", "allow": ["fonts.googleapis.com"]}
```

`full` 只允许 `trust: "signed"` 的 Style Pack，不作为默认。  
外部素材应尽量下载一次并缓存到项目内，避免每次渲染都联网。






