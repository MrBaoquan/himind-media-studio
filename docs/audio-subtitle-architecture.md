# TTS / 字幕 / BGM 音频底座架构

> 目标：把“旁白、字幕、背景音乐、混音、音频 QA”做成可复用、可替换、可审计的底座能力，而不是每条视频单独写脚本。
> 当前状态：P0 已落地，并用「合肥人造太阳」72 秒成片完成真实闭环。

## 一、分层

```text
CreativePackage
  ├─ script.beats           旁白、屏幕文字、时长
  ├─ production.voice       provider、voice pack、rate、license
  ├─ production.subtitles   时间轴字幕 cues
  ├─ production.bgm         BGM 预设、gain、loop、license
  └─ production.mix         voice/bgm 混音参数与产物
```

```text
com.himind.audio-production（External Pack plugin）
  ├─ audio.catalog          provider / voice / BGM 预设 / license
  ├─ audio.runtime.install  安装并校验离线 TTS 运行时与 voice pack
  ├─ audio.speech           文本 → 逐 beat WAV
  ├─ audio.bgm.generate     通用 BGM 预设 → WAV
  ├─ audio.mix.render       voice clips + BGM → M4A
  ├─ audio.qa.signal        时长、响度、真峰值、削波风险
  ├─ audio.manifest.get     读取音频 manifest
  └─ audio.pipeline.render  一键编排：旁白 + BGM + 混音 + QA

com.himind.media-workbench（应用型插件）
  ├─ media.workbench.subtitle.generate  脚本 → 时间轴字幕 cues
  ├─ media.workbench.audio.attach       把 mix 与字幕挂到项目
  └─ AUDIO_REQUIRED_FOR_NARRATION       有旁白时必须传 audio_path
```

> 旧的 `media.workbench.audio.*` 保留一版兼容，但创作台 UI 已切换到 `audio.*` 能力；后续版本会移除旧入口。

边界：

- TTS 不进入画面渲染器，只产出标准 WAV；
- 字幕是独立数据轨，渲染器按时间绘制；
- BGM 是通用音频资产，不绑定视频题材；
- 混音由 ffmpeg 完成，渲染器只负责 mux；
- `audio.build` 是编排入口，不替代底层单项能力。

## 二、TTS

### Provider 适配器

| Adapter | 状态 | 说明 |
| --- | --- | --- |
| `piper` | **默认，已实现** | 离线神经 TTS，CPU，无 API Key；运行时与 voice pack 可校验、可替换 |
| `windows-sapi` | 实验 | Windows 系统语音。插件进程在 Agent 沙箱中可能无法访问语音注册表，不作为默认路径 |
| `ark-speech` | 待开启 | 火山方舟 Agent Plan 语音模型；需要用户先开启语音模型抵扣 |
| `openai-speech` | 预留 | OpenAI 兼容 TTS 服务 |
| `external-pack` | 预留 | 第三方 TTS External Pack |

### Runtime Asset Pack

运行时不写死在插件二进制里，而是登记在：

```text
catalog/audio-runtimes.json
```

每个运行时包含：

- `id` / `version` / `license`；
- 每个制品的 URL、目标路径、大小、SHA-256；
- 每个 voice pack 的语言、质量、模型/配置、许可证、是否可商用、兼容性说明。

安装：

```json
{
  "workspace_root": "...",
  "runtime_id": "piper",
  "dry_run": false
}
```

安装落点：

```text
<workspace>/.himind-media/runtimes/piper-2023.11.14-2/
  piper/piper.exe
  voices/zh_CN-huayan-medium.onnx
  voices/zh_CN-huayan-medium.onnx.json
  runtime.json
```

`runtime.json` 记录安装时间、许可证与已校验制品。`dry_run=true` 只返回计划，不写文件。

### Voice Pack 许可策略

当前登记两个中文声音：

| Voice | 许可 | 商用 | 状态 |
| --- | --- | --- | --- |
| `zh_CN-huayan-medium` | 数据集许可未知 | 否，仅开发/演示 | 可用；默认声音 |
| `zh_CN-chaowen-medium` | CC0 | 是 | 当前 C++ Piper 对多码点音素不兼容，标记不可用；需 Python Piper 1.4+ 或更新运行时 |

因此：

- 受限声音必须显式传 `allow_restricted_voice=true` 才能合成；
- manifest 写入 `voice_license` 与 `commercial_voice`；
- 正式商用前必须替换为可商用 voice pack；
- 后续 Voice Pack 市场/资产包只允许登记来源、许可证、SHA-256 与兼容运行时。

### 逐 beat 合成

```text
.himind-media/creatives/<creative_id>/audio/voice-<beat>.wav
```

每个 beat 保留：

- 文本、voice、provider、rate；
- 原始时长与适配后时长；
- `tempo`（旁白超出镜头时最多 1.5x 压缩）；
- `sha256`、许可证、是否可商用；
- `fit_warning`（仍溢出时明确提示）。

## 三、字幕

### 数据

字幕写在项目 `input/data.json` 的顶层 `subtitles`：

```json
{
  "subtitles": [
    {"start": 0.0, "end": 3.28, "text": "在合肥，有一轮“人造太阳”。它不挂在天空，", "beat_id": "b1"},
    {"start": 3.28, "end": 5.0, "text": "而是在一座实验装置里。", "beat_id": "b1"}
  ]
}
```

同时保留独立副本：

```text
.himind-media/creatives/<creative_id>/audio/subtitles.json
```

### 生成策略

- 按 beat 旁白切分；
- 按中文标点与最大字数 `max_chars`（默认 24）切分；
- cue 不跨 beat；
- 最短 cue `min_cue_seconds`（默认 0.9s）；
- 用实际旁白有效时长决定字幕结束点，最多留 0.55s 尾巴；
- 字幕文本与 TTS 文本来自同一份 `script.beats`，避免两边漂移。

