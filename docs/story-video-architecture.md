# 故事视频能力演进（LibTV 类）

> 结论：现有底座可以逐步支持故事视频，但不能把它当成一个新的 Style Pack 硬塞进现有链路。
> 需要把 CreativePackage 扩展为“故事制作包”，把 Style Pack 扩展为 Story / Character / Asset / Timeline 能力族。

## 一、现有底座已经具备什么

- CreativePackage：Brief、研究、脚本、分镜、制作规格、审批、revision、materialize；
- Project：项目、预览、确定性、渲染、QA、产物；
- Renderer：Remotion 主引擎 + Browser-Code 风格化片段；
- Asset Pack：风格、模板、预览、来源与许可证；
- Skill / Workflow：AI 判断与多步骤编排；
- 创作脉络台：项目、创意、逻辑检查、来源、revision、成片溯源的 UI 骨架。

这些能力已经覆盖“从想法到一条风格化短片”的大部分骨架。故事视频缺的不是项目壳，而是**角色、场景、资产、音频、长视频时间线**。

## 二、故事视频的目标链路

```text
主题 / 故事
→ 故事圣经（角色、世界、视觉规则）
→ 剧本（场、对白、旁白）
→ 分场（scene）
→ 分镜（shot）
→ 资产（角色图、场景图、道具、配音、音乐）
→ 时间线（视频、配音、音乐、字幕）
→ Remotion 组装
→ 连续性 / 字幕 / 音频 / 画面 QA
→ 成片 / 系列
```

## 三、数据模型扩展

CreativePackage 需要从 5 个 section 扩展为：

```text
brief
research
story
characters
script
scenes
storyboard
assets
production
timeline
approval
links
```

### story

```json
{
  "logline": "一句话故事",
  "synopsis": "故事梗概",
  "theme": "主题",
  "tone": "语气",
  "world_rules": ["世界规则"],
  "episode_arc": ["起", "承", "转", "合"]
}
```

### characters

```json
{
  "id": "char_01",
  "name": "角色名",
  "role": "主角 / 配角",
  "appearance": "外观描述",
  "wardrobe": "服装",
  "personality": "性格",
  "voice_ref": "voice_01",
  "reference_assets": ["asset_char_01_front"],
  "consistency_prompt": "固定角色描述",
  "negative_prompt": "不要出现的特征"
}
```

### scenes

```json
{
  "id": "scene_01",
  "location": "地点",
  "time_of_day": "白天 / 夜晚",
  "characters": ["char_01"],
  "action": "发生什么",
  "dialogue": "对白",
  "mood": "情绪",
  "duration_seconds": 12
}
```

### storyboard.shots 扩展

```json
{
  "id": "shot_01",
  "scene_id": "scene_01",
  "shot": "wide / medium / close",
  "camera": "推 / 拉 / 摇 / 移",
  "composition": "构图",
  "prompt": "图像或视频生成提示词",
  "negative_prompt": "负面提示词",
  "reference_assets": ["asset_char_01_front"],
  "duration_seconds": 3,
  "transition": "cut / fade",
  "audio_cue": "voice_01 / music_01"
}
```

### assets

```json
{
  "id": "asset_char_01_front",
  "type": "character_ref | location_ref | image | video | voice | music | sfx",
  "path": "assets/char_01/front.png",
  "source": "generated / stock / user",
  "license": "MIT / CC0 / commercial",
  "sha256": "…",
  "status": "draft / approved"
}
```

### timeline

```json
{
  "tracks": [
    {"id": "video", "clips": [{"shot_id": "shot_01", "start": 0, "end": 3}]},
    {"id": "voice", "clips": [{"asset_id": "voice_01", "start": 0, "end": 3}]},
    {"id": "music", "clips": [{"asset_id": "music_01", "start": 0, "end": 60}]},
    {"id": "subtitle", "clips": [{"text": "字幕", "start": 0, "end": 3}]}
  ]
}
```

> 镜头级能力契约（provider / technique / prompt / fallback / timeline.plan）见 [shot-capability-architecture.md](shot-capability-architecture.md)。

## 四、能力族与扩展分工

