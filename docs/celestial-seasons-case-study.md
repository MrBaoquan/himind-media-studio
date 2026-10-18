# 案例：30 秒《为什么地球有四季？》

> 状态：已跑通（2026-10-18）  
> 项目：`video-project-5f5d8aa5`（创作台修订 `r8`）  
> 风格包：`celestial-seasons` `1.0.0`  
> 渲染器：`browser-code`（three.js + Canvas 2D overlay）  
> 用途：用一期科学 3D 科普片，验证「AI 对话 → 创作台项目 → 审查 → 确定性 → 成片 → 资产沉淀」完整链路。

## 一、目标与结论

这期视频回答一个常见但容易被讲错的问题：

```text
为什么地球有四季？
```

核心结论：

```text
四季由地轴倾斜 + 公转造成，不是日地距离变化造成的。
```

选择这个题目，是因为它同时具备：

- 明确的科学事实与可核对的公开来源；
- 适合用 3D 空间关系解释（公转、地轴、太阳光线、南北半球）；
- 有常见的错误直觉（“夏天是因为离太阳近”），适合做科普纠偏；
- 能覆盖 3D、文字排版、科学约束、音频、竖屏交付等核心能力。

## 二、科学约束

| 项目 | 数值 / 约束 | 说明 |
| --- | --- | --- |
| 地轴倾角 | `23.44°` | 相对公转轨道法线 |
| 轨道偏心率 | `0.0167` | 视觉上接近圆形 |
| 日地距离 | 约 `1.47–1.52 亿 km` | 近日点 / 远日点范围 |
| 画面比例 | `SCHEMATIC / NOT TO SCALE` | 不按真实天体比例绘制 |
| 主因 | 地轴倾斜 + 公转 | 距离变化不是四季主因 |
| 来源 | NASA Earth Fact Sheet；NOAA Seasons | 画面底部持续显示来源 |

这些约束不是文案建议，而是 Style Pack 的资产内容：`STORYBOARD.md`、`reel.js` 中的数据契约和最终画面必须一致。

## 三、叙事分镜

| 时间 | 场景 | 观众必须读到 |
| --- | --- | --- |
| 0–6s | 地球在公转 | 轨道接近圆形；日地距离变化不是四季主因 |
| 6–12s | 地轴倾斜 23.44° | 地轴方向在一年中几乎保持不变 |
| 12–20s | 北半球夏至 | 北半球更朝向太阳：太阳更高、白昼更长 |
| 20–26s | 南半球夏至 | 南北半球季节相反 |
| 26–30s | 结论 | 四季 = 地轴倾斜 + 公转；NOT DISTANCE |

画面使用统一的科学信息层级：

- 顶部：`SCHEMATIC / NOT TO SCALE` 与频道标识；
- 左上：场景标题、解释句、`01 / 05` 章节标记；
- 中部：3D 太阳、地球、轨道、地轴、太阳光线；
- 下部：`23.44°`、`0.0167`、`1.47–1.52 亿 km` 事实卡；
- 结尾：结论与来源；
- 全程：时间轴进度条。

## 四、技术实现

### 4.1 3D 场景

- three.js `0.160.0`（MIT，作为 npm 依赖）；
- NASA 地球贴图：昼夜地表、城市灯光、云层、法线、海洋高光；
- 自定义 PBR 地球 Shader：日间/夜间过渡、云层混合、海洋镜面高光、法线细节、大气 Fresnel 边缘光；
- 大气壳层、太阳临边昏暗与表面颗粒 Shader、双层日冕、水平变形宽银幕光晕；
- 1800 颗带色温变化的星点；
- 轨道线、地轴、赤道环、南北半球高亮环；
- 相机在开场做推近与横向漂移，在地轴段落做轻微环绕，在结论段缓慢拉远；
- 所有随机量使用固定种子 / 确定性伪随机，不使用运行时随机数。

### 4.2 画面合成

- WebGL 渲染 3D 场景；
- Canvas 2D 叠加标题、解释、事实卡、标签、进度条与来源；
- 开场 0–2.7s 显示数据契约中的主标题/副标题（`为什么地球有四季？` / `30 秒看懂：不是离太阳远近`），再切换到第一场景标题；
- 标签通过边界 clamp 保证不越出 1080×1920 安全区；
- 画面明确标注 `SCHEMATIC / NOT TO SCALE`。

### 4.3 声音

- 默认生成确定性的程序化环境声床：A 大调低频铺底 + 柔和空气噪声 + 缓慢起伏；
- 无第三方音乐，无版权风险；
- 无旁白；中文旁白应由可选 TTS External Pack 生成后通过 `--audio` 注入；
- 需要静音交付时使用 `--silent`。

### 4.4 渲染契约

Browser-Code 固定 CLI：

```bash
node render.mjs --sheet=1.5,4.8,8.1,11.4,14.7,18,21.3,24.6,27.9 --cols=3 --w=540 --out=out/check/sheet.jpg
node render.mjs --strip=2:3 --cols=6 --w=320 --out=out/check/strip.jpg
node render.mjs --stills=2,9,16,23,28 --out=out/stills
node render.mjs --frames --workers=1 --duration=30
node render.mjs --encode --out=out/video.mp4
```

