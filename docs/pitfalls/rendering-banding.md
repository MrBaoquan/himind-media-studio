# 渲染坑：暗部色带 / 连片色块

> 首次记录：2026-10-18  
> 触发案例：`celestial-seasons` 天体四季科普  
> 结论：最终母版不要用 JPEG 中间帧；暗部渐变优先 PNG + dither。

## 现象

深色天空、地球边缘、vignette、星云或大面积渐变区域出现：

- 一整片一整片的色块；
- 色带（banding）；
- 8×8 或更大的压缩块；
- 预览静帧正常，但成片更明显。

## 根因

问题通常不在 three.js / WebGL 本身，而在**帧捕获格式**：

```text
renderAt()
→ page.screenshot(JPEG 0.94)
→ f00000.jpg
→ H.264
```

JPEG 在暗部渐变上会强烈量化。实测同一帧：

| 输入 | 背景亮度级数 | 最大连续平坦段 |
| --- | ---: | ---: |
| PNG 截图 | 51 级 | 5px |
| JPEG 中间帧 | 2 级 | 648px |
| H.264 成片 | 2 级 | 648px |

H.264 只是把 JPEG 已经压平的色块保留下来，并不是视频编码器单独造成的。

叠加因素：

- Canvas overlay 的径向 vignette；
- WebGL canvas 上的 CSS `contrast()`；
- 低码率 Web copy；
- 大面积纯黑/近黑背景没有 dither。

## 正确修复顺序

1. 移除不必要的 CSS `contrast()` 和 overlay vignette；
2. 给暗部背景/星云加低频渐变和 dither；
3. Earth / Atmosphere / Sun Shader 输出加 1/255 dither；
4. 最终帧改为 PNG：`--frame-format=png`（默认）；
5. `--encode` 自动识别 PNG/JPG；
6. Web copy 再按码率兜底，不能反过来用 Web copy 掩盖母版问题；
7. 以母版审片，不以 720p Web copy 作为质量依据。

## 已沉淀的位置

| 层 | 位置 | 作用 |
| --- | --- | --- |
| 约束 | `catalog/constraints.json` `LOSSLESS_FINAL_FRAMES` | 最终帧必须 PNG/无损 |
| 约束 | `catalog/constraints.json` `DARK_GRADIENT_BANDING` | 暗部渐变必须通过色带检查 |
| 测试 | `quality_contract_test.go` `TestStylePacksUseLosslessFinalFrames` | 防止新 Style Pack 退回 JPEG |
| 能力 | `media.workbench.quality.banding` | AI 可直接调用的色带量化与 finding |
| 能力 | `media.workbench.quality.visual` | 成片黑帧/静帧/色带/音频综合 QA |
| 自动 QA | render completion | 自动写入 job 与母版 artifact 的 quality_status / quality_report |
| 能力 | `media.workbench.project.upgrade-style` | 旧项目原地升级 renderer，保留项目代码 |
| 脚本 | `dev/verify-banding.mjs` | 本地量化亮度级数与最大平坦段 |
| 文字审计 | `renderer/browser-code/text-audit.mjs` | 通用 Canvas fillText bounding box 审计 |
| Skill | `skills/video-code-project/SKILL.md` | 告诉 AI 何时用 PNG、如何检查 |
| Skill | `skills/video-creator/SKILL.md` | 质量反馈先查帧格式 |
| Skill | `skills/video-style-curator/SKILL.md` | 把 banding 反馈沉淀为规则/样例/版本 |
| 案例 | `docs/celestial-seasons-case-study.md` | 完整前后对比与证据 |
| 交付 | 19 个 Style Pack + `examples/code-project-demo` | 已统一支持 PNG 最终帧 |

## 新风格 Checklist

- [ ] `render.mjs --frames` 默认写 PNG；
- [ ] `--encode` 能识别 PNG/JPG；
- [ ] 有暗部渐变时，`--frame-format=jpg` 只用于快速测试；
- [ ] 有 vignette / CSS contrast 时，先跑 `dev/verify-banding.mjs`；
- [ ] 最大平坦段不超过画面宽度 8%，亮度级数不少于 12；
- [ ] 母版审片通过后再生成 Web copy；
- [ ] 新坑写入 `catalog/constraints.json`，并补测试/样例；
- [ ] 已有项目不会自动继承 Style Pack 更新，需要重新创建或同步 `render.mjs`。

## 边界

- 该脚本是启发式检查，不能替代人眼审片；
- 平面设计风格（白板、极简排版）可能出现大面积纯色，不能机械套用阈值；
- 720p Web copy 仍可能因码率出现轻微色带，关键交付以母版为准；
- 真正的电影级暗部还需要更高位深/离线渲染器，PNG 只解决“中间帧压平”这一层。








## 白底平面风格的误报

`transit-network` 首版白底线网图曾被色带门禁判为“严重色带”：

```text
1 个亮度级 / 1728px 平坦段
```

实际画面是大面积纯白背景，不是暗部渐变缺陷。处理方式：

```json
{"quality": {"banding_gate": false}}
```

- 只对确实以平面纯色为主的 Style Pack 使用；
- `banding` 变为 `skipped`，仍保留平坦段指标供人工复核；
- 暗色、3D、粒子、vignette 风格必须保持严格检查；
- 不要用 `banding_gate=false` 掩盖 JPEG 中间帧或低码率造成的真实色带。
