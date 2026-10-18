# 渲染坑：Browser-Code 多页面并发截图停滞

> 首次记录：2026-10-18  
> 触发案例：`retro-vhs` 复古 VHS  
> 结论：重型 Canvas 风格先从 `runtime.max_workers=1` 开始；不要为了提速把多个截图页面塞进同一个 Chrome。

## 现象

同一台机器、同一份 VHS 项目：

- `node render.mjs --frames --workers=1 --duration=10`：300 帧约 55 秒完成；
- `--workers=2`：约 50 秒只写出 2 帧，后续长时间没有进展；
- 预览静帧 1.5 秒左右即可生成，单独 3 帧探测约 2 秒。

也就是说，问题不在 Canvas 本身，而在**同一个浏览器实例里同时截图多个 1080×1920 页面**。

## 当前根因判断

`render.mjs` 的并发实现是：

```text
一个 puppeteer browser
  ├─ page 0 → renderAt(t) → page.screenshot()
  └─ page 1 → renderAt(t) → page.screenshot()
```

VHS 页面有大量渐变、扫描线、噪点与高频重绘。两个页面同时请求截图时，Chrome 的帧提交/截图等待会互相拖住，表现为进程仍活着、CPU 不高、没有报错，但帧文件不再增长。

这不是“渲染器死锁”的唯一可能原因，所以先做保守约束，不把它当成所有风格的通用结论。

## 修复

Style Pack 声明运行时上限：

```json
{
  "runtime": {
    "max_workers": 1
  }
}
```

`media.workbench.render.start` 会把调用方传入的 `workers` 收敛到 `max_workers`。当前 `retro-vhs` 已使用该配置；默认 worker 也是 1。

执行点：

- `code_project.go` `effectiveWorkerCount()`
- `style_pack.go` 校验 `runtime.max_workers` 必须在 0–8 之间
- `code_project_test.go` `TestEffectiveWorkerCountRespectsStylePackLimit`
- `schemas/style-pack.schema.json` `runtime.max_workers`

## 新风格 Checklist

- [ ] 先用 `workers=1` 跑完整 10 秒或目标时长；
- [ ] 只有确认单 worker 稳定后，才尝试提高并发；
- [ ] 提高并发时同时观察帧文件增长速度，不只看 Job 的 `progress`；
- [ ] 重型渐变/噪点/滤镜/多层 Canvas 优先声明 `runtime.max_workers=1`；
- [ ] 若未来真正需要并发，走“多浏览器实例或多进程”，不要继续增加同一浏览器页面数；
- [ ] 出现“进程活着但帧不增长”时，先查 Chrome 截图，而不是先怀疑 ffmpeg。

## 证据

- VHS 单 worker 渲染 Job：`render-1792332233401933100`，300 帧，约 55 秒；
- VHS 双 worker 试验：约 50 秒仅 2 帧，随后取消；
- 自动 QA 结果：`quality_status=warning`，原因是轻微平坦段 239px，不阻断；文字审计已开启并通过 47 条文字检查。

## 边界

- 该坑目前只在高频 Canvas 页面观察到，不能直接推断所有 Style Pack 都不能并发；
- `runtime.max_workers` 是保底策略，不是性能优化的终点；
- 正式性能优化应优先减少每帧重绘成本、复用离屏 Canvas、降低不必要的噪点循环。
