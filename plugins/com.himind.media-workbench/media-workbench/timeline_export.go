package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func timelineExportCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if len(in.MediaPaths) == 0 {
		return nil, himindjsonrpc.InvalidParams("media_paths is required")
	}
	doc := in.Timeline
	blockers := validateTimeline(doc)
	if len(blockers) > 0 {
		return map[string]any{"state": "blocked", "ok": false, "blockers": blockers}, nil
	}
	qa := timelineQAEvaluation(root, in)
	if !in.SkipQA && qa["passed"] != true {
		return map[string]any{"state": "blocked", "ok": false, "blockers": qa["blockers"], "qa": qa}, nil
	}
	videoTrack := primaryVideoTrack(doc)
	if videoTrack == nil {
		return nil, himindjsonrpc.InvalidParams("timeline has no non-empty video track")
	}
	clips := append([]timelineClip(nil), videoTrack.Clips...)
	sort.SliceStable(clips, func(left, right int) bool { return clips[left].StartFrame < clips[right].StartFrame })
	num := doc.Timebase.Num
	if num <= 0 {
		num = 30
	}
	den := doc.Timebase.Den
	if den <= 0 {
		den = 1
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg not found in PATH")
	}
	outputPath := strings.TrimSpace(in.OutputPath)
	if outputPath == "" {
		outputPath = filepath.ToSlash(filepath.Join(".himind-media", "timeline", "exports", fmt.Sprintf("timeline-%d.mp4", time.Now().UnixNano())))
	}
	resolvedOutput, err := workspaceFile(root, outputPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output_path: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(resolvedOutput), 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	preset := deliveryPresetByID(in.DeliveryPreset)
	if in.DeliveryPreset == "" {
		preset = deliveryPresetByID("master")
	}
	targetWidth, targetHeight := deliveryDimensions(preset, doc.Canvas)
	args := []string{"-hide_banner", "-loglevel", "error"}
	filterParts := make([]string, 0)
	videoInputCount := 0
	videoLabels := make([]string, 0, len(clips))
	for _, clip := range clips {
		mediaPath := strings.TrimSpace(in.MediaPaths[clip.MediaID])
		if mediaPath == "" {
			return nil, himindjsonrpc.InvalidParams("media_paths missing for " + clip.MediaID)
		}
		resolvedMedia, pathErr := workspaceFile(root, mediaPath)
		if pathErr != nil {
			return nil, himindjsonrpc.InvalidParams("media_paths: " + pathErr.Error())
		}
		start := float64(clip.SourceInFrame) * float64(den) / float64(num)
		duration := float64(clip.DurationFrames) * float64(den) / float64(num)
		sourceFrames := clip.SourceOutFrame - clip.SourceInFrame
		if sourceFrames <= 0 {
			sourceFrames = clip.DurationFrames
		}
		sourceDuration := float64(sourceFrames) * float64(den) / float64(num)
		isImage := isImagePath(resolvedMedia)
		if isImage {
			args = append(args, "-loop", "1", "-t", fmt.Sprintf("%.6f", duration), "-i", resolvedMedia)
		} else {
			args = append(args, "-ss", fmt.Sprintf("%.6f", start), "-t", fmt.Sprintf("%.6f", sourceDuration), "-i", resolvedMedia)
		}
		label := fmt.Sprintf("[v%d]", videoInputCount)
		filterParts = append(filterParts, buildVideoClipFilter(root, label, videoInputCount, clip, targetWidth, targetHeight, num, den))
		videoLabels = append(videoLabels, label)
		videoInputCount++
	}
	videoSequence, sequenceOut, sequenceFrames := buildVideoSequence(videoLabels, clips, num, den)
	filterParts = append(filterParts, videoSequence...)
	videoOutLabel := sequenceOut
	overlayParts, overlayOut, nextInput := buildOverlayTracks(root, in, doc, videoTrack, videoOutLabel, num, den, videoInputCount, &args)
	if len(overlayParts) > 0 {
		filterParts = append(filterParts, overlayParts...)
		videoOutLabel = overlayOut
	}
	videoInputCount = nextInput
	if in.BurnSubtitles {
		cues := subtitleCuesFromTimeline(doc)
		if len(cues) == 0 && len(in.Subtitles) > 0 {
			cues = in.Subtitles
		}
		if len(cues) > 0 {
			subtitlePath := strings.TrimSpace(in.SubtitlePath)
			if subtitlePath == "" {
				subtitlePath = filepath.ToSlash(filepath.Join(filepath.Dir(outputPath), "subtitle-"+fmt.Sprintf("%d", time.Now().UnixNano())+".srt"))
			}
			resolvedSubtitle, pathErr := workspaceFile(root, subtitlePath)
			if pathErr != nil {
				return nil, himindjsonrpc.InvalidParams("subtitle_path: " + pathErr.Error())
			}
			if writeErr := writeSRT(resolvedSubtitle, cues); writeErr != nil {
				return nil, himindjsonrpc.InternalError("write subtitles: " + writeErr.Error())
			}
			style := subtitleASS(in.SubtitleStyle)
			filterParts = append(filterParts, fmt.Sprintf("%ssubtitles=filename='%s':force_style='%s'[vsub]", videoOutLabel, ffmpegFilterPath(resolvedSubtitle), style))
			videoOutLabel = "[vsub]"
		}
	}
	audioInputStart := videoInputCount
	audioFilters, audioLabels, audioRoles := buildAudioFilters(root, in, doc, num, den, audioInputStart, &args)
	if len(audioFilters) > 0 {
		filterParts = append(filterParts, audioFilters...)
	}
	filterParts = append(filterParts, buildVideoEncodeFilter(videoOutLabel, "[vout]", targetWidth, targetHeight, num, den, preset)...)
	maps := []string{"-map", "[vout]"}
	hasAudio := false
	if len(audioLabels) > 0 {
		filterParts = append(filterParts, buildAudioMixFilter(audioLabels, audioRoles, preset.LoudnessLUFS)...)
		maps = append(maps, "-map", "[aout]", "-c:a", "aac", "-b:a", preset.AudioBitrate)
		hasAudio = true
	}
	args = append(args, "-filter_complex", strings.Join(filterParts, ";"))
	args = append(args, maps...)
	args = append(args, "-r", fmt.Sprintf("%d/%d", num, den), "-c:v", "libx264", "-preset", "medium", "-crf", fmt.Sprintf("%d", preset.CRF), "-pix_fmt", "yuv420p")
	if preset.MaxRate != "" {
		args = append(args, "-maxrate", preset.MaxRate, "-bufsize", doubleRate(preset.MaxRate))
	}
	args = append(args,
		"-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1",
		"-movflags", "+faststart", "-y", resolvedOutput,
	)
	command := exec.Command(ffmpeg, args...)
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		return nil, himindjsonrpc.InternalError("timeline export failed: " + strings.TrimSpace(string(output)))
	}
	hash, _ := fileSHA256(resolvedOutput)
	relPath, _ := filepath.Rel(root, resolvedOutput)
	info, _ := os.Stat(resolvedOutput)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	durationSeconds := float64(sequenceFrames) * float64(den) / float64(num)
	return map[string]any{
		"state": "ready", "ok": true,
		"path": filepath.ToSlash(relPath), "sha256": hash, "size": size,
		"clip_count": len(clips), "has_audio": hasAudio, "duration_seconds": durationSeconds,
		"timebase": fmt.Sprintf("%d/%d", num, den), "preset": preset.ID,
		"width": targetWidth, "height": targetHeight, "qa": qa,
		"next": "调用 delivery.qa 校验平台交付；保留 Timeline 与媒体引用以便重渲染",
	}, nil
}

