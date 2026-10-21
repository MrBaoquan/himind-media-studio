# 剪辑能力 M1-M4 实施报告

> 日期：2026-10-20
> 范围：`docs/editing-implementation-plan.md` 的 M1-M4，以及 backlog E-10…E-29 / C-01…C-07
> 定位：AI 创作系统的精修面，而不是复刻 NLE。

## 一、完成结论

剪辑能力已经形成一条可运行闭环：

```text
创作/分镜产物
  → 一键成片 / 模板槽位 / 去静音 / 文本粗剪
  → 同一 Timeline（帧时间、revision、命令、undo/redo）
  → 简易 / 专业 / 交付三模式
  → 时间轴 QA
  → 单一 render.start（ffmpeg）
  → ducking + loudnorm + 字幕烧录 + 平台预设
  → delivery.qa
```

创作台默认进入简易模式；专业模式保留显式剪辑工具；交付模式负责平台预设与 QC。三种模式共享同一个 Timeline，不使用第二套工程文件。

## 二、核心架构

### 2.1 单一真值

- Timeline 为唯一剪辑真值，保留 `revision` 与 `expected_revision`。
- 命令统一走 `media.workbench.timeline.apply`。
- UI undo/redo 使用成功命令后的快照；失败命令不会污染撤销栈。
- 时间使用帧，避免浮点漂移。
- 渲染只有 `media.workbench.render.start` 一个入口。

### 2.2 新增能力

| 能力 | 用途 |
| --- | --- |
| `timeline.autoedit` | 按素材与目标时长生成确定性初剪 |
| `timeline.templates.list` / `timeline.template.apply` | 槽位模板（钩子-论证-收束、六拍故事、五段讲解） |
| `timeline.transcript.search` | 字幕/转写文本检索 |
| `timeline.roughcut` | 按命中文本生成 AI 粗剪 |
| `timeline.silence.detect` / `timeline.silence.trim` | 静音检测与去停顿 |
| `timeline.beats.detect` / `timeline.beats.markers` | 音频能量节拍与卡点 markers |
| `timeline.media.catalog` | 统一 MediaRef：uri/sha/fps/分辨率/时长 |
| `timeline.proxy.build` | 540p 代理；成片仍使用原素材 |
| `timeline.subtitle.attach` | 项目字幕 → S1 轨 |
| `timeline.qa` | 结构、媒体存在、同步、字幕检查 |
| `timeline.comment.*` | 时间码批注与解决 |
| `delivery.presets` | 母版/Web/抖音/B 站/YouTube/西瓜 |
| `delivery.qa` | 容器、画布、时长、响度 QC |

### 2.3 专业操作

Timeline 命令新增：

- `ripple_trim`
- `roll`
- `slip`
- `slide`
- `update_clip`（transform / transition / opacity / volume）
- 多 clip `link` / `unlink`
- S1 字幕 clip 的文本轨道校验

### 2.4 渲染

`timeline-ffmpeg` 现在支持：

- 多视频 clip 顺序拼接；
- 交叉淡化（`xfade`）；
- 9:16 / 1:1 / 16:9 目标画布 cover/contain；
- 图片素材；
- 多 audio track；
- narration/music 分角色混音；
- sidechain ducking；
- loudnorm 响度归一；
- SRT 字幕烧录；
- 平台分辨率与码率；
- 渲染前 QA 门禁。

## 三、真实验收证据

### 3.1 UI

| 场景 | 结果 | 证据 |
| --- | --- | --- |
| 撤销/重做 + 快捷键表 | 删除 5→4 clips；undo/redo 均成功；快捷键表打开 | `.extension-state/bridge-e2e-full/undo-redo-v2.png` |
| 一键成片 → 智能字幕 → QA → Web 渲染 → QC | 3 clips；字幕挂载；QA 通过；QC passed | `.extension-state/bridge-e2e-full/easy-deliver-e2e.png` |
| 100 clips 时间轴 | 220ms 出现 100 个 clip；DOM 5585 节点 | `.extension-state/bridge-e2e-full/timeline-100-clips.png` |
| 模板 / 字幕 / 文本检索 / 批注 | 模板套用、字幕挂载、文本命中、AI 粗剪、批注写入 | `.extension-state/bridge-e2e-full/editing-macros-e2e.png` |
| 画中画 / 1:1 重构 / 暖调 | V2 出现、画布 1080×1080、连续操作 revision 7 | `.extension-state/bridge-e2e-full/pro-pip-reframe-color.png` |
| 卡点 markers | 合成 click track 检出 7 个 beat 并写入 | `.extension-state/bridge-e2e-full/beat-markers-e2e.png` |

