package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

const assetCatalogSchema = "himind.asset-catalog.v1"

type assetCatalog struct {
	Schema        string              `json:"schema"`
	Name          string              `json:"name,omitempty"`
	Version       string              `json:"version,omitempty"`
	Repository    string              `json:"repository,omitempty"`
	ReleasePrefix string              `json:"release_prefix,omitempty"`
	Assets        []assetCatalogEntry `json:"assets"`
}

type assetCatalogEntry struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	Version          string   `json:"version"`
	FileName         string   `json:"file_name"`
	FileSize         int64    `json:"file_size"`
	SHA256           string   `json:"sha256"`
	DownloadURL      string   `json:"download_url"`
	License          string   `json:"license,omitempty"`
	Status           string   `json:"status,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Installed        bool     `json:"installed,omitempty"`
	InstalledVersion string   `json:"installed_version,omitempty"`
	UpdateAvailable  bool     `json:"update_available,omitempty"`
}

func assetCatalogCandidates(workspaceRoot string) []string {
	candidates := []string{}
	if custom := strings.TrimSpace(os.Getenv("HIMIND_ASSET_CATALOG")); custom != "" {
		candidates = append(candidates, custom)
	}
	if workspaceRoot != "" {
		candidates = append(candidates, filepath.Join(workspaceRoot, "catalog", "asset-packs.json"))
	}
	repoRoot := filepath.Clean(filepath.Join(pluginRoot(), "..", "..", ".."))
	candidates = append(candidates, filepath.Join(repoRoot, "catalog", "asset-packs.json"))
	if installRoot, err := assetPackInstallRoot(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(installRoot), "asset-packs-catalog.json"))
		candidates = append(candidates, filepath.Join(installRoot, "catalog.json"))
	}
	return candidates
}

func loadAssetCatalog(workspaceRoot string) (assetCatalog, error) {
	for _, path := range assetCatalogCandidates(workspaceRoot) {
		if !fileExists(path) {
			continue
		}
		var catalog assetCatalog
		if err := readJSON(path, &catalog); err != nil {
			return catalog, fmt.Errorf("read asset catalog %s: %w", path, err)
		}
		if catalog.Schema != assetCatalogSchema {
			return catalog, fmt.Errorf("asset catalog schema must be %s", assetCatalogSchema)
		}
		if len(catalog.Assets) == 0 {
			return catalog, fmt.Errorf("asset catalog has no assets")
		}
		return catalog, nil
	}
	return assetCatalog{}, fmt.Errorf("asset catalog not found")
}

func assetMarketList(in requestInput) (any, *himindjsonrpc.Error) {
	root := ""
	if strings.TrimSpace(in.WorkspaceRoot) != "" {
		var err error
		root, err = workspace(in.WorkspaceRoot)
		if err != nil {
			return nil, himindjsonrpc.InvalidParams(err.Error())
		}
	}
	catalog, err := loadAssetCatalog(root)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	versionsByID := map[string][]string{}
	for key := range installedAssetPackPaths() {
		parts := strings.SplitN(key, "@", 2)
		if len(parts) != 2 {
			continue
		}
		versionsByID[parts[0]] = append(versionsByID[parts[0]], parts[1])
	}
	for i := range catalog.Assets {
		item := &catalog.Assets[i]
		installedVersion := highestVersion(versionsByID[item.ID])
		item.Installed = installedVersion != ""
		item.InstalledVersion = installedVersion
		item.UpdateAvailable = installedVersion != "" && versionLess(installedVersion, item.Version)
	}
	return map[string]any{
		"schema":     catalog.Schema,
		"name":       catalog.Name,
		"version":    catalog.Version,
		"repository": catalog.Repository,
		"assets":     catalog.Assets,
		"total":      len(catalog.Assets),
	}, nil
}

func assetMarketInstall(in requestInput) (any, *himindjsonrpc.Error) {
	root, err := workspace(in.WorkspaceRoot)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	catalog, err := loadAssetCatalog(root)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	entry, ok := findCatalogEntry(catalog, strings.TrimSpace(in.PackID), strings.TrimSpace(in.Version))
	if !ok {
		return nil, himindjsonrpc.InvalidParams("asset pack not found in catalog")
	}
	if entry.Status != "published" {
		return nil, himindjsonrpc.InvalidParams("asset pack is not published: " + entry.Status)
	}
	if strings.TrimSpace(entry.DownloadURL) == "" {
		return nil, himindjsonrpc.InvalidParams("asset catalog entry has no download_url")
	}
	target, err := workspaceFile(root, filepath.Join("dist", "asset-packages", entry.FileName))
	if err != nil {
		return nil, himindjsonrpc.InvalidParams("download target: " + err.Error())
	}
	if err := downloadAssetPackage(entry.DownloadURL, target); err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	hash, err := fileSHA256(target)
	if err != nil {
		return nil, himindjsonrpc.InternalError(err.Error())
	}
	if entry.FileSize > 0 && info.Size() != entry.FileSize {
		return nil, himindjsonrpc.InvalidParams("downloaded asset size mismatch")
	}
	if !strings.EqualFold(hash, entry.SHA256) {
		return nil, himindjsonrpc.InvalidParams("downloaded asset sha256 mismatch")
	}
	installed, err := installAssetPackage(target, in.Force)
	if err != nil {
		return nil, himindjsonrpc.InvalidParams(err.Error())
	}
	return map[string]any{
		"state":          "installed",
		"pack_id":        entry.ID,
		"version":        entry.Version,
		"package_path":   target,
		"package_sha256": hash,
		"install":        installed,
	}, nil
}

func findCatalogEntry(catalog assetCatalog, id, version string) (assetCatalogEntry, bool) {
	for _, entry := range catalog.Assets {
		if entry.ID == id && (version == "" || entry.Version == version) {
			return entry, true
		}
	}
	return assetCatalogEntry{}, false
}

func highestVersion(versions []string) string {
	best := ""
	for _, version := range versions {
		if best == "" || versionLess(best, version) {
			best = version
		}
	}
	return best
}

func versionLess(a, b string) bool {
	aParts := versionParts(a)
	bParts := versionParts(b)
	for i := 0; i < 3; i++ {
		if aParts[i] != bParts[i] {
			return aParts[i] < bParts[i]
		}
	}
	return false
}

func versionParts(value string) [3]int {
	var parts [3]int
	for i, raw := range strings.SplitN(value, ".", 3) {
		if i >= 3 {
			break
		}
		parts[i], _ = strconv.Atoi(raw)
	}
	return parts
}

func downloadAssetPackage(rawURL, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".download-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()
	switch {
	case strings.HasPrefix(rawURL, "file://"):
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		source := parsed.Path
		if runtime.GOOS == "windows" && strings.HasPrefix(source, "/") && len(source) > 2 && source[2] == ':' {
			source = source[1:]
		}
		input, err := os.Open(filepath.FromSlash(source))
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(temp, input)
		_ = input.Close()
		if copyErr != nil {
			return copyErr
		}
	case strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://"):
		client := &http.Client{Timeout: 120 * time.Second}
		response, err := client.Get(rawURL)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("download asset: HTTP %d", response.StatusCode)
		}
		if _, err := io.Copy(temp, response.Body); err != nil {
			return err
		}
	case filepath.IsAbs(rawURL):
		input, err := os.Open(rawURL)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(temp, input)
		_ = input.Close()
		if copyErr != nil {
			return copyErr
		}
	default:
		return fmt.Errorf("unsupported asset download url: %s", rawURL)
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, target)
}
