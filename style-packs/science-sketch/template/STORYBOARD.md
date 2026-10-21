# Science Sketch · 天空为什么是蓝色的？

| Time | Scene | What the viewer must understand |
| --- | --- | --- |
| 0–5s | Hook | 天空是空的，为什么抬头是蓝色的？ |
| 5–12s | Spectrum | 太阳白光包含多种颜色 |
| 12–23s | Scattering | 空气分子更容易把短波长的蓝光散向四面八方；瑞利散射强度约与 1/λ⁴ 成正比 |
| 23–31s | Observer | 蓝光从天空各处进入眼睛；紫光散射更强，但人眼对蓝更敏感 |
| 31–40s | Sunset | 日落时光线穿过更厚大气，蓝光被散掉，红橙色留下 |
| 40–45s | Takeaway | 天空的蓝色来自散射，不是海洋反射 |

## Sources

- NASA Space Place, *Why Is the Sky Blue?*
  https://spaceplace.nasa.gov/blue-sky/en/
- NOAA JetStream, *The Atmosphere*
  https://www.noaa.gov/jetstream/atmosphere

发布前仍需人工复核原文表述；视频内只放来源名称，完整 URL 放在本文件与 CreativePackage 的 `research.sources` 中。

## Visual rules

- 米白纸张、墨线、淡彩天空，Rough.js 手绘线条；
- 所有图形由固定 seed 程序化绘制，不依赖外部图片或网络字体；
- 单屏最多三层信息：场景标题、核心图解、注释；
- 使用有限图元：光谱、波形、分子、光线、观察者、昼夜对比、来源卡；
- 示意图必须标注来源；发布前复核科学表述。

## Motion rules

- 手绘线条以 7.5fps 做轻微 “boil”（seed 抖动），模拟逐帧手绘，不改变构图；
- 纸张颗粒与光晕做低频漂移，避免整屏完全静止；
- 太阳、分子、光谱射线、观察者箭头保持呼吸式微动；
- 不用无意义高频抖动掩盖真实问题；静帧只允许出现在有意的停顿；
- 自动 QA 使用 `freezedetect=n=0.002:d=1.5`，出现 warning 时必须人工确认原因。
