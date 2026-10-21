# 达芬奇心智的剪辑能力方案

> 日期：2026-10-19
> 目标：按 DaVinci Resolve「Edit 页」的心智模型，为创作台补齐通用片段剪辑；保留 AI 优先与可回退，不做完整 NLE。

## 一、核心映射

| 达芬奇 | 创作台 | 说明 |
| --- | --- | --- |
| Media Pool | 媒体池 | 项目、创意包、Asset Pack、生成产物、音频、字幕 |
| Source Viewer | 源监视器 | 选中素材的入点/出点与预览 |
| Timeline Viewer | 节目监视器 | 当前时间轴输出 |
| Inspector | 片段检查器 | 变换、速度、音频、转场、字幕 |
| Edit 页 | 剪辑模式 | 时间轴 + 双监视器 + 工具栏 |
| Deliver 页 | 交付模式 | 渲染预设、QA、导出 |
| Timeline | 多轨时间轴 | V1/V2、A1/A2、S1 |
| Storyboard | 装配视图 | 生成初始时间轴，不替代精剪 |

创作台模式固定为：

```text
媒体 → 剪辑 → 交付
```

分镜作为“装配视图”进入剪辑模式，不再单独占一个顶层模式。

## 二、信息架构

```text
Topbar：媒体 / 剪辑 / 交付 · 项目面包屑 · 任务中心 · 主题
Left：媒体池（项目 / 资产 / 产物 / 音频 / 字幕）
Center：源监视器 | 节目监视器（窄屏可切换单监视器）
Right：Inspector（片段 / 变换 / 速度 / 音频 / 转场）
Bottom：时间轴（V2 / V1 / A1 / A2 / S1）+ 工具栏 + 走带
```

## 三、时间轴数据模型

```json
{
  "schema": "himind.timeline.v1",
  "fps": 30,
  "canvas": {"width": 1080, "height": 1920},
  "duration": 72,
  "tracks": [
    {"id": "V2", "kind": "video", "clips": []},
    {"id": "V1", "kind": "video", "clips": [
      {"id": "c1", "asset_id": "a1", "source_in": 2.0, "source_out": 8.5,
       "start": 0, "duration": 6.5, "speed": 1.0, "linked_group": "g1"}
    ]},
    {"id": "A1", "kind": "audio", "clips": []},
    {"id": "A2", "kind": "audio", "clips": []},
    {"id": "S1", "kind": "subtitle", "clips": []}
  ],
  "markers": [],
  "revision": 3
}
```

关键字段：

- `source_in/source_out`：素材入出点；
- `start/duration`：时间轴位置与长度；
- `speed`：变速；
- `linked_group`：音视频链接；
- `transition_in/out`：转场；
- `revision`：每次操作可 diff、可回退。

## 四、剪辑操作语义

| 操作 | 行为 | 达芬奇对应 |
| --- | --- | --- |
| Insert | 在播放头插入，后续片段后移 | Insert |
| Overwrite | 覆盖目标区间 | Overwrite |
| Lift | 删除片段，保留空隙 | Lift |
| Ripple Delete | 删除并让后续片段前移 | Ripple Delete |
| Blade | 在播放头切开 | Blade |
| Ripple Trim | 改入/出点并推动后续 | Ripple Trim |
| Roll | 移动相邻片段的共同编辑点，总长不变 | Roll |
| Slip | 改素材内容，不移动片段 | Slip |
| Slide | 移动片段，调整相邻片段 | Slide |
| Link/Unlink | 音视频链接 | Link |
| Transition | 默认转场 / 手动时长 | Add Transition |

所有操作都是命令：

```json
{"op": "ripple_delete", "clip_id": "c2", "at": 12.4}
```

命令可校验、可撤销、可由 AI 生成。

## 五、交互与快捷键（Resolve 对齐）

| 键 | 行为 |
| --- | --- |
| Space | 播放/暂停 |
| J / K / L | 倒放 / 停 / 正放 |
| I / O | 标记入点 / 出点 |
| ← / → | 逐帧 |
| Shift + ← / → | 逐秒 |
| B（或 Ctrl+B） | 刀片切开 |
| A | 选择工具 |
| T | 修剪工具 |
| N | 吸附开关 |
| Delete | Lift |
| Shift + Delete | Ripple Delete |
| Ctrl/Cmd + Z / Y | 撤销 / 重做 |
| Ctrl/Cmd + T | 默认转场 |

说明：与宿主冲突的快捷键可配置；AI 对话不依赖快捷键。

## 六、AI 与手工的边界

- **AI 粗剪**：从 Storyboard 生成初始 Timeline；
- **AI 精剪**：输出命令序列（不是直接改像素），例如：
  - “把第 2 段剪到 5 秒”
  - “在 10 秒处切开，删掉后半段”
  - “给第 3 段加 0.3 秒淡入”
- 每条命令先校验、再预览、再提交 revision；
- 手工操作与 AI 操作共用同一条命令总线与撤销栈。

## 七、实施阶段

### P0：可用精剪

1. `himind.timeline.v1` + 命令模型；
2. 媒体池 + 双监视器；
3. 时间轴：轨道、播放头、选择、吸附、缩放；
4. Blade / Lift / Ripple Delete / Insert / Overwrite；
5. `clip.trim/split/concat/retime` + `timeline.render`；
6. 撤销/重做。

### P1：专业修剪

7. Ripple Trim / Roll / Slip / Slide；
8. 音视频 Link 与轨道目标；
9. 转场与 Inspector；
10. 竖屏重构（transform）；
11. 字幕烧录、画中画、ducking。

### P2：交付与调色

12. Deliver 预设（母版/Web/竖屏）；
13. 基础调色/LUT；
14. 关键帧与速度曲线；
15. 多机位。

## 八、验收

- 三点编辑：源入出点 → 插入时间轴；
- Ripple Delete 后后续片段前移；
- Roll 改变编辑点但总长不变；
- Slip 改变素材内容但片段位置不变；
- 撤销/重做 10 步一致；
- AI 命令“把第二段剪到 5 秒并加 0.3s 淡入”产出确定性命令与结果；
- 同一 Timeline 两次渲染 SHA 一致。

## 九、风险

- 不要照搬 Fusion/Color/Fairlight 全部页面；
- Storyboard 与 Timeline 必须主从清晰；
- 长素材需要代理文件；
- 变速/裁剪后字幕与旁白要重新对齐；
- 快捷键与宿主冲突需可配置。
