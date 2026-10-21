# 剪辑能力完整实施方案

> 日期：2026-10-19
> 定位：AI 视频生产系统的精修面；一键成片是入口，专业剪辑是退路。
> 原则：单一时间轴、单一渲染入口、命令驱动、帧时间、可回退、可沉淀。

## 一、冻结的 7 个决策（ADR）

| ADR | 决策 | 结论 |
| --- | --- | --- |
| ADR-EDIT-01 | Timeline 内核归属 | 先留在 Workbench，以 `media.workbench.timeline.*` 暴露；出现第二消费者再拆插件 |
| ADR-EDIT-02 | 渲染入口 | 只保留 `media.workbench.render.start`；不新增第二渲染器 |
| ADR-EDIT-03 | 媒体身份 | 统一 MediaRef：id/uri/sha/时长/帧率/色彩/许可证 |
| ADR-EDIT-04 | Revision/并发/撤销 | Timeline revision + append-only 命令日志 + 乐观锁 |
| ADR-EDIT-05 | 模式矩阵 | 创作 / 剪辑（简易↔专业）/ 交付；媒体池全局面板 |
| ADR-EDIT-06 | 宏/模板/预设 | 宏与模板有 schema；预设作为 Asset Pack |
| ADR-EDIT-07 | QA 门禁 | 每个 revision 先过结构/同步/响度 QA，再允许渲染 |

## 二、目标架构

```text
Workbench（应用领域）
├─ Creative / Storyboard（意图）
├─ Timeline 内核（纯命令 + 校验 + 命令日志 + 撤销）
├─ Media Pool（MediaRef + Proxy）
├─ Easy Macros（声明式）
├─ UI（创作 / 剪辑简易 / 剪辑专业 / 交付）
└─ Render 入口（media.workbench.render.start）

Plugins
├─ audio-production / audio-intelligence
├─ media-visual（QA/切点/联系表/放大）
└─ Asset Packs（模板/预设/字体/音乐/转场）
```

## 三、数据模型

### 3.1 MediaRef

```json
{
  "id": "media-...",
  "kind": "video|image|audio",
  "uri": "workspace://relative/path.mp4",
  "proxy_uri": "workspace://relative/proxy.mp4",
  "sha256": "...",
  "duration_frames": 1800,
  "timebase": {"num": 30, "den": 1},
  "width": 1080,
  "height": 1920,
  "color_space": "sdr|hdr",
  "license": "CC-BY-4.0",
  "commercial_use": true
}
```

### 3.2 Timeline v1

```json
{
  "schema": "himind.timeline.v1",
  "revision": 3,
  "timebase": {"num": 30, "den": 1},
  "duration_frames": 2160,
  "canvas": {"width": 1080, "height": 1920},
  "tracks": [
    {"id": "V1", "kind": "video", "index": 1, "muted": false, "locked": false, "clips": []},
    {"id": "A1", "kind": "audio", "index": 1, "muted": false, "locked": false, "clips": []},
    {"id": "S1", "kind": "subtitle", "index": 1, "muted": false, "locked": false, "clips": []}
  ],
  "markers": []
}
```

Clip：

```json
{
  "id": "clip-...",
  "media_id": "media-...",
  "track_id": "V1",
  "start_frame": 0,
  "duration_frames": 195,
  "source_in_frame": 60,
  "source_out_frame": 255,
  "speed": 1.0,
  "linked_group": "g1",
  "locked": false,
  "transform": {"scale": 1.0, "x": 0, "y": 0, "crop": null},
  "transition_in": null,
  "transition_out": null
}
```

### 3.3 命令

```json
{
  "command_id": "cmd-...",
  "op": "blade|insert|overwrite|lift|ripple_delete|trim|roll|slip|slide|link|unlink",
  "expected_revision": 3,
  "payload": {"clip_id": "clip-...", "at_frame": 300}
}
```

结果：

```json
{
  "applied": true,
  "revision": 4,
  "diff": {"added": [], "removed": [], "changed": []},
  "blockers": []
}
```

