package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeMarketCatalog(t *testing.T, root, packagePath string, status string) string {
	t.Helper()
	catalogPath := filepath.Join(root, "catalog", "asset-packs.json")
	if err := os.MkdirAll(filepath.Dir(catalogPath), 0755); err != nil {
		t.Fatalf("mkdir catalog: %v", err)
	}
	packageInfo, err := os.Stat(packagePath)
	if err != nil {
		t.Fatalf("stat package: %v", err)
	}
	hash, err := fileSHA256(packagePath)
	if err != nil {
		t.Fatalf("hash package: %v", err)
	}
	catalog := assetCatalog{
		Schema: assetCatalogSchema,
		Name:   "Test asset catalog",
		Assets: []assetCatalogEntry{{
			ID:          "demo",
			Name:        "Demo",
			Version:     "1.0.0",
			FileName:    "demo-1.0.0.hmpack",
			FileSize:    packageInfo.Size(),
			SHA256:      hash,
			DownloadURL: "file:///" + filepath.ToSlash(packagePath),
			License:     "MIT",
			Status:      status,
		}},
	}
	raw, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	if err := os.WriteFile(catalogPath, raw, 0644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return catalogPath
}

func TestAssetMarketListAndInstall(t *testing.T) {
	root := t.TempDir()
	writeAssetPackFixture(t, root)
	packageResult, rpcErr := assetPackPackage(requestInput{
		WorkspaceRoot: root,
		PackID:        "demo",
		Version:       "1.0.0",
		Output:        "dist/original.hmpack",
	})
	if rpcErr != nil {
		t.Fatalf("asset.pack: %s", rpcErr.Message)
	}
	packagePath := packageResult.(map[string]any)["package_path"].(string)
	catalogPath := writeMarketCatalog(t, root, packagePath, "published")
	installRoot := filepath.Join(t.TempDir(), "asset-packs")
	t.Setenv("HIMIND_ASSET_CATALOG", catalogPath)
	t.Setenv("HIMIND_ASSET_PACK_HOME", installRoot)

	listResult, rpcErr := assetMarketList(requestInput{WorkspaceRoot: root})
	if rpcErr != nil {
		t.Fatalf("asset.market.list: %s", rpcErr.Message)
	}
	items := listResult.(map[string]any)["assets"].([]assetCatalogEntry)
	if len(items) != 1 || items[0].Installed || items[0].UpdateAvailable {
		t.Fatalf("unexpected initial market state: %+v", items)
	}

	installResult, rpcErr := assetMarketInstall(requestInput{WorkspaceRoot: root, PackID: "demo", Version: "1.0.0"})
	if rpcErr != nil {
		t.Fatalf("asset.market.install: %s", rpcErr.Message)
	}
	if installResult.(map[string]any)["state"] != "installed" {
		t.Fatalf("unexpected market install result: %#v", installResult)
	}

	listResult, rpcErr = assetMarketList(requestInput{WorkspaceRoot: root})
	if rpcErr != nil {
		t.Fatalf("asset.market.list after install: %s", rpcErr.Message)
	}
	items = listResult.(map[string]any)["assets"].([]assetCatalogEntry)
	if !items[0].Installed || items[0].InstalledVersion != "1.0.0" || items[0].UpdateAvailable {
		t.Fatalf("installed market state not merged: %+v", items[0])
	}
}

func TestAssetMarketRejectsPreparedEntry(t *testing.T) {
	root := t.TempDir()
	writeAssetPackFixture(t, root)
	packageResult, rpcErr := assetPackPackage(requestInput{
		WorkspaceRoot: root,
		PackID:        "demo",
		Version:       "1.0.0",
		Output:        "dist/original.hmpack",
	})
	if rpcErr != nil {
		t.Fatalf("asset.pack: %s", rpcErr.Message)
	}
	packagePath := packageResult.(map[string]any)["package_path"].(string)
	catalogPath := writeMarketCatalog(t, root, packagePath, "prepared")
	t.Setenv("HIMIND_ASSET_CATALOG", catalogPath)
	t.Setenv("HIMIND_ASSET_PACK_HOME", filepath.Join(t.TempDir(), "asset-packs"))
	if _, rpcErr := assetMarketInstall(requestInput{WorkspaceRoot: root, PackID: "demo", Version: "1.0.0"}); rpcErr == nil {
		t.Fatal("prepared market entry should not be installable")
	}
}

func TestVersionComparison(t *testing.T) {
	if !versionLess("1.0.0", "1.0.1") || versionLess("1.0.1", "1.0.0") {
		t.Fatal("version comparison is incorrect")
	}
	if highestVersion([]string{"1.0.0", "1.2.0", "1.1.5"}) != "1.2.0" {
		t.Fatal("highestVersion picked the wrong version")
	}
}