func clipSpeed(clip timelineClip) float64 {
	if clip.Speed <= 0 {
		return 1
	}
	if clip.Speed < 0.25 {
		return 0.25
	}
	if clip.Speed > 4 {
		return 4
	}
	return clip.Speed
}

func colorKeyframeValue(keyframe timelineColorKeyframe, field string) float64 {
	switch field {
	case "brightness":
		return math.Max(-1, math.Min(1, keyframe.Brightness))
	case "contrast":
		if keyframe.Contrast <= 0 {
			return 1
		}
		return keyframe.Contrast
	case "saturation":
		if keyframe.Saturation <= 0 {
			return 1
		}
		return keyframe.Saturation
	case "gamma":
		if keyframe.Gamma <= 0 {
			return 1
		}
		return keyframe.Gamma
	}
	return 1
}

func colorKeyframeExpr(keyframes []timelineColorKeyframe, num, den int, field string) string {
	if len(keyframes) == 0 {
		return "0"
	}
	times := make([]float64, len(keyframes))
	values := make([]float64, len(keyframes))
	for index, keyframe := range keyframes {
		times[index] = float64(keyframe.Frame) * float64(den) / float64(num)
		values[index] = colorKeyframeValue(keyframe, field)
	}
	last := len(keyframes) - 1
	expr := fmt.Sprintf("%.6f", values[last])
	for index := last - 1; index >= 0; index-- {
		t0, t1 := times[index], times[index+1]
		v0, v1 := values[index], values[index+1]
		if t1 <= t0 {
			expr = fmt.Sprintf("%.6f", v1)
			continue
		}
		interpolated := fmt.Sprintf("%.6f+(%.6f-%.6f)*(t-%.6f)/(%.6f-%.6f)", v0, v1, v0, t0, t1, t0)
		expr = fmt.Sprintf("if(lt(t,%.6f),%s,%s)", t1, interpolated, expr)
	}
	return expr
}