### 3.4 宏与模板

- `himind.edit-macro.v1`：inputs、preconditions、commands（带变量）、preview；
- `himind.edit-template.v1`：slots（media/text/music）、timeline_template、constraints；
- 预设：`himind.asset-pack.v1` 中的 caption/transition/text/music/export 资源。

## 四、能力面

### Workbench（时间轴与项目）

| 能力 | 说明 |
| --- | --- |
| `media.workbench.timeline.get` | 读取 Timeline + revision |
| `media.workbench.timeline.apply` | 应用命令批次，返回 diff |
| `media.workbench.timeline.validate` | 结构/重叠/越界/链接校验 |
| `media.workbench.timeline.undo` / `redo` | 命令日志回放 |
| `media.workbench.timeline.diff` | 两个 revision 的差异 |
| `media.workbench.render.start` | 接收 timeline 或 timeline_path，单一渲染入口 |

### 插件

| 能力 | Owner |
| --- | --- |
| `visual.video.qa` / `scenes` / `contact-sheet` | media-visual |
| `audio.align` / `subtitle.from-alignment` | audio-intelligence |
| `audio.pipeline.render` / `mix` | audio-production |
| `visual.upscale` | media-visual |

### 媒体操作（后续）

- proxy / transcode / extract-audio / burn-in：优先放 media-visual 或独立 media-ops，不进入时间轴内核。

## 五、UI 实施

### 模式矩阵

```text
创作：Brief / 研究 / 脚本 / 分镜
剪辑：简易（默认） ↔ 专业
交付：QA / 预设 / 导出
```

媒体池为全局面板。

### 剪辑布局

```text
Topbar：模式 · 项目面包屑 · 任务中心 · 主题
Left：媒体池（项目/资产/产物/音频/字幕）
Center：源监视器 | 节目监视器
Right：Inspector（片段/变换/速度/音频/转场）
Bottom：时间轴（V2/V1/A1/A2/S1）+ 工具栏 + 走带
```

### 简易模式

- 只露出：素材托盘、预览、简化时间轴、一键成片、自动字幕、卡点、精修、导出；
- 不露出轨道目标、Trim 模式、复杂参数；
- 所有智能动作先生成 diff 预览，再提交 revision。

### 专业模式

- 多轨、Blade/Lift/Ripple Delete/Insert/Overwrite；
- Inspector 与转场；
- 快捷键对齐 Resolve 子集。

## 六、里程碑与工作分解

### M1 剪辑内核（约 9 个开发日）

| 任务 | 交付 | 验收 |
| --- | --- | --- |
| T1.1 | MediaRef + Timeline v1 schema | schema/校验测试 |
| T1.2 | 命令：insert/overwrite/blade/lift/ripple_delete | 表驱动测试 |
| T1.3 | 命令日志 + revision + undo/redo | 10 步撤销一致 |
| T1.4 | timeline.validate | 重叠/越界/空轨/链接 |
| T1.5 | render.start 接受 timeline | 单视频轨+音频轨输出 |
| T1.6 | 真实素材 60s→30s、3 段拼接 | SHA 一致 |

**门禁**：Go test、golden fixture、两次渲染 SHA 一致。

### M2 媒体与时间轴 UI（约 10 个开发日）

| 任务 | 交付 | 验收 |
| --- | --- | --- |
| T2.1 | 媒体池 + MediaRef 列表 | 拖入轨道 |
| T2.2 | proxy/thumbnail 缓存 | 100 片段不卡 |
| T2.3 | 源/节目双监视器 | 入出点与三点编辑 |
| T2.4 | 时间轴轨道/播放头/吸附/缩放 | 键盘与鼠标 |
| T2.5 | 基础编辑工具 | Blade/Lift/Ripple/Insert/Overwrite |
| T2.6 | 撤销/重做 UI | 与命令日志一致 |

**门禁**：100 镜头项目 e2e、时间轴操作截图、性能基线。

### M3 易用层与专业修剪（约 11 个开发日）

