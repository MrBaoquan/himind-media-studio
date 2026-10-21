package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"himind-plugin/media-visual/internal/himindjsonrpc"
)

type assetResource struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Path   string `json:"path"`
	Role   string `json:"role,omitempty"`
	MIME   string `json:"mime,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Source string `json:"source,omitempty"`
}

type assetPackManifest struct {
	Schema      string          `json:"schema"`
	ID          string          `json:"id"`
	Version     string          `json:"version"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	License     string          `json:"license,omitempty"`
	LicenseURL  string          `json:"license_url,omitempty"`
	Attribution string          `json:"attribution,omitempty"`
	Resources   []assetResource `json:"resources"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
	Provenance  map[string]any  `json:"provenance,omitempty"`
	Permissions map[string]any  `json:"permissions,omitempty"`
	Consumers   []string        `json:"consumers,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

func referencePackCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	packID := strings.TrimSpace(in.PackID)
	if !idPattern.MatchString(packID) {
		return nil, himindjsonrpc.InvalidParams("pack_id is required and must be lowercase")
	}
	version := firstNonEmpty(in.Version, "1.0.0")
	if !idPattern.MatchString(version) {
		return nil, himindjsonrpc.InvalidParams("version is invalid")
	}
	if len(in.ReferenceImages) == 0 {
		return nil, himindjsonrpc.InvalidParams("reference_images is required")
	}
	packRoot := filepath.Join(root, "asset-packs", packID, version)
	resources := make([]assetResource, 0, len(in.ReferenceImages))
	for index, reference := range in.ReferenceImages {
		source, err := workspaceFile(root, reference)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams("reference_images: " + err.Error())
		}
		info, err := os.Stat(source)
		if err != nil || info.IsDir() {
			return nil, himindjsonrpc.InvalidParams("reference image not found: " + reference)
		}
		base := filepath.Base(source)
		destRel := filepath.ToSlash(filepath.Join("resources", fmt.Sprintf("%02d-%s", index+1, base)))
		dest := filepath.Join(packRoot, filepath.FromSlash(destRel))
		if err := copyFile(source, dest); err != nil {
			return nil, himindjsonrpc.InternalError(err.Error())
		}
		hash, _ := fileSHA256(dest)
		width, height, mime := imageMetadata(dest)
		if mime == "" {
			return nil, himindjsonrpc.InvalidParams("reference image cannot be decoded: " + reference)
		}
		resources = append(resources, assetResource{
			ID: fmt.Sprintf("reference-%02d", index+1), Type: "image", Path: destRel,
			Role: firstNonEmpty(in.Role, "character-reference"), MIME: mime,
			Width: width, Height: height, SHA256: hash, Size: info.Size(), Source: reference,
		})
	}
	license := firstNonEmpty(in.License, "unknown")
	tags := append([]string{"reference", "character", "image"}, in.Tags...)
	manifest := assetPackManifest{
		Schema: "himind.asset-pack.v1", ID: packID, Version: version,
		Name: firstNonEmpty(in.Name, packID), Description: in.Description,
		License: license, LicenseURL: in.LicenseURL,
		Attribution: firstNonEmpty(in.LicenseURL, license),
		Resources:   resources,
		Metadata: map[string]any{
			"domains": []string{"image", "reference"}, "provider": "media-visual.reference",
			"character": firstNonEmpty(in.Character, in.Name), "prompt": in.Prompt,
			"negative_prompt": in.NegativePrompt, "seed": in.Seed,
			"commercial_use": in.CommercialUse, "created_at": now(),
		},
		Provenance: map[string]any{
			"source": "user-provided", "license": license, "license_url": in.LicenseURL,
			"commercial_use": in.CommercialUse, "resources": len(resources),
		},
		Permissions: map[string]any{"filesystem": "workspace", "process": []string{}, "network": map[string]any{"mode": "none", "allow": []string{}}},
		Consumers:   []string{"com.himind.media-visual", "com.himind.media-workbench"},
		Tags:        tags,
	}
	if err := writeJSON(filepath.Join(packRoot, "asset-pack.json"), manifest); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	return map[string]any{
		"state": "ready", "pack": manifest, "path": workspaceRelative(root, packRoot),
		"resource_count": len(resources),
		"next":           "把 pack_id@version 写入 shot.reference_assets，并在生成前调用 visual.reference.validate",
	}, nil
}

func referenceValidateCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	packPath := strings.TrimSpace(in.PackPath)
	if packPath == "" {
		return nil, himindjsonrpc.InvalidParams("pack_path is required")
	}
	resolved, err := workspaceFile(root, packPath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("pack_path: " + err.Error())
	}
	info, statErr := os.Stat(resolved)
	if statErr != nil || !info.IsDir() {
		return nil, himindjsonrpc.InvalidParams("pack_path must be a directory")
	}
	var manifest assetPackManifest
	if err := readJSON(filepath.Join(resolved, "asset-pack.json"), &manifest); err != nil {
		return nil, himindjsonrpc.InvalidParams("read asset-pack.json: " + err.Error())
	}
	blockers := make([]map[string]any, 0)
	warnings := make([]map[string]any, 0)
	if manifest.Schema != "" && manifest.Schema != "himind.asset-pack.v1" {
		blockers = append(blockers, issue("REFERENCE_SCHEMA_UNSUPPORTED", "reference", "不支持的资产包 schema: "+manifest.Schema, "使用 himind.asset-pack.v1"))
	}
	if len(manifest.Resources) == 0 {
		blockers = append(blockers, issue("REFERENCE_EMPTY", "reference", "资产包没有资源", "至少放一张角色参考图"))
	}
	for _, resource := range manifest.Resources {
		path := filepath.Join(resolved, filepath.FromSlash(resource.Path))
		hash, hashErr := fileSHA256(path)
		if hashErr != nil {
			blockers = append(blockers, issue("REFERENCE_RESOURCE_MISSING", "reference", "资源不存在: "+resource.Path, "重新打包 reference pack"))
			continue
		}
		if !strings.EqualFold(hash, resource.SHA256) {
			blockers = append(blockers, issue("REFERENCE_HASH_MISMATCH", "reference", "资源哈希不一致: "+resource.Path, "重新打包 reference pack"))
		}
	}
	for _, reference := range in.ReferenceAssets {
		path, pathErr := workspaceFile(root, reference)
		if pathErr != nil {
			blockers = append(blockers, issue("REFERENCE_ASSET_INVALID", "reference", "reference_assets 越界: "+reference, "把参考资产放在工作区内"))
			continue
		}
		if _, statErr := os.Stat(path); statErr != nil {
			blockers = append(blockers, issue("REFERENCE_ASSET_MISSING", "reference", "reference_assets 不存在: "+reference, "补齐参考图或重新打包"))
		}
	}
	if strings.EqualFold(firstNonEmpty(manifest.License, "unknown"), "unknown") {
		warnings = append(warnings, issue("REFERENCE_LICENSE_UNKNOWN", "license", "参考图许可证未知，商业发布前需补齐", "在 asset-pack.json 的 license 字段填写来源许可证"))
	}
	state := "ready"
	if len(blockers) > 0 {
		state = "blocked"
	}
	return map[string]any{"state": state, "ok": len(blockers) == 0, "pack": manifest, "blockers": blockers, "warnings": warnings}, nil
}

