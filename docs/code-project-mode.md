# Browser-Code 项目模式

> 状态：已落地（MVP）  
> 日期：2026-10-17  
> 关联：`skills/video-code-project`、`examples/code-project-demo`、`media.workbench.project.*`、`media.workbench.preview.review`

## 一、为什么需要第二种模式

模板模式解决“数据型、可参数化、可批量”的视频：

```text
Plan → Recipe → Shot → Technique → Contract → Template → Fixture
```

但动态字体、手绘动画、p5/three.js 这类视频更像“AI 写了一个可继续修改的代码项目”，不是一组 Recipe 参数。  
把它们硬塞进 Shot 模型会失去作者的表达空间，也会让模板目录失去边界。

所以创作台采用双模式：

```text
项目
├─ mode=template  → Remotion + Recipe + Shot
└─ mode=code      → Browser-Code + Style Pack + render.mjs
```

两种模式共享：

- 项目列表与修订
- 预览产物
- 反馈
- 质量门禁
- 最终视频 Artifact
- Candidate（后续）

## 二、项目模型

代码项目的 `project.json`：

```json
{
  "id": "code-demo",
  "name": "code-demo",
  "mode": "code",
  "renderer": "browser-code",
  "style_pack": "kinetic-mini",
  "style_version": "0.1.0",
  "source_dir": "code-project-demo",
  "entry": "render.mjs",
  "source_hash": "sha256:...",
  "preview_status": "ready",
  "preview_hash": "sha256:...",
  "video_out": "out/video.mp4",
  "sheet_out": "out/check/sheet.jpg",
  "strip_out": "out/check/strip.jpg",
  "stills_out": "out/stills",
  "duration_seconds": 10,
  "revision": 4
}
```

- `source_dir` 必须在 `workspace_root` 内；
- `source_hash` 只覆盖源码与素材，排除 `out/`、`node_modules/`、`.cache/`、`package-lock.json`；
- 源码变化后 `preview_status=stale`；
- 旧预览不能当作最新结果。

## 三、Style Pack 契约

```text
style-pack.json
render.mjs
STORYBOARD.md
package.json
reel/ 或 src/
```

`style-pack.json`：

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

`render.mjs` 固定 CLI：

```bash
node render.mjs --sheet=1,3,5,7,9 --cols=3 --w=540 --out=out/check/sheet.jpg
node render.mjs --strip=2:3 --cols=6 --w=320 --out=out/check/strip.jpg
node render.mjs --stills=2,5,8 --out=out/stills
node render.mjs --frames --workers=1 --duration=10
node render.mjs --encode --out=out/video.mp4
```

插件只执行这些固定子命令，不接受 AI 临时拼接的任意 shell。

## 四、AI 对话到创作台的闭环

```text
用户描述需求
→ AI 选择 Style Pack
→ 复制模板到工作区
→ 写 STORYBOARD.md 与代码
→ project.import
→ preview.review(sheet)
→ 创作台显示审查图
→ 用户在创作台记录反馈
→ AI 调用 project.context 读取反馈
→ 修改代码
→ project.sync
→ preview.review
→ quality.check
→ render.start
→ 创作台显示 MP4
```

关键能力：

| 能力 | 作用 |
| --- | --- |
| `media.workbench.stylepack.validate` | 登记前校验 Style Pack 契约 |
| `media.workbench.project.import` | 登记代码项目 |
| `media.workbench.project.sync` | 检测源码变化，标记预览过期 |
| `media.workbench.project.context` | 给 AI 返回项目、分镜、反馈、产物、下一步 |
| `media.workbench.preview.review` | 生成 sheet / strip / stills |
| `media.workbench.quality.check` | 检查源码、分镜、预览同步 |
| `media.workbench.render.start` | 后台渲染 MP4 |

## 五、创作台 UI

代码项目详情：

```text
审阅 / 分镜
```

