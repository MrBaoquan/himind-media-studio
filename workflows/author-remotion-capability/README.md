# 沉淀 Remotion 能力

把“现有模板不够用”的瞬间，变成一次可复用的能力建设。

```text
能力缺口
→ 提案（AI 判断 reuse / extend / new_capability）
→ 用户确认方向
→ 确认数据契约
→ 实现 Shot + Technique + 注册
→ capability.audit 机械审计
→ Candidate / ready
```

## 入口

- `gap`：从需求开始，先读取模板与能力目录，再由 Runtime 产出 `himind.video.capability-proposal.v1`。
- `audit`：给定 `capability_id`，直接审计已声明的能力。

## 能力六件套

```text
Shot
Technique
Contract
Template
Fixture
Metadata（catalog/capabilities/*.json）
```

`media.workbench.capability.audit` 会检查：

- descriptor 的 schema 与路径是否合法；
- Template/Composition/Contract 是否一致；
- Shot 是否注册到 `index.jsx` 与 `shots/registry.js`；
- Technique、Contract、Fixture 配方与基线帧是否存在；
- ready 能力是否声明验收标准。

## 依赖

- `com.himind.media-workbench`（含 capability.list/describe/audit）
- Skill：`video-capability-author`

## 边界

- 不做运行时任意 JSX 执行；
- 不把 Native Shot 拆成 plugin；
- 没有 Contract、Fixture、Template 与审计证据的能力不得标记 ready；
- 当前 Workflow 已通过 `extension.test`（0.1.1）；确认步骤仍是人工门禁，代码修改由 AI 客户端按 Skill 与约束完成。