### 渲染

`science-fusion` 的 Canvas 渲染器已加入 `drawSubtitles(time)`：

- 固定在底部安全区，约 y=1536–1718；
- 最多两行，超长自动换行；
- 图片场景用深色半透明卡片，纸面场景用浅色纸卡；
- 与来源行 y=1778、底部边框、标题/说明保持距离；
- 字幕样式由 Style Pack 决定，渲染器只消费 `subtitles` 数据。

`text-audit.mjs` 会拦截 `fillText`，因此字幕会自然进入文字边界、安全区、重叠与标题命中检查。

## 四、BGM

### 预设目录

```text
catalog/audio-presets.json
```

预设是**通用音频资产**，不绑定视频题材。当前四个：

- `documentary-drone`：纪录片低频；
- `hopeful-pulse`：希望脉冲；
- `minimal-tech`：极简科技；
- `warm-memory`：温暖回忆。

每个预设声明：

- 名称、描述、mood；
- 许可证 `generated-original`；
- `loop`；
- 音层频率/音量/低通；
- 粉噪/白噪/棕噪；
- tremolo、echo、gain、淡入淡出。

### 生成

`audio.bgm` 用 ffmpeg `lavfi` 实时合成：

```text
sine layers + noise → lowpass/highpass → amix
→ tremolo → echo → stereo → limiter → fade → WAV
```

输出：

```text
.himind-media/creatives/<id>/audio/bgm-<preset>.wav
```

无第三方音乐版权风险。后续可替换为 CC0 / 商业授权音乐 / 品牌定制 BGM，但必须写入来源与许可证。

## 五、混音

### `audio.mix`

输入：

```json
{
  "voice_clips": [
    {"path": "audio/voice-b1.wav", "start": 0, "window_seconds": 5, "fit_to_window": true}
  ],
  "bgm_path": "audio/bgm-documentary-drone.wav",
  "duration_seconds": 72,
  "voice_gain_db": 0,
  "bgm_gain_db": -20,
  "ducking": true
}
```

处理：

```text
每个 voice clip → aformat → atempo（按镜头窗口）→ volume → adelay
→ amix → alimiter
BGM → volume → atrim → fade → sidechaincompress(voice)
voice + bgmduck → amix → apad/atrim → loudnorm → limiter → AAC/M4A
```

当前门禁参数：

- 目标响度 `-16 LUFS`；
- `loudnorm TP=-2.5`；
- 最终 `alimiter=limit=0.78:level=disabled`；
- 输出 AAC 192k / 48kHz / stereo。

合肥案例实测：

```text
duration        72.00 s
integrated      -16.28 LUFS
true peak       -2.49 dBTP
max volume      -2.50 dB
status          passed
```

## 六、与渲染链路的关系

```text
CreativePackage
→ shot.produce / media.image.generate
→ audio.build（或 audio.speech + subtitle.generate + audio.bgm + audio.mix）
→ 字幕写回项目 input/data.json
→ preview.review
→ preview.determinism
→ render.start(audio_path = mix.m4a)
→ 成片 + quality.visual + audio.qa
→ artifact.export
```

`render.mjs` 已支持：

```text
--audio=<mix.m4a>
ffmpeg -c:a aac -b:a 192k -shortest
```

## 七、创作台 UI

创作脉络 → 制作规格 → 「旁白 / 字幕 / BGM」：

- Provider / voice 下拉；
- 语速、BGM 预设、BGM 音量、ducking；
- 受限声音的许可证确认；
- 「安装 Piper 运行时」；
- 「生成音频底座」；
- manifest 摘要：片段数、字幕 cues、BGM、混音路径、时长；
- 音频 QA：LUFS、真峰值、采样峰值、warning/blocker；
- 「复查音频 QA」。

AI 对话侧只需调用一个能力：

```text
media.workbench.audio.build
```

## 八、质量门禁

- 有旁白时必须有音频轨；
- 音频时长与视频时长误差 ≤ 0.2s；
- 整体响度 -16±2 LUFS；
- 真峰值 ≤ -1.5 dBTP；
- 无削波；
- 字幕不出安全区、不与画面文字重叠、不遮挡来源；
- 字幕 cue 不跨 beat；
- BGM 必须记录来源/许可证；
- TTS 文本与字幕文本来自同一逐步旁白；
- 受限 voice pack 不得在未确认时用于正式交付。

## 九、分阶段

### P0（已完成）

- Piper 运行时安装、SHA-256 校验、voice pack 许可策略；
- 逐 beat TTS、语速适配、manifest；
- 字幕切分、写回项目、Canvas 字幕；
- 四个程序化 BGM 预设；
- ffmpeg 定位/ducking/响度归一化/限幅；
- 音频 QA；
- 创作台音频面板；
- 合肥案例真实成片。

### P1

- Ark Speech / OpenAI Speech adapter；
- word/segment timestamps，逐词高亮；
- 可商用中文 voice pack；
- BGM/SFX/Voice Pack 资产市场。

### P2

- 角色对白与旁白分离；
- 多轨混音、音乐情绪曲线；
- 唇形同步；
- 音频指纹与去重。

## 十、已知边界

- Windows SAPI 在 Agent 插件沙箱中可能无法访问语音注册表；Piper 是默认离线路径。
- `zh_CN-huayan-medium` 仅用于开发/演示，商用必须替换。
- `zh_CN-chaowen-medium` 虽为 CC0，但当前 C++ Piper 运行时不兼容多码点音素；需更新运行时。
- 字幕目前是 cue 级时间轴，尚未做逐词 karaoke。
- 首次安装 Piper 约需下载 150 MB，之后完全离线。
