# 剪辑 M1-M4 完成审计

> 日期：2026-10-20
> 对象：Workbench 0.57.0
> 结论：`docs/editing-implementation-plan.md` 的 M1-M4，以及 backlog `E-01…E-29`、`C-01…C-07` 均已落到代码、测试、真实页面或真实产物。

## 一、里程碑审计

| 里程碑 | 任务 | 状态 | 证据 |
| --- | --- | --- | --- |
| M1 | MediaRef + Timeline schema | PASS | `timeline_media.go`、timeline 校验与 catalog 调用 |
| M1 | insert/overwrite/blade/lift/ripple_delete | PASS | `timeline_core_test.go` 表驱动测试 |
| M1 | 命令日志 + revision + undo/redo | PASS | `timeline_history_test.go` 10 步一致；`timeline-commands.jsonl` |
| M1 | timeline.validate | PASS | 重叠/越界/空轨/链接/字幕/关键帧检查 |
| M1 | render.start 接收 timeline | PASS | 单一入口；时间轴成片与直接 export SHA 一致 |
| M1 | 真实素材 60s→30s | PASS | 历史证据：72s 成片取两段 30s，SHA `607d98d0…` |
| M1 | 确定性 | PASS | 当前 0.57 同 Timeline 两次输出 SHA 均为 `e92cdb83…` |
| M2 | 媒体池 + MediaRef 列表 + 拖入轨道 | PASS | 拖拽 e2e：5→7 clips（落点分割+插入） |
| M2 | proxy + 100 clips 性能 | PASS | 540p proxy；100 clips 220ms |
| M2 | 源/节目双监视器 | PASS | 源预览 Blob、节目最近成片、三点插入 e2e |
| M2 | 时间轴轨道/播放头/吸附/缩放 | PASS | 鼠标 + 缩放 + 15 帧吸附 |
| M2 | 基础编辑工具 | PASS | Blade/Lift/Ripple/Insert/Overwrite |
| M2 | 撤销/重做 UI | PASS | 删除→撤销→重做 e2e |
| M3 | 简易模式壳 | PASS | 简易/专业/交付三模式 |
| M3 | 自动字幕/去静音/卡点 | PASS | 12 cues；静音 2s 去除；7 beat markers |
| M3 | 自动重构 | PASS | 9:16/1:1/16:9；内容感知非黑边中心 |
| M3 | 模板槽位 | PASS | 3/5/6 槽模板；六段故事模板 |
| M3 | RippleTrim + Link A/V | PASS | 内核测试 + 专业 UI |
| M3 | 转场/画中画 Inspector | PASS | xfade、PIP、淡入淡出、保存/回放 |
| M3 | 简易↔专业双向交接 | PASS | 同一 Timeline 状态与持久化 |
| M4 | revision QA 门禁 | PASS | timeline.qa + render 前阻断 |
| M4 | 字幕烧录 | PASS | `timeline-m3.mp4` 12s；`timeline-color-keyframes.mp4` |
| M4 | 音频总线/ducking | PASS | narration/music sidechain；-16.05 LUFS |
| M4 | 平台预设 | PASS | 母版/Web/抖音/B站/YouTube/西瓜 |
| M4 | 性能与缓存 | PASS | 100 clips；proxy；媒体 catalog |
| M4 | 命令日志/审计 | PASS | append-only JSONL + timeline.diff |

## 二、Backlog 审计

- `E-01…E-29`：全部 `DONE`，包含 retime、PIP、颜色关键帧、速度曲线、timeline.diff、自然语言精剪命令计划。
- `C-01…C-07`：全部 `DONE`，包含简易模式、智能字幕/去静音/卡点、内容感知重构、模板、Asset Pack、简易↔专业、平台导出。

## 三、验收场景

1. 60s→30s 快剪：PASS（历史 30s 输出 SHA `607d98d0…`）
2. 三点编辑：PASS（源入出点 → insert）
3. Ripple vs Roll vs Slip/Slide：PASS（内核测试 + UI）
4. 9:16 重构：PASS（9:16/1:1/16:9 + 内容感知）
5. 一键成片：PASS（3 clips、QA、QC）
6. 六段剪同款：PASS（六拍模板）
7. AI 精剪：PASS（“第二段剪到 1.5s 并加 0.3s 淡入” → 2 条命令 + preview）
8. 确定性：PASS（两次 SHA 完全一致）
9. QA 失败恢复：PASS（blocker 可定位、未静默渲染）

## 四、最终制品

- Workbench 0.57.0：`dist/plugins/com.himind.media-workbench-0.57.0.hmpkg`
- SHA-256：`38927a5186f9d19d951aa38b2b133227f687ef1c8165271012e55756820d558a`
- Asset Pack：`editing-kit@1.0.0`，SHA `7311cf7085c50f5c420b270e38c393c9ad567192aed8b522be97fd12c14b7c2e`

## 五、已知边界

- 内容感知重构是确定性非黑边中心对齐，不是逐帧语义主体分割。
- 速度曲线使用分段恒定速度烘焙，不是单条任意 Bézier 速度曲线。
- 这些边界不阻塞 M1-M4 当前契约，已在 backlog 与报告中显式记录。
