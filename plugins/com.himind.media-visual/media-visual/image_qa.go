package main

import (
	"image"
	"math"
	"os"
	"strings"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type imageMetrics struct {
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	UniqueLuma     int     `json:"unique_luma_levels"`
	FlatRatio      float64 `json:"flat_ratio_smooth"`
	StepRatio      float64 `json:"step_ratio_smooth"`
	BandingScore   float64 `json:"banding_score"`
	EdgeDensity    float64 `json:"edge_density"`
	Sharpness      float64 `json:"sharpness"`
	Blockiness     float64 `json:"blockiness"`
	ClipHigh       float64 `json:"clip_high_ratio"`
	ClipLow        float64 `json:"clip_low_ratio"`
	MeanLuma       float64 `json:"mean_luma"`
	DominantLevels int     `json:"dominant_luma_levels"`
}

func imageQACapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	path, err := workspaceFile(root, in.ImagePath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("image_path: " + err.Error())
	}
	info, statErr := os.Stat(path)
	if statErr != nil || info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("image_path must be a file")
	}
	img, decodeErr := decodeImage(path)
	if decodeErr != nil {
		return nil, himindjsonrpc.InvalidParams("decode image: " + decodeErr.Error())
	}
	metrics := analyzeImage(img)
	blockers, warnings := evaluateImageMetrics(metrics, "")
	state := "ready"
	if len(blockers) > 0 {
		state = "blocked"
	}
	return map[string]any{
		"state": state, "ok": len(blockers) == 0, "image": map[string]any{
			"path": workspaceRelative(root, path), "width": metrics.Width, "height": metrics.Height,
		},
		"metrics":    metrics,
		"blockers":   blockers,
		"warnings":   warnings,
		"thresholds": map[string]any{"clip_high_block": 0.30, "banding_warn": 0.60, "banding_high": 0.78, "blockiness_warn": 1.8, "sharpness_min": 0.01},
		"note":       "确定性像素统计：用于发现色带/色块、过曝与失焦风险，不做语义审美判断",
	}, nil
}

func analyzeImage(img image.Image) imageMetrics {
	const maxSide = 384
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	sampleW, sampleH := width, height
	if width > maxSide || height > maxSide {
		if width >= height {
			sampleW = maxSide
			sampleH = int(math.Round(float64(height) * float64(maxSide) / float64(width)))
		} else {
			sampleH = maxSide
			sampleW = int(math.Round(float64(width) * float64(maxSide) / float64(height)))
		}
	}
	if sampleW < 1 {
		sampleW = 1
	}
	if sampleH < 1 {
		sampleH = 1
	}
	rgb := resizeRGB(img, sampleW, sampleH)
	luma := make([]float64, sampleW*sampleH)
	hist := [256]int{}
	clipHigh, clipLow := 0, 0
	totalLuma := 0.0
	for index := 0; index < len(luma); index++ {
		value := 0.299*float64(rgb[index*3]) + 0.587*float64(rgb[index*3+1]) + 0.114*float64(rgb[index*3+2])
		luma[index] = value
		totalLuma += value
		bucket := int(math.Round(value))
		if bucket < 0 {
			bucket = 0
		}
		if bucket > 255 {
			bucket = 255
		}
		hist[bucket]++
		if bucket >= 250 {
			clipHigh++
		}
		if bucket <= 5 {
			clipLow++
		}
	}
	pixels := float64(len(luma))
	unique := 0
	dominant := 0
	for _, count := range hist {
		if count > 0 {
			unique++
		}
		if float64(count) >= pixels*0.001 {
			dominant++
		}
	}
	// 色带只发生在"有低频渐变"的区域：先看左右/上下 5px 区块均值是否存在
	// 缓慢变化（0.4..12），再统计该区域内相邻像素是否长段为 0。
	// 纯色背景没有低频渐变，不参与色带统计，避免误报。
	smoothPairs, flatPairs, stepPairs := 0, 0, 0
	const radius = 2
	const block = 4
	for y := block; y < sampleH-block; y++ {
		for x := block; x < sampleW-block; x++ {
			center := luma[y*sampleW+x]
			leftMean := regionMean(luma, sampleW, x-block, y-radius, block, radius*2+1)
			rightMean := regionMean(luma, sampleW, x+1, y-radius, block, radius*2+1)
			topMean := regionMean(luma, sampleW, x-radius, y-block, radius*2+1, block)
			bottomMean := regionMean(luma, sampleW, x-radius, y+1, radius*2+1, block)
			if gradientInRange(leftMean, rightMean) {
				diff := math.Abs(center - luma[y*sampleW+x+1])
				smoothPairs++
				if diff < 0.5 {
					flatPairs++
				} else if diff >= 1.5 {
					stepPairs++
				}
			}
			if gradientInRange(topMean, bottomMean) {
				diff := math.Abs(center - luma[(y+1)*sampleW+x])
				smoothPairs++
				if diff < 0.5 {
					flatPairs++
				} else if diff >= 1.5 {
					stepPairs++
				}
			}
		}
	}
	flatRatio, stepRatio := 0.0, 0.0
	if smoothPairs > 0 {
		flatRatio = float64(flatPairs) / float64(smoothPairs)
		stepRatio = float64(stepPairs) / float64(smoothPairs)
	}
	levelScore := 0.0
	if dominant < 128 {
		levelScore = float64(128-dominant) / 128
	}
	bandingScore := 0.0
	// 渐变样本太少时，说明画面以纯色/硬边为主，不应判为色带。
	if smoothPairs > (sampleW*sampleH)/400 {
		bandingScore = 0.7*flatRatio + 0.3*levelScore
	}
	if bandingScore > 1 {
		bandingScore = 1
	} // Laplacian 方差衡量细节量；归一化到 0..1 便于跨分辨率比较。
	laplacianSum, laplacianSqSum := 0.0, 0.0
	count := 0
	for y := 1; y < sampleH-1; y++ {
		for x := 1; x < sampleW-1; x++ {
			center := luma[y*sampleW+x]
			value := 4*center - luma[(y-1)*sampleW+x] - luma[(y+1)*sampleW+x] - luma[y*sampleW+x-1] - luma[y*sampleW+x+1]
			laplacianSum += value
			laplacianSqSum += value * value
			count++
		}
	}
	sharpness, edgeDensity := 0.0, 0.0
	if count > 0 {
		mean := laplacianSum / float64(count)
		variance := laplacianSqSum/float64(count) - mean*mean
		sharpness = math.Sqrt(math.Max(variance, 0)) / 64
		if sharpness > 1 {
			sharpness = 1
		}
	}
	if count > 0 {
		edges := 0
		for y := 1; y < sampleH-1; y++ {
			for x := 1; x < sampleW-1; x++ {
				value := math.Abs(4*luma[y*sampleW+x] - luma[(y-1)*sampleW+x] - luma[(y+1)*sampleW+x] - luma[y*sampleW+x-1] - luma[y*sampleW+x+1])
				if value >= 24 {
					edges++
				}
			}
		}
		edgeDensity = float64(edges) / float64(count)
	}
	blockiness := blockinessRatio(luma, sampleW, sampleH)
	return imageMetrics{
		Width: width, Height: height, UniqueLuma: unique,
		FlatRatio: round4(flatRatio), StepRatio: round4(stepRatio),
		BandingScore: round4(bandingScore), EdgeDensity: round4(edgeDensity),
		Sharpness: round4(sharpness), Blockiness: round4(blockiness),
		ClipHigh: round4(float64(clipHigh) / pixels), ClipLow: round4(float64(clipLow) / pixels),
		MeanLuma: round4(totalLuma / pixels), DominantLevels: dominant,
	}
}

