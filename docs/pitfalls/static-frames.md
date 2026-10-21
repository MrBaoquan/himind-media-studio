# 渲染坑：长静帧 / 画面“死掉”

> 首次记录：2026-10-19
> 触发案例：`science-sketch`《天空为什么是蓝色的？》v0.1.0
> 结论：手绘/白板解释类风格必须有低频持续动效；不能只做一次性入场动画。

## 现象

成片自动 QA：

```text
freeze: warning — 9 个 ≥1.5s 静帧区间
```

画面并不黑、文字也都在，但连续 1.5 秒以上像素几乎不变，观感像幻灯片：

- 场景标题和图表在 0.7 秒内入场完毕；
- 之后只有 `sceneProgress` 不再变化，整屏完全静止；
- `ffmpeg freezedetect=n=0.002:d=1.5` 将这些区间判为 freeze。

## 根因

不是编码器问题，而是渲染函数在入场动画结束后没有持续动效：

```text
renderAt(t)
→ p = sceneProgress(scene, t)
→ p 很快到达 1
→ 所有位置/透明度固定
→ 后续帧完全相同
```

手绘风格尤其容易踩这个坑，因为线条本身是静态图元，天然缺少“呼吸”。

## 修复

`science-sketch` 0.1.1 用三层低成本动效解决：

| 层 | 做法 | 作用 |
| --- | --- | --- |
| 手绘 boil | `boil = floor(t * 7.5)`，Rough.js seed 随 boil 偏移 | 线条像逐帧手绘一样轻微抖动 |
| 纸张漂移 | 光晕中心、纸纹颗粒做低频正弦位移 | 整屏始终有极轻的生命感 |
| 图元呼吸 | 太阳半径、分子位置、光谱射线、观察者箭头微动 | 让注意力焦点保持活性 |

关键实现位置：

```text
style-packs/science-sketch/template/reel/reel.js
  const rc = new Proxy(rawRc, ...)   // 给所有 rough 图元加 boil seed
  function paper(t)                   // 纸张颗粒低频漂移
  function sun(cx, cy, r, alpha)      // 太阳呼吸
  function renderAt(t)                // NOW / boil 由时间驱动
```

## 不要这样做

- 不要用每帧随机噪点制造“动”的假象：压缩后闪烁、码率上升、观感廉价；
- 不要为了过 freezedetect 给整屏加高频抖动；
- 不要把有意的 2 秒结论停顿强行改成抖动；QA warning 允许人工确认；
- 不要在入场动画结束后就让所有元素完全冻结。

## 新风格 Checklist

- [ ] 入场动画结束后，至少有一个持续的、低频的画面变化源；
- [ ] 手绘/白板风格使用 6–10fps 的 seed boil，而不是每帧随机；
- [ ] 动效幅度小到不改变构图和文字可读性；
- [ ] 关键帧仍然确定性可复现（时间驱动，不用 `Math.random()`）；
- [ ] 成片跑 `quality.visual`，确认 freeze 不是大面积 warning；
- [ ] 若保留长静帧，必须在分镜里标注为“有意停顿”并在交付说明中解释。

## 已沉淀的位置

| 层 | 位置 | 作用 |
| --- | --- | --- |
| 约束 | `catalog/constraints.json` `MOTION_CONTINUITY` | 成片不得出现非有意长静帧 |
| 能力 | `media.workbench.quality.visual` | freezedetect 自动检测 |
| 风格 | `style-packs/science-sketch` 0.1.1 | 首个实现 boil + 纸张漂移的样例 |
| 文档 | 本文件 | 现象、根因、修复与 checklist |

## 边界

- `freezedetect` 是启发式检测，不能替代人眼审片；
- 平面排版风格的大面积纯色本身不是问题，关键是“时间轴上是否完全不动”；
- 真正的电影级运动仍需要镜头运动、景深、光照变化，boil 只解决“手绘静止”这一层。
