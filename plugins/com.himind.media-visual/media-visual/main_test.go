package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, path string, shade uint8) {
	t.Helper()
	writeTestPNGPattern(t, path, shade, false)
}

func writeTestPNGPattern(t *testing.T, path string, shade uint8, inverted bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			r := shade
			g := uint8((x * 4) % 256)
			b := uint8((y * 4) % 256)
			if inverted {
				r, g, b = 255-r, 255-g, 255-b
			}
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func TestReferencePackAndValidate(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "refs", "hero.png")
	writeTestPNG(t, source, 32)
	result, rpcErr := referencePackCapability(requestInput{
		WorkspaceRoot:   root,
		PackID:          "hefei-hero",
		Version:         "1.0.0",
		Name:            "合肥人造太阳主角",
		ReferenceImages: []string{"refs/hero.png"},
		Character:       "tokamak-host",
		License:         "CC-BY-4.0",
		CommercialUse:   true,
	})
	if rpcErr != nil {
		t.Fatalf("pack failed: %+v", rpcErr)
	}
	payload := result.(map[string]any)
	packPath, _ := payload["path"].(string)
	if packPath == "" {
		t.Fatal("pack path missing")
	}
	validated, rpcErr := referenceValidateCapability(requestInput{
		WorkspaceRoot:   root,
		PackPath:        packPath,
		ReferenceAssets: []string{"refs/hero.png"},
	})
	if rpcErr != nil {
		t.Fatalf("validate failed: %+v", rpcErr)
	}
	report := validated.(map[string]any)
	if report["ok"] != true {
		t.Fatalf("expected ok report, got %+v", report)
	}
	manifest := report["pack"].(assetPackManifest)
	if manifest.Schema != "himind.asset-pack.v1" || len(manifest.Resources) != 1 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if manifest.Resources[0].SHA256 == "" || manifest.Resources[0].Width != 64 {
		t.Fatalf("resource metadata incomplete: %+v", manifest.Resources[0])
	}
}