创作台只执行这些固定子命令，不接受 AI 临时拼接的任意 shell。

## 五、创作台链路

```text
AI 客户端对话提出需求
→ 选择 celestial-seasons Style Pack
→ 复制模板到工作区
→ 写入 STORYBOARD / 数据 / 代码
→ project.import
→ preview.review（按 30s 时长均匀采样 9 帧）
→ 用户在创作台审阅、记录反馈
→ AI 读取 project.context 修改
→ project.sync
→ preview.determinism（15% / 50% / 85% 三帧重复渲染哈希）
→ quality.check（46/46）
→ render.start（后台渲染 900 帧 + 编码 + Web copy）
→ 创作台显示母版 / Web copy / 审查图
```

这次链路中，创作台负责的是**项目管理与证据**，而不是替 AI 做创作判断：

- 项目列表、修订、风格包版本、源码目录；
- 审查图、连续帧、静帧；
- 反馈记录与 AI 下一轮上下文；
- 确定性、质量门禁、渲染任务状态；
- 产物索引与最新成片入口。

## 六、质量门禁与证据

### 6.1 成片

| 项目 | 结果 |
| --- | --- |
| 母版 | `1080×1920` / `30.000s` / `30fps` / `900` 帧 |
| 母版大小 | `22,144,909` bytes |
| 母版 SHA-256 | `F20B0DC8A2FC051641CF6C7CA974A38F5BB44CA659AFF5F67532903856014E80` |
| Web copy | `720×1280` / `3,627,412` bytes |
| Web copy SHA-256 | `1D751DAF822B71CB3C8B9A2CA9410EAF3D978CE440D604EE71254E5702CE559B` |
| 视频编码 | H.264 / `yuv420p` / 30fps |
| 音频编码 | AAC / 约 `128–192 kbps` |
| 响度 | `-22.2 LUFS`，true peak `-13.6 dBFS` |

### 6.2 自动检查

| 检查 | 结果 |
| --- | --- |
| 黑帧检测 | 无 `black_start` |
| 静帧检测（≥1.5s） | 无 `freeze_start` |
| 帧数 | `900 / 900` |
| 确定性 | 15% / 50% / 85% 采样静帧重复渲染 SHA-256 一致，`passed` |
| 创作台母版 | 与独立渲染母版 SHA-256 完全一致 |
| 质量门禁 | `46/46` |
| 审查图 | 按 30s 时长采样 9 帧，`246,032` bytes |
| 动态预览 | 2s WebP，`93,760` bytes |
| 文字审计 | 开场主标题/副标题 + 5 个场景标题 / 章节标记 / 事实卡 / 来源全部命中；0 越界；0 重叠 |
| 通用文字审计 | 6 场景 / 89 条文字；`text_bounds` / `safe_area` / `text_overlap` / `title_hit` 全部通过 |

文字审计结果：

```text
dist/verify/celestial-seasons/overlay-audit.json
ok: true
6/6 samples（开场 + 5 个场景）
missing: []
outOfBounds: []
overlaps: []
```

独立渲染参考：`dist/verify/celestial-seasons/celestial-seasons-standalone.mp4`，SHA-256 `F20B0DC8A2FC051641CF6C7CA974A38F5BB44CA659AFF5F67532903856014E80`（与创作台母版完全一致）。

### 6.3 证据文件
```text
dist/verify/celestial-seasons/celestial-seasons-master.mp4
dist/verify/celestial-seasons/celestial-seasons-standalone.mp4
dist/verify/celestial-seasons/sheet-final.jpg
dist/verify/celestial-seasons/keyframes/keyframe-01-intro.jpg
dist/verify/celestial-seasons/keyframes/keyframe-03-orbit.jpg
dist/verify/celestial-seasons/keyframes/keyframe-09-tilt.jpg
dist/verify/celestial-seasons/keyframes/keyframe-16-north.jpg
dist/verify/celestial-seasons/keyframes/keyframe-23-south.jpg
dist/verify/celestial-seasons/keyframes/keyframe-28-conclusion.jpg
dist/verify/celestial-seasons/overlay-audit.json
dist/verify/celestial-seasons/qa.txt
dist/verify/celestial-seasons-workbench-final-18084.png
dist/verify/style-library-science-filter-18084.png
dist/verify/style-library-3d-filter-18084.png
```

## 七、这次补强的通用能力

这期视频不是只多做了一支片子，它反向补强了创作台的通用能力：

1. **审查图按项目时长采样**：30 秒项目不再只采样前 10 秒，而是均匀覆盖 1.5–27.9 秒；
2. **确定性检查按比例采样**：15% / 50% / 85%，长视频不再固定采前几秒；
3. **确定性旧哈希识别**：源码或分镜变化后，即使持久化状态仍是 `passed`，界面也会比较 `determinism_hash` 与 `source_hash`，显示 `stale（待重检）`；
4. **一键检查并渲染**：先重检确定性，再启动渲染，避免旧哈希误放行；
5. **竖屏 Web copy**：按长边缩放，1080×1920 自动输出 720×1280，而不是错误地保持 1080×1920；
6. **程序化环境声床**：确定性、无版权风险，补齐视频的音频轨；
7. **画面文字审计**：用 Canvas `fillText` 钩子检查场景标题、事实卡、来源、越界与重叠，作为可复用的 QA 手段；
8. **风格库分类**：新增「3D」「科学」筛选，当前 19 个风格中 3D 3 个、科学 1 个。

