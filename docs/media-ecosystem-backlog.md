# 媒体生态可持续任务清单

> 状态：living backlog。每次自主推进只领取一个 `TODO` 且未被阻塞的任务，完成后更新状态与证据。
> 原则：一个 Capability 只有一个 Owner；能确定性实现的先确定性实现；模型能力只做 adapter；坑必须沉淀为规则/测试。

## 一、执行协议

1. 读取本文件与 `ultimate-media-architecture-progress.md`；
2. 选择优先级最高、依赖已满足的 `TODO`；
3. 只做一个可验证切片：代码 + 测试 + 真实调用 + 打包（如需）；
4. 更新本文件状态、证据与下一步；
5. 遇到阻塞（权限、外部服务、凭据）记录原因并跳到下一个非阻塞任务；
6. 不新增“只为架构漂亮”的插件；新能力优先进入现有领域插件。

## 二、任务总表

| ID | 优先级 | 工作流 | 状态 | 交付物 | 验收 |
| --- | --- | --- | --- | --- | --- |
| V-01 | P0 | Visual | DONE | `visual.video.qa` / `visual.video.scenes` @0.4.0 | 真实 72s 成片 9 帧、0 blocker、4 切点 |
| V-02 | P0 | Visual | DONE | `visual.video.contact-sheet` @0.5.0 | 真实 72s 成片 3×3 联系表 1080×1920，9 格时间映射 |
| V-03 | P1 | Visual | DONE | `visual.upscale` @0.6.0 | 图片 1080×1920→1620×2880；视频 256→384；SHA 与时长通过 |
| V-04 | P1 | Visual | TODO | `visual.video.keyframes`（切点关键帧） | 每个切点输出 1 帧 + 索引 |
| V-05 | P2 | Visual | TODO | `visual.matte` / `visual.segment` | 需要模型 adapter，先定义契约 |
| V-06 | P2 | Visual | TODO | `visual.video.interpolate` | ffmpeg minterpolate 确定性版本 + 模型 adapter 预留 |
| A-01 | P1 | Audio | TODO | `audio.ducking`（旁白自动压低 BGM） | 旁白区间 BGM 降 6–12 dB，输出新混音 |
| A-02 | P2 | Audio | TODO | `audio.qa.balance`（人声/BGM 平衡） | 返回语音占比与建议 |
| W-01 | P0 | Workflow | TODO | `story-video` Workflow | 主题 → 脚本 → 分镜 → 生成 → QA → 成片 |
| W-02 | P1 | Workflow | TESTED/CONFIRMED（安装受权限阻塞） | `media-qa` Workflow 0.1.1 | 6 能力 preflight ready；contract dry-run passed；安装返回 os error 5 |
| W-03 | P2 | Workflow | TODO | `image-shot` / `video-shot` | 单镜头生成与替换 |
| S-01 | P1 | Skill | TODO | `character-continuity` Skill | 角色参考包 + 一致性检查流程 |
| S-02 | P2 | Skill | TODO | `rights-license` Skill | 素材来源/许可证/商用检查 |
| Q-01 | P0 | Harness | DONE | contact sheet 纳入 `media-qa` | Workflow 第一步 MQ-SHEET 已编排 |
| M-01 | P1 | Market | TODO | Plugin/Asset/Skill/Workflow 市场条目 | 版本、SHA、license、回滚 |
| M-02 | P1 | Market | TODO | commercial profile | 禁止 unknown license 进入商用交付 |
| R-01 | P2 | Renderer | TODO | Renderer 插件抽取复核 | 渲染器独立发版、创作台不硬依赖 |
| E-01 | P0 | Edit | DONE | `himind.timeline.v1` schema + validate | schema/排序/重叠/时长校验；0.45 结构门禁通过 |
| E-02 | P0 | Edit | DONE | `clip.trim/split/concat/retime` | retime 命令 + 0.5×/1×/2× UI + ffmpeg setpts/atempo 渲染，timeline-retime.mp4 通过 |
| E-03 | P0 | Edit | DONE | Timeline 渲染（0.30.0） | V1+A1 经 render.start 输出 H.264+AAC；字幕烧录归 E-20 |
| E-04 | P1 | Edit | DONE | `clip.transform` 竖屏重构 | transform cover/contain 与 9:16 渲染真实通过 |
| E-05 | P1 | Edit | DONE | `clip.transition/overlay/audio` | V2 画中画 overlay、xfade、多音轨与字幕烧录；timeline-pip.mp4 真实通过 |
| E-06 | P1 | Edit | DONE | 时间轴交互：分割/波纹删除/撤销 | 鼠标 + B/Delete/I/O/N/Ctrl+Z 快捷键 e2e 通过 |
| E-07 | P2 | Edit | DONE | `clip.color` / 关键帧 | 颜色关键帧 eq eval=frame + 速度曲线分段烘焙；timeline-color-keyframes / timeline-speed-curve 通过 |
| E-10 | P0 | Edit | DONE | Timeline v1 + 命令/历史/持久化（0.31.0） | schema/命令/校验/undo/get/save 全部通过 |
| E-11 | P0 | Edit | DONE | 媒体池 + 双监视器（0.39.0） | 源预览、节目最近成片、三点插入闭环 |
| E-12 | P0 | Edit | DONE | 时间轴 UI（0.33.0） | 100 clips 220ms 渲染，截图 timeline-100-clips.png |
| E-13 | P0 | Edit | DONE | 基础命令 UI（0.33.0） | Blade/Lift/Ripple/Insert/Overwrite 全部接入 |
| E-14 | P2 | Edit | DONE | RippleTrim/Roll/Slip/Slide（后置） | ripple_trim/roll/slip/slide 内核 + 专业模式 RippleTrim/Roll/Slip UI |
| E-15 | P1 | Edit | DONE | Link/轨道目标/转场 Inspector | A/V 联动、淡入淡出按钮与 clip update 契约 |
| E-16 | P1 | Edit | DONE | 竖屏 transform/字幕烧录/ducking | 竖屏 transform、SRT 烧录、narration/music ducking 真实渲染 |
| E-17 | P0 | Edit | DONE | 撤销/重做 UI（0.40.0） | UI 删除→撤销→重做 e2e 全通过；快捷键表打开 |
| E-18 | P1 | Edit | DONE | Deliver 预设 + QC | 6 个交付预设 + delivery.qa 响度/画布/容器 QC |
| E-19 | P0 | Edit | DONE | 媒体池/代理/元数据 | MediaRef catalog、540p proxy、媒体池插入 |
| E-20 | P1 | Edit | DONE | 字幕/标题烧录 | S1 字幕轨 + ffmpeg 烧录，timeline-m3.mp4 12s 成片 |
| E-21 | P1 | Edit | DONE | 音频总线/ducking/响度 | 多音轨、sidechain ducking、loudnorm；-16.05 LUFS 实测 |
| E-22 | P1 | Edit | DONE | 交付预设/QC | 母版/Web/抖音/B站/YouTube/西瓜预设与 QC |
| E-23 | P1 | Edit | DONE | 审阅批注/版本 | 时间码批注 add/list/resolve + UI 批注弹窗 |
| E-24 | P1 | Edit | DONE | AI 粗剪/转写编辑/语义检索 | 转写检索 + 按文本确定性粗剪 + 去静音 |
| E-25 | P2 | Edit | DONE | 色彩管理/LUT（后置） | eq 冷暖调 + 3D LUT 字段与 lut3d 渲染；timeline-lut.mp4 通过 |
| C-01 | P0 | Easy | DONE | 简易模式壳（素材托盘/预览/简化时间轴） | 简易模式壳、素材托盘、目标时长、一键成片 |
| C-02 | P0 | Easy | DONE | 自动字幕/去静音/卡点 | 智能字幕、去静音、卡点 markers 均真实调用 |
| C-03 | P1 | Easy | DONE | 自动重构 9:16/1:1/16:9 | 9:16/1:1/16:9 + 内容感知非黑边中心重构；letterbox 实测 offset 正确 |
| C-04 | P1 | Easy | DONE | 模板槽位与“剪同款” | 3 个槽位模板 + 模板套用 e2e |
| C-05 | P1 | Easy | DONE | 字幕/转场/文字/音乐/导出预设 Asset Pack | editing-kit@1.0.0：caption/transition/delivery/template/preview |
| C-06 | P0 | Easy | DONE | 简易↔专业交接 | 简易/专业/交付共享同一 Timeline 状态与持久化 |
| C-07 | P1 | Easy | DONE | 抖音/B站/YouTube 导出预设 | 平台预设渲染 + QC 一次完成 |
| E-26 | P0 | Edit | DONE | MediaRef 身份模型 | timeline.media.catalog 返回 id/uri/sha/fps/尺寸/时长 |
| E-27 | P0 | Edit | DONE | Timeline revision/命令日志/并发 | revision + append-only JSONL 命令日志 + UI 写入，日志查询返回 retime/revision |
| E-28 | P1 | Edit | DONE | 每 revision QA 门禁 | timeline.qa + render 前门禁 + delivery.qa |
| E-29 | P0 | Edit | DONE | 模式矩阵 + 宏/模板契约 | 创作/简易剪辑/专业/交付三模式 + 宏/模板契约 |
| U-01 | P0 | UI | DONE | 深/浅双主题与高级感配色 @0.17.9 | 浅色/深色切换、持久化、刷新保持 |
| U-02 | P0 | UI/IA | TODO | 固定主视图四标签：剧本/分镜/预览/交付 | 标签与顺序跨项目类型一致 |
| U-03 | P0 | UI/IA | TODO | 顶栏动作分层 + 命令面板壳 | 主操作 ≤3，次级两次点击可达 |
| U-04 | P1 | UI/IA | TODO | 左栏分区：项目/创意包/资产/产物 | 对象可全局发现 |
| U-05 | P1 | UI/IA | TODO | 常驻面包屑与创作包→项目血缘 | 任意页面可见位置 |
| U-06 | P1 | UI/IA | TODO | 选择驱动 Inspector + 空态契约 | 任意选中对象有属性/空态 |
| U-07 | P1 | UI/IA | TODO | 可调高度多轨时间轴 | 画面/字幕/旁白/BGM/SFX |
| U-08 | P2 | UI/IA | TODO | 全局任务中心 + QA 报告入口 | 任务可定位、报告可打开 |
| U-09 | P2 | UI/IA | TODO | aria/focus/快捷键表 | 键盘完成核心操作 |
| U-10 | P0 | UX | DONE | AI 协作可见性 @0.18.2 | 已同步/未保存/已更新状态 + AI 对话深链，真实页面通过 |
| U-11 | P0 | UX | DONE | 创意包→项目血缘 @0.18.1 | 创意 r8 徽章、来源创意包、点击打开创作脉络 |
| U-12 | P0 | UX | DONE | 大分镜可用性 @0.20.0 | 100 镜头、批量时长、锁定、场景导航 |
| U-13 | P0 | UX | DONE | QA 体检报告 @0.18.0 | Inspector 展示 checks/findings；真实项目 9 项检查通过 |
| U-14 | P1 | UX | DONE | 反馈收件箱 @0.21.0 | 待处理/已解决/已忽略筛选与解决/忽略操作，真实页面通过 |
| U-15 | P1 | UX | DONE | 任务中心与失败恢复 @0.22.0 | 6 任务列表、cancelled 重试、阻塞原因可见 |
| U-16 | P1 | UX | DONE | 字幕轨与逐句编辑 @0.23.0 | 12 cues 轨道、编辑保存、重叠/空档告警 |
| U-17 | P1 | UX | DONE | 交接摘要 @0.24.0 | 项目/创意包/QA/反馈摘要，复制与 AI 深链 |
| U-18 | P2 | UX | DONE | 新手引导 @0.25.0 | 三步上手弹窗 + AI/模板入口，真实页面通过 |