func buildVideoClipFilter(root, label string, index int, clip timelineClip, width, height, num, den int) string {
	fit := "cover"
	positionX, positionY := 0.0, 0.0
	if clip.Transform != nil {
		if clip.Transform.Fit != "" {
			fit = clip.Transform.Fit
		}
		positionX = clip.Transform.PositionX
		positionY = clip.Transform.PositionY
	}
	var filter string
	if fit == "contain" {
		filter = fmt.Sprintf("[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1,setpts=PTS/%.6f,fps=%d/%d", index, width, height, width, height, clipSpeed(clip), num, den)
	} else {
		x := fmt.Sprintf("(in_w-out_w)/2+%.4f*in_w", positionX)
		y := fmt.Sprintf("(in_h-out_h)/2+%.4f*in_h", positionY)
		filter = fmt.Sprintf("[%d:v]scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d:%s:%s,setsar=1,setpts=PTS/%.6f,fps=%d/%d", index, width, height, width, height, x, y, clipSpeed(clip), num, den)
	}
	if clip.Opacity > 0 && clip.Opacity < 1 {
		filter += fmt.Sprintf(",format=rgba,colorchannelmixer=aa=%.4f", math.Max(0, math.Min(1, clip.Opacity)))
	}
	if clip.Color != nil {
		brightness := math.Max(-1, math.Min(1, clip.Color.Brightness))
		contrast := clip.Color.Contrast
		if contrast <= 0 {
			contrast = 1
		}
		saturation := clip.Color.Saturation
		if saturation <= 0 {
			saturation = 1
		}
		gamma := clip.Color.Gamma
		if gamma <= 0 {
			gamma = 1
		}
		if len(clip.Color.Keyframes) > 0 {
			brightnessExpr := colorKeyframeExpr(clip.Color.Keyframes, num, den, "brightness")
			contrastExpr := colorKeyframeExpr(clip.Color.Keyframes, num, den, "contrast")
			saturationExpr := colorKeyframeExpr(clip.Color.Keyframes, num, den, "saturation")
			gammaExpr := colorKeyframeExpr(clip.Color.Keyframes, num, den, "gamma")
			filter += fmt.Sprintf(",eq=eval=frame:brightness='%s':contrast='%s':saturation='%s':gamma='%s'", brightnessExpr, contrastExpr, saturationExpr, gammaExpr)
		} else if brightness != 0 || contrast != 1 || saturation != 1 || gamma != 1 {
			filter += fmt.Sprintf(",eq=brightness=%.4f:contrast=%.4f:saturation=%.4f:gamma=%.4f", brightness, contrast, saturation, gamma)
		}
		if clip.Color.LUT != "" {
			if lutPath, err := workspaceFile(root, clip.Color.LUT); err == nil {
				filter += fmt.Sprintf(",lut3d='%s'", ffmpegFilterPath(lutPath))
			}
		}
	}
	return filter + ",format=yuv420p" + label
}