func blockinessRatio(luma []float64, width, height int) float64 {
	boundarySum, boundaryCount := 0.0, 0
	interiorSum, interiorCount := 0.0, 0
	for y := 0; y < height; y++ {
		for x := 0; x+1 < width; x++ {
			diff := math.Abs(luma[y*width+x] - luma[y*width+x+1])
			if (x+1)%8 == 0 {
				boundarySum += diff
				boundaryCount++
			} else {
				interiorSum += diff
				interiorCount++
			}
		}
	}
	for x := 0; x < width; x++ {
		for y := 0; y+1 < height; y++ {
			diff := math.Abs(luma[y*width+x] - luma[(y+1)*width+x])
			if (y+1)%8 == 0 {
				boundarySum += diff
				boundaryCount++
			} else {
				interiorSum += diff
				interiorCount++
			}
		}
	}
	if boundaryCount == 0 || interiorCount == 0 {
		return 1
	}
	boundary := boundarySum / float64(boundaryCount)
	interior := interiorSum / float64(interiorCount)
	if interior < 0.01 {
		return 1
	}
	return boundary / interior
}

func round4(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*10000) / 10000
}

func regionMean(luma []float64, width, x, y, w, h int) float64 {
	if w <= 0 || h <= 0 {
		return 0
	}
	sum := 0.0
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			sum += luma[(y+dy)*width+(x+dx)]
		}
	}
	return sum / float64(w*h)
}

func gradientInRange(left, right float64) bool {
	delta := math.Abs(right - left)
	return delta >= 0.4 && delta <= 12
}

// evaluateImageMetrics 是 image.qa 与 video.qa 共用的阈值判断。
// at 非空时用于给每条 finding 加时间前缀。
func evaluateImageMetrics(metrics imageMetrics, at string) ([]map[string]any, []map[string]any) {
	prefix := ""
	if strings.TrimSpace(at) != "" {
		prefix = at + "s "
	}
	blockers := make([]map[string]any, 0)
	warnings := make([]map[string]any, 0)
	if metrics.ClipHigh > 0.30 {
		blockers = append(blockers, issue("IMAGE_CLIPPING_HIGH", "exposure", prefix+"高光溢出面积过大", "降低发光强度或使用 tone mapping，避免大片纯白"))
	} else if metrics.ClipHigh > 0.12 {
		warnings = append(warnings, issue("IMAGE_CLIPPING_WARN", "exposure", prefix+"高光溢出偏高", "检查发光体/天空是否丢失细节"))
	}
	if metrics.Sharpness < 0.002 {
		blockers = append(blockers, issue("IMAGE_SOFT", "sharpness", prefix+"画面几乎没有可辨识结构", "检查素材是否为空帧或严重失焦"))
	}
	if metrics.BandingScore >= 0.78 {
		warnings = append(warnings, issue("IMAGE_BANDING_HIGH", "banding", prefix+"检测到明显色带风险", "加入确定性 dither/grain、提高色彩位深或改用渐变噪声"))
	} else if metrics.BandingScore >= 0.60 {
		warnings = append(warnings, issue("IMAGE_BANDING_MEDIUM", "banding", prefix+"存在中等色带风险", "在暗部渐变处加入轻微噪声，避免纯平滑色块"))
	}
	if metrics.Blockiness >= 1.8 {
		warnings = append(warnings, issue("IMAGE_BLOCKY", "compression", prefix+"8px 块边界差异偏高", "提高编码码率或改用无损中间帧"))
	}
	if metrics.Sharpness < 0.01 {
		warnings = append(warnings, issue("IMAGE_SOFT_WARN", "sharpness", prefix+"画面细节偏软", "检查景深、降噪强度与缩放算法"))
	}
	return blockers, warnings
}
