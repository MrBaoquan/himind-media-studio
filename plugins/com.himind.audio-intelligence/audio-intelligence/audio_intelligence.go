package main

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"himind-plugin/audio-intelligence/internal/himindjsonrpc"
)

type alignClip struct {
	BeatID            string  `json:"beat_id"`
	Text              string  `json:"text"`
	Start             float64 `json:"start"`
	DurationSeconds   float64 `json:"duration_seconds"`
	EffectiveDuration float64 `json:"effective_duration_seconds"`
	Tempo             float64 `json:"tempo,omitempty"`
	ClipID            string  `json:"clip_id,omitempty"`
}

type manifestClip struct {
	BeatID            string  `json:"beat_id"`
	Text              string  `json:"text"`
	Start             float64 `json:"start"`
	DurationSeconds   float64 `json:"duration_seconds"`
	EffectiveDuration float64 `json:"effective_duration_seconds"`
	Path              string  `json:"path"`
}

type audioManifest struct {
	Clips []manifestClip `json:"clips"`
}

type alignmentWord struct {
	Text   string  `json:"text"`
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	BeatID string  `json:"beat_id,omitempty"`
	ClipID string  `json:"clip_id,omitempty"`
}

type alignmentPayload struct {
	Schema      string          `json:"schema"`
	CreativeID  string          `json:"creative_id"`
	GeneratedAt string          `json:"generated_at"`
	Words       []alignmentWord `json:"words"`
}

type subtitleCue struct {
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	Text   string  `json:"text"`
	BeatID string  `json:"beat_id,omitempty"`
}

type syncFinding struct {
	Code         string  `json:"code"`
	Severity     string  `json:"severity"`
	Message      string  `json:"message"`
	CueIndex     int     `json:"cue_index,omitempty"`
	DriftSeconds float64 `json:"drift_seconds,omitempty"`
}

func audioAlignCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	creativeID := strings.TrimSpace(in.CreativeID)
	if creativeID == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	item, err := loadCreative(root, creativeID)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	beats := asMapSlice(asMap(item.Script)["beats"])
	beatText := map[string]string{}
	beatStart := map[string]float64{}
	beatDuration := map[string]float64{}
	for _, beat := range beats {
		id := strings.TrimSpace(asString(beat["id"]))
		if id == "" {
			continue
		}
		beatText[id] = strings.TrimSpace(asString(beat["narration"]))
		beatStart[id] = numberFloat(beat["start"])
		beatDuration[id] = numberFloat(beat["duration_seconds"])
	}
	clips := in.VoiceClips
	if len(clips) == 0 {
		var manifest audioManifest
		if err := readJSON(filepath.Join(creativeRoot(root, creativeID), "audio", "audio-manifest.json"), &manifest); err != nil {
			return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("ALIGN_CLIPS_MISSING", "audio", "没有 voice clips 或 audio manifest", "先调用 audio.pipeline.render 或 audio.speech")}, "retryable": true}, nil
		}
		for index, clip := range manifest.Clips {
			clips = append(clips, alignClip{
				BeatID: clip.BeatID, Text: clip.Text, Start: clip.Start,
				DurationSeconds: clip.DurationSeconds, EffectiveDuration: clip.EffectiveDuration,
				ClipID: fmt.Sprintf("clip-%d", index+1),
			})
		}
	}
	if len(clips) == 0 {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("ALIGN_CLIPS_MISSING", "audio", "没有可对齐的 voice clips", "先生成旁白")}, "retryable": true}, nil
	}
	words := make([]alignmentWord, 0, 256)
	for index, clip := range clips {
		beatID := strings.TrimSpace(clip.BeatID)
		text := strings.TrimSpace(clip.Text)
		if text == "" {
			text = beatText[beatID]
		}
		if text == "" {
			continue
		}
		start := clip.Start
		if start <= 0 && beatStart[beatID] > 0 {
			start = beatStart[beatID]
		}
		duration := clip.EffectiveDuration
		if duration <= 0 {
			duration = clip.DurationSeconds
		}
		if duration <= 0 {
			duration = beatDuration[beatID]
		}
		if duration <= 0 {
			duration = 1
		}
		clipID := clip.ClipID
		if clipID == "" {
			clipID = fmt.Sprintf("clip-%d", index+1)
		}
		words = append(words, alignText(text, start, duration, beatID, clipID)...)
	}
	payload := alignmentPayload{
		Schema: "himind.audio.alignment.v1", CreativeID: creativeID,
		GeneratedAt: now(), Words: words,
	}
	path := filepath.Join(creativeRoot(root, creativeID), "audio", "alignment.json")
	if err := writeJSON(path, payload); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"state": "ready", "creative_id": creativeID, "word_count": len(words),
		"path": workspaceRelative(root, path), "words": words,
		"next": "把 words 传给 audio.qa.sync，或用于逐词字幕",
	}, nil
}

func audioAlignmentGetCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	creativeID := strings.TrimSpace(in.CreativeID)
	if creativeID == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	path := filepath.Join(creativeRoot(root, creativeID), "audio", "alignment.json")
	if strings.TrimSpace(in.AlignmentPath) != "" {
		path, err = workspaceFile(root, in.AlignmentPath)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams("alignment_path: " + err.Error())
		}
	}
	var payload alignmentPayload
	if err := readJSON(path, &payload); err != nil {
		return map[string]any{"state": "missing", "creative_id": creativeID, "next": "调用 audio.align"}, nil
	}
	return map[string]any{"state": "ready", "alignment": payload, "path": workspaceRelative(root, path)}, nil
}

func audioQASyncCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	creativeID := strings.TrimSpace(in.CreativeID)
	if creativeID == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	var alignment alignmentPayload
	alignmentPath := strings.TrimSpace(in.AlignmentPath)
	if alignmentPath != "" {
		path, pathErr := workspaceFile(root, alignmentPath)
		if pathErr != nil {
			return nil, himindjsonrpc.InvalidParams("alignment_path: " + pathErr.Error())
		}
		if err := readJSON(path, &alignment); err != nil {
			return nil, himindjsonrpc.InvalidParams("read alignment: " + err.Error())
		}
	} else if err := readJSON(filepath.Join(creativeRoot(root, creativeID), "audio", "alignment.json"), &alignment); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("ALIGNMENT_MISSING", "audio", "没有 alignment.json", "先调用 audio.align")}, "retryable": true}, nil
	}
	cues := in.Subtitles
	if len(cues) == 0 {
		var payload struct {
			Subtitles []subtitleCue `json:"subtitles"`
		}
		if err := readJSON(filepath.Join(creativeRoot(root, creativeID), "audio", "subtitles.json"), &payload); err == nil {
			cues = payload.Subtitles
		}
	}
	if len(cues) == 0 {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("SUBTITLES_MISSING", "audio", "没有字幕 cues", "先调用 media.workbench.subtitle.generate")}, "retryable": true}, nil
	}
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].Start < cues[j].Start })
	findings := make([]syncFinding, 0)
	maxDrift := 0.0
	for index, cue := range cues {
		if cue.End <= cue.Start {
			findings = append(findings, syncFinding{Code: "SUBTITLE_INVALID_RANGE", Severity: "error", Message: "字幕结束时间必须大于开始时间", CueIndex: index})
			continue
		}
		if index > 0 && cue.Start < cues[index-1].End-0.01 {
			findings = append(findings, syncFinding{Code: "SUBTITLE_OVERLAP", Severity: "error", Message: "字幕 cue 与上一条重叠", CueIndex: index})
		}
		words := overlapWords(alignment.Words, cue.Start, cue.End)
		if len(words) == 0 {
			findings = append(findings, syncFinding{Code: "SUBTITLE_NO_ALIGNMENT", Severity: "warning", Message: "字幕区间没有对应逐词时间戳", CueIndex: index})
			continue
		}
		expected := strings.Join(words, "")
		if textSimilarity(expected, cue.Text) < 0.98 {
			findings = append(findings, syncFinding{Code: "SUBTITLE_TEXT_MISMATCH", Severity: "error", Message: "字幕文本与逐词对齐文本不一致", CueIndex: index})
		}
		startDrift := math.Abs(wordsStart(alignment.Words, cue.Start, cue.End) - cue.Start)
		endDrift := math.Abs(wordsEnd(alignment.Words, cue.Start, cue.End) - cue.End)
		drift := math.Max(startDrift, endDrift)
		if drift > maxDrift {
			maxDrift = drift
		}
		if drift > 0.25 {
			findings = append(findings, syncFinding{Code: "SUBTITLE_DRIFT", Severity: "warning", Message: "字幕与逐词时间戳存在漂移", CueIndex: index, DriftSeconds: math.Round(drift*1000) / 1000})
		}
	}
	ok := true
	for _, finding := range findings {
		if finding.Severity == "error" {
			ok = false
		}
	}
	return map[string]any{
		"state": "ready", "ok": ok, "cue_count": len(cues),
		"max_drift_seconds": math.Round(maxDrift*1000) / 1000,
		"findings":          findings,
	}, nil
}

func audioSubtitleFromAlignmentCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	creativeID := strings.TrimSpace(in.CreativeID)
	if creativeID == "" {
		return nil, himindjsonrpc.InvalidParams("creative_id is required")
	}
	var alignment alignmentPayload
	path := filepath.Join(creativeRoot(root, creativeID), "audio", "alignment.json")
	if strings.TrimSpace(in.AlignmentPath) != "" {
		path, err = workspaceFile(root, in.AlignmentPath)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams("alignment_path: " + err.Error())
		}
	}
	if err := readJSON(path, &alignment); err != nil {
		return map[string]any{"state": "blocked", "blockers": []map[string]any{issue("ALIGNMENT_MISSING", "audio", "没有 alignment.json", "先调用 audio.align")}, "retryable": true}, nil
	}
	maxChars := in.MaxChars
	if maxChars <= 0 {
		maxChars = 24
	}
	cues := subtitlesFromAlignment(alignment.Words, maxChars)
	output := filepath.Join(creativeRoot(root, creativeID), "audio", "subtitles-aligned.json")
	if err := writeJSON(output, map[string]any{
		"version": "himind.subtitles.aligned.v1", "creative_id": creativeID,
		"generated_at": now(), "subtitles": cues,
	}); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"state": "ready", "creative_id": creativeID, "subtitles": cues,
		"count": len(cues), "path": workspaceRelative(root, output),
		"next": "调用 media.workbench.audio.attach 挂载 aligned cues",
	}, nil
}