## 八、对架构的意义

这期案例验证了当前架构的三个关键判断：

### 8.1 Style Pack 是「可复用能力资产」，不是一次性代码

`celestial-seasons` 可以被多次复制成不同项目，项目内可以继续改代码、改数据、改分镜；Style Pack 本身保持版本化、可校验、可打包、可分发。

### 8.2 AI 客户端负责判断，创作台负责证据

AI 负责选题、科学约束、叙事和代码修改；创作台负责把每一步变成可审阅、可复现、可追溯的项目状态。  
这避免了“AI 说已经做完了，但用户看不到证据”的问题。

### 8.3 确定性不是渲染器的附属功能，而是交付契约

只有在以下三者一致时，才允许最终渲染：

```text
preview_hash = determinism_hash = source_hash
```

这次创作台母版与独立渲染母版的 SHA-256 完全一致，说明这条契约在 3D / WebGL / PBR 材质 / PNG 帧 / 音频 / 竖屏场景下同样成立。

## 九、边界与下一步

### 当前边界

- **没有中文旁白**：当前是文字驱动的科学讲解 + 3D 动画 + 环境声床。TTS 属于可选 External Pack，不应塞进创作台核心。
- **不是真实比例**：画面是解释性示意图，明确标注 `SCHEMATIC / NOT TO SCALE`；如果需要真实星历，应接入 NASA/JPL 数据与科学可视化适配器。
- **没有发布能力**：发布到抖音 / 视频号 / YouTube 属于可选 External Pack，当前只交付母版与 Web copy。
- **资产包仍是 `prepared`**：`celestial-seasons-1.0.0.hmpack` 已打包并更新目录哈希，但尚未发布 GitHub Release / 签名。

### 下一步建议

1. **TTS External Pack**：接入可授权的中文旁白，把旁白音频作为 `--audio` 注入，保留字幕与时间轴对齐；
2. **科学数据适配器**：把“示意动画”与“真实星历 / 真实比例”分成两种模式，避免科普画面误导；
3. **更多 3D 科学模板**：潮汐、日食月食、板块运动、气候变化等，复用同一套 three.js + Canvas overlay + 确定性管线；
4. **平台交付预设**：竖屏 9:16、横屏 16:9、方形 1:1 的长边 / 码率 / 响度预设；
5. **资产市场发布**：把 `celestial-seasons` 作为第一个 3D 科学 Style Pack 发布，验证签名、更新与回滚。

## 十、色带 / 色块问题与修复

现象：暗背景出现连片色块/色带，尤其在地球周围和画面底部。

诊断（逐层隔离）：

- PNG 截图：背景渐变有 51 个亮度级，最大平坦段 5px；
- JPEG 帧：同一位置只有 2 个亮度级，最大平坦段 648px；
- H.264 视频：编码后仍是 2 个亮度级，说明主要问题在帧捕获阶段的 JPEG 压缩，而不是渲染器或视频编码器；
- 叠加层径向 vignette 与 CSS contrast 进一步放大了暗部色带。

修复：

1. 移除 overlay 的径向 vignette；
2. 移除 WebGL canvas 的 CSS contrast；
3. 在 3D 背景加入低频星云渐变 + 每像素 dither；
4. Earth / atmosphere / sun shader 输出加入 1/255 dither；
5. 最终成片改用 PNG 帧序列，不再用 JPEG 中间帧；
6. Web copy CRF 从 27 提升到 25。

验证：

- PNG 帧背景：51 个亮度级，最大平坦段 5px；
- PNG 帧编码后的视频：46 个亮度级，最大平坦段 6px；
- 画面底部：26 个亮度级，最大平坦段 40px；
- 无黑帧、无 ≥1.5s 静帧。

代价：

- PNG 帧约 2GB 临时磁盘；
- 900 帧渲染约 8–10 分钟；
- 母版体积明显增大，但暗部色带被消除。

该 PNG 帧策略已推广到全部 19 个 Style Pack 与 `examples/code-project-demo`；`--frame-format=jpg` 仅建议用于快速测试。

## 十一、复现

```powershell
cd C:\Users\mrma6\Documents\HiMindExtensions\himind-media-studio\style-packs\celestial-seasons\template
npm install
node render.mjs --frames --workers=1 --duration=30
node render.mjs --encode --out=..\..\..\dist\verify\celestial-seasons\celestial-seasons-standalone.mp4
```

独立复现得到 `celestial-seasons-standalone.mp4`（`F20B0DC8...`），与创作台母版 SHA-256 完全一致。

创作台内：

```text
打开「自媒体创作台」
→ 天体四季科普 项目
→ 重新生成（同步源码并生成审查图）
→ 检查确定性并渲染
→ 等待 completed 100%
→ 打开最新成片
```



