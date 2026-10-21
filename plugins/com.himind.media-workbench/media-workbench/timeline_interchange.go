package main

// timeline_interchange.go — OpenTimelineIO / EDL exchange adapters.
//
// Internal truth stays himind.timeline.v1. OTIO and EDL are project-boundary
// formats. Exact Himind extensions are carried in metadata / EDL comments so a
// round-trip never loses timing or transform / color / transition fields.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

const interchangeSchema = "himind.timeline.interchange.v1"

type interchangeFile struct {
	Format  string `json:"format"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Bytes   int    `json:"bytes"`
	Content string `json:"content,omitempty"`
}

func framesToTimecode(frame, fps int) string {
	if frame < 0 {
		frame = 0
	}
	if fps <= 0 {
		fps = 30
	}
	ff := frame % fps
	totalSeconds := frame / fps
	ss := totalSeconds % 60
	mm := (totalSeconds / 60) % 60
	hh := totalSeconds / 3600
	return fmt.Sprintf("%02d:%02d:%02d:%02d", hh, mm, ss, ff)
}

func timecodeToFrames(value string, fps int) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 4 {
		return 0, false
	}
	numbers := make([]int, 4)
	for i, part := range parts {
		number, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return 0, false
		}
		numbers[i] = number
	}
	return ((numbers[0]*60+numbers[1])*60+numbers[2])*fps + numbers[3], true
}

func otioRationalTime(rate, value int) map[string]any {
	return map[string]any{"OTIO_SCHEMA": "RationalTime.1", "rate": rate, "value": value}
}

func timelineToOTIO(doc timelineDoc) map[string]any {
	rate := doc.Timebase.Num
	if rate <= 0 {
		rate = 30
	}
	tracks := make([]any, 0, len(doc.Tracks))
	for _, track := range doc.Tracks {
		kind := "Video"
		if track.Kind == "audio" {
			kind = "Audio"
		}
		children := make([]any, 0, len(track.Clips))
		clips := append([]timelineClip(nil), track.Clips...)
		sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
		for _, clip := range clips {
			children = append(children, map[string]any{
				"OTIO_SCHEMA": "Clip.2",
				"name":        clip.ID,
				"source_range": map[string]any{
					"OTIO_SCHEMA": "TimeRange.1",
					"start_time":  otioRationalTime(rate, clip.StartFrame),
					"duration":    otioRationalTime(rate, clip.DurationFrames),
				},
				"metadata": map[string]any{"himind": map[string]any{"clip": clip}},
			})
		}
		tracks = append(tracks, map[string]any{
			"OTIO_SCHEMA": "Track.1",
			"name":        track.ID,
			"kind":        kind,
			"metadata":    map[string]any{"himind": map[string]any{"id": track.ID, "kind": track.Kind, "role": track.Role, "muted": track.Muted, "locked": track.Locked}},
			"children":    children,
		})
	}
	return map[string]any{
		"OTIO_SCHEMA":       "Timeline.1",
		"name":              "Himind Timeline",
		"global_start_time": nil,
		"metadata": map[string]any{"himind": map[string]any{
			"schema":      interchangeSchema,
			"source":      doc.Schema,
			"revision":    doc.Revision,
			"timeline":    doc,
			"exported_at": time.Now().UTC().Format(time.RFC3339Nano),
		}},
		"tracks": map[string]any{"OTIO_SCHEMA": "Stack.1", "name": "tracks", "children": tracks},
	}
}

func otioToTimeline(payload map[string]any) (timelineDoc, error) {
	var doc timelineDoc
	metadata, _ := payload["metadata"].(map[string]any)
	himind, _ := metadata["himind"].(map[string]any)
	if timeline, ok := himind["timeline"]; ok {
		data, err := json.Marshal(timeline)
		if err != nil {
			return doc, err
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return doc, err
		}
		return doc, nil
	}
	return doc, fmt.Errorf("OTIO payload has no metadata.himind.timeline; standard-only import is not supported yet")
}

func timelineToEDL(doc timelineDoc) string {
	rate := doc.Timebase.Num
	if rate <= 0 {
		rate = 30
	}
	lines := []string{"TITLE: Himind Timeline", "FCM: NON-DROP FRAME", ""}
	meta, _ := json.Marshal(map[string]any{"schema": doc.Schema, "revision": doc.Revision, "timebase": doc.Timebase, "canvas": doc.Canvas, "media": doc.Media})
	lines = append(lines, "* HIMIND_TIMELINE: "+string(meta))
	event := 1
	for _, track := range doc.Tracks {
		clips := append([]timelineClip(nil), track.Clips...)
		sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
		for _, clip := range clips {
			start := clip.StartFrame
			end := clip.StartFrame + clip.DurationFrames
			sourceIn := clip.SourceInFrame
			sourceOut := clip.SourceOutFrame
			if sourceOut <= sourceIn {
				sourceOut = sourceIn + clip.DurationFrames
			}
			reel := "AX"
			if clip.MediaID != "" {
				reel = strings.ToUpper(strings.ReplaceAll(clip.MediaID, " ", "_"))
				if len(reel) > 32 {
					reel = reel[:32]
				}
			}
			trackType := "V"
			if track.Kind == "audio" {
				trackType = "A"
			}
			lines = append(lines, fmt.Sprintf("%03d  %-8s %s     C        %s %s %s %s", event, reel, trackType, framesToTimecode(sourceIn, rate), framesToTimecode(sourceOut, rate), framesToTimecode(start, rate), framesToTimecode(end, rate)))
			lines = append(lines, " * FROM CLIP NAME: "+clip.ID)
			clipData, _ := json.Marshal(map[string]any{"track_id": track.ID, "kind": track.Kind, "clip": clip})
			lines = append(lines, " * HIMIND_CLIP: "+string(clipData))
			event++
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func edlToTimeline(content string) (timelineDoc, error) {
	doc := timelineDoc{Schema: "himind.timeline.v1", Timebase: timelineTimebase{Num: 30, Den: 1}, Canvas: timelineCanvas{Width: 1080, Height: 1920}}
	tracks := map[string]*timelineTrack{}
	order := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "* HIMIND_TIMELINE: ") {
			var meta struct {
				Schema   string             `json:"schema"`
				Revision int                `json:"revision"`
				Timebase timelineTimebase   `json:"timebase"`
				Canvas   timelineCanvas     `json:"canvas"`
				Media    []timelineMediaRef `json:"media"`
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "* HIMIND_TIMELINE: ")), &meta); err == nil {
				if meta.Schema != "" {
					doc.Schema = meta.Schema
				}
				doc.Revision = meta.Revision
				if meta.Timebase.Num > 0 {
					doc.Timebase = meta.Timebase
				}
				if meta.Canvas.Width > 0 {
					doc.Canvas = meta.Canvas
				}
				doc.Media = meta.Media
			}
			continue
		}
		if !strings.HasPrefix(line, "* HIMIND_CLIP: ") {
			continue
		}
		var payload struct {
			TrackID string       `json:"track_id"`
			Kind    string       `json:"kind"`
			Clip    timelineClip `json:"clip"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "* HIMIND_CLIP: ")), &payload); err != nil {
			return doc, err
		}
		if payload.TrackID == "" {
			continue
		}
		track := tracks[payload.TrackID]
		if track == nil {
			kind := payload.Kind
			if kind == "" {
				kind = "video"
			}
			index := len(order) + 1
			track = &timelineTrack{ID: payload.TrackID, Kind: kind, Index: index, Clips: []timelineClip{}}
			tracks[payload.TrackID] = track
			order = append(order, payload.TrackID)
		}
		clip := payload.Clip
		clip.TrackID = payload.TrackID
		track.Clips = append(track.Clips, clip)
	}
	if len(order) == 0 {
		return doc, fmt.Errorf("EDL contains no Himind clips")
	}
	for _, id := range order {
		sortTrackClips(tracks[id])
		doc.Tracks = append(doc.Tracks, *tracks[id])
	}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return doc, fmt.Errorf("imported timeline invalid: %s", strings.Join(blockers, "; "))
	}
	return doc, nil
}

func timelineInterchangeExportCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	doc := in.Timeline
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "ok": false, "blockers": blockers}, nil
	}
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if format == "" {
		format = "otio"
	}
	if format != "otio" && format != "edl" && format != "both" {
		return nil, himindjsonrpc.InvalidParams("format must be otio, edl or both")
	}
	base := strings.TrimSpace(in.Output)
	if base == "" {
		base = filepath.ToSlash(filepath.Join(".himind-media", "timeline", "interchange", fmt.Sprintf("timeline-%d", time.Now().UnixNano())))
	}
	resolvedBase, err := workspaceFile(root, base)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("output: " + err.Error())
	}
	_ = os.MkdirAll(filepath.Dir(resolvedBase), 0755)
	files := make([]interchangeFile, 0, 2)
	write := func(kind, suffix, content string) error {
		path := resolvedBase + suffix
		if strings.HasSuffix(strings.ToLower(base), ".json") || strings.HasSuffix(strings.ToLower(base), ".edl") {
			path = resolvedBase
		}
		resolved, err := workspaceFile(root, filepath.ToSlash(path))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(content))
		files = append(files, interchangeFile{Format: kind, Path: filepath.ToSlash(path), SHA256: hex.EncodeToString(sum[:]), Bytes: len([]byte(content)), Content: content})
		return nil
	}
	if format == "otio" || format == "both" {
		data, err := json.MarshalIndent(timelineToOTIO(doc), "", "  ")
		if err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
		if err := write("otio", ".otio.json", string(data)); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	if format == "edl" || format == "both" {
		if err := write("edl", ".edl", timelineToEDL(doc)); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
	}
	return map[string]any{"state": "ready", "ok": true, "schema": interchangeSchema, "format": format, "files": files, "clip_count": countTimelineClips(doc)}, nil
}

