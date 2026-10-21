# 生成产品宣传片

这是「产品宣传片导演」Skill 的确定性生产后半程。上游调研、需求问询、风格样帧、脚本、分镜和 CreativePackage 审批仍由 AI 在客户端完成；本 Workflow 只接管已经批准的创作包，避免把创作判断伪装成固定流水线。

```text
已批准 CreativePackage
→ 物化 promo-video 项目
→ 生成预览
→ 人工审片
→ 质量门禁
→ 审批后渲染
→ AI 轮询 render.status / artifact.export
```

## 与十阶段 Skill 的关系

Skill 负责需要判断的十个阶段：

1. 预调研
2. 需求问询
3. 建 CreativePackage
4. 深度调研与真实素材
5. 风格样帧
6. 脚本与分镜
7. 校验与批准
8. 落成项目
9. 预览、音频、质检
10. 渲染与交付

Workflow 从第 8 阶段开始执行可确定的部分，并把第 9 阶段的审片与质量门禁、第 10 阶段的渲染审批做成显式卡点。旁白、字幕、BGM 与证据素材由 Skill 在调用本 Workflow 前写入已批准的 CreativePackage 或项目输入。

## 输入

- `workspace_root`：创作台工作区；
- `creative_id`：已批准的 CreativePackage ID；
- `project_id`：从已有项目重新预览时使用；
- `audio_path`：可选，混音后的音轨；
- `subtitle_path`：可选，SRT/ASS 字幕；
- `burn_subtitles`：可选，是否烧录字幕；
- `output_path`：可选，工作区内的最终输出路径。

## 门禁

- `creative.materialize` 会拒绝未批准当前 revision 的创作包；
- 预览后必须人工审片；
- `quality.check` 未通过不得渲染；
- `render.start` 需要单独审批；
- 渲染返回的是任务而非完成事实，AI 必须轮询到 `completed` 并用 `artifact.export` 取得真实 MP4。

## 安装

本 Workflow 与 `com.himind.media.promo` Skill 需要分别安装。Feature Pack 当前不能直接包含 Workflow，这是平台限制；安装自媒体创作台 Feature Pack 后仍应补充安装本 Workflow。
