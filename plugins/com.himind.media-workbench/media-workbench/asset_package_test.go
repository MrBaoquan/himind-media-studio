package main

import (
	"path/filepath"
	"testing"
)

func TestAssetPackPackageInstallScanAndUninstall(t *testing.T) {
	root := t.TempDir()
	writeAssetPackFixture(t, root)
	installRoot := filepath.Join(t.TempDir(), "asset-packs")
	t.Setenv("HIMIND_ASSET_PACK_HOME", installRoot)

	packageResult, rpcErr := assetPackPackage(requestInput{
		WorkspaceRoot: root,
		PackID:        "demo",
		Version:       "1.0.0",
		Output:        "dist/demo-1.0.0.hmpack",
	})
	if rpcErr != nil {
		t.Fatalf("asset.pack: %s", rpcErr.Message)
	}
	packaged := packageResult.(map[string]any)
	if !packaged["valid"].(bool) || packaged["state"] != "packaged" {
		t.Fatalf("unexpected package result: %#v", packaged)
	}
	packagePath := packaged["package_path"].(string)
	if !fileExists(packagePath) {
		t.Fatalf("package was not written: %s", packagePath)
	}
	if packaged["package_sha256"] == "" {
		t.Fatal("package sha256 is empty")
	}
	secondResult, rpcErr := assetPackPackage(requestInput{
		WorkspaceRoot: root,
		PackID:        "demo",
		Version:       "1.0.0",
		Output:        "dist/demo-copy.hmpack",
	})
	if rpcErr != nil {
		t.Fatalf("asset.pack second run: %s", rpcErr.Message)
	}
	if secondResult.(map[string]any)["package_sha256"] != packaged["package_sha256"] {
		t.Fatal("asset package is not deterministic")
	}

	installResult, rpcErr := assetPackInstall(requestInput{PackagePath: packagePath})
	if rpcErr != nil {
		t.Fatalf("asset.install: %s", rpcErr.Message)
	}
	installed := installResult.(map[string]any)
	if installed["state"] != "installed" {
		t.Fatalf("unexpected install result: %#v", installed)
	}
	installedPath := installed["installed_path"].(string)
	if !fileExists(filepath.Join(installedPath, "asset-pack.json")) || !fileExists(filepath.Join(installedPath, "install.json")) {
		t.Fatalf("installed pack is incomplete: %s", installedPath)
	}
	installedList, rpcErr := assetList(requestInput{WorkspaceRoot: root})
	if rpcErr != nil {
		t.Fatalf("asset.list with installed metadata: %s", rpcErr.Message)
	}
	markedInstalled := false
	for _, pack := range installedList.(map[string]any)["items"].([]assetPack) {
		if pack.ID == "demo" && pack.Version == "1.0.0" {
			markedInstalled = true
			if !pack.Installed || pack.InstallPath == "" {
				t.Fatalf("installed metadata missing: %+v", pack)
			}
		}
	}
	if !markedInstalled {
		t.Fatal("demo pack missing from asset.list")
	}

	// Use a workspace without the source pack so discovery must come from the
	// versioned installed directory.
	otherWorkspace := t.TempDir()
	listResult, rpcErr := assetList(requestInput{WorkspaceRoot: otherWorkspace})
	if rpcErr != nil {
		t.Fatalf("asset.list: %s", rpcErr.Message)
	}
	foundInstalled := false
	for _, pack := range listResult.(map[string]any)["items"].([]assetPack) {
		if pack.ID == "demo" && pack.Version == "1.0.0" && pack.Source == "installed" {
			foundInstalled = true
			if !filepath.IsAbs(pack.Path) {
				t.Fatalf("installed pack path must be absolute: %s", pack.Path)
			}
		}
	}
	if !foundInstalled {
		t.Fatalf("installed demo pack was not discovered from %s", installRoot)
	}

	if _, rpcErr := assetPackInstall(requestInput{PackagePath: packagePath}); rpcErr == nil {
		t.Fatal("reinstalling without force should fail")
	}
	if _, rpcErr := assetPackInstall(requestInput{PackagePath: packagePath, Force: true}); rpcErr != nil {
		t.Fatalf("force reinstall: %s", rpcErr.Message)
	}

	if _, rpcErr := assetPackUninstall(requestInput{PackID: "demo", Version: "1.0.0"}); rpcErr != nil {
		t.Fatalf("asset.uninstall: %s", rpcErr.Message)
	}
	listAfter, rpcErr := assetList(requestInput{WorkspaceRoot: otherWorkspace})
	if rpcErr != nil {
		t.Fatalf("asset.list after uninstall: %s", rpcErr.Message)
	}
	for _, pack := range listAfter.(map[string]any)["items"].([]assetPack) {
		if pack.ID == "demo" && pack.Source == "installed" {
			t.Fatalf("demo pack still installed: %+v", pack)
		}
	}
}

func TestSafeAssetPackagePath(t *testing.T) {
	bad := []string{"", "..", "../escape", "/absolute", `C:\\escape`}
	for _, value := range bad {
		if _, err := safeAssetPackagePath(value); err == nil {
			t.Fatalf("expected unsafe path to be rejected: %q", value)
		}
	}
	good := map[string]string{"template/reel.js": "template/reel.js", "preview\\sheet.jpg": "preview/sheet.jpg"}
	for input, expected := range good {
		actual, err := safeAssetPackagePath(input)
		if err != nil || actual != expected {
			t.Fatalf("safe path %q => %q, %v; want %q", input, actual, err, expected)
		}
	}
}
