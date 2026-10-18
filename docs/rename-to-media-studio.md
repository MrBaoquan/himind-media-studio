# 更名记录：从 short-video 到 media-studio

> 执行时间：2026-10-18  
> 状态：预发布阶段硬切换，不保留旧能力别名。

## 映射

| 旧 | 新 |
| --- | --- |
| `himind-short-video` | `himind-media-studio` |
| `com.himind.short-video-creation` | `com.himind.media-workbench` |
| `short.video.*` | `media.workbench.*` |
| `plugins/com.himind.short-video-creation/short-video-creation` | `plugins/com.himind.media-workbench/media-workbench` |
| `short-video-creation.exe` | `media-workbench.exe` |
| `.himind-video/` | `.himind-media/` |
| `com.himind.short-video.director` | `com.himind.media.director` |
| `com.himind.short-video.creator` | `com.himind.media.creator` |
| `com.himind.short-video.style-curator` | `com.himind.media.style-curator` |
| `com.himind.short-video.capability-author` | `com.himind.media.capability-author` |
| `com.himind.short-video.code-project` | `com.himind.media.code-project` |
| `com.himind.workflow.generate-video` | `com.himind.media.workflow.generate-video` |
| `com.himind.workflow.author-remotion-capability` | `com.himind.media.workflow.author-remotion-capability` |

## 说明

- 当前没有正式发布制品，因此不保留 `short.video.*` 兼容别名。
- 已有本地视频项目需要将 `.himind-video` 目录改名为 `.himind-media`，并同步更新其中的绝对路径。
- Skill、Workflow、Asset Pack 的 provider / consumer 字段已同步迁移。
- 仓库、插件展示名与 UI 标题统一为“自媒体创作台”；底层仍可以产出短视频，但产品边界不再由“短视频”四个字定义。