func primaryVideoTrack(doc timelineDoc) *timelineTrack {
	for index := range doc.Tracks {
		if doc.Tracks[index].Kind == "video" && !doc.Tracks[index].Muted && len(doc.Tracks[index].Clips) > 0 {
			return &doc.Tracks[index]
		}
	}
	return nil
}

func buildOverlayTracks(root string, in requestInput, doc timelineDoc, primary *timelineTrack, baseLabel string, num, den, startIndex int, args *[]string) ([]string, string, int) {
	parts := make([]string, 0)
	acc := baseLabel
	inputIndex := startIndex
	for _, track := range doc.Tracks {
		if track.Kind != "video" || track.Muted || track.ID == primary.ID || len(track.Clips) == 0 {
			continue
		}
		clips := append([]timelineClip(nil), track.Clips...)
		sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
		for _, clip := range clips {
			mediaPath := strings.TrimSpace(in.MediaPaths[clip.MediaID])
			if mediaPath == "" {
				continue
			}
			resolved, err := workspaceFile(root, mediaPath)
			if err != nil {
				continue
			}
			sourceFrames := clip.SourceOutFrame - clip.SourceInFrame
			if sourceFrames <= 0 {
				sourceFrames = clip.DurationFrames
			}
			sourceDuration := float64(sourceFrames) * float64(den) / float64(num)
			start := float64(clip.SourceInFrame) * float64(den) / float64(num)
			timelineStart := float64(clip.StartFrame) * float64(den) / float64(num)
			timelineEnd := timelineStart + float64(clip.DurationFrames)*float64(den)/float64(num)
			if isImagePath(resolved) {
				*args = append(*args, "-loop", "1", "-t", fmt.Sprintf("%.6f", timelineEnd-timelineStart), "-i", resolved)
			} else {
				*args = append(*args, "-ss", fmt.Sprintf("%.6f", start), "-t", fmt.Sprintf("%.6f", sourceDuration), "-i", resolved)
			}
			inputLabel := fmt.Sprintf("[ovin%d]", inputIndex)
			outputLabel := fmt.Sprintf("[ovout%d]", inputIndex)
			scale := 0.35
			posX, posY := 0.0, 0.0
			opacity := 1.0
			if clip.Transform != nil {
				if clip.Transform.Scale > 0 {
					scale = clip.Transform.Scale
				}
				posX = clip.Transform.PositionX
				posY = clip.Transform.PositionY
			}
			if clip.Opacity > 0 && clip.Opacity < 1 {
				opacity = clip.Opacity
			}
			parts = append(parts, fmt.Sprintf("[%d:v]scale=trunc(iw*%.4f/2)*2:-2,format=rgba,colorchannelmixer=aa=%.4f,setpts=PTS-STARTPTS+%.6f/TB%s", inputIndex, scale, opacity, timelineStart, inputLabel))
			parts = append(parts, fmt.Sprintf("%s%soverlay=x=(W-w)/2+%.4f*W:y=(H-h)/2+%.4f*H:enable='between(t,%.6f,%.6f)':eof_action=pass%s", acc, inputLabel, posX, posY, timelineStart, timelineEnd, outputLabel))
			acc = outputLabel
			inputIndex++
		}
	}
	return parts, acc, inputIndex
}

