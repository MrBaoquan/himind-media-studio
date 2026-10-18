---
name: video-capability-author
description: 当现有 Shot 无法满足需求时，设计并实现新的 Remotion 视觉能力，完成契约、模板、黄金样例和门禁，再沉淀为可复用能力。
---

# 视频能力作者

## 目标

把“现有模板不够用”的瞬间，变成一次可沉淀的能力建设，而不是一次性写死一条视频。

```text
需求
→ 能力缺口判断
→ 能力提案
→ 用户确认
→ Shot + Technique
→ Contract
→ Template 注册
→ Fixture
→ 预览 / 渲染 / QC
→ Candidate
```

## 九阶段流程

### 0. 开源 prior art（新增能力前必做）

在写 Shot、Style Pack 或 Renderer 之前，先搜索是否已有专门的开源库/项目：

- 布局：线网示意/地理布局、八向网格、站名避让；
- 动画：SVG stroke-dashoffset、时间轴引擎、视频导出；
- 数据：GTFS / OSM / 官方线网数据；
- 导出：PNG 帧 + ffmpeg / Remotion / Motion Canvas。

调研结论写入 `docs/oss-evaluation/<topic>.md`，按 `docs/oss-evaluation/README.md` 模板记录。

硬规则：

- MIT / ISC / Apache-2.0 优先；
- GPL / AGPL 不直接 vendor，只能作为可选外部工具或仅参考；
- 商业求解器依赖不能成为默认安装前提；
- 无许可证的仓库视为不可复用；
- 先区分“布局库”和“动画库”，不要用动画引擎替代线网布局；
- 能包装成 External Pack / Renderer Plugin 的，不写进创作台核心。

参考：`docs/oss-evaluation/transit-map.md`、`docs/oss-evaluation/label-placement.md`。


### 1. 能力缺口

先调用 `media.workbench.catalog.list` 和 `media.workbench.template.describe`，判断：

- 现有 ready Shot 是否足够；
- 是否只是样式差异；
- 是否需要新的结构或技法；
- 是否需要新数据契约。

输出 `himind.video.capability-proposal.v1`。

### 2. 能力提案

提案必须包含：

- `id`、`name`；
- `structure`；
- `reuse_for`；
- `data_contract`；
- `visual_direction`；
- `acceptance`。

**确认点 1：用户确认方向。**

### 3. 数据契约

先写 contract，再写画面：

```text
catalog/contracts/<capability>.schema.json
```

**确认点 2：用户确认数据口径。**

### 4. 视觉方案

明确：

- 构图与层级；
- 字号与安全区；
- 颜色令牌；
- 动效节奏；
- 标签避让；
- 空态/错误态。

### 5. 实现 Shot

一个镜头一个文件：

```text
renderer/remotion/shots/<Shot>.jsx
renderer/remotion/techniques/<technique>.js
```

硬规则：

- 动画是帧号纯函数；
- 布局算法放 technique；
- 不测 DOM；
- 不全屏随机；
- 颜色只来自 style tokens；
- 文字用 `foundation/text.js` 估算。

### 6. 注册

补齐：

```text
shots/registry.js
index.jsx
catalog/templates.json
catalog/fixtures/manifest.json
```

**确认点 3：用户确认预览。**

### 7. 门禁

必须跑：

```text
go test
recipe.validate
fixture 存在性与 SSIM
verify-render.py
两次渲染确定性
```

### 8. Candidate

生成 `himind.video.capability-descriptor.v1`：

- Shot；
- Technique；
- Contract；
- Template；
- Fixture；
- QC 证据；
- 标签。

### 9. 提升

只有通过门禁后才能把 status 改为 `ready`。

**确认点 4：用户确认沉淀。**

## 首个参考案例

`SankeyFlow`：

- Shot：`renderer/remotion/shots/SankeyFlow.jsx`
- Technique：`renderer/remotion/techniques/sankey.js`
- Contract：`catalog/contracts/sankey.schema.json`
- Template：`sankey-flow`
- Fixture：`catalog/fixtures/sankey-flow/`

它不是高铁路线专用，而是“总量流向多个目标”的通用视觉能力。

## 禁止事项

- 不在渲染时执行任意用户 JSX；
- 不把 Native Shot 拆成 plugin；
- 不复制 anything2explainer 的固定风格或受许可限制的代码；
- 不在没有 contract、fixture、template 时把能力标成 ready；
- 不把一次性镜头当成可复用能力。