## 三、当前执行顺序

1. ~~V-02 contact sheet~~ DONE；
2. ~~V-03 upscale~~ DONE；
3. **W-02 media-qa**：把 image/video/audio/subtitle QA 编排成一次调用；
4. **W-01 story-video**：串起主题到成片；
5. **A-01 ducking**、**V-04 keyframes**、**S-01 character-continuity**；
6. 最后做市场与治理（M-01/M-02）。

## 四、完成定义（DoD）

- 能力有 `input_schema`、超时、风险等级；
- 有 Go 测试或可重复的真实调用证据；
- workspace 边界、失败态、空态有明确返回；
- 打包为 `.hmpkg` 并通过 candidate test/confirm（技能受权限限制时至少 package + candidate.save）；
- `ultimate-media-architecture-progress.md` 记录证据与 SHA；
- 若产生新坑，写入 `catalog/constraints.json` 或 Workbench constraints。


## 审查报告

- UI / 信息架构审查：`docs/ui-ia-review.md`
- 场景化 UX 改进：`docs/ux-scenarios.md`
- 通用剪辑能力缺口：`docs/editing-capability-architecture.md`
- 达芬奇心智剪辑方案：`docs/davinci-style-editing-plan.md`
- 剪辑架构复核：`docs/editing-architecture-review.md`
- 剪辑价值与非目标：`docs/editing-value-positioning.md`
- 剪映式易用性融合：`docs/capcut-ease-hybrid-plan.md`
- 剪辑架构复核（第二轮）：`docs/editing-architecture-review-round2.md`
- 剪辑完整实施方案：`docs/editing-implementation-plan.md`