func buildVideoSequence(labels []string, clips []timelineClip, num, den int) ([]string, string, int) {
	if len(labels) == 0 {
		return nil, "[vcat]", 0
	}
	parts := make([]string, 0, len(labels)+1)
	acc := labels[0]
	durationFrames := clips[0].DurationFrames
	for index := 1; index < len(labels); index++ {
		transitionFrames := 0
		if clips[index].TransitionIn != nil {
			transitionFrames = clips[index].TransitionIn.DurationFrames
		}
		if transitionFrames > 0 && transitionFrames < durationFrames && transitionFrames < clips[index].DurationFrames {
			output := fmt.Sprintf("[vx%d]", index)
			offset := float64(durationFrames-transitionFrames) * float64(den) / float64(num)
			duration := float64(transitionFrames) * float64(den) / float64(num)
			parts = append(parts, fmt.Sprintf("%s%sxfade=transition=fade:duration=%.6f:offset=%.6f%s", acc, labels[index], duration, offset, output))
			acc = output
			durationFrames += clips[index].DurationFrames - transitionFrames
		} else {
			output := fmt.Sprintf("[vc%d]", index)
			parts = append(parts, fmt.Sprintf("%s%sconcat=n=2:v=1:a=0%s", acc, labels[index], output))
			acc = output
			durationFrames += clips[index].DurationFrames
		}
	}
	return parts, acc, durationFrames
}

func buildVideoEncodeFilter(inLabel, outLabel string, width, height, num, den int, preset deliveryPreset) []string {
	parts := []string{fmt.Sprintf("%sscale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1%s", inLabel, width, height, width, height, outLabel), fmt.Sprintf("%ssetpts=N/(%d/%d)/TB%s", inLabel, num, den, outLabel)}
	return parts[:1]
}

func audioClipGain(track timelineTrack, clip timelineClip) float64 {
	totalDB := track.VolumeDB + clip.VolumeDB
	if totalDB == 0 {
		return 1
	}
	return math.Pow(10, totalDB/20)
}

func anyAudioSolo(doc timelineDoc) bool {
	for _, track := range doc.Tracks {
		if track.Kind == "audio" && track.Solo {
			return true
		}
	}
	return false
}

func audioTrackAudible(doc timelineDoc, track timelineTrack) bool {
	if anyAudioSolo(doc) {
		return track.Solo
	}
	return !track.Muted
}

func buildAudioFilters(root string, in requestInput, doc timelineDoc, num, den, inputStart int, args *[]string) ([]string, []string, []string) {
	filters := make([]string, 0)
	labels := make([]string, 0)
	roles := make([]string, 0)
	inputIndex := inputStart
	for _, track := range doc.Tracks {
		if track.Kind != "audio" || !audioTrackAudible(doc, track) {
			continue
		}
		clips := append([]timelineClip(nil), track.Clips...)
		sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
		for _, clip := range clips {
			mediaPath := strings.TrimSpace(in.MediaPaths[clip.MediaID])
			if mediaPath == "" {
				continue
			}
			resolved, err := workspaceFile(root, mediaPath)
			if err != nil {
				continue
			}
			start := float64(clip.SourceInFrame) * float64(den) / float64(num)
			sourceFrames := clip.SourceOutFrame - clip.SourceInFrame
			if sourceFrames <= 0 {
				sourceFrames = clip.DurationFrames
			}
			duration := float64(sourceFrames) * float64(den) / float64(num)
			delayMs := int(math.Round(float64(clip.StartFrame) * float64(den) / float64(num) * 1000))
			*args = append(*args, "-ss", fmt.Sprintf("%.6f", start), "-t", fmt.Sprintf("%.6f", duration), "-i", resolved)
			label := fmt.Sprintf("[a%d]", inputIndex)
			gain := audioClipGain(track, clip)
			speed := clipSpeed(clip)
			filters = append(filters, fmt.Sprintf("[%d:a]atrim=duration=%.6f,asetpts=PTS-STARTPTS,atempo=%.6f,volume=%.6f,adelay=%d|%d%s", inputIndex, duration, speed, gain, delayMs, delayMs, label))
			labels = append(labels, label)
			role := track.Role
			if role == "" {
				role = "mix"
			}
			roles = append(roles, role)
			inputIndex++
		}
	}
	return filters, labels, roles
}

