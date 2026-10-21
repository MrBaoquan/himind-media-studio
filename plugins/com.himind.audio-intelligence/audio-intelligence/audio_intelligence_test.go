package main

import (
	"math"
	"testing"
)

func TestTokenizeChineseKeepsPunctuationWithPrevious(t *testing.T) {
	tokens := tokenize("在合肥，有一轮人造太阳。")
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}
	if tokens[2] != "肥，" {
		t.Fatalf("punctuation should stay with previous token: %#v", tokens)
	}
	if tokens[len(tokens)-1] != "阳。" {
		t.Fatalf("final punctuation should stay with previous token: %#v", tokens)
	}
}

func TestAlignTextCoversClipDuration(t *testing.T) {
	words := alignText("在合肥，有一轮人造太阳。", 2.5, 4.0, "b1", "clip-1")
	if len(words) == 0 {
		t.Fatal("expected words")
	}
	if math.Abs(words[0].Start-2.5) > 0.001 {
		t.Fatalf("first word start = %v", words[0].Start)
	}
	if math.Abs(words[len(words)-1].End-6.5) > 0.001 {
		t.Fatalf("last word end = %v", words[len(words)-1].End)
	}
	for index := 1; index < len(words); index++ {
		if words[index].Start < words[index-1].Start {
			t.Fatalf("word timing not sorted: %#v", words)
		}
	}
}

func TestTextSimilarityDetectsMissingText(t *testing.T) {
	full := "在合肥，有一轮人造太阳。"
	same := textSimilarity(full, full)
	if same < 0.999 {
		t.Fatalf("same text score = %v", same)
	}
	missing := textSimilarity(full, "在合肥，有一轮太阳。")
	if missing >= 0.99 {
		t.Fatalf("missing text should reduce score: %v", missing)
	}
}

func TestLevenshtein(t *testing.T) {
	if got := levenshtein("kitten", "sitting"); got != 3 {
		t.Fatalf("levenshtein = %d", got)
	}
	if got := levenshtein("", "abc"); got != 3 {
		t.Fatalf("levenshtein empty = %d", got)
	}
}
func TestSubtitlesFromAlignmentUsesWordBoundaries(t *testing.T) {
	words := []alignmentWord{
		{Text: "在", Start: 0.0, End: 0.1, BeatID: "b1"},
		{Text: "合", Start: 0.1, End: 0.2, BeatID: "b1"},
		{Text: "肥", Start: 0.2, End: 0.3, BeatID: "b1"},
		{Text: "，", Start: 0.3, End: 0.4, BeatID: "b1"},
		{Text: "有", Start: 0.4, End: 0.5, BeatID: "b1"},
		{Text: "光", Start: 0.5, End: 0.6, BeatID: "b1"},
		{Text: "明", Start: 0.6, End: 0.7, BeatID: "b1"},
	}
	cues := subtitlesFromAlignment(words, 6)
	if len(cues) != 2 {
		t.Fatalf("expected 2 cues, got %#v", cues)
	}
	if cues[0].Start != 0.0 || cues[0].Text != "在合肥，有光" {
		t.Fatalf("unexpected first cue: %+v", cues[0])
	}
	if cues[len(cues)-1].End != 0.7 {
		t.Fatalf("last cue must end at last word: %+v", cues[len(cues)-1])
	}
}
