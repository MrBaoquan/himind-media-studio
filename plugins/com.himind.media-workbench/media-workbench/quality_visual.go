package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type visualCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Level  string `json:"level,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type visualReport struct {
	OK        bool             `json:"ok"`
	ProjectID string           `json:"project_id,omitempty"`
	VideoPath string           `json:"video_path,omitempty"`
	ImagePath string           `json:"image_path,omitempty"`
	Metrics   map[string]any   `json:"metrics"`
	Checks    []visualCheck    `json:"checks"`
	Findings  []map[string]any `json:"findings,omitempty"`
	Warnings  []string         `json:"warnings,omitempty"`
}

type visualMediaProbe struct {
	Width    int
	Height   int
	Frames   int
	FPS      float64
	Duration float64
	HasAudio bool
}

// qualityVisual is the post-render QA gate: frames, black/freeze, banding and
// audio loudness. Text bounds/overlap are reported as skipped until a Style Pack
// provides a text-audit adapter, so the capability never pretends to have checked
// something it cannot measure.
func qualityVisual(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	imagePath := strings.TrimSpace(in.ImagePath)
	videoPath := strings.TrimSpace(in.VideoPath)
	if imagePath != "" {
		path, pathErr := workspaceFile(root, imagePath)
		if pathErr != nil {
			return nil, himindjsonrpc.InvalidParams("image_path: " + pathErr.Error())
		}
		return qualityVisualImage(path, in), nil
	}
	if videoPath == "" && strings.TrimSpace(in.ProjectID) != "" {
		for _, item := range projectArtifacts(root, in.ProjectID) {
			if item.Kind == "video/mp4" && item.Ready {
				videoPath = item.Path
				break
			}
		}
	}
	if videoPath == "" {
		return nil, himindjsonrpc.InvalidParams("video_path or project_id with a video/mp4 artifact is required")
	}
	path, pathErr := workspaceFile(root, videoPath)
	if pathErr != nil {
		return nil, himindjsonrpc.InvalidParams("video_path: " + pathErr.Error())
	}
	probe, probeErr := probeVisualMedia(path)
	if probeErr != nil {
		return nil, himindjsonrpc.InvalidParams("probe video: " + probeErr.Error())
	}
	report := visualReport{
		ProjectID: in.ProjectID,
		VideoPath: path,
		Metrics: map[string]any{
			"width":    probe.Width,
			"height":   probe.Height,
			"frames":   probe.Frames,
			"fps":      probe.FPS,
			"duration": probe.Duration,
			"audio":    probe.HasAudio,
		},
		Checks: make([]visualCheck, 0, 8),
	}
	report.Checks = append(report.Checks, visualCheck{
		ID: "frames", Status: statusFor(probe.Frames > 0 && probe.FPS > 0), Level: "error",
		Detail: fmt.Sprintf("%d 帧 / %.3f fps / %.3fs", probe.Frames, probe.FPS, probe.Duration),
	})
	if !(probe.Frames > 0 && probe.FPS > 0) {
		report.Findings = append(report.Findings, issue("VISUAL_FRAMES", "output", "视频帧数或 fps 无效", "检查渲染参数与输出文件"))
	}

	blackOutput, blackErr := runMediaFilter(path, "blackdetect=d=0.2:pix_th=0.10")
	blackEvents := countDetectorEvents(blackOutput, "black_start")
	if blackErr != nil {
		report.Checks = append(report.Checks, visualCheck{ID: "black_frames", Status: "failed", Level: "error", Detail: blackErr.Error()})
		report.Findings = append(report.Findings, issue("VISUAL_BLACK_CHECK", "output", "黑帧检测失败: "+blackErr.Error(), "检查 ffmpeg 与源视频"))
	} else {
		report.Checks = append(report.Checks, visualCheck{
			ID: "black_frames", Status: statusFor(blackEvents == 0), Level: "error",
			Detail: fmt.Sprintf("%d 个黑帧区间", blackEvents),
		})
		if blackEvents > 0 {
			report.Findings = append(report.Findings, issue("VISUAL_BLACK_FRAMES", "output", fmt.Sprintf("检测到 %d 个黑帧区间", blackEvents), "检查渲染任务与素材加载"))
		}
	}

	freezeOutput, freezeErr := runMediaFilter(path, "freezedetect=n=0.002:d=1.5")
	freezeEvents := countDetectorEvents(freezeOutput, "freeze_start")
	if freezeErr != nil {
		report.Checks = append(report.Checks, visualCheck{ID: "freeze", Status: "warning", Level: "warn", Detail: "freeze 检测失败: " + freezeErr.Error()})
		report.Warnings = append(report.Warnings, "freeze 检测失败: "+freezeErr.Error())
	} else if freezeEvents > 0 {
		report.Checks = append(report.Checks, visualCheck{ID: "freeze", Status: "warning", Level: "warn", Detail: fmt.Sprintf("%d 个 ≥1.5s 静帧区间", freezeEvents)})
		report.Warnings = append(report.Warnings, fmt.Sprintf("检测到 %d 个 ≥1.5s 静帧区间，请确认是否为有意停顿", freezeEvents))
	} else {
		report.Checks = append(report.Checks, visualCheck{ID: "freeze", Status: "passed", Level: "warn", Detail: "无 ≥1.5s 静帧"})
	}

	times := parseTimeList(in.Times)
	if len(times) == 0 {
		times = relativeSampleTimes(probe.Duration)
	}
	worstRun := 0
	worstLevels := 0
	bandingOK := true
	for _, sampleTime := range times {
		img, frameErr := decodeVideoFrame(path, sampleTime)
		if frameErr != nil {
			bandingOK = false
			report.Findings = append(report.Findings, issue("VISUAL_BANDING_SAMPLE", "output", fmt.Sprintf("%.2fs 抽帧失败: %v", sampleTime, frameErr), "检查 ffmpeg 与视频文件"))
			continue
		}
		banding := analyzeBanding(img, in.MaxRunRatio, in.MinLevels)
		for _, line := range banding.Lines {
			if line.MaxRunPX > worstRun {
				worstRun = line.MaxRunPX
			}
			if worstLevels == 0 || line.Unique < worstLevels {
				worstLevels = line.Unique
			}
		}
		if !banding.OK {
			bandingOK = false
		}
	}
	report.Metrics["banding_samples"] = times
	report.Metrics["banding_worst_flat_run_px"] = worstRun
	report.Metrics["banding_worst_levels"] = worstLevels
	ratio := in.MaxRunRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 0.08
	}
	minLevels := in.MinLevels
	if minLevels <= 0 {
		minLevels = 12
	}
	maxRunLimit := int(math.Max(24, math.Round(float64(probe.Width)*ratio)))
	bandingGate := bandingGateForProject(root, in.ProjectID)
	severe := worstLevels < minLevels && worstRun > maxRunLimit*2
	bandingStatus := "passed"
	bandingLevel := "error"
	if !bandingGate {
		bandingStatus = "skipped"
		bandingLevel = "warn"
	} else if !bandingOK {
		if severe {
			bandingStatus = "failed"
			report.Findings = append(report.Findings, issue("VISUAL_BANDING", "output", fmt.Sprintf("采样帧出现严重色带：最小 %d 级亮度，最大平坦段 %dpx（阈值 %dpx）", worstLevels, worstRun, maxRunLimit), "最终帧改用 PNG；移除 CSS contrast/vignette；暗部加 dither；提高 Web copy 码率"))
		} else {
			bandingStatus = "warning"
			bandingLevel = "warn"
			report.Warnings = append(report.Warnings, fmt.Sprintf("检测到轻微平坦段 %dpx（阈值 %dpx），未达到严重色带阈值，请人工复核", worstRun, maxRunLimit))
		}
	}
	bandingDetail := fmt.Sprintf("采样 %d 帧；最小亮度级数 %d；最大平坦段 %dpx（阈值 %dpx）", len(times), worstLevels, worstRun, maxRunLimit)
	if !bandingGate {
		bandingDetail = "Style Pack 声明 banding_gate=false（平面风格）；仅记录平坦段指标"
	}
	report.Checks = append(report.Checks, visualCheck{
		ID: "banding", Status: bandingStatus, Level: bandingLevel,
		Detail: bandingDetail,
	})

	if !probe.HasAudio {
		report.Checks = append(report.Checks, visualCheck{ID: "audio_loudness", Status: "skipped", Detail: "视频没有音频轨"})
	} else {
		loudnessOutput, loudnessErr := runLoudness(path)
		if loudnessErr != nil {
			report.Checks = append(report.Checks, visualCheck{ID: "audio_loudness", Status: "warning", Level: "warn", Detail: loudnessErr.Error()})
			report.Warnings = append(report.Warnings, "音频响度检测失败: "+loudnessErr.Error())
		} else {
			integrated := lastFloatMatch(loudnessOutput, `I:\s*(-?\d+(?:\.\d+)?)\s*LUFS`)
			peak := lastFloatMatch(loudnessOutput, `Peak:\s*(-?\d+(?:\.\d+)?)\s*dBFS`)
			report.Metrics["integrated_lufs"] = integrated
			report.Metrics["true_peak_dbfs"] = peak
			audioOK := integrated >= -30 && integrated <= -12 && peak <= -1
			report.Checks = append(report.Checks, visualCheck{
				ID: "audio_loudness", Status: statusFor(audioOK), Level: "error",
				Detail: fmt.Sprintf("I=%.1f LUFS / peak=%.1f dBFS", integrated, peak),
			})
			if !audioOK {
				report.Findings = append(report.Findings, issue("VISUAL_AUDIO_LOUDNESS", "output", fmt.Sprintf("响度 %.1f LUFS / true peak %.1f dBFS 超出建议范围", integrated, peak), "调整混音到 -30..-12 LUFS，true peak ≤ -1 dBFS"))
			}
		}
	}

	textChecks, textFindings, textWarnings := textAuditChecks(root, in.ProjectID, in.Times)
	report.Checks = append(report.Checks, textChecks...)
	report.Findings = append(report.Findings, textFindings...)
	report.Warnings = append(report.Warnings, textWarnings...)
	report.OK = true
	for _, check := range report.Checks {
		if check.Status == "failed" && check.Level == "error" {
			report.OK = false
		}
	}
	return report, nil
}

func applyVisualQuality(item *job, report visualReport) {
	item.QualityStatus = "passed"
	if !report.OK {
		item.QualityStatus = "failed"
	} else if len(report.Warnings) > 0 {
		item.QualityStatus = "warning"
	}
	item.QualityReport = map[string]any{
		"ok":       report.OK,
		"metrics":  report.Metrics,
		"checks":   report.Checks,
		"findings": report.Findings,
		"warnings": report.Warnings,
	}
}

func qualityVisualImage(path string, in requestInput) any {
	file, err := os.Open(path)
	if err != nil {
		return visualReport{OK: false, ImagePath: path, Metrics: map[string]any{}, Checks: []visualCheck{{ID: "image", Status: "failed", Level: "error", Detail: err.Error()}}}
	}
	defer file.Close()
	img, _, decodeErr := image.Decode(file)
	if decodeErr != nil {
		return visualReport{OK: false, ImagePath: path, Metrics: map[string]any{}, Checks: []visualCheck{{ID: "image", Status: "failed", Level: "error", Detail: decodeErr.Error()}}}
	}
	banding := analyzeBanding(img, in.MaxRunRatio, in.MinLevels)
	worstRun := 0
	worstLevels := 0
	for _, line := range banding.Lines {
		if line.MaxRunPX > worstRun {
			worstRun = line.MaxRunPX
		}
		if worstLevels == 0 || line.Unique < worstLevels {
			worstLevels = line.Unique
		}
	}
	report := visualReport{
		ImagePath: path,
		Metrics: map[string]any{
			"width": banding.Width, "height": banding.Height,
			"banding_worst_flat_run_px": worstRun, "banding_worst_levels": worstLevels,
		},
		Checks: []visualCheck{
			{ID: "image", Status: "passed", Detail: fmt.Sprintf("%dx%d", banding.Width, banding.Height)},
			{ID: "banding", Status: statusFor(banding.OK), Level: "error", Detail: fmt.Sprintf("最小亮度级数 %d；最大平坦段 %dpx", worstLevels, worstRun)},
			{ID: "black_frames", Status: "skipped", Detail: "图片输入只做色带检查"},
			{ID: "freeze", Status: "skipped", Detail: "图片输入无时间轴"},
			{ID: "audio_loudness", Status: "skipped", Detail: "图片输入无音频"},
		},
	}
	report.OK = banding.OK
	if !banding.OK {
		report.Findings = banding.Findings
	}
	return report
}

func probeVisualMedia(path string) (visualMediaProbe, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "ffprobe",
		"-v", "error", "-count_frames",
		"-show_entries", "stream=codec_type,width,height,avg_frame_rate,nb_read_frames",
		"-show_entries", "format=duration",
		"-of", "json", path,
	)
	output, err := command.Output()
	if err != nil {
		return visualMediaProbe{}, err
	}
	var payload struct {
		Streams []struct {
			CodecType    string `json:"codec_type"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			AvgFrameRate string `json:"avg_frame_rate"`
			NbReadFrames string `json:"nb_read_frames"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return visualMediaProbe{}, err
	}
	probe := visualMediaProbe{}
	for _, stream := range payload.Streams {
		if stream.CodecType == "audio" {
			probe.HasAudio = true
			continue
		}
		if stream.CodecType != "video" {
			continue
		}
		probe.Width = stream.Width
		probe.Height = stream.Height
		probe.FPS = parseFrameRate(stream.AvgFrameRate)
		probe.Frames, _ = strconv.Atoi(stream.NbReadFrames)
	}
	probe.Duration, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	return probe, nil
}

func parseFrameRate(value string) float64 {
	parts := strings.Split(value, "/")
	if len(parts) == 2 {
		num, _ := strconv.ParseFloat(parts[0], 64)
		den, _ := strconv.ParseFloat(parts[1], 64)
		if den > 0 {
			return num / den
		}
	}
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func runMediaFilter(path, filter string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-nostdin", "-i", path,
		"-vf", filter, "-an", "-f", "null", "-",
	)
	output, err := command.CombinedOutput()
	return string(output), err
}

func countDetectorEvents(output, marker string) int {
	pattern := regexp.MustCompile(marker + `:\s*[0-9.]+`)
	return len(pattern.FindAllString(output, -1))
}

func runLoudness(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-nostdin", "-i", path,
		"-map", "0:a:0", "-af", "ebur128=peak=true",
		"-f", "null", "-",
	)
	output, err := command.CombinedOutput()
	return string(output), err
}

func lastFloatMatch(output, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(matches[len(matches)-1][1], 64)
	return value
}

func parseTimeList(value string) []float64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]float64, 0, len(parts))
	for _, part := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err == nil && number >= 0 {
			result = append(result, number)
		}
	}
	return result
}

func relativeSampleTimes(duration float64) []float64 {
	if duration <= 0 {
		return []float64{2, 5, 8}
	}
	return []float64{duration * 0.15, duration * 0.5, duration * 0.85}
}

func statusFor(ok bool) string {
	if ok {
		return "passed"
	}
	return "failed"
}
