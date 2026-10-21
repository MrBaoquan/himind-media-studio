package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func makeColorClip(t *testing.T, dir, name, color string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c="+color+":s=160x90:d=1:r=10", "-pix_fmt", "yuv420p", "-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1", "-y", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture %s: %v: %s", name, err, output)
	}
	return path
}

func TestTimelineExportDeterministic(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	root := t.TempDir()
	m1 := makeColorClip(t, root, "a.mp4", "red")
	m2 := makeColorClip(t, root, "b.mp4", "green")
	m3 := makeColorClip(t, root, "c.mp4", "blue")
	_ = m1
	_ = m2
	_ = m3
	doc := timelineDoc{
		Schema: "himind.timeline.v1", Timebase: timelineTimebase{Num: 10, Den: 1},
		Canvas: timelineCanvas{Width: 160, Height: 90},
		Tracks: []timelineTrack{{ID: "V1", Kind: "video", Index: 1, Clips: []timelineClip{
			{ID: "c1", MediaID: "m1", TrackID: "V1", StartFrame: 0, DurationFrames: 5, SourceInFrame: 0, SourceOutFrame: 10, Speed: 1},
			{ID: "c2", MediaID: "m2", TrackID: "V1", StartFrame: 5, DurationFrames: 5, SourceInFrame: 0, SourceOutFrame: 10, Speed: 1},
			{ID: "c3", MediaID: "m3", TrackID: "V1", StartFrame: 10, DurationFrames: 5, SourceInFrame: 0, SourceOutFrame: 10, Speed: 1},
		}}},
	}
	media := map[string]string{"m1": "a.mp4", "m2": "b.mp4", "m3": "c.mp4"}
	first, rpcErr := timelineExportCapability(requestInput{WorkspaceRoot: root, Timeline: doc, MediaPaths: media, OutputPath: "out1.mp4"})
	if rpcErr != nil {
		t.Fatalf("export 1: %+v", rpcErr)
	}
	second, rpcErr := timelineExportCapability(requestInput{WorkspaceRoot: root, Timeline: doc, MediaPaths: media, OutputPath: "out2.mp4"})
	if rpcErr != nil {
		t.Fatalf("export 2: %+v", rpcErr)
	}
	a := first.(map[string]any)
	b := second.(map[string]any)
	if a["clip_count"] != 3 {
		t.Fatalf("clip_count = %v", a["clip_count"])
	}
	if a["sha256"] != b["sha256"] {
		t.Fatalf("non-deterministic: %v vs %v", a["sha256"], b["sha256"])
	}
	if _, err := os.Stat(filepath.Join(root, "out1.mp4")); err != nil {
		t.Fatal(err)
	}
}