func countTimelineClips(doc timelineDoc) int {
	total := 0
	for _, track := range doc.Tracks {
		total += len(track.Clips)
	}
	return total
}

func timelineInterchangeImportCapability(in requestInput) (any, *himindjsonrpc.Error) {
	content := in.Content
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if content == "" {
		if strings.TrimSpace(in.Path) == "" {
			return nil, himindjsonrpc.InvalidParams("path or content is required")
		}
		root, err := workspace(in.WorkspaceRoot)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
		resolved, err := workspaceFile(root, in.Path)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams("path: " + err.Error())
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
		content = string(data)
		if format == "" {
			if strings.HasSuffix(strings.ToLower(resolved), ".edl") {
				format = "edl"
			} else {
				format = "otio"
			}
		}
	}
	content = strings.TrimSpace(content)
	if format == "" || format == "auto" {
		if strings.HasPrefix(content, "{") {
			format = "otio"
		} else {
			format = "edl"
		}
	}
	var doc timelineDoc
	var err error
	if format == "otio" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			return nil, himindjsonrpc.InvalidParams("invalid OTIO JSON: " + err.Error())
		}
		doc, err = otioToTimeline(payload)
	} else if format == "edl" {
		doc, err = edlToTimeline(content)
	} else {
		return nil, himindjsonrpc.InvalidParams("format must be otio or edl")
	}
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	if blockers := validateTimeline(doc); len(blockers) > 0 {
		return map[string]any{"state": "blocked", "ok": false, "blockers": blockers}, nil
	}
	return map[string]any{"state": "ready", "ok": true, "format": format, "timeline": doc, "clip_count": countTimelineClips(doc)}, nil
}