| 任务 | 交付 | 验收 |
| --- | --- | --- |
| T3.1 | 简易模式壳 | 3 分钟一键成片 |
| T3.2 | 自动字幕/去静音/卡点宏 | 可预览可撤销 |
| T3.3 | 自动重构 9:16 | 主体跟随/安全区 |
| T3.4 | 模板槽位与剪同款 | 6 段素材适配 |
| T3.5 | RippleTrim + Link A/V | 修剪语义正确 |
| T3.6 | 转场/画中画 Inspector | 可保存/回放 |
| T3.7 | 简易↔专业双向交接 | 同一 Timeline |

**门禁**：模板替换 e2e、宏 diff、专业模式可回退。

### M4 交付与硬化（约 9 个开发日）

| 任务 | 交付 | 验收 |
| --- | --- | --- |
| T4.1 | 每 revision QA 门禁 | 结构/同步/响度 |
| T4.2 | 字幕烧录 | 成片字幕可校验 |
| T4.3 | 音频总线/ducking | 旁白压 BGM |
| T4.4 | 抖音/B站/YouTube 预设 | 平台规范 |
| T4.5 | 性能与缓存 | 长素材/100 片段 |
| T4.6 | 命令日志/审计/权限 | 可追溯 |

**门禁**：平台预设导出、长素材压力、权限与审批。

## 七、验收场景（端到端）

1. **60s → 30s 快剪**：trim + split + ripple delete；
2. **三点编辑**：源入出点 → insert；
3. **Ripple vs Roll vs Slip**：语义与总长正确；
4. **9:16 重构**：横屏素材主体跟随；
5. **一键成片**：素材 + 音乐 + 时长 → 可预览；
6. **模板剪同款**：替换 6 段素材自动适配；
7. **AI 精剪**：“把第二段剪到 5 秒并加 0.3s 淡入”生成命令并执行；
8. **确定性**：同一 Timeline 两次渲染 SHA 一致；
9. **失败恢复**：QA 失败可定位、可重试、不丢稿。

## 八、测试策略

- 时间轴代数：表驱动 + property tests（命令序列幂等/可逆）；
- golden timelines：固定 fixture，比较 JSON diff；
- 渲染确定性：同帧集两次渲染哈希；
- UI e2e：18082 + CDP，覆盖简易/专业/交付；
- AI 命令：自然语言 → 命令序列的契约测试；
- 性能：100 片段、30 分钟素材、代理开关。

## 九、迁移与兼容

- 现有项目继续使用 storyboard；新 Timeline 通过“编译”生成；
- 功能开关 `timeline_v1`，默认关闭，M2 后逐步开启；
- 不改变 CreativePackage / Project 现有字段；
- 旧项目打开时显示“生成时间轴”入口，不静默迁移。

## 十、风险与缓解

| 风险 | 缓解 |
| --- | --- |
| 双真值 | storyboard 意图 / timeline 结果；rebase diff |
| 第二渲染器 | 单一 render.start；媒体操作与渲染分层 |
| 时间漂移 | 帧时间 + 有理数 timebase |
| 长素材卡顿 | proxy + 缩略图缓存 + 虚拟化 |
| AI 误操作 | 命令预览/diff/撤销/审批 |
| 范围膨胀 | 价值决策规则；Deferred 清单 |

## 十一、自主执行协议

每次推进：

1. 读本文件与 backlog，领取第一个未阻塞任务；
2. 只做一个可验证切片（代码 + 测试 + 真实调用）；
3. 跑门禁：Go test / typecheck / build / plugin validate/package/test/confirm；
4. UI 改动做 18082 e2e；
5. 更新 backlog 与 progress 台账；
6. 遇到环境阻塞记录并转下一项。

## 十二、完成定义

- 9 个验收场景全部通过；
- 单一 Timeline 与单一 render 入口；
- 命令可撤销、可 diff、可回放；
- 简易模式 3 分钟出第一版；
- 专业模式可完成基本精剪；
- 交付预设通过平台规范；
- 文档、制品、截图、SHA 齐全。
