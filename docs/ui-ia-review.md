# 自媒体创作台 UI / 信息架构审查

> 日期：2026-10-19
> 范围：`com.himind.media-workbench` 创作台（项目工作台 + 创作脉络）
> 方法：源码结构走查 + 18082 真实实例 DOM/样式采样 + 行业同类产品对标
> 结论：视觉已从“纯黑工具”提升到可用的深/浅双主题；下一阶段主要矛盾是**信息架构**，不是配色。

## 一、对标对象

| 产品 | 类型 | 可借鉴 | 不该照搬 |
| --- | --- | --- | --- |
| DaVinci Resolve | 专业剪辑/调色/交付 | 页面级模式（Media/Edit/Fusion/Color/Fairlight/Deliver）、常驻 Media Pool、Inspector、Timeline | 页面过多、学习曲线陡 |
| Premiere Pro | 专业剪辑 | 工作区预设、面板可停靠、时间轴为核心 | 面板自由度带来的复杂度 |
| Final Cut Pro | 专业剪辑 | 磁性时间轴、Browser/Inspector 双层结构、角色（Roles） | 强绑定的剪辑心智 |
| CapCut | 大众短视频 | 左侧素材/文本/音频/贴纸、中央画布、底部时间轴、模板优先 | 功能堆叠、层级浅 |
| Descript | AI 优先创作 | 文稿即时间轴、Scene/Storyboard 视图、AI 编辑闭环 | 过度依赖转写 |
| Runway | AI 生成 | 资产库 + 生成画布 + 时间轴、模型能力可见 | 生成参数前台化 |
| Figma | 设计协作 | 左侧 Pages/Layers/Assets、右侧属性、命令面板 | 画布优先的心智 |
| Linear | 高效工具 | 命令面板、键盘优先、密度与节奏 | 纯列表、不适合媒体 |

我们的定位应介于 **Descript（AI 优先叙事）** 与 **CapCut（短视频交付）** 之间，同时保留 Resolve 式的“审阅/交付”严谨度。

## 二、当前信息架构

```text
App Shell（48px 顶栏）
├─ 左：项目列表（232px，仅项目）
├─ 中：审阅/预览 | 计划 | 分镜（标签随项目类型变化）
├─ 右：Inspector（296px）
├─ 下：DurationRuler（112px）+ StatusBar（30px）
└─ 模态：新建项目、风格库

创作脉络（全屏覆盖，56px 顶栏）
├─ 左：创作脉络（300px，步骤 1..N）
├─ 中：Brief / Research / Script / Storyboard / Production
├─ 右：Inspector（320px，逻辑/来源/版本/计划）
└─ 底部：保存栏
```

证据位置：

- 主壳：`ui/src/app.tsx`（顶栏动作、`mainView`、三栏布局）
- 项目树：`ui/src/features/project-tree.tsx`
- 创作脉络：`ui/src/features/creative.tsx`（顶栏、五段内容、右栏四页签）
- 状态栏：`ui/src/features/status-bar.tsx`
- 主题 token：`ui/src/styles.css`

## 三、关键问题（按严重度）

### P0-1 两套并行 IA，项目与创作包关系不可见

创作脉络是全屏覆盖层，进入/退出靠“返回”；创作包与项目之间没有常驻的面包屑或血缘视图。

影响：

- 用户不知道“我现在在创作包还是项目”；
- materialize 之后，项目与创意包的 revision 关系只存在于数据里；
- 对标产品都保持“同一工作区 + 模式切换”，而不是两个应用。

建议：

- 统一为一个工作区壳：`Workspace > Project > Creative` 常驻面包屑；
- 创作包成为左栏一等对象，materialize 后显示 `creative@rN → project` 血缘；
- 用模式切换（Script / Storyboard / Preview / Deliver）替代全屏覆盖。

### P0-2 主视图标签语义不稳定

同一位置在不同项目类型下叫“审阅 / 预览 / 分镜”，且“分镜”对应两个不同组件（`CodeStoryboardPane` 与 `StoryboardEditor`）。

影响：

- 用户无法建立稳定心智模型；
- AI 与文档里的“分镜”指代不一致；
- 同一操作在不同项目类型下位置不同。

建议：固定四个模式，标签与顺序在所有项目类型一致：

```text
剧本 → 分镜 → 预览 → 交付
```

### P0-3 缺少全局对象导航

左栏只有项目；创意包、资产包、风格、模板、产物分散在模态和其他入口。

对标：

- Figma：Pages / Layers / Assets；
- Resolve：Media Pool / Effects；
- CapCut：Media / Audio / Text / Stickers。

建议左栏改为分区式：

```text
项目
创意包
资产（Asset Packs / 风格 / 模板）
产物（预览 / 成片 / QA 报告）
```

### P1-1 顶栏动作过载

主顶栏 8 个动作，创作顶栏 10+ 个动作（返回、新建、脉络、检查、主题、保存、关闭、状态 badges），没有溢出菜单或命令面板。

建议：

- 顶栏只保留：主操作（生成/保存）+ 任务中心 + 主题 + 更多；
- 次级动作进入 `Ctrl/Cmd+K` 命令面板；
- 批量操作放入选择态工具条。

### P1-2 Inspector 不是“选中对象属性”

右栏是固定页签，而不是随选中对象变化；空态没有告诉用户“选中什么能看到什么”。