## 五、踩坑记录

- **Capability ID 命名（ADR 0119）**：层级用 `.`，词内复合用 `-`，禁止下划线。
  - 反例：`visual.video.contact_sheet` 会被 candidate test 拒绝；
  - 正例：`visual.video.contact-sheet`。
- **场景切点不等于分镜**：ffmpeg 场景分数只抓视觉硬切，同背景手绘场景会漏检；分镜时间轴才是真值。
- **色带阈值必须按风格解释**：纸张/平面风格 0.60–0.78 只作待复核，不能直接判死。


## 六、本轮阻塞

- `media-qa` Workflow 已 validate / package / candidate.test / candidate.confirm 全部通过；
- 但 `install_local_workflow_archive` 返回 `拒绝访问。 (os error 5)`，工作流未进入 `workflow.catalog.list`；
- 与 Skill 客户端注册同类环境权限问题，不是工作流内容错误；
- **根因已定位**：不是 `agent_home/workflows` ACL，而是安装器用 `std::env::temp_dir()` 解包；受限令牌下 `%TEMP%` 写入返回 os error 5。
- **修复已落地**：`store::paths::agent_temp_dir()` 优先 `<agent_home>/tmp`；`workflow_manager.rs` 5 处生产暂存改用该目录；`cargo check --locked` 通过。
- 运行中的 18082 实例需重启后重新安装验证。

































