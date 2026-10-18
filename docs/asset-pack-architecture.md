# Asset Pack 架构

> 状态：已落地 Phase 1  
> 日期：2026-10-18

## 一、定义

Asset Pack 是**插件优先、异构资源、可被宿主和多类消费者解析的版本化资产容器**。

它不是 Plugin、Skill、Workflow 或 Feature Pack：

```text
Plugin       提供 Capability
Skill        保存 AI 判断
Workflow     编排多步骤
Asset Pack   提供领域资产
Feature Pack 组合安装
```

一个 Asset Pack 可以同时包含：

- 模板
- 字体
- 图片
- 音频
- 视频
- 数据集
- 提示词
- 文档
- 可选执行入口

顶层不区分 `video/audio/image`。资源类型由 `resources[].type` 表达。

## 二、清单

```json
{
  "schema": "himind.asset-pack.v1",
  "id": "data-kinetic",
  "version": "1.0.0",
  "name": "数据动态字卡",
  "resources": [
    {"id":"template","type":"template","path":"template","role":"render-template","runtime":"browser-code","entry":"render.mjs"},
    {"id":"preview","type":"image","path":"preview/sheet.jpg","role":"preview","mime":"image/jpeg"}
  ],
  "entrypoints": [
    {"id":"render","resource":"template","runtime":"browser-code","entry":"render.mjs"}
  ],
  "permissions": {"filesystem":"workspace","process":["node","chrome","ffmpeg"],"network":{"mode":"none","allow":[]}},
  "tags": ["video","style","data"]
}
```

## 三、消费者

主要消费者：

- Plugin：按 `resource.type/role` 选择资源
- 自媒体创作台：消费 `type=template` + `runtime=browser-code`

插件外消费者：

- Skill：通过宿主 `asset.list/read` 引用资源，不直接读文件
- Workflow：物化资源作为步骤输入
- 人类资产库：浏览、预览、复制
- 跨插件品牌套件：多个插件共享
- 外部应用：通过宿主 API 消费

执行边界：

> Asset Pack 可以被多种消费者使用，但执行权只属于 Plugin / Runtime Adapter。

## 四、当前能力

```text
media.workbench.asset.list
media.workbench.asset.describe
media.workbench.asset.read
media.workbench.asset.validate
media.workbench.asset.materialize
media.workbench.asset.pack
media.workbench.asset.install
media.workbench.asset.uninstall
media.workbench.asset.market.list
media.workbench.asset.market.install
media.workbench.stylepack.scaffold
media.workbench.stylepack.validate
```

`stylepack.scaffold` 会：

```text
读取 template 资源
→ 复制到 workspace
→ 确保 style-pack.json
→ project.import
```

## 五、安装与发现

当前扫描：

```text
<workspace>/asset-packs/
<workspace>/style-packs/
<repository>/asset-packs/
<repository>/style-packs/
%LOCALAPPDATA%/HiMindAgent/asset-packs/
```

优先级：workspace → repository → installed。

长期安装目录：

```text
%LOCALAPPDATA%\HiMindAgent\asset-packs\<id>\<version>\
```

## 六、市场边界

Market 是 Asset Pack 的分发层，不是执行层：

```text
GitHub Release
→ .hmpack / .hmstyle
→ SHA-256
→ 签名
→ catalog
→ Agent 安装
→ 插件消费
```

Market 不直接执行 Pack 中的代码。

## 七、非目标

- 不在 Asset Pack 顶层引入 `domain`
- 不把 Asset Pack 做成 Plugin
- 不把执行逻辑塞进 Market
- 不让 Skill 直接读取任意文件系统
- 不允许未声明 Runtime 的可执行入口


## 五之一、.hmpack 制品

`.hmpack` 是确定性 ZIP：

```text
package.json     文件清单、大小与 SHA-256
asset-pack.json  资产包 Manifest
...              资源文件
```

打包规则：

- 文件按路径排序后写入；
- ZIP 时间固定为 `1980-01-01`，同一输入得到稳定制品；
- 排除 `node_modules/`、`out/`、`.cache/`、`dist/`、`.git/`；
- 拒绝符号链接和非普通文件；
- 输出必须位于调用方 workspace 内。

安装流程：

```text
读取 package.json
→ 逐文件校验 SHA-256 与大小
→ 拒绝未声明文件与路径穿越
→ 解包到临时目录
→ validateAssetPack
→ 移动到 asset-packs/<id>/<version>/
→ 写入 install.json
```

版本选择：

```text
asset.list               返回全部已安装版本
asset.describe(version)  选择指定版本
asset.uninstall          删除指定版本
```

当前还没有自动“激活版本”指针；回滚通过显式选择旧版本实现。

## 七之一、风格库发现层

风格库除了来源与安装状态，还提供：

- 搜索：名称、描述、ID 与标签；
- 分类：数据、地图路线、品牌编辑、插画手绘、复古、技术、字体歌词；
- 收藏：本机 localStorage 保存；
- 最近：选择或创建项目后记录最近 8 个风格；
- 动态预览：优先读取 `preview/loop.webp`，卡片显示 2 秒循环动画。

这层只负责发现，不改变 Asset Pack 契约。

## 八、资产目录与更新检查

```text
catalog/asset-packs.json
  → asset.market.list
  → 合并本机 installed_version
  → update_available
  → asset.market.install（仅 status=published）
```

条目状态：

- `prepared`：包和 SHA-256 已准备，尚未发布 Release；
- `published`：Release 与 download_url 已可下载；
- `deprecated`：保留历史版本，不推荐新项目使用。

`dev/prepare-asset-release.ps1` 会重新打包、回填大小/SHA-256、生成 Release 命令，但不会自动向 GitHub 发布。


