# Asset Pack 实施计划

## Phase 1：本地 Registry（已完成）

- [x] `himind.asset-pack.v1` schema
- [x] `asset.list / describe / read / materialize`
- [x] `stylepack.scaffold`
- [x] 扫描 workspace / repository / installed
- [x] 五个视频风格：kinetic-reel / painted-animation / data-kinetic / paper-collage / whiteboard-explainer
- [x] 创作台「风格库」入口
- [x] 一键创建项目并打开审阅
- [x] 风格库显示来源 / 安装状态
- [x] 风格库安装到本机 / 卸载本机副本
- [x] 2 秒循环 WebP 动态预览（19 个风格）

## Phase 1.5：代码型项目模式（已完成）

- [x] `project.import / sync / context`
- [x] `project.data.write` 写入 `input/data.json` 与 `input/brief.json`
- [x] `preview.review` 审查图
- [x] `preview.determinism` 双渲染 SHA-256 一致性
- [x] `quality.check` 与渲染前门禁
- [x] `feedback.record / list / resolve` 审阅反馈闭环
- [x] Browser-Code 固定 CLI 渲染契约
- [x] 网络策略与 Node 网络守卫
- [x] Web copy 交付（母版 + 下采样副本）
- [x] 审查图按项目实际时长均匀采样（9 帧）
- [x] 确定性检查按 15% / 50% / 85% 采样
- [x] 创作台一键「检查确定性并渲染」
- [x] 最终帧 PNG（19 个风格 + code-project-demo），避免暗部 JPEG 色带

## Phase 2：风格质量门禁

- [x] `asset.pack` 打包
- [x] `asset.validate` 统一校验
- [ ] smoke 渲染 2 秒片段
- [ ] determinism 短片段
- [x] 资源许可证与归属检查
- [ ] Web copy 大小门禁
- [x] Web copy 按长边缩放（竖屏 1080×1920 → 720×1280）
- [x] 程序化环境声床（无第三方音乐版权风险）
- [x] 成片视觉 QA（黑帧/静帧/色带/响度/文字审计）
- [x] 质量失败母版阻断 artifact.export
- [x] 重型 Style Pack runtime.max_workers 并发上限
- [x] 安装包 constraint evidence（52/52 resolved）

## Phase 3：本地安装与更新

- [x] `.hmpack` 安装
- [x] SHA-256 校验
- [x] 版本目录
- [x] 卸载
- [ ] 自动激活 / 回滚策略

## Phase 4：GitHub 市场

- [x] `catalog/asset-packs.json` 本地资产目录
- [ ] `.himind/catalog.json` 增加 `asset_packs`
- [ ] GitHub Release 制品
- [ ] 签名校验
- [x] `asset.market.list` / `asset.market.install`（插件层）
- [ ] Agent `extension.asset.install`
- [ ] 市场 UI

## Phase 5：更多风格

- [x] kinetic-reel
- [x] painted-animation
- [x] data-kinetic
- [x] paper-collage
- [x] whiteboard-explainer
- [x] data-chart-reel
- [x] retro-vhs
- [x] lyric-karaoke
- [x] comic-panel
- [x] blueprint-tech
- [x] editorial-minimal
- [x] isometric-infographic
- [x] neon-arcade
- [x] brand-lookbook
- [x] cinematic-map-route
- [x] 3d-product-reel
- [x] data-story-3d
- [x] celestial-seasons（three.js PBR / 30s / 科学事实卡 / NASA 地球贴图）
- [x] transit-network（白底线网 / 线路生长 / 站名避让 / 合肥示意数据 / PNG 帧）
- [x] science-sketch 0.1.1（手绘线条 / 淡彩 / 光谱 / 散射 / 45 秒科普场景 / 逐帧 boil / 纸张漂移 / 成片无长静帧）

## Phase 6：跨领域验证

- [ ] 图片模板包
- [ ] 音效包
- [ ] 数据集包
- [ ] 提示词包
- [ ] 品牌套件（跨插件共享）
