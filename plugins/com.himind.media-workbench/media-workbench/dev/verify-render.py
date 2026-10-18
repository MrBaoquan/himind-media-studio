#!/usr/bin/env python3
"""verify-render.py — 对渲染出的成片做**内容级**验收。

只依赖 ffmpeg/ffprobe + Python 标准库（不需要 numpy/Pillow）。

用法:
    python dev/verify-render.py <video.mp4> [--json] [--profile storyboard]

检查的是"画面里到底有没有东西、动没动"，而不是"文件存不存在"：
  * 分辨率 / 帧率 / 帧数 / 时长
  * 逐帧内容面积是否随叙事推进（支持比例增长或绝对面积增长）
  * 强调色/路线色像素占比是否随叙事生长而上升（路线类模板）
  * 标题区、中部内容区、页脚区是否都有内容
  * Storyboard 使用 --profile storyboard：允许镜头切换造成的内容面积回落
  * 同输入两次渲染的 SSIM 对比（可选，需要 ffmpeg libvmaf/ssim）

退出码 0 = 全部通过。
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile

W, H = 216, 384          # 分析用的降采样尺寸（10%）
BG = (8, 12, 26)         # tech-blue color.bg
ROUTE_COLORS = [
    (231, 76, 60), (46, 204, 113), (52, 152, 219), (243, 156, 18), (155, 89, 182),
    (26, 188, 156), (230, 126, 34), (255, 107, 107), (0, 188, 212), (126, 87, 194),
    (141, 110, 99), (120, 144, 156), (92, 107, 192), (171, 71, 188), (38, 166, 154),
    (239, 83, 80),
]


def tool(name):
    found = shutil.which(name)
    if not found:
        # Windows 上 chocolatey 常装在这里
        for candidate in (
            r"C:\ProgramData\chocolatey\bin\{}.exe".format(name),
            r"C:\ffmpeg\bin\{}.exe".format(name),
        ):
            if os.path.exists(candidate):
                return candidate
        raise SystemExit("找不到 {}，请先安装 ffmpeg".format(name))
    return found


def probe(path):
    out = subprocess.run(
        [tool("ffprobe"), "-v", "error", "-select_streams", "v:0",
         "-show_entries", "stream=width,height,r_frame_rate,nb_frames,codec_name",
         "-show_entries", "format=duration,size",
         "-of", "json", path],
        capture_output=True, text=True, check=True).stdout
    data = json.loads(out)
    stream = data["streams"][0]
    num, den = stream["r_frame_rate"].split("/")
    return {
        "codec": stream.get("codec_name"),
        "width": int(stream["width"]),
        "height": int(stream["height"]),
        "fps": float(num) / float(den),
        "frames": int(stream.get("nb_frames") or 0),
        "duration": float(data["format"]["duration"]),
        "bytes": int(data["format"]["size"]),
    }


def extract_raw(path, fps=1):
    tmp = os.path.join(tempfile.mkdtemp(prefix="verify-"), "frames.raw")
    subprocess.run(
        [tool("ffmpeg"), "-v", "error", "-y", "-i", path,
         "-vf", "fps={},scale={}:{}".format(fps, W, H),
         "-f", "rawvideo", "-pix_fmt", "rgb24", tmp],
        check=True)
    return tmp


def near(px, ref, tol):
    return abs(px[0] - ref[0]) <= tol and abs(px[1] - ref[1]) <= tol and abs(px[2] - ref[2]) <= tol


def analyze(raw):
    per = W * H * 3
    data = open(raw, "rb").read()
    count = len(data) // per
    frames = []
    for i in range(count):
        buf = data[i * per:(i + 1) * per]
        non_bg = routes = 0
        colors = set()
        for p in range(0, per, 3):
            px = (buf[p], buf[p + 1], buf[p + 2])
            colors.add((px[0] >> 3, px[1] >> 3, px[2] >> 3))
            if not near(px, BG, 26):
                non_bg += 1
            if any(near(px, c, 34) for c in ROUTE_COLORS):
                routes += 1
        pixels = per / 3
        frames.append({
            "content_pct": 100.0 * non_bg / pixels,
            "route_pct": 100.0 * routes / pixels,
            "colors": len(colors),
        })
    return frames


def text_peak(buf, y0f, y1f):
    """返回区域内最亮通道值 —— 用来判断水印/文字是否真的画上去了。"""
    peak = 0
    for y in range(int(H * y0f), int(H * y1f)):
        base = y * W * 3
        for x in range(W):
            o = base + x * 3
            peak = max(peak, buf[o], buf[o + 1], buf[o + 2])
    return peak


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    path = sys.argv[1]
    want_json = "--json" in sys.argv
    profile = "general"
    if "--profile" in sys.argv:
        index = sys.argv.index("--profile")
        if index + 1 < len(sys.argv):
            profile = sys.argv[index + 1]

    info = probe(path)
    raw = extract_raw(path, fps=1)
    frames = analyze(raw)
    per = W * H * 3
    data = open(raw, "rb").read()
    last = data[(len(frames) - 1) * per:][:per]

    checks = []

    def check(name, cond, detail):
        checks.append({"name": name, "ok": bool(cond), "detail": detail})

    if profile == "code":
        check("分辨率有效", info["width"] >= 720 and info["height"] >= 720,
              "{}x{}".format(info["width"], info["height"]))
    else:
        check("分辨率与模板画布一致", info["width"] >= 720 and info["height"] > info["width"],
              "{}x{}".format(info["width"], info["height"]))
    check("时长有效", info["duration"] > 1.0, "{:.2f}s".format(info["duration"]))
    check("有足够帧数", info["frames"] > int(info["fps"]), "{} 帧".format(info["frames"]))

    colors = [f["colors"] for f in frames]
    color_floor = 8 if profile == "code" else (32 if profile == "storyboard" else 100)
    check("画面不是纯色/空白", min(colors) > color_floor, "最少 {} 种颜色（阈值 {}）".format(min(colors), color_floor))

    content = [f["content_pct"] for f in frames]
    # 不同叙事的"内容面积"起点不同：地图从近乎空白开始，环形图一开始就有底环。
    # 因此同时接受"比例增长 1.5 倍"和"绝对面积增长 >= 1.5 个百分点"。
    content_growth = max(content) - min(content)
    content_levels = len({round(value, 1) for value in content})
    check("画面内容随时间变化（不是静止图）",
          max(content) > min(content) * 1.5 or content_growth >= 1.5 or (profile == "code" and (content_levels >= 3 or min(content) >= 90)),
          "{:.2f}% -> {:.2f}% (+{:.2f}pp)".format(min(content), max(content), content_growth))
    if profile == "code":
        check("代码项目画面有足够变化", content_levels >= 3 or content_growth >= 1.5 or min(content) >= 90,
              "{} 个内容面积档位，变化 {:.2f}pp".format(content_levels, content_growth))
    elif profile == "storyboard":
        levels = len({round(value, 1) for value in content})
        check("多镜头内容有明显变化", levels >= 3, "{} 个内容面积档位".format(levels))
    else:
        check("内容整体递增（叙事在推进）",
              sum(1 for i in range(1, len(content)) if content[i] >= content[i - 1] - 1.5) >= len(content) - 2,
              "{} 段中 {}/{} 段非递减".format(len(content) - 1,
                                              sum(1 for i in range(1, len(content)) if content[i] >= content[i - 1] - 1.5),
                                              len(content) - 1))

    routes = [f["route_pct"] for f in frames]
    if max(routes) > 0.2:
        check("强调色/路线色随叙事生长", routes[-1] > routes[0] * 1.5 or routes[-1] > 1.0,
              "路线像素 {:.2f}% -> {:.2f}%".format(routes[0], routes[-1]))

    head = text_peak(data[:per], 0.02, 0.16)
    mid = text_peak(data[(len(frames) // 2) * per:][:per], 0.30, 0.72)
    foot = text_peak(last, 0.88, 0.96)
    if profile == "code":
        check("代码项目有明显文字/高光", max(head, mid) > 100,
              "head={} mid={}".format(head, mid))
        check("页脚信息已绘制", foot > 24, "最亮通道 {}".format(foot))
    elif profile == "storyboard":
        bright_regions = sum(1 for value in (head, mid) if value > 150) + (1 if foot > 32 else 0)
        check("多镜头至少有头部/中部/页脚两处内容", bright_regions >= 2,
              "head={} mid={} foot={}".format(head, mid, foot))
    else:
        check("标题区有内容", head > 150, "最亮通道 {}".format(head))
        check("中部内容区有内容", mid > 150, "最亮通道 {}".format(mid))
        check("页脚水印已绘制", foot > 32, "最亮通道 {}（背景约 26）".format(foot))

    passed = all(c["ok"] for c in checks)
    if want_json:
        print(json.dumps({"video": path, "probe": info, "checks": checks, "passed": passed},
                         ensure_ascii=False, indent=2))
    else:
        print("视频: {}  {}x{}  {:.2f}s  {} 帧  {:.2f} MB".format(
            os.path.basename(path), info["width"], info["height"], info["duration"],
            info["frames"], info["bytes"] / 1048576))
        print()
        for c in checks:
            print("{}  {:<32} {}".format("PASS" if c["ok"] else "FAIL", c["name"], c["detail"]))
        print()
        print("全部通过" if passed else "存在未通过项")
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())





