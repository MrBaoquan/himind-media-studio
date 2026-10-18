# OSS Prior Art 评估

在新增 Style Pack、Renderer、External Pack 或专用能力前，先做一次开源 prior art 调研，并把结论写进本目录。

## 必填字段

```text
需求：
候选项目：
Stars / License / 语言：
解决的子问题（布局 / 标签 / 动画 / 导出）：
运行时依赖：
输出格式：
许可证兼容性：
确定性 / headless：
结论：直接复用 / 包装为 External Pack / 可选外部工具 / 仅参考 / 自建
重新评估条件：
```

## 硬规则

- MIT / ISC / Apache-2.0 优先；
- GPL / AGPL 代码不直接 vendor 进插件，只能作为可选外部工具或仅参考；
- 商业求解器（如 Gurobi）依赖不能成为默认安装前提；
- 无许可证的仓库视为不可复用；
- 先确认它解决的是“布局”还是“动画”，不要把动画引擎当成线网布局库；
- 记录调研日期、Stars 和 last commit，避免把过期结论当现状。