## 六、剪辑 M1-M4 实施记录（2026-10-20）

- 报告：`docs/editing-m1-m4-implementation-report.md`；
- Workbench 0.67.0 候选 SHA：`4dc3940792d57894e3304315adb65708ca02e01fb1898cc28acaaab7ac52b204`；
- UI e2e：撤销/重做、快捷键、一键成片、智能字幕、模板、文本粗剪、批注、交付 QC、100 clips；
- 视频证据：12s 字幕成片、9s ducking、8.5s xfade、3.7s 去静音、5s PIP、3s retime、3s color-keyframes、3s speed-curve、LUT/调色；媒体池拖拽、timeline.diff、自然语言精剪命令均有 e2e/调用证据；UI 编辑态 800×600 无溢出，审查见 `docs/ui-editing-review-2026-10-20.md`；
- 通用资产包：`editing-kit@1.0.0`，validate `valid=true`；
- 当前边界：逐帧语义主体跟踪仍是后续升级；颜色关键帧、速度曲线、版本 diff 与媒体池拖拽已完成。


## 七、剪辑工作台成熟度任务（UI / OSS）

> 完整方案：`docs/editing-ui-maturity-and-oss-review.md`。这些是 0.67 后的产品化任务，不改变已有 M1-M4 能力契约。