- **审阅**：关键帧总览、连续帧、静帧、审查产物列表、反馈输入
- **成片渲染**：先执行确定性检查，再启动 
ender.start；页面显示渲染进度，完成后出现可打开的成片入口
- **产物回读**：审阅页按当前 workspace_root 读取 Artifact，不会读取另一个工作区的成片
- **分镜**：只读展示 `STORYBOARD.md`
- **右栏**：Style Pack、源码目录、入口、修订、预览状态、产物数量、质量门禁
- **项目树**：代码项目带「代码」徽标

## 五点五、视觉 QA 与交付门禁

渲染完成后自动执行 `media.workbench.quality.visual`，结果写入 Job 与母版 Artifact。

- `quality_status=failed/error`：`artifact.export` 返回 `blocked`，母版不能进入正式交付；
- `quality_status=warning`：允许交付，但要在交付说明中记录 warning；
- `artifact.open` 仍可用于调试审片；
- Style Pack 声明 `quality.text_audit` 后会自动检查文字越界、安全区、重叠与标题命中；
- 重型 Canvas 使用 `runtime.max_workers=1`，避免单浏览器多页面截图停滞。

## 六、安全边界

- 不执行任意 JSX / 任意命令；
- 只执行 `node <entry>` 的固定子命令；
- 所有路径必须在 `workspace_root` 内；
- 首次渲染需要审批；
- 默认不访问网络；
- 默认单 worker；调用方可以请求更多 worker，但会被 Style Pack 的 `runtime.max_workers` 收敛；
- 缺少 Node / Chrome / ffmpeg 时返回结构化 blocker。

## 七、当前样例

`examples/code-project-demo/`：

- 1080×1920
- 10 秒 / 30fps
- Canvas 2D 动态字体
- 一镜到底能力叙事
- 支持 sheet / strip / stills / frames / encode

它证明了闭环，但不是最终视觉上限。

已接入真实 Style Pack：

- `style-packs/kinetic-reel/`：动态字体 Reel，MIT，来自 `tuzhechen2005/opus-video-skills`。
- `style-packs/painted-animation/`：p5.brush 手绘动画，MIT，来自同一来源。
- 17 秒 / 1920×1080 / 510 帧 / three.js + Canvas 2D + WebAudio 合成配乐。
- 已通过 `stylepack.validate`、`project.import`、`preview.review`、`quality.check`、`render.start`。
- 成片验收使用 `dev/verify-render.py --profile code`，9/9 PASS。

`painted-animation` 已验证：

- 11 秒 / 1920×1080 / 264 帧。
- 母版 13.38 MB；Web copy 1280×720 / 1.91 MB。
- 静帧确定性检查通过。
- `verify-render.py --profile code` 9/9 PASS。

## 八、边界

- 代码项目不会自动变成 Remotion Shot；
- Style Pack 不是 plugin，不新增扩展安装类型；
- 当前没有 Style Pack 市场；Style Pack 通过 Skill 复制到工作区并使用 `stylepack.validate` 校验；
- 当前没有自动 TTS / 字幕 / 发布；
- 当前代码修改仍由 AI 客户端完成，创作台负责项目、审查、反馈、门禁和产物。
## 九、用户怎么用

在 AI 客户端直接说：

```text
请用代码项目模式做一支 10 秒动态字体短片，
主题是“AI 能力沉淀”，暗色科技风，竖屏 1080×1920。
做完后让我在创作台审查，我要继续调。
```

AI 应执行：

```text
复制 examples/code-project-demo 到工作区
→ 改 STORYBOARD.md 与 reel.js
→ project.import
→ preview.review(sheet)
```

随后创作台会出现：

```text
code-demo  [代码]
审阅 / 分镜
关键帧 / 连续帧 / 静帧
审查产物
反馈输入
```

用户在创作台写反馈后，AI 下一轮应调用 `project.context` 读取反馈，改代码，再 `project.sync` + `preview.review`。最终通过 `quality.check` 后调用 `render.start`。

