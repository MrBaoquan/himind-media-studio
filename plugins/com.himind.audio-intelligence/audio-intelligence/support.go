package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	metadataDir = ".himind-media"
	creativeDir = "creatives"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

type creativePackage struct {
	ID     string         `json:"id"`
	Script map[string]any `json:"script"`
}

func workspace(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("workspace_root is required")
	}
	root, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("workspace_root must be an existing directory")
	}
	return root, nil
}

func workspaceFile(root, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("file path is required")
	}
	path := input
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("file path must stay inside workspace_root")
	}
	return path, nil
}

func creativeRoot(root, id string) string {
	return filepath.Join(root, metadataDir, creativeDir, id)
}

func loadCreative(root, id string) (creativePackage, error) {
	if !strings.HasPrefix(id, "creative-") || !idPattern.MatchString(id) {
		return creativePackage{}, fmt.Errorf("invalid creative_id")
	}
	var item creativePackage
	if err := readJSON(filepath.Join(creativeRoot(root, id), "creative.json"), &item); err != nil {
		return creativePackage{}, fmt.Errorf("creative not found: %s", id)
	}
	if item.ID == "" {
		item.ID = id
	}
	if item.Script == nil {
		item.Script = map[string]any{}
	}
	return item, nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func asString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return fmt.Sprint(value)
}

func asMap(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func asMapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return []map[string]any{}
	}
	items := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		if object, ok := entry.(map[string]any); ok {
			items = append(items, object)
		}
	}
	return items
}

func numberFloat(value any) float64 {
	switch number := value.(type) {
	case int:
		return float64(number)
	case int64:
		return float64(number)
	case float64:
		return number
	case json.Number:
		parsed, _ := number.Float64()
		return parsed
	case string:
		var parsed float64
		_, _ = fmt.Sscanf(strings.TrimSpace(number), "%f", &parsed)
		return parsed
	default:
		return 0
	}
}

func issue(code, stage, message, remediation string) map[string]any {
	return map[string]any{"code": code, "stage": stage, "message": message, "remediation": remediation}
}