func imageCompareCapability(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	referencePath, err := workspaceFile(root, in.ReferencePath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("reference_path: " + err.Error())
	}
	candidatePath, err := workspaceFile(root, in.CandidatePath)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("candidate_path: " + err.Error())
	}
	reference, refErr := decodeImage(referencePath)
	candidate, candErr := decodeImage(candidatePath)
	if refErr != nil || candErr != nil {
		return nil, himindjsonrpc.InvalidParams(fmt.Sprintf("decode image: %v / %v", refErr, candErr))
	}
	score := imageSimilarity(reference, candidate)
	threshold := in.Threshold
	if threshold <= 0 {
		threshold = 0.75
	}
	return map[string]any{
		"state": "ready", "ok": score >= threshold, "score": math.Round(score*10000) / 10000,
		"threshold": threshold,
		"reference": map[string]any{"path": workspaceRelative(root, referencePath), "width": reference.Bounds().Dx(), "height": reference.Bounds().Dy()},
		"candidate": map[string]any{"path": workspaceRelative(root, candidatePath), "width": candidate.Bounds().Dx(), "height": candidate.Bounds().Dy()},
		"note":      "确定性结构+颜色相似度，不是人脸或角色识别；用于候选筛选与漂移预警",
	}, nil
}

func imageMetadata(path string) (int, int, string) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, ""
	}
	defer file.Close()
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, ""
	}
	mime := "image/" + format
	if format == "jpg" {
		mime = "image/jpeg"
	}
	return config.Width, config.Height, mime
}

func decodeImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}

// imageSimilarity 组合两项确定性指标：
//   - 结构：32x32 灰度零均值归一化互相关（NCC），不受整体明暗影响；
//   - 颜色：RGB 平均绝对差，防止「结构相同但配色跑偏」。
//
// 这不是人脸/角色识别，只用于候选筛选与漂移预警。
func imageSimilarity(left, right image.Image) float64 {
	const size = 32
	leftPixels := resizeRGB(left, size, size)
	rightPixels := resizeRGB(right, size, size)
	count := len(leftPixels) / 3
	leftLuma := make([]float64, count)
	rightLuma := make([]float64, count)
	leftMean, rightMean := 0.0, 0.0
	colorDiff := 0.0
	for index := 0; index < count; index++ {
		lr, lg, lb := leftPixels[index*3], leftPixels[index*3+1], leftPixels[index*3+2]
		rr, rg, rb := rightPixels[index*3], rightPixels[index*3+1], rightPixels[index*3+2]
		leftLuma[index] = 0.299*float64(lr) + 0.587*float64(lg) + 0.114*float64(lb)
		rightLuma[index] = 0.299*float64(rr) + 0.587*float64(rg) + 0.114*float64(rb)
		leftMean += leftLuma[index]
		rightMean += rightLuma[index]
		colorDiff += math.Abs(float64(lr)-float64(rr)) + math.Abs(float64(lg)-float64(rg)) + math.Abs(float64(lb)-float64(rb))
	}
	leftMean /= float64(count)
	rightMean /= float64(count)
	numerator, leftEnergy, rightEnergy := 0.0, 0.0, 0.0
	for index := 0; index < count; index++ {
		lc := leftLuma[index] - leftMean
		rc := rightLuma[index] - rightMean
		numerator += lc * rc
		leftEnergy += lc * lc
		rightEnergy += rc * rc
	}
	structure := 0.0
	if leftEnergy > 0 && rightEnergy > 0 {
		structure = numerator / math.Sqrt(leftEnergy*rightEnergy)
	}
	if structure < 0 {
		structure = 0
	}
	colorSimilarity := 1 - colorDiff/(float64(count)*3*255)
	if colorSimilarity < 0 {
		colorSimilarity = 0
	}
	similarity := 0.6*structure + 0.4*colorSimilarity
	if similarity < 0 {
		similarity = 0
	}
	if similarity > 1 {
		similarity = 1
	}
	return similarity
}

func resizeRGB(img image.Image, width, height int) []uint8 {
	bounds := img.Bounds()
	out := make([]uint8, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := bounds.Min.X + int(float64(x)*float64(bounds.Dx())/float64(width))
			sy := bounds.Min.Y + int(float64(y)*float64(bounds.Dy())/float64(height))
			r, g, b, _ := img.At(sx, sy).RGBA()
			offset := (y*width + x) * 3
			out[offset] = uint8(r >> 8)
			out[offset+1] = uint8(g >> 8)
			out[offset+2] = uint8(b >> 8)
		}
	}
	return out
}

func workspaceRelative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