## 十、当前验证证据

- 项目：`code-demo`
- Style Pack：`kinetic-mini`
- 审查图：`dist/verify/code-project/sheet.jpg`
- 连续帧：`dist/verify/code-project/strip.jpg`
- 最终视频：`dist/verify/code-project/code-project-demo.mp4`
- 视频：1080×1920 / 10.00s / 300 帧
- `dev/verify-render.py`：10/10 PASS
- 创作台审查页：`dist/verify/code-project-review-18084.png`
- 创作台分镜页：`dist/verify/code-project-storyboard-18084.png`
- 约束清单：43/43 已解析
## 十一、网络策略与确定性门禁

### 网络策略

Style Pack 声明：

```json
"network": {"mode": "none", "allow": []}
```

- `none`：默认；只允许 `127.0.0.1`
- `declared`：只允许白名单域名
- `full`：仅 `trust: "signed"` 的 Style Pack

插件通过 Node `--require network_guard.cjs` 拦截 `fetch / http / https / net / dns / WebSocket`，并通过 `HIMIND_CHROME_ARGS_JSON` 给 Chrome 注入 `--host-resolver-rules`。  
这是 best-effort 隔离；硬隔离仍应放到独立 Renderer 进程/容器。

### 帧格式与暗部色带

最终成片默认使用 **PNG 帧序列**（`--frame-format=png`），避免 JPEG 中间帧把暗部渐变压成大片色块。

- `--frames` 默认写 `f00000.png`；`--encode` 自动识别 PNG/JPG；
- 需要快速测试时可显式传 `--frame-format=jpg`；
- PNG 帧约 2GB/900 帧、渲染约 8–10 分钟，属于最终交付质量档；
- 19 个 Style Pack 与 `examples/code-project-demo` 已统一支持该参数。

量化检查：

```text
media.workbench.quality.banding
  workspace_root
  image_path | video_path
  time
  max_run_ratio
  min_levels
```

已有项目升级 renderer：

```text
media.workbench.project.upgrade-style
  workspace_root
  project_id
  target_version?
  dry_run?
```

该能力只替换 `render.mjs` 等 renderer 契约文件，保留 `reel/`、`input/`、`STORYBOARD.md` 与项目数据；成功后自动把预览/确定性标记为 `stale`，并备份旧 renderer。

成片后的视觉 QA：

```text
media.workbench.quality.visual
  workspace_root
  project_id | video_path
  times?
```

当前检查：帧数/fps、黑帧、静帧、色带、音频响度。未声明 `quality.text_audit` 的风格，`text_bounds` / `safe_area` / `text_overlap` / `title_hit` 返回 `skipped`；声明后走下面的通用文字审计。

渲染完成后插件会自动执行一次视觉 QA，把结果写入作业与母版 artifact 的 `quality_status` / `quality_report`；失败不会删除成片，但创作台会以错误色显示，必须修复后重新交付。

### 文字审计（可选）

Style Pack 可以声明：

```json
"quality": {
  "text_audit": {
    "enabled": true,
    "page": "reel/index.html",
    "times": "1.2,3.5,9,16,23,28"
  }
}
```

`quality.visual` 会调用通用：

```text
renderer/browser-code/text-audit.mjs
  --source=<project source_dir>
  --page=<reel/index.html>
  --times=<...>
```

检查：

- `text_bounds`：文字是否越出画布；
- `safe_area`：标题/大字号文字是否进入 5% 安全区外；
- `text_overlap`：18px 以上文字是否重叠；
- `title_hit`：场景标题是否在画面文字中命中。

未声明 `quality.text_audit` 的风格仍返回 `skipped`，不会伪装成已检查。

### 风格感知的 banding gate

平面/白底风格的大块纯色不应被当成“严重色带”。Style Pack 可以声明：

```json
"quality": { "banding_gate": false }
```

效果：

