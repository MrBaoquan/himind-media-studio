package main

import (
	"image"
	"os"
	"path/filepath"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type candidateConsistency struct {
	Path          string  `json:"path"`
	BestReference string  `json:"best_reference"`
	BestScore     float64 `json:"best_score"`
	MeanScore     float64 `json:"mean_score"`
	OK            bool    `json:"ok"`
}

func referenceConsistencyCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	packPath := firstNonEmpty(in.PackPath, filepath.ToSlash(filepath.Join("asset-packs", in.PackID, firstNonEmpty(in.Version, "1.0.0"))))
	resolved, err := workspaceFile(root, packPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("pack_path: " + err.Error())
	}
	var manifest assetPackManifest
	if err := readJSON(filepath.Join(resolved, "asset-pack.json"), &manifest); err != nil {
		return nil, himindjsonrpc.InvalidParams("read asset-pack.json: " + err.Error())
	}
	if len(in.CandidatePaths) == 0 {
		return nil, himindjsonrpc.InvalidParams("candidate_paths is required")
	}
	referencePaths := make([]string, 0, len(manifest.Resources))
	for _, resource := range manifest.Resources {
		if resource.Type != "image" {
			continue
		}
		referencePaths = append(referencePaths, filepath.Join(resolved, filepath.FromSlash(resource.Path)))
	}
	if len(referencePaths) == 0 {
		return nil, himindjsonrpc.InvalidParams("reference pack has no image resources")
	}
	references := make([]image.Image, 0, len(referencePaths))
	for _, path := range referencePaths {
		img, decodeErr := decodeImage(path)
		if decodeErr != nil {
			return nil, himindjsonrpc.InvalidParams("decode reference: " + decodeErr.Error())
		}
		references = append(references, img)
	}
	threshold := in.Threshold
	if threshold <= 0 {
		threshold = 0.75
	}
	candidates := make([]candidateConsistency, 0, len(in.CandidatePaths))
	blockers := make([]map[string]any, 0)
	warnings := make([]map[string]any, 0)
	lowest := 1.0
	for _, candidate := range in.CandidatePaths {
		path, pathErr := workspaceFile(root, candidate)
		if pathErr != nil {
			return nil, himindjsonrpc.InvalidParams("candidate_paths: " + pathErr.Error())
		}
		if _, statErr := os.Stat(path); statErr != nil {
			return nil, himindjsonrpc.InvalidParams("candidate not found: " + candidate)
		}
		img, decodeErr := decodeImage(path)
		if decodeErr != nil {
			return nil, himindjsonrpc.InvalidParams("decode candidate: " + decodeErr.Error())
		}
		bestScore, meanScore := 0.0, 0.0
		bestReference := ""
		for index, reference := range references {
			score := imageSimilarity(reference, img)
			meanScore += score
			if score > bestScore {
				bestScore = score
				bestReference = referencePaths[index]
			}
		}
		meanScore /= float64(len(references))
		ok := bestScore >= threshold
		candidates = append(candidates, candidateConsistency{
			Path: workspaceRelative(root, path), BestReference: workspaceRelative(root, bestReference),
			BestScore: round4(bestScore), MeanScore: round4(meanScore), OK: ok,
		})
		if bestScore < lowest {
			lowest = bestScore
		}
		if !ok {
			blockers = append(blockers, issue("REFERENCE_DRIFT", "consistency",
				"候选图与参考包最佳相似度低于阈值: "+workspaceRelative(root, path),
				"换用参考一致的候选，或把确认可用的角度补进参考包"))
		} else if bestScore < threshold+0.08 {
			warnings = append(warnings, issue("REFERENCE_NEAR_THRESHOLD", "consistency",
				"候选图与参考包接近阈值: "+workspaceRelative(root, path),
				"发布前人工复核角色/主体细节"))
		}
	}
	state := "consistent"
	if len(blockers) > 0 {
		state = "drift"
	}
	return map[string]any{
		"state": state, "ok": len(blockers) == 0, "threshold": threshold,
		"pack":       map[string]any{"id": manifest.ID, "version": manifest.Version, "references": len(referencePaths)},
		"candidates": candidates, "lowest_score": round4(lowest),
		"blockers": blockers, "warnings": warnings,
		"note": "以参考包为基准的最佳匹配相似度；确定性结构+颜色指标，不是人脸识别",
	}, nil
}
