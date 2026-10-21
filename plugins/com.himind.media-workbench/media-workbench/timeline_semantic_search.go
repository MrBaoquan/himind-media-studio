package main

// timeline_semantic_search.go — deterministic, local semantic-ish retrieval.
//
// The model supplies the intent; this capability provides a stable retriever
// over clip text, media metadata and path names. It is deliberately local and
// explainable: every hit carries the terms that matched. Model embeddings can
// be added later behind the same contract.

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

type semanticHit struct {
	Kind           string  `json:"kind"`
	ClipID         string  `json:"clip_id,omitempty"`
	TrackID        string  `json:"track_id,omitempty"`
	MediaID        string  `json:"media_id,omitempty"`
	StartFrame     int     `json:"start_frame,omitempty"`
	DurationFrames int     `json:"duration_frames,omitempty"`
	Text           string  `json:"text,omitempty"`
	Score          float64 `json:"score"`
	Reason         string  `json:"reason"`
}

var semanticSynonymGroups = [][]string{
	{"视频", "画面", "镜头", "video", "clip", "shot", "footage"},
	{"音频", "声音", "音乐", "audio", "music", "bgm", "sound"},
	{"图片", "图像", "照片", "image", "photo", "picture", "still"},
	{"字幕", "文字", "文本", "subtitle", "caption", "text"},
	{"人物", "角色", "人", "person", "people", "character", "actor"},
	{"风景", "场景", "环境", "landscape", "scene", "environment"},
	{"动画", "动效", "animation", "motion", "animated"},
	{"科普", "解释", "讲解", "explainer", "science", "education"},
	{"片头", "开头", "hook", "intro", "opening"},
	{"结尾", "收束", "outro", "ending", "payoff"},
}

func semanticTokenize(query string) []string {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return nil
	}
	seen := map[string]bool{}
	tokens := make([]string, 0, 12)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		tokens = append(tokens, value)
	}
	fields := strings.FieldsFunc(lower, func(r rune) bool { return unicode.IsSpace(r) || strings.ContainsRune(",，。.;；/|、", r) })
	for _, field := range fields {
		add(field)
	}
	if len([]rune(lower)) >= 2 && len(tokens) <= 1 {
		runes := []rune(lower)
		for i := 0; i+1 < len(runes); i++ {
			add(string(runes[i : i+2]))
		}
	}
	return tokens
}

func semanticExpand(tokens []string) map[string][]string {
	expanded := map[string][]string{}
	for _, token := range tokens {
		expanded[token] = append(expanded[token], token)
		for _, group := range semanticSynonymGroups {
			found := false
			for _, word := range group {
				if token == strings.ToLower(word) || strings.Contains(token, strings.ToLower(word)) || strings.Contains(strings.ToLower(word), token) {
					found = true
					break
				}
			}
			if found {
				for _, word := range group {
					word = strings.ToLower(word)
					if word != token {
						expanded[token] = append(expanded[token], word)
					}
				}
			}
		}
	}
	return expanded
}

func timelineSemanticScore(haystack string, tokens []string, expanded map[string][]string) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, len(tokens))
	for _, token := range tokens {
		matched := ""
		if strings.Contains(haystack, token) {
			score += 3
			matched = token
		}
		for _, synonym := range expanded[token] {
			if synonym == token {
				continue
			}
			if strings.Contains(haystack, synonym) {
				score += 1.5
				if matched == "" {
					matched = synonym
				}
			}
		}
		if matched != "" {
			reasons = append(reasons, matched)
		}
	}
	return score, reasons
}

func timelineSemanticSearchCapability(in requestInput) (any, *himindjsonrpc.Error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return map[string]any{"state": "blocked", "blockers": []string{"QUERY_EMPTY"}}, nil
	}
	tokens := semanticTokenize(query)
	expanded := semanticExpand(tokens)
	hits := make([]semanticHit, 0)
	for _, track := range in.Timeline.Tracks {
		for _, clip := range track.Clips {
			haystack := strings.ToLower(strings.Join([]string{clip.ID, clip.MediaID, clip.Text, track.ID, track.Kind, track.Role}, " "))
			if path := strings.TrimSpace(in.MediaPaths[clip.MediaID]); path != "" {
				haystack += " " + strings.ToLower(path)
			}
			score, reasons := timelineSemanticScore(haystack, tokens, expanded)
			if strings.Contains(strings.ToLower(query), track.Kind) {
				score += 0.75
				reasons = append(reasons, track.Kind)
			}
			if score <= 0 {
				continue
			}
			hits = append(hits, semanticHit{
				Kind: "clip", ClipID: clip.ID, TrackID: track.ID, MediaID: clip.MediaID,
				StartFrame: clip.StartFrame, DurationFrames: clip.DurationFrames, Text: clip.Text,
				Score: score, Reason: "命中 " + strings.Join(semanticUniqueStrings(reasons), " / "),
			})
		}
	}
	for mediaID, path := range in.MediaPaths {
		haystack := strings.ToLower(mediaID + " " + path)
		score, reasons := timelineSemanticScore(haystack, tokens, expanded)
		if score <= 0 {
			continue
		}
		hits = append(hits, semanticHit{Kind: "media", MediaID: mediaID, Score: score - 0.25, Reason: "素材命中 " + strings.Join(semanticUniqueStrings(reasons), " / ")})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].StartFrame < hits[j].StartFrame
		}
		return hits[i].Score > hits[j].Score
	})
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return map[string]any{
		"state": "ready", "query": query, "tokens": tokens, "expanded": expanded,
		"hits": hits, "total": len(hits),
		"summary": fmt.Sprintf("“%s”命中 %d 个素材/片段", query, len(hits)),
	}, nil
}

func semanticUniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
