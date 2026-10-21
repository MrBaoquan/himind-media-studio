# 对话生成短视频

把自然语言需求转成可验证的短视频项目：

```text
需求
→ AI 导演计划
→ Plan 规范化与估算
→ 创建项目与预览
→ 质量门禁
→ 审批后提交渲染
```

## 输入

- `request`：创作需求；
- `name`：项目名称；
- `workspace_root`：项目工作区；
- `style_id`：风格；
- `path`（可选）：CSV/JSON 数据文件；真实排名/路线类请求应提供。

## 依赖

- `com.himind.media-workbench`
- `himind.builtin` Runtime（只用于导演计划这一同步步骤）

## 说明

- **联网语义**：`GV-PLAN.runtime.allow_network=true` 只表示允许调用模型服务；`tool_policy=none` 禁止模型做工具探索或外部检索。真实数据必须来自 `input.path`，缺失时 Plan 只记录数据风险，不编造数字。
- 计划必须通过 `media.workbench.plan.normalize`；
- 项目创建和渲染都需要审批；
- **镜头支持**：Plan 与 `generate.run` 支持单镜头和多镜头（`storyboard`）；`series.generate` 目前只支持单镜头 Plan，系列生成前需先拆分。
- **安装方式**：本 Workflow 是独立安装单元。当前 Feature Pack 只能组合 plugin/skill/agent preset，**不能直接包含 workflow**；短期请单独安装本 Workflow，长期再扩展 Feature Pack 的 `workflow_ids`。
- 质量门禁通过不等于自动交付；建议在渲染完成后用 `media-qa` Workflow 做最终整片体检。

## 示例

```text
请制作安徽省高铁路线动画示意图
```

使用 `route-growth`，随包数据为 17 条线路 / 97 个站点 / 2972 km。