func buildAudioMixFilter(labels, roles []string, targetLUFS float64) []string {
	voiceLabels := make([]string, 0)
	musicLabels := make([]string, 0)
	otherLabels := make([]string, 0)
	for index, label := range labels {
		switch roles[index] {
		case "narration", "voice":
			voiceLabels = append(voiceLabels, label)
		case "music", "bgm":
			musicLabels = append(musicLabels, label)
		default:
			otherLabels = append(otherLabels, label)
		}
	}
	parts := make([]string, 0)
	if len(otherLabels) > 0 {
		parts = append(parts, strings.Join(otherLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[aother]", len(otherLabels)))
	}
	if len(voiceLabels) > 0 && len(musicLabels) > 0 {
		parts = append(parts,
			strings.Join(voiceLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0,asplit=2[vmix][vkey]", len(voiceLabels)),
			strings.Join(musicLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[music]", len(musicLabels)),
			"[music][vkey]sidechaincompress=threshold=0.035:ratio=6:attack=20:release=350[ducked]",
		)
		if len(otherLabels) > 0 {
			parts = append(parts, "[vmix][ducked][aother]amix=inputs=3:duration=longest:normalize=0[loudin]")
		} else {
			parts = append(parts, "[vmix][ducked]amix=inputs=2:duration=longest:normalize=0[loudin]")
		}
	} else if len(voiceLabels) > 0 {
		parts = append(parts, strings.Join(voiceLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[loudin]", len(voiceLabels)))
	} else if len(musicLabels) > 0 {
		parts = append(parts, strings.Join(musicLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0[loudin]", len(musicLabels)))
	} else if len(otherLabels) > 0 {
		parts = append(parts, "[aother]anull[loudin]")
	}
	if len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("[loudin]loudnorm=I=%.1f:TP=-1.5:LRA=11[aout]", targetLUFS))
	}
	return parts
}

func timelineDurationSeconds(doc timelineDoc, num, den int) float64 {
	if doc.DurationFrames > 0 {
		return float64(doc.DurationFrames) * float64(den) / float64(num)
	}
	maxEnd := 0
	for _, track := range doc.Tracks {
		for _, clip := range track.Clips {
			if clipEnd(clip) > maxEnd {
				maxEnd = clipEnd(clip)
			}
		}
	}
	return float64(maxEnd) * float64(den) / float64(num)
}

func isImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".bmp", ".gif":
		return true
	}
	return false
}

func subtitleASS(style string) string {
	if strings.TrimSpace(style) == "" {
		style = "FontName=Microsoft YaHei,FontSize=20,PrimaryColour=&H00FFFFFF,OutlineColour=&H00000000,BackColour=&H80000000,BorderStyle=1,Outline=2,Shadow=1,Alignment=2,MarginV=60"
	}
	return strings.ReplaceAll(style, "'", "")
}

func ffmpegFilterPath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, ":", "\\:")
	path = strings.ReplaceAll(path, "'", "\\'")
	path = strings.ReplaceAll(path, ",", "\\,")
	return path
}

func doubleRate(rate string) string {
	rate = strings.TrimSpace(rate)
	if strings.HasSuffix(rate, "M") {
		var n float64
		_, _ = fmt.Sscanf(rate, "%fM", &n)
		return fmt.Sprintf("%.0fM", n*2)
	}
	return "16M"
}

// renderTimeline keeps a single render entry: timeline input is handled by the
// same render.start capability, using the deterministic ffmpeg adapter.
func renderTimeline(root string, in requestInput) (any, *himindjsonrpc.Error) {
	jobID := newID("timeline-render")
	artifactID := newID("video-artifact")
	artifactPath := filepath.Join(metadataDir, artifactDir, artifactID+".mp4")
	in.OutputPath = filepath.ToSlash(artifactPath)
	result, rpcErr := timelineExportCapability(in)
	if rpcErr != nil {
		return nil, rpcErr
	}
	payload, _ := result.(map[string]any)
	if payload == nil || payload["state"] != "ready" {
		return payload, nil
	}
	return map[string]any{
		"state": "completed", "adapter": "timeline-ffmpeg",
		"job": map[string]any{
			"job_id": jobID, "project_id": "timeline", "stage": "render",
			"status": "completed", "progress": 100, "artifact_id": artifactID,
			"artifact_path": artifactPath, "message": "Timeline 已导出",
		},
		"artifact": payload,
		"next":     "使用 artifact.export 获取路径与 SHA，或继续编辑 Timeline",
	}, nil
}