| ID | 优先级 | 状态 | 交付 | 验收 |
| --- | --- | --- | --- | --- |
| UIP-01 | P0 | DONE | 可拖拽三栏编辑壳 | 800px 抽屉；1280/1440 三栏；宽度持久化 |
| UIP-02 | P0 | DONE | 媒体库 v1 | 缩略图/类型/时长/搜索；100 项不阻塞 |
| UIP-03 | P0 | DONE | Clip Inspector v1 | 选中即显示 Transform/Speed/Audio/Transition/Color |
| UIP-04 | P0 | DONE | 播放头与节目监视器联动 | 拖动播放头 100ms 内更新当前帧；播放/暂停 |
| UIP-05 | P0 | DONE | 时间轴直接操作 | 鼠标完成 move/blade/ripple delete/trim/转场手柄 |
| UIP-06 | P0 | DONE | 工具态与上下文菜单 | 选择/刀片/修剪工具、右键菜单、cursor 反馈 |
| UIP-07 | P1 | IMPLEMENTED | 多选/框选/批量命令 | 10+ 片段批量操作可撤销 |
| UIP-08 | P1 | IMPLEMENTED | 波形与视频缩略图条 | 200 片段滚动稳定 |
| UIP-09 | P1 | IMPLEMENTED | 撤销历史与版本对比 | 查看/比较/恢复 revision |
| UIP-10 | P1 | IMPLEMENTED | 素材市场/模板浏览器 | 搜索、预览、槽位替换 |
| UIP-11 | P1 | IMPLEMENTED | 面板宽高拖拽 | 布局保存与重置 |
| UIP-12 | P1 | IMPLEMENTED | 键盘与可访问性 | 核心操作无鼠标可完成 |
| UIP-13 | P2 | IMPLEMENTED | AI diff 预览层 | 新增/删除/移动 clip 高亮并可局部应用 |
| UIP-14 | P2 | IMPLEMENTED | 文本-时间轴双向编辑 | 修改文稿定位并调整字幕/镜头 |
| UIP-15 | P2 | IMPLEMENTED | 语义素材检索 | 自然语言定位素材和片段 |
| UIP-16 | P2 | IMPLEMENTED | PIP/字幕/音频关键帧 lane | 专业精修可视化 |
| UIP-17 | P2 | IMPLEMENTED | 时间轴窗口化渲染 | 1000+ 片段可交互 |
| OSS-01 | P1 | DONE | OpenTimelineIO 交换适配 | OTIO/EDL round-trip 不改变 clip 时间 |
| OSS-02 | P1 | TODO | WebAV 预览 POC | WebView2 拖动播放头延迟与能力评估；不替换 render.start |
| OSS-03 | P2 | TODO | MLT 可行性评估 | 仅在自研预览/渲染成为硬瓶颈时启动 |

> 2026-10-20 复验（0.69.8）：UIP-01…UIP-06 与 OSS-01 已完成真实 18082 e2e；UIP-07…UIP-17 为已实现、待专项 e2e；OSS-02/OSS-03 仍待启动。证据见 `docs/ui-editing-review-2026-10-20.md` 第七节。

## 轨道成熟度后续（0.70.4 审查后）

| ID | 优先级 | 状态 | 能力 | 验收 |
| --- | --- | --- | --- | --- |
| UIP-18 | P0 | DONE | 逐轨高度调整 | 拖拽轨底或紧凑/标准/高预设，刷新后保持 |
| UIP-19 | P0 | DONE | Solo | 多音轨独听；solo 优先于 mute，预览与导出一致 |
| UIP-20 | P0 | DONE | 轨道排序/重命名/颜色 | 拖拽排序持久化；重命名与色条可编辑 |
| UIP-21 | P1 | DONE | 音频表与音量 | 轨道头实时电平、dB 数值、音量滑杆 |
| UIP-22 | P1 | TODO | Marker lane 与 mini-map | 标记独立 lane；长项目全局缩略图 |
| UIP-23 | P1 | TODO | 转场拖拽创建 | 从素材面板/clip 间隙拖入转场 |
| UIP-24 | P2 | TODO | 源/节目单双监视器切换 | 紧凑模式手动切换双/单监视器 |

> 0.70.4 已完成：轨道 M/L、增删轨道、视频轨隐藏语义、播放头复位、五轨首屏、低高度紧凑模式。0.74.1 已完成 UIP-18…UIP-21：逐轨高度、Solo、排序/重命名/颜色、音频表与轨道音量。证据见 `docs/timeline-track-maturity-review-2026-10-20.md` 第九节。