func TestReferenceValidateDetectsTamperAndMissing(t *testing.T) {
	root := t.TempDir()
	writeTestPNG(t, filepath.Join(root, "refs", "hero.png"), 32)
	result, rpcErr := referencePackCapability(requestInput{
		WorkspaceRoot: root, PackID: "tamper-pack",
		ReferenceImages: []string{"refs/hero.png"}, License: "MIT",
	})
	if rpcErr != nil {
		t.Fatalf("pack failed: %+v", rpcErr)
	}
	packPath := result.(map[string]any)["path"].(string)
	resourcePath := filepath.Join(root, filepath.FromSlash(packPath), "resources", "01-hero.png")
	if err := os.WriteFile(resourcePath, []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	validated, rpcErr := referenceValidateCapability(requestInput{WorkspaceRoot: root, PackPath: packPath})
	if rpcErr != nil {
		t.Fatalf("validate failed: %+v", rpcErr)
	}
	report := validated.(map[string]any)
	if report["ok"] != false || report["state"] != "blocked" {
		t.Fatalf("tamper not detected: %+v", report)
	}
	blockers := report["blockers"].([]map[string]any)
	if len(blockers) == 0 || blockers[0]["code"] != "REFERENCE_HASH_MISMATCH" {
		t.Fatalf("expected hash mismatch, got %+v", blockers)
	}
	if err := os.Remove(resourcePath); err != nil {
		t.Fatal(err)
	}
	missing, rpcErr := referenceValidateCapability(requestInput{WorkspaceRoot: root, PackPath: packPath})
	if rpcErr != nil {
		t.Fatalf("validate failed: %+v", rpcErr)
	}
	missingReport := missing.(map[string]any)
	missingBlockers := missingReport["blockers"].([]map[string]any)
	if len(missingBlockers) == 0 || missingBlockers[0]["code"] != "REFERENCE_RESOURCE_MISSING" {
		t.Fatalf("expected missing resource, got %+v", missingBlockers)
	}
}

func TestImageCompareDeterministic(t *testing.T) {
	root := t.TempDir()
	same := filepath.Join(root, "same.png")
	other := filepath.Join(root, "other.png")
	writeTestPNG(t, same, 32)
	writeTestPNGPattern(t, other, 240, true)
	identical, rpcErr := imageCompareCapability(requestInput{WorkspaceRoot: root, ReferencePath: "same.png", CandidatePath: "same.png"})
	if rpcErr != nil {
		t.Fatalf("compare failed: %+v", rpcErr)
	}
	if score := identical.(map[string]any)["score"].(float64); score != 1 {
		t.Fatalf("identical score = %v", score)
	}
	different, rpcErr := imageCompareCapability(requestInput{WorkspaceRoot: root, ReferencePath: "same.png", CandidatePath: "other.png"})
	if rpcErr != nil {
		t.Fatalf("compare failed: %+v", rpcErr)
	}
	report := different.(map[string]any)
	if report["ok"] != false {
		t.Fatalf("expected below-threshold result: %+v", report)
	}
	if score := report["score"].(float64); score >= 1 {
		t.Fatalf("different images scored too high: %v", score)
	}
}

func TestWorkspaceEscapeRejected(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestPNG(t, filepath.Join(parent, "outside.png"), 64)
	_, rpcErr := referencePackCapability(requestInput{
		WorkspaceRoot: root, PackID: "escape-pack",
		ReferenceImages: []string{"../outside.png"},
	})
	if rpcErr == nil {
		t.Fatal("expected workspace escape rejection")
	}
}

func writeBandedPNG(t *testing.T, path string, steps int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			bucket := (x * steps) / 256
			value := uint8(bucket * (255 / maxInt(steps-1, 1)))
			img.Set(x, y, color.RGBA{R: value, G: value, B: value, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func writeSmoothPNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			jitter := ((x*7 + y*13) % 3) - 1
			value := x + jitter
			if value < 0 {
				value = 0
			}
			if value > 255 {
				value = 255
			}
			img.Set(x, y, color.RGBA{R: uint8(value), G: uint8(value), B: uint8(value), A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func TestImageQADetectsBanding(t *testing.T) {
	root := t.TempDir()
	banded := filepath.Join(root, "banded.png")
	smooth := filepath.Join(root, "smooth.png")
	writeBandedPNG(t, banded, 16)
	writeSmoothPNG(t, smooth)
	bandedResult, rpcErr := imageQACapability(requestInput{WorkspaceRoot: root, ImagePath: "banded.png"})
	if rpcErr != nil {
		t.Fatalf("qa failed: %+v", rpcErr)
	}
	smoothResult, rpcErr := imageQACapability(requestInput{WorkspaceRoot: root, ImagePath: "smooth.png"})
	if rpcErr != nil {
		t.Fatalf("qa failed: %+v", rpcErr)
	}
	bandedMetrics := bandedResult.(map[string]any)["metrics"].(imageMetrics)
	smoothMetrics := smoothResult.(map[string]any)["metrics"].(imageMetrics)
	if bandedMetrics.BandingScore <= smoothMetrics.BandingScore {
		t.Fatalf("banded score %v should exceed smooth score %v", bandedMetrics.BandingScore, smoothMetrics.BandingScore)
	}
	if bandedMetrics.BandingScore < 0.6 {
		t.Fatalf("banded image not flagged: %+v", bandedMetrics)
	}
	warnings := bandedResult.(map[string]any)["warnings"].([]map[string]any)
	if len(warnings) == 0 || warnings[0]["code"] != "IMAGE_BANDING_HIGH" {
		t.Fatalf("expected banding warning, got %+v", warnings)
	}
}

func TestReferenceConsistency(t *testing.T) {
	root := t.TempDir()
	writeTestPNG(t, filepath.Join(root, "ref", "hero.png"), 40)
	writeTestPNGPattern(t, filepath.Join(root, "cand", "good.png"), 40, false)
	writeTestPNGPattern(t, filepath.Join(root, "cand", "drift.png"), 40, true)
	packed, rpcErr := referencePackCapability(requestInput{
		WorkspaceRoot: root, PackID: "hero-pack", ReferenceImages: []string{"ref/hero.png"}, License: "MIT",
	})
	if rpcErr != nil {
		t.Fatalf("pack failed: %+v", rpcErr)
	}
	packPath := packed.(map[string]any)["path"].(string)
	report, rpcErr := referenceConsistencyCapability(requestInput{
		WorkspaceRoot: root, PackPath: packPath, CandidatePaths: []string{"cand/good.png"}, Threshold: 0.75,
	})
	if rpcErr != nil {
		t.Fatalf("consistency failed: %+v", rpcErr)
	}
	if state := report.(map[string]any)["state"]; state != "consistent" {
		t.Fatalf("expected consistent, got %v", state)
	}
	drift, rpcErr := referenceConsistencyCapability(requestInput{
		WorkspaceRoot: root, PackPath: packPath, CandidatePaths: []string{"cand/drift.png"}, Threshold: 0.75,
	})
	if rpcErr != nil {
		t.Fatalf("consistency failed: %+v", rpcErr)
	}
	driftReport := drift.(map[string]any)
	if driftReport["state"] != "drift" {
		t.Fatalf("expected drift, got %+v", driftReport)
	}
	blockers := driftReport["blockers"].([]map[string]any)
	if len(blockers) == 0 || blockers[0]["code"] != "REFERENCE_DRIFT" {
		t.Fatalf("expected drift blocker, got %+v", blockers)
	}
}

func TestParseFrameTimes(t *testing.T) {
	times := parseFrameTimes("3, 1.5,1.5, bad, -2, 0")
	want := []float64{0, 1.5, 3}
	if len(times) != len(want) {
		t.Fatalf("parseFrameTimes = %v, want %v", times, want)
	}
	for index := range want {
		if times[index] != want[index] {
			t.Fatalf("parseFrameTimes = %v, want %v", times, want)
		}
	}
}

func TestDefaultFrameTimes(t *testing.T) {
	times := defaultFrameTimes(72, 5)
	if len(times) != 5 {
		t.Fatalf("expected 5 samples, got %v", times)
	}
	if times[0] <= 0 || times[len(times)-1] >= 72 {
		t.Fatalf("samples out of range: %v", times)
	}
	for index := 1; index < len(times); index++ {
		if times[index] <= times[index-1] {
			t.Fatalf("samples not increasing: %v", times)
		}
	}
	if len(defaultFrameTimes(0, 5)) != 0 {
		t.Fatal("zero duration should yield no samples")
	}
}

func TestVideoQADetectsBanding(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	root := t.TempDir()
	makeMixedVideo(t, root)
	result, rpcErr := videoQACapability(requestInput{
		WorkspaceRoot: root, VideoPath: "mixed.mp4", Times: "0.5,1.5",
	})
	if rpcErr != nil {
		t.Fatalf("video qa failed: %+v", rpcErr)
	}
	payload := result.(map[string]any)
	frames := payload["frames"].([]frameQA)
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[0].Metrics.BandingScore <= frames[1].Metrics.BandingScore {
		t.Fatalf("banded frame %v should exceed smooth frame %v", frames[0].Metrics.BandingScore, frames[1].Metrics.BandingScore)
	}
	summary := payload["summary"].(map[string]any)
	worst := summary["worst_banding"].(map[string]any)
	if worst["score"].(float64) < 0.6 {
		t.Fatalf("worst banding too low: %+v", worst)
	}
	if payload["state"] != "ready" {
		t.Fatalf("unexpected state: %+v", payload["state"])
	}
}
func makeMixedVideo(t *testing.T, root string) string {
	t.Helper()
	banded := filepath.Join(root, "banded.png")
	smooth := filepath.Join(root, "smooth.png")
	writeBandedPNG(t, banded, 16)
	writeSmoothPNG(t, smooth)
	video := filepath.Join(root, "mixed.mp4")
	command := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-loop", "1", "-t", "1", "-i", banded, "-loop", "1", "-t", "1", "-i", smooth,
		"-filter_complex", "[0:v][1:v]concat=n=2:v=1[v]",
		"-map", "[v]", "-t", "2", "-r", "10", "-pix_fmt", "yuv420p", "-y", video)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg fixture failed: %v: %s", err, output)
	}
	return video
}

func TestParseSceneTimes(t *testing.T) {
	output := "[Parsed_showinfo_1] n:0 pts:30 pts_time:1.0\n" +
		"[Parsed_showinfo_1] n:1 pts:75 pts_time:2.5\n" +
		"[Parsed_showinfo_1] n:2 pts:30 pts_time:1.0\n" +
		"[Parsed_showinfo_1] n:3 pts:0 pts_time:0.0"
	times := parseSceneTimes(output)
	want := []float64{1, 2.5}
	if len(times) != len(want) {
		t.Fatalf("parseSceneTimes = %v, want %v", times, want)
	}
	for index := range want {
		if times[index] != want[index] {
			t.Fatalf("parseSceneTimes = %v, want %v", times, want)
		}
	}
}

func TestVideoScenesDetectsCut(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	root := t.TempDir()
	makeCutVideo(t, root)
	result, rpcErr := videoScenesCapability(requestInput{
		WorkspaceRoot: root, VideoPath: "cut.mp4", SceneThreshold: 0.2,
	})
	if rpcErr != nil {
		t.Fatalf("scene detection failed: %+v", rpcErr)
	}
	payload := result.(map[string]any)
	cuts := payload["cuts"].([]sceneCut)
	if len(cuts) == 0 {
		t.Fatal("expected at least one scene cut")
	}
	found := false
	for _, cut := range cuts {
		if cut.Time >= 0.7 && cut.Time <= 1.3 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a cut near 1s, got %+v", cuts)
	}
}
func makeCutVideo(t *testing.T, root string) string {
	t.Helper()
	video := filepath.Join(root, "cut.mp4")
	command := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=black:s=256x256:d=1:r=10",
		"-f", "lavfi", "-i", "color=c=white:s=256x256:d=1:r=10",
		"-filter_complex", "[0:v][1:v]concat=n=2:v=1[v]",
		"-map", "[v]", "-pix_fmt", "yuv420p", "-y", video)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg cut fixture failed: %v: %s", err, output)
	}
	return video
}

func TestVideoContactSheet(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	root := t.TempDir()
	makeMixedVideo(t, root)
	result, rpcErr := videoContactSheetCapability(requestInput{
		WorkspaceRoot: root, VideoPath: "mixed.mp4", Times: "0.5,1.5",
		Columns: 2, CellWidth: 160, OutputPath: "out/sheet.png",
	})
	if rpcErr != nil {
		t.Fatalf("contact sheet failed: %+v", rpcErr)
	}
	payload := result.(map[string]any)
	if payload["path"] != "out/sheet.png" {
		t.Fatalf("unexpected path: %v", payload["path"])
	}
	if payload["width"] != 320 || payload["height"] != 160 {
		t.Fatalf("unexpected sheet size: %vx%v", payload["width"], payload["height"])
	}
	cells := payload["cells"].([]contactCell)
	if len(cells) != 2 || cells[0].Time != 0.5 || cells[1].Time != 1.5 {
		t.Fatalf("unexpected cells: %+v", cells)
	}
	file, err := os.Open(filepath.Join(root, "out", "sheet.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 320 || config.Height != 160 {
		t.Fatalf("encoded sheet is %dx%d", config.Width, config.Height)
	}
}

func TestVisualUpscaleImage(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	root := t.TempDir()
	writeTestPNG(t, filepath.Join(root, "src.png"), 40)
	result, rpcErr := visualUpscaleCapability(requestInput{
		WorkspaceRoot: root, ImagePath: "src.png", Scale: 2, OutputPath: "out/up.png",
	})
	if rpcErr != nil {
		t.Fatalf("upscale failed: %+v", rpcErr)
	}
	output := result.(map[string]any)["output"].(map[string]any)
	if output["width"] != 128 || output["height"] != 128 {
		t.Fatalf("unexpected output size: %+v", output)
	}
	if output["sha256"] == "" {
		t.Fatal("missing output sha256")
	}
}

func TestVisualUpscaleVideo(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	root := t.TempDir()
	makeCutVideo(t, root)
	result, rpcErr := visualUpscaleCapability(requestInput{
		WorkspaceRoot: root, VideoPath: "cut.mp4", Scale: 1.5, OutputPath: "out/up.mp4",
	})
	if rpcErr != nil {
		t.Fatalf("upscale failed: %+v", rpcErr)
	}
	output := result.(map[string]any)["output"].(map[string]any)
	if output["width"] != 384 || output["height"] != 384 {
		t.Fatalf("unexpected output size: %+v", output)
	}
	probe, err := probeVideo(mustLookPath(t, "ffprobe"), filepath.Join(root, "out", "up.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if probe.Width != 384 || probe.Height != 384 {
		t.Fatalf("encoded video is %dx%d", probe.Width, probe.Height)
	}
}

func TestVisualUpscaleRequiresOneInput(t *testing.T) {
	if _, rpcErr := visualUpscaleCapability(requestInput{WorkspaceRoot: t.TempDir()}); rpcErr == nil {
		t.Fatal("expected missing input error")
	}
	if _, rpcErr := visualUpscaleCapability(requestInput{WorkspaceRoot: t.TempDir(), ImagePath: "a.png", VideoPath: "b.mp4"}); rpcErr == nil {
		t.Fatal("expected mutually exclusive error")
	}
}

func mustLookPath(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not installed", name)
	}
	return path
}
