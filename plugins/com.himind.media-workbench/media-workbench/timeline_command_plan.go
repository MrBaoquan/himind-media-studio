package main

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var durationInstructionPattern = regexp.MustCompile(`(?:剪到|缩短到|时长)\s*([0-9.]+)\s*秒`)
var fadeInstructionPattern = regexp.MustCompile(`(?:([0-9.]+)\s*秒\s*淡入|淡入\s*([0-9.]+))`)
var ordinalInstructionPattern = regexp.MustCompile(`第([一二三四五六七八九十0-9]+)(?:段|个|镜)`)

func parseOrdinal(value string) int {
	if number, err := strconv.Atoi(value); err == nil {
		return number
	}
	runes := []rune(value)
	digits := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	if len(runes) == 1 {
		if digit, ok := digits[runes[0]]; ok {
			return digit
		}
		if runes[0] == '十' {
			return 10
		}
	}
	if len(runes) == 2 && runes[0] == '十' {
		if digit, ok := digits[runes[1]]; ok {
			return 10 + digit
		}
	}
	if len(runes) == 2 && runes[1] == '十' {
		if digit, ok := digits[runes[0]]; ok {
			return digit * 10
		}
	}
	return 0
}

func timelineCommandPlanCapability(in requestInput) any {
	instruction := strings.TrimSpace(in.CommandText)
	if instruction == "" {
		return map[string]any{"state": "blocked", "blockers": []string{"INSTRUCTION_EMPTY"}}
	}
	doc := in.Timeline
	var videoTrack *timelineTrack
	for index := range doc.Tracks {
		if doc.Tracks[index].Kind == "video" && len(doc.Tracks[index].Clips) > 0 {
			videoTrack = &doc.Tracks[index]
			break
		}
	}
	if videoTrack == nil {
		return map[string]any{"state": "blocked", "blockers": []string{"VIDEO_TRACK_EMPTY"}}
	}
	clips := append([]timelineClip(nil), videoTrack.Clips...)
	sort.SliceStable(clips, func(i, j int) bool { return clips[i].StartFrame < clips[j].StartFrame })
	index := 0
	if match := ordinalInstructionPattern.FindStringSubmatch(instruction); len(match) == 2 {
		index = parseOrdinal(match[1]) - 1
	}
	if index < 0 || index >= len(clips) {
		return map[string]any{"state": "blocked", "blockers": []string{"TARGET_CLIP_NOT_FOUND"}}
	}
	target := clips[index]
	commands := make([]timelineCommand, 0, 2)
	if match := fadeInstructionPattern.FindStringSubmatch(instruction); len(match) == 3 {
		secondsText := match[1]
		if secondsText == "" {
			secondsText = match[2]
		}
		seconds, _ := strconv.ParseFloat(secondsText, 64)
		if seconds > 0 {
			updated := target
			updated.TransitionIn = &timelineTransition{Type: "fade", DurationFrames: int(math.Round(seconds * float64(doc.Timebase.Num) / float64(doc.Timebase.Den)))}
			commands = append(commands, timelineCommand{Op: "update_clip", TrackID: target.TrackID, ClipID: target.ID, Clip: &updated})
		}
	}
	if match := durationInstructionPattern.FindStringSubmatch(instruction); len(match) == 2 {
		seconds, _ := strconv.ParseFloat(match[1], 64)
		frames := int(math.Round(seconds * float64(doc.Timebase.Num) / float64(doc.Timebase.Den)))
		if frames > 0 {
			commands = append(commands, timelineCommand{Op: "ripple_trim", TrackID: target.TrackID, ClipID: target.ID, DurationFrames: frames, Ripple: true})
		}
	}
	if len(commands) == 0 {
		return map[string]any{"state": "blocked", "blockers": []string{"NO_SUPPORTED_EDIT"}}
	}
	preview, blockers := applyTimelineCommands(doc, commands)
	return map[string]any{
		"state": "ready", "instruction": instruction, "target_clip": target.ID, "target_index": index + 1,
		"commands": commands, "preview": preview, "blockers": blockers,
		"formatted": fmt.Sprintf("对第 %d 段生成 %d 条精剪命令", index+1, len(commands)),
	}
}
