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
- `style_id`：风格。

## 依赖

- `com.himind.media-workbench`

## 说明

- 规划由 `himind.builtin` Runtime 完成，禁止联网和工具探索；
- 计划必须通过 `media.workbench.plan.normalize`；
- 项目创建和渲染都需要审批；
- 当前只支持单镜头计划；
- 真实数据缺失时，Plan 会记录数据风险，不编造数字。

## 示例

```text
请制作安徽省高铁路线动画示意图
```

使用 `route-growth`，随包数据为 17 条线路 / 97 个站点 / 2972 km。