建议：

- 选择驱动的 Inspector：镜头 / 场景 / 字幕 / 音频 / 素材 / 项目；
- 每个类型有明确的空态与必填项；
- 校验结果与历史放在同一栏的“检查/历史”页。

### P1-3 时间轴缺位

底部只有 112px 的 DurationRuler；没有轨道、片段、音频、字幕。

影响：

- 专业用户无法做音画对位；
- 字幕/旁白/BGM 的关系不可见；
- 与“AI 生成 + 人工打磨”的定位不匹配。

建议：

- 底部时间轴可拖拽调整高度（最小 160px）；
- 轨道：画面 / 字幕 / 旁白 / BGM / SFX；
- 分镜模式与时间轴双向联动。

### P1-4 任务与 QA 反馈分散

StatusBar 只显示 job/产物/约束/连接；渲染、预览、音频 QA、工作流运行没有统一任务中心。

建议：

- 顶栏任务中心（进行中/失败/完成）；
- 点击任务定位到对应项目/步骤；
- QA 报告作为一等产物，可从任务中心打开。

### P2-1 搜索范围窄

项目树只搜 name/id/template/style；没有跨对象搜索。

建议：全局搜索（项目、创意、资产、产物、命令）。

### P2-2 排版与密度

13px 正文、10px 标签在专业工具里可接受，但元信息偏小且对比不足。

建议：

- 类型刻度 11/12/13/15/20；
- 元信息不小于 11px；
- 提供舒适/紧凑两档密度。

### P2-3 可访问性与键盘

多数 icon button 只有 `title`，没有 `aria-label`；缺少 `:focus-visible` 样式与快捷键映射。

建议：

- 所有图标按钮补 `aria-label`；
- 统一 focus ring（`accent-line + accent-soft`）；
- 建立快捷键表：播放/暂停、上一/下一镜头、保存、命令面板、切换主题。

## 四、目标信息架构

```text
Workspace（常驻）
├─ Topbar：面包屑 · 全局搜索/命令面板 · 任务中心 · 主题 · 更多
├─ Left rail（可折叠分区）
│   ├─ 项目
│   ├─ 创意包
│   ├─ 资产（Asset Packs / 风格 / 模板）
│   └─ 产物（预览 / 成片 / QA）
├─ Center：Script | Storyboard | Preview | Deliver（稳定标签）
├─ Right：Selection Inspector（镜头/场景/字幕/音频/素材）+ 检查/历史
└─ Bottom：Timeline（可调高度）+ StatusBar（任务/QA/连接）
```

对象模型：

```text
Workspace
└─ CreativePackage (revision, approval)
   └─ Project (materialized_from)
      ├─ Scenes / Shots
      ├─ Audio (narration, BGM, SFX)
      ├─ Subtitles
      └─ Artifacts (preview, master, web, QA)
```

## 五、设计系统建议

| 维度 | 现状 | 建议 |
| --- | --- | --- |
| 颜色 | 双主题已建立，仍有内联 rgba | 全部收敛到语义 token；对比度 ≥ AA |
| 间距 | 4/8/12 混用 | 统一 4/8/12/16/24 |
| 圆角 | 7/8/10/12 混用 | 控件 8，面板 12 |
| 字号 | 10/11/12/13 混用 | 11/12/13/15/20 |
| 阴影 | 仅暗色少量 | 两级 elevation，浅色更克制 |
| 状态 | 部分空态为纯文字 | 每个面板定义 empty/loading/error/blocked |
| 动效 | 120ms 过渡 | 120–180ms，尊重 reduced motion |

## 六、实施路线

### Phase 1（低风险，先做）

1. 固定主视图四标签：剧本 / 分镜 / 预览 / 交付；
2. 顶栏动作分层：主操作 + 任务中心 + 更多；
3. 图标按钮补 `aria-label` 与 `:focus-visible`；
4. 清理剩余内联颜色，补浅色对比度。

### Phase 2（IA 重构）

5. 左栏分区：项目 / 创意包 / 资产 / 产物；
6. 常驻面包屑与创作包→项目血缘；
7. 选择驱动 Inspector + 空态契约。

### Phase 3（专业能力）

8. 可调时间轴与多轨；
9. 全局任务中心与 QA 报告入口；
10. 全局搜索 + 命令面板 + 快捷键表。

### Phase 4（成熟度）

11. 舒适/紧凑密度；
12. 工作区预设（创作 / 剪辑 / 审阅 / 交付）；
13. 引导与渐进披露。

## 七、验收标准

- 同一项目类型下，主视图标签与顺序稳定；
- 任意页面都能看到 `Workspace > Project > Creative` 位置；
- 任意选中对象都有明确的 Inspector 内容或空态；
- 顶栏主操作 ≤ 3 个，其余可在两次点击内到达；
- 键盘可完成：播放/暂停、切镜头、保存、命令面板、切主题；
- 浅色/深色均通过对比度与截图走查。

## 八、证据与截图

- 浅色：`.extension-state/bridge-e2e-full/theme-light.png`
- 深色：`.extension-state/bridge-e2e-full/theme-dark.png`
- 关键源码：`ui/src/app.tsx`、`ui/src/features/creative.tsx`、`ui/src/features/project-tree.tsx`、`ui/src/features/status-bar.tsx`、`ui/src/styles.css`