func subtitlesFromAlignment(words []alignmentWord, maxChars int) []subtitleCue {
	if maxChars < 6 {
		maxChars = 6
	}
	cues := make([]subtitleCue, 0, len(words)/8+1)
	current := ""
	start := 0.0
	end := 0.0
	beatID := ""
	flush := func() {
		if strings.TrimSpace(current) == "" {
			return
		}
		cues = append(cues, subtitleCue{Start: roundMillis(start), End: roundMillis(end), Text: current, BeatID: beatID})
		current = ""
	}
	for _, word := range words {
		next := current + word.Text
		if current != "" && len([]rune(next)) > maxChars {
			flush()
		}
		if current == "" {
			start = word.Start
			beatID = word.BeatID
		}
		current += word.Text
		end = word.End
	}
	flush()
	return cues
}
func audioQAIntelligibilityCapability(in requestInput) (any, *himindjsonrpc.Error) {
	script := strings.TrimSpace(in.Script)
	if script == "" && strings.TrimSpace(in.CreativeID) != "" {
		root, err := workspace(in.WorkspaceRoot)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
		item, err := loadCreative(root, in.CreativeID)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
		parts := make([]string, 0)
		for _, beat := range asMapSlice(asMap(item.Script)["beats"]) {
			if text := strings.TrimSpace(asString(beat["narration"])); text != "" {
				parts = append(parts, text)
			}
		}
		script = strings.Join(parts, "")
	}
	transcript := strings.TrimSpace(in.Transcript)
	if script == "" || transcript == "" {
		return nil, himindjsonrpc.InvalidParams("script/creative_id and transcript are required")
	}
	left := normalizeSpeech(script)
	right := normalizeSpeech(transcript)
	distance := levenshtein(left, right)
	denominator := len([]rune(left))
	if len([]rune(right)) > denominator {
		denominator = len([]rune(right))
	}
	score := 1.0
	if denominator > 0 {
		score = 1 - float64(distance)/float64(denominator)
	}
	if score < 0 {
		score = 0
	}
	return map[string]any{
		"state": "ready", "ok": score >= 0.85,
		"score":    math.Round(score*10000) / 10000,
		"distance": distance, "script_chars": len([]rune(left)),
		"transcript_chars": len([]rune(right)),
		"next":             "score < 0.85 时检查漏读、错读或 ASR 语言/模型",
	}, nil
}

func alignText(text string, start, duration float64, beatID, clipID string) []alignmentWord {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return nil
	}
	weights := make([]float64, len(tokens))
	total := 0.0
	for index, token := range tokens {
		weight := float64(len([]rune(token)))
		if weight < 1 {
			weight = 1
		}
		weights[index] = weight
		total += weight
	}
	words := make([]alignmentWord, 0, len(tokens))
	cursor := start
	for index, token := range tokens {
		next := cursor + duration*weights[index]/total
		if index == len(tokens)-1 {
			next = start + duration
		}
		words = append(words, alignmentWord{
			Text: token, Start: roundMillis(cursor), End: roundMillis(next),
			BeatID: beatID, ClipID: clipID,
		})
		cursor = next
	}
	return words
}

func tokenize(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	hasHan := false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			hasHan = true
			break
		}
	}
	if !hasHan {
		return strings.Fields(text)
	}
	tokens := make([]string, 0, len([]rune(text)))
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.IsPunct(r) && len(tokens) > 0 {
			tokens[len(tokens)-1] += string(r)
			continue
		}
		tokens = append(tokens, string(r))
	}
	return tokens
}

func normalizeSpeech(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func textSimilarity(left, right string) float64 {
	left = normalizeSpeech(left)
	right = normalizeSpeech(right)
	denominator := len([]rune(left))
	if len([]rune(right)) > denominator {
		denominator = len([]rune(right))
	}
	if denominator == 0 {
		return 1
	}
	return 1 - float64(levenshtein(left, right))/float64(denominator)
}

func overlapWords(words []alignmentWord, start, end float64) []string {
	items := make([]string, 0)
	for _, word := range words {
		if word.Start < end && word.End > start {
			items = append(items, word.Text)
		}
	}
	return items
}

func wordsStart(words []alignmentWord, start, end float64) float64 {
	value := math.Inf(1)
	for _, word := range words {
		if word.Start < end && word.End > start && word.Start < value {
			value = word.Start
		}
	}
	if math.IsInf(value, 1) {
		return start
	}
	return value
}

func wordsEnd(words []alignmentWord, start, end float64) float64 {
	value := math.Inf(-1)
	for _, word := range words {
		if word.Start < end && word.End > start && word.End > value {
			value = word.End
		}
	}
	if math.IsInf(value, -1) {
		return end
	}
	return value
}

func levenshtein(left, right string) int {
	a := []rune(left)
	b := []rune(right)
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = minInt(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

func minInt(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func roundMillis(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func workspaceRelative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