- `banding` 返回 `skipped`；
- 仍记录最小亮度级数与最大平坦段，便于人工复核；
- 不阻断视觉 QA。

已启用文字审计：`celestial-seasons`、`3d-product-reel`、`whiteboard-explainer`、`isometric-infographic`、`cinematic-map-route`；`paper-collage` 因 Canvas transform 复杂暂不启用，`retro-vhs` 待独立验证。




### 确定性门禁

```text
media.workbench.preview.determinism
```

两次渲染同一组静帧并比较 SHA-256。

- 不一致：`determinism_status=failed`
- 一致：`determinism_status=passed`
- 源码变化：`project.sync` 会把确定性标记为 `stale`
- 仅写入源码或分镜后，持久化的 `determinism_status` 可能仍是 `passed`；创作台会同时比较 `determinism_hash` 与 `source_hash`，不一致时显示 `stale（待重检）`
- 审查图按项目实际时长均匀采样 9 帧；确定性检查按 15% / 50% / 85% 采样
- 「检查确定性并渲染」会先重检确定性，再启动渲染，避免用旧哈希误放行

`render.start` 要求：

```text
preview_status=ready
determinism_status=passed
preview_hash = determinism_hash = source_hash
```

当前 `kinetic-reel` 两次独立 17 秒渲染得到完全相同的 MP4 SHA-256，确定性验证通过。
## 十二、交付预设

Style Pack 可以声明：

```json
"delivery": {
  "web": true,
  "web_width": 1280,
  "web_crf": 23,
  "max_master_mb": 200
}
```

渲染完成后：

- 母版 `video/mp4`
- Web copy `video/web`

Web copy 使用 ffmpeg 重新缩放与编码，不改变母版。缩放按**长边**计算：横屏 `1920×1080` → `1280×720`，竖屏 `1080×1920` → `720×1280`；`web_crf` 可按风格覆盖。当前 `painted-animation` 母版 13.38 MB，Web copy 1.91 MB。
## 十三、数据驱动 Style Pack

代码项目可以通过：

```text
media.workbench.project.data.write
  project_id
  data
  brief
```

把规范化数据写入：

```text
<source_dir>/input/data.json
<source_dir>/input/brief.json
```

写入后：

- `source_hash` 更新
- `preview_status=stale`
- `determinism_status=stale`
- `revision++`

Style Pack 可以从 `input/data.json` 读取：

```json
{
  "title": "2025 省会 GDP TOP 3",
  "subtitle": "数据驱动 · 示例",
  "unit": "亿元",
  "items": [
    {"name": "合肥", "value": 12000},
    {"name": "芜湖", "value": 9800},
    {"name": "滁州", "value": 7600}
  ]
}
```

`examples/code-project-demo` 已支持该输入：没有数据时使用示例内容，有数据时渲染 `title/subtitle/items`。

实测：

```text
data.write
→ preview.review sheet 146,825 bytes
→ preview.determinism valid=true
→ quality.check passed=true
→ render.start completed
→ 10 秒 MP4 418,068 bytes
```


### 创作台内渲染验收（2026-10-18）

```text
项目：video-project-63773485
工作区：%TEMP%\himind-video-check\ws2
操作：检查确定性 -> 渲染成片
页面状态：确定性 passed -> rendering 10% -> completed 100% -> 成片 2 个
入口状态：打开最新成片 (2) 可用
```

成片：

```text
母版：443,589 bytes
SHA-256: 20B37CCA00056C63237D44F1721D53B8E039C621737A5D9094AE9E1762242471
Web ：345,769 bytes
SHA-256: E6CCCB32C3DD9AC1DF0070187DC98F6A17A41F20025901B52F9DFEB3007284DE
```

截图：

```text
dist/verify/whiteboard-render-complete-18084.png
SHA-256: 8F3397421FA346AE0C8F5A7D7F918B5DDD897FEB9F7F80D589FA611AB5CB0E79
```