| 能力族 | 形态 | 负责 |
| --- | --- | --- |
| Story Pack | Asset Pack | 故事结构、类型语法、集数弧线 |
| Character Pack | Asset Pack | 角色设定、参考图、固定提示词、种子 |
| Location Pack | Asset Pack | 场景、世界观、地点参考 |
| Voice Pack | Asset Pack | TTS 音色、语速、情绪 |
| Music / SFX Pack | Asset Pack | 配乐、音效、混音预设 |
| Storyboard Pack | Asset Pack | 镜头语法、构图、转场 |
| Image Gen Adapter | External Pack plugin | 文生图 / 图生图 |
| Video Gen Adapter | External Pack plugin | 图生视频 / 视频延长 |
| TTS Adapter | External Pack plugin | 旁白、对白、字幕时间戳 |
| Remotion Renderer | Renderer plugin | 时间线组装、字幕、音频、转场 |
| Story Director | Skill | 故事结构、角色弧线、节奏 |
| Character Continuity | Skill | 角色一致性检查与修正 |
| Shot Planner | Skill | 剧本 → 分场 → 分镜 |
| Audio Director | Skill | 配音、音乐、混音 |
| story-video | Workflow | 多步骤编排与审批门禁 |

## 五、创作台 UI 的演进

创作脉络从 7 段扩展为：

```text
故事 → 角色 → 剧本 → 分场 → 分镜 → 资产 → 音频 → 时间线 → 渲染 → QA
```

右栏“逻辑检查”增加：

- 连续性：角色外观、服装、道具、色彩；
- 资产覆盖：每个 shot 是否有参考资产；
- 音频覆盖：每个 scene 是否有配音/音乐；
- 时间线：视频、配音、字幕是否对齐；
- 版权：资产来源与许可证是否齐全。

中栏增加：

- 故事 / 角色 / 分场编辑器；
- 资产库（角色卡、场景图、参考图、音频）；
- 多轨时间线（视频、配音、音乐、字幕）；
- 镜头级预览与重生成。

## 六、分阶段落地

### P0：故事骨架 + 静态图 + 旁白 + 字幕

- 扩展 `story / characters / scenes` schema；
- 用现有 Remotion 组装静态图、Ken Burns、字幕、TTS；
- 先做 60–90 秒故事短片：3 个角色、5 个场景、12 个镜头。

### P1：图像生成 + 角色一致性

- Image Gen Adapter；
- Character Pack（参考图、固定提示词、种子）；
- 角色一致性检查；
- shot 级资产替换与重生成。

### P2：图生视频 + 唇形 + 音乐

- Video Gen Adapter；
- 唇形同步；
- Music / SFX Pack；
- 多轨混音与字幕样式。

### P3：系列化

- 多集项目、角色/世界资产复用；
- 剧集 bible、角色弧线、连续性看板；
- 批量生成与版本对比。

### P4：长视频与审片

- 长视频时间线、多轨播放器；
- 审片批注、镜头级反馈；
- 资产清理与存储配额。

## 七、关键风险

- **角色一致性**：参考图、固定提示词、种子、检查器缺一不可；
- **版权与授权**：图像/视频/声音来源、模型许可、声音克隆授权；
- **成本与时长**：生成任务必须异步 Job，设置预算与超时；
- **审核**：故事、角色、资产、成片多级审批；
- **存储**：大量图片/视频资产需要资产库、哈希、清理策略；
- **可复现性**：生成模型输出不确定，必须保存 prompt、seed、模型版本和参考资产。

## 八、与现有底座的关系

不需要重做项目/渲染/QA 骨架。建议按以下顺序复用：

1. CreativePackage 作为故事制作包；
2. 创作脉络台作为故事/角色/分场/资产/时间线的 UI；
3. Remotion 作为时间线组装器；
4. External Pack 承担图像/视频/TTS/音乐生成；
5. Asset Pack 承担角色包、场景包、声音包、音乐包；
6. Skill 承担故事/连续性/镜头/音频判断；
7. Workflow 承担多步骤编排与审批。

## 九、下一步

先做 P0，不引入图像/视频生成模型：

```text
story → characters → scenes → shots
→ 静态图 / 占位图
→ TTS 旁白
→ 字幕
→ Remotion 时间线组装
→ QA
→ 60–90 秒故事短片
```

P0 验证通过后，再接 Image Gen Adapter 和 Character Pack。这样风险和成本最低，也能最快验证“故事视频”这条产品线是否成立。
