# 剪辑架构复核（第二轮）：结构性决策

> 日期：2026-10-19
> 结论：方向仍然成立，但在开工前有 **7 个结构性决策未冻结**。不冻结就写代码，必然出现三套表示、两套渲染、归属不清。

## 一、当前事实（证据）

- 插件只有：`audio-production`、`audio-intelligence`、`media-visual`、`media-workbench`；
- 没有独立 Renderer 插件；渲染入口仍是 `media.workbench.render.start/status`；
- `media.workbench.timeline.plan` 是“镜头制作计划”，不是 Timeline；
- 没有 `timeline.*` / `clip.*` 能力；
- 现有表示至少三种：CreativePackage.storyboard、项目 recipe/storyboard、未来 Timeline。

## 二、七个未冻结决策

### D1. Timeline 内核归谁

选项：

- A：留在 `media-workbench`，以 `media.workbench.timeline.*` 暴露；
- B：拆 `com.himind.media-edit` 插件，Workbench 依赖它。

建议：**先 A**。Timeline 是应用领域模型，当前只有 Workbench 一个消费者；按“出现第二消费者再拆”的原则，不应先拆插件。等 Skill/Workflow 或第二个应用需要时再抽。

### D2. 渲染入口只能有一个

不要再造 `timeline.render` 作为第二渲染器。正确做法：

- 保留 `media.workbench.render.start` 作为唯一入口；
- 增加 `timeline` 输入模式（或 `render.timeline` 的内部适配）；
- Renderer 插件抽取按 R-01 单独推进，不在 M0 里做。

### D3. 媒体身份模型

现在 `asset_id`、`artifact_id`、`media_id`、`path` 混用。必须先定义：

```json
{
  "media_ref": {
    "id": "media-...",
    "kind": "video|image|audio",
    "uri": "workspace://...",
    "sha256": "...",
    "duration_frames": 1800,
    "fps": 30,
    "color_space": "sdr|hdr",
    "license": "..."
  }
}
```

没有 MediaRef，trim/proxy/relink 都无法稳定。

### D4. Revision / 并发 / 撤销

需要同时定义：

- Creative revision、Project revision、Timeline revision、Artifact revision；
- 命令日志（append-only）与撤销栈归属；
- AI 客户端与 UI 同时编辑时的乐观锁与冲突提示；
- 每次提交的 diff 与批准状态。

建议：Timeline revision 与命令日志放在 Workbench；AI 只能提交命令，不能直接写文件。

### D5. 模式矩阵

现有模式已很多：项目、创作脉络、预览、分镜、代码审阅。再加“简易/专业”会失控。

建议固定：

```text
创作（Brief/研究/脚本/分镜）
剪辑（简易 ↔ 专业）
交付（QA/导出）
```

媒体池为全局面板；分镜是剪辑里的装配视图，不再占顶层。

### D6. 宏 / 模板 / 预设契约

- Easy Action = 宏：输入、前置条件、命令模板、预览、失败回退；
- 模板 = 带槽位的 Timeline；
- 预设 = Asset Pack（字幕样式/转场/文字/音乐/导出）。

三者都需要 schema，不能靠代码硬编码。

### D7. 每个 revision 的 QA 门禁

现在 QA 在渲染后。时间轴编辑后应先跑：

- clip 重叠/越界/空轨；
- 音画链接一致性；
- 字幕重叠/空档/漂移；
- 响度与峰值。

QA 失败不得进入渲染。

## 三、结构性矛盾（必须消除）

1. **media-edit 插件 vs Workbench 领域模型**：先 A，别急着拆；
2. **timeline.render vs 现有 render.start**：只保留一个入口；
3. **媒体 → 剪辑 → 交付 vs 创作/剪辑/交付**：采用后者；
4. **storyboard / timeline / template slots 三套表示**：明确 storyboard 为意图、timeline 为结果、template 为模式；
5. **浮点秒 vs 帧时间**：统一帧。

## 四、修正后的目标架构

```text
Workbench（应用领域）
├─ Creative / Storyboard（意图）
├─ Timeline 内核（纯命令 + 命令日志 + 撤销）
├─ Media Pool（MediaRef）
├─ Easy Macros（声明式）
└─ UI（创作 / 剪辑简易 / 剪辑专业 / 交付）

Renderer（唯一入口 media.workbench.render.start）
Plugins：audio.* / visual.* / Asset Packs
```

## 五、M0 重新定义

1. MediaRef + Timeline IR（帧）；
2. 命令：insert / overwrite / blade / lift / ripple_delete；
3. 校验 + 命令日志 + 撤销；
4. 单一 render 入口消费 Timeline；
5. 真实素材 60s→30s、3 段拼接、SHA 一致；
6. 通过后再做时间轴 UI 与简易模式。

## 六、结论

方向正确，但当前最大风险不是“功能不够”，而是“归属不清”。
**先冻结 7 个决策，再写 M0；否则剪辑层会把创作、渲染、交付一起拖乱。**