### 3.2 视频产物

| 产物 | 规格 | SHA-256 |
| --- | --- | --- |
| `dist/verify/timeline-m3.mp4` | 12s，720×1280，字幕烧录 | `e63a32ca590d45378d8e7934b4946e9f30a1715566e5f643381dd943fa62c4b5` |
| `dist/verify/timeline-audio-ducking.mp4` | 9s，1080×1920，voice+music ducking，-16.05 LUFS | `73f53675014dcb68a78a874da33d7c90a593612c5903da5ea397ce59d91271ac` |
| `dist/verify/timeline-xfade.mp4` | 8.5s，720×1280，15 帧交叉淡化 | `6a8ad69c447558e8eb3a36387c7a1bf743585e22e08f6f3531a86a7910c3ffea` |
| `dist/verify/silence-trim.mp4` | 3.7s，从 6s 素材去除 2s 静音 | `32074b475812acc9d11dd62dec19d7a87b33e98428a34be5394738872f4a2c7a` |

### 3.2.1 专业硬化产物

| 产物 | 规格 | SHA-256 |
| --- | --- | --- |
| `dist/verify/timeline-retime.mp4` | 2× retime，3s，命令日志 revision 写入 | `1f56d64a3001b2dc61fb42b157de82d9d01ecac8d8861073c2fe7d089960b06f` |
| `dist/verify/timeline-pip.mp4` | V1+V2 画中画，5s，720×1280 | `987bc090144e187034a07f879d55b0e32a3e04529af1427986ae6a2a702f7485` |
| `dist/verify/timeline-color.mp4` | 暖调 eq 调色，3s | `7e7a63f8799fb59335e5f3e31e16e8c481a562c11d910236d0aeffca0cd6937a` |
| `dist/verify/timeline-lut.mp4` | identity 3D LUT 渲染，2s | `e7608416be8bf2f6c5e90c21064d80b858e91bbd7013bf077febf90611846c4d` |

### 3.3 制品

- Workbench 0.67.0：`.hmpkg` SHA `4dc3940792d57894e3304315adb65708ca02e01fb1898cc28acaaab7ac52b204`。
- 通用资产包 `editing-kit@1.0.0`：`dist/asset-packages/editing-kit-1.0.0.hmpack`，SHA `7311cf7085c50f5c420b270e38c393c9ad567192aed8b522be97fd12c14b7c2e`。
- 资产包 validate：`valid=true / issues=0`；资源包含 caption、transition、delivery、template、preview。

## 四、当前边界

- 自动重构提供 9:16/1:1/16:9 与内容感知非黑边中心对齐；不做逐帧语义分割跟随。
- 卡点提供节拍 markers，不自动把任意模板重新切成逐拍剪辑。
- 色彩管理 / LUT 尚未实现（P2）。
- AI 粗剪当前基于转写文本确定性匹配，不包含语义 embedding。
- 交叉淡化会按转场时长缩短实际成片；渲染返回真实时长，避免静默误差。
- 颜色关键帧与分段速度曲线已实现；内容感知重构为非黑边中心对齐，不是逐帧语义跟踪。
- `timeline.diff` 支持 revision 间新增/删除/修改对比，可服务审阅版本对比。
- 命令日志为 JSONL append-only，可在项目 metadata 中追溯。

## 五、复现命令

```powershell
# 插件门禁
cd plugins/com.himind.media-workbench/media-workbench
go test ./...
go build -o bin/media-workbench.exe .

# UI 门禁
cd ../../../ui
npm run typecheck
npm run build

# 打包 / candidate / confirm
cd ..
powershell -ExecutionPolicy Bypass -File scripts/package-workbench.ps1 -Version 0.67.0

# 资产包
# 调用 media.workbench.asset.validate / asset.pack
```
