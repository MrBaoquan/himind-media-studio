# 整片体检

对一条成片做一次可重复的 QA：

```text
联系表（一张图审全片）
→ 抽帧画质（色带/色块/过曝/失焦）
→ 场景切点
→ 可选：关键帧图片 QA
→ 可选：角色/主体一致性
→ 可选：项目级文字/音频 QA
```

## 输入

- `workspace_root`（必填）：工作区目录；
- `video_path`（必填）：工作区内成片路径；
- `times`：逗号分隔采样秒数；留空按 `sample_count` 均匀采样；
- `sample_count`：默认 5，最大 12；
- `project_id`：传入时追加项目级 `quality.visual`（文字越界/重叠、音频响度等）；
- `image_path`：传入时追加单帧 `visual.image.qa`；
- `pack_path` + `candidate_paths`：传入时追加参考包一致性检查。

## 依赖

- `com.himind.media-visual`（0.6.0+）
- `com.himind.media-workbench`（0.17.8+）
- 本机 `ffmpeg` / `ffprobe`

## 说明

- 联系表与场景检测是确定性能力，不调用模型；
- 场景切点只反映视觉硬切，同风格场景以分镜时间轴为准；
- 色带 0.60–0.78 为待复核，≥0.78 才按高危处理；
- 本 Workflow 只读媒体，联系表写回 `.himind-media/qa/`。

## 交付门禁（no-go）

- 本 Workflow 是**独立终检**，不会自动挂在 `generate-video` 的渲染步骤里；渲染完成后应单独运行。
- 没有 `video_path` 时是 **no-go**：不得声称“整片体检通过”。
- 出现高危色带/明显色块/大面积过曝或失焦、字幕越界、旁白削波或响度异常时，**不得交付**；先回到时间线或渲染参数修复。
- 画面 QA 通过只代表技术质量，不代表事实、版权或叙事正确；这些仍需 CreativePackage 审批与人工复核。
- 本 Workflow 是 capability-only，不声明 Runtime 依赖。
