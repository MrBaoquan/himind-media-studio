package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type upscaleTarget struct {
	Width  int
	Height int
	Scale  float64
}

func visualUpscaleCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if strings.TrimSpace(in.ImagePath) != "" && strings.TrimSpace(in.VideoPath) != "" {
		return nil, himindjsonrpc.InvalidParams("image_path and video_path are mutually exclusive")
	}
	if strings.TrimSpace(in.ImagePath) == "" && strings.TrimSpace(in.VideoPath) == "" {
		return nil, himindjsonrpc.InvalidParams("image_path or video_path is required")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg not found in PATH")
	}
	isVideo := strings.TrimSpace(in.VideoPath) != ""
	inputPath := in.ImagePath
	if isVideo {
		inputPath = in.VideoPath
	}
	resolvedInput, err := workspaceFile(root, inputPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("input: " + err.Error())
	}
	info, statErr := os.Stat(resolvedInput)
	if statErr != nil || info.IsDir() || info.Size() == 0 {
		return nil, himindjsonrpc.InvalidParams("input must be a non-empty file")
	}
	scale := in.Scale
	if scale <= 0 {
		scale = 2
	}
	if scale < 1.05 || scale > 8 {
		return nil, himindjsonrpc.InvalidParams("scale must be within 1.05..8")
	}
	target := upscaleTarget{Scale: scale, Width: in.Width, Height: in.Height}
	if isVideo {
		ffprobe, probeErr := exec.LookPath("ffprobe")
		if probeErr != nil {
			return nil, himindjsonrpc.InternalError("ffprobe not found in PATH")
		}
		probe, probeErr := probeVideo(ffprobe, resolvedInput)
		if probeErr != nil {
			return nil, himindjsonrpc.InvalidParams("probe video: " + probeErr.Error())
		}
		target = resolveTarget(probe.Width, probe.Height, target, true)
	} else {
		width, height, mime := imageMetadata(resolvedInput)
		if mime == "" {
			return nil, himindjsonrpc.InvalidParams("input image cannot be decoded")
		}
		target = resolveTarget(width, height, target, false)
	}
	if target.Width < 2 || target.Height < 2 || target.Width > 8192 || target.Height > 8192 {
		return nil, himindjsonrpc.InvalidParams("target dimensions out of range (2..8192)")
	}
	outputPath := strings.TrimSpace(in.OutputPath)
	if outputPath == "" {
		extension := filepath.Ext(resolvedInput)
		if isVideo {
			extension = ".mp4"
		} else if extension == "" {
			extension = ".png"
		}
		base := strings.TrimSuffix(filepath.Base(resolvedInput), filepath.Ext(resolvedInput))
		outputPath = filepath.ToSlash(filepath.Join(filepath.Dir(workspaceRelative(root, resolvedInput)), fmt.Sprintf("%s@%gx%s", base, scale, extension)))
	}
	resolvedOutput, err := workspaceFile(root, outputPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output_path: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(resolvedOutput), 0755); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	filter := fmt.Sprintf("scale=%d:%d:flags=lanczos", target.Width, target.Height)
	args := []string{"-hide_banner", "-loglevel", "error", "-i", resolvedInput, "-vf", filter}
	if isVideo {
		crf := in.CRF
		if crf <= 0 {
			crf = 18
		}
		preset := strings.TrimSpace(in.Preset)
		if preset == "" {
			preset = "medium"
		}
		args = append(args, "-c:v", "libx264", "-crf", strconv.Itoa(crf), "-preset", preset, "-pix_fmt", "yuv420p")
		if in.Mute {
			args = append(args, "-an")
		} else {
			args = append(args, "-c:a", "aac", "-b:a", "192k")
		}
		args = append(args, "-movflags", "+faststart")
	} else {
		args = append(args, "-frames:v", "1")
	}
	args = append(args, "-y", resolvedOutput)
	command := exec.Command(ffmpeg, args...)
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		return nil, himindjsonrpc.InternalError("ffmpeg upscale failed: " + strings.TrimSpace(string(output)))
	}
	hash, _ := fileSHA256(resolvedOutput)
	fileInfo, _ := os.Stat(resolvedOutput)
	size := int64(0)
	if fileInfo != nil {
		size = fileInfo.Size()
	}
	return map[string]any{
		"state": "ready", "ok": true,
		"input": map[string]any{"path": workspaceRelative(root, resolvedInput), "size": info.Size()},
		"output": map[string]any{
			"path": workspaceRelative(root, resolvedOutput), "width": target.Width, "height": target.Height,
			"size": size, "sha256": hash,
		},
		"scale": target.Scale, "media": map[string]any{"kind": map[bool]string{true: "video", false: "image"}[isVideo]},
		"note": "确定性 Lanczos 放大，不增加真实细节；需要生成式超分时由模型 adapter 接管",
	}, nil
}

func resolveTarget(width, height int, target upscaleTarget, even bool) upscaleTarget {
	if width <= 0 || height <= 0 {
		return target
	}
	if target.Width > 0 && target.Height > 0 {
		if even {
			target.Width = evenFloor(target.Width)
			target.Height = evenFloor(target.Height)
		}
		return target
	}
	if target.Width > 0 {
		target.Height = int(float64(height) * float64(target.Width) / float64(width))
	} else if target.Height > 0 {
		target.Width = int(float64(width) * float64(target.Height) / float64(height))
	} else {
		target.Width = int(float64(width) * target.Scale)
		target.Height = int(float64(height) * target.Scale)
	}
	if even {
		target.Width = evenFloor(target.Width)
		target.Height = evenFloor(target.Height)
	}
	if target.Width < 2 {
		target.Width = 2
	}
	if target.Height < 2 {
		target.Height = 2
	}
	return target
}

func evenFloor(value int) int {
	if value%2 == 1 {
		return value - 1
	}
	return value
}
